package wcb_pool

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// 定时触发事件，用来设置从连接池获取连接的超时事件，使用对象池的方式
var timers = sync.Pool{
	New: func() interface{} {
		t := time.NewTimer(time.Hour)
		// 执行timer的Stop() 不会关闭channel C；但是会把Timer从runtime中删除
		// 可以使用Reset代替
		// 这里之所以要立即stop，是为了减少gc压力，具体可参考https://studygolang.com/articles/9289
		// stop了之后并没有什么负面影


		t.Stop()
		return t
	},
}




var ErrorClosed = errors.New("redis: client is closed")
var ErrPoolTimeout = errors.New("redis: connection pool timeout")

type Pooler interface {
	NewConn() (*Conn, error)
	CloseConn(*Conn) error

	Get() (*Conn,error)
	Put(*Conn)
	Remove(*Conn)

	Len() int
	IdleLen() int

}

type Options struct {
	// 和服务器建立连接的方法
	Dialer func() (net.Conn, error)
	// 连接池上限
	PoolSize int
	// 最少的空闲连接数
	MinIdleConns int

	//连接最大持续时间(连接now-最近使用时间超过这个时间，认为连接时一个太陈旧的连接)
	MaxConnAge         time.Duration
	// 获取连接池的一个连接超时时间
	PoolTimeout        time.Duration
	// 一个连接多久未使用认为该连接是一个空闲连接
	IdleTimeout        time.Duration
}

type ConnPool struct {
	opt *Options

	//建立连接错误次数(原子操作) 如果建立连接错误次数超过配置的opt.Poolsize则会单独生成协程无限重试
	dialErrorsNum uint32 // atomic

	lastDialErrorMu sync.RWMutex
	lastDialError   error

	//控制同时连接数达到上限，再多就需要等待
	queue chan struct{}

	//锁
	connsMu      sync.Mutex

	//当前连接池内连接的个数
	poolSize     int

	// 所有连接存放数组,包括在线程池中的连接和不在线程池中的连接
	conns        []*Conn
	//空闲连接存放数组
	idleConns    []*Conn
	//空闲连接个数
	idleConnsLen int

	_closed uint32 // atomic
}

// 新建连接池
func NewConnPool(opt *Options) *ConnPool {
	p := &ConnPool{
		opt:opt,
		queue: make(chan struct{}, opt.PoolSize),
		conns: make([]*Conn,0,opt.PoolSize),
		idleConns:make([]*Conn,0,opt.PoolSize),
	}
	// 在初始化时创建配置的最小连接数
	for i:=0;i<opt.MinIdleConns;i++ {
		p.checkMinIdleConns()
	}
	return p
}

// 新建一个不在连接池内的连接
func (p *ConnPool) NewConn() (*Conn, error)  {
	return p._NewCoon(false)
}

func (p *ConnPool) CloseConn(*Conn) error {
	return nil
}

// 从连接池获取一个可用连接
func (p *ConnPool) Get() (*Conn, error)  {
	if p.closed() {
		fmt.Println("ConnPool_Get err:",ErrorClosed)
		return nil, ErrorClosed
	}
	err := p.waitTurn() //占用一个连接
	if err !=nil {
		return nil,err
	}

	for  {	//从连接池获取一个连接
		p.connsMu.Lock()
		cn := p.popIdle()
		p.connsMu.Unlock()
		if cn == nil {
			break //说明没有就跳出循环新建
		}
		//如果是一个陈旧的连接，就关闭连接并继续获取一个连接

		return cn,nil
	}


	newcn, err := p._NewCoon(true)
	if err != nil {
		p.freeTurn() // 如果出错释放这个连接占用名额
		return nil,err
	}

	return newcn,nil
}

func (p *ConnPool) Put(c *Conn)  {
	//如果该连接不属于连接池直接remove掉
	if !c.pooled {
		p.Remove(c)
		return
	}
	p.connsMu.Lock()
	p.idleConns = append(p.idleConns, c)
	//空闲连接数+1
	p.idleConnsLen++
	p.connsMu.Unlock()
	//占用的服务已结束
	p.freeTurn()
}

func (p *ConnPool) Remove(c *Conn) {
	p.removeConn(c)
	p.freeTurn()
	_ = p.closeConn(c)
}

func (p *ConnPool) Len() int  { //连接池存在的连接数
	p.connsMu.Lock()
	n := len(p.conns)
	p.connsMu.Unlock()
	return n
}

func (p *ConnPool) IdleLen() int {	// 连接池空闲连接数
	p.connsMu.Lock()
	n := len(p.idleConns)
	p.connsMu.Unlock()
	return n
}

func (p *ConnPool) removeConn(c *Conn)  {
	p.connsMu.Lock()
	for i, cn := range p.conns {
		if c == cn {
			p.conns = append(p.conns[:i], p.conns[i+1:]...)
			if c.pooled {
				p.poolSize--
				p.checkMinIdleConns()
			}
			break
		}
	}
	p.connsMu.Unlock()
}

func (p *ConnPool) closeConn(c *Conn) error {
	return c.Close()
}
// 检查空闲连接数，如果小于设定的最小空闲连接数，就创建新的连接
func (p *ConnPool) checkMinIdleConns()  {
	if p.opt.MinIdleConns == 0 {
		return
	}

	if p.poolSize < p.opt.PoolSize && p.idleConnsLen < p.opt.MinIdleConns {
		// 异步创建
		p.createIdleConn()
	}
}

// 创建一个空闲连接
func (p *ConnPool) createIdleConn()  {
	cn, err := p.newConn(true)
	if err != nil {
		fmt.Println("ConnPool_createIdleConn err: ",err)
		return
	}
	p.connsMu.Lock()
	p.conns = append(p.conns,cn)
	p.idleConns = append(p.idleConns,cn)
	p.poolSize++
	p.idleConnsLen++
	p.connsMu.Unlock()
}

// 等待自己的时机
func (p *ConnPool) waitTurn() error  {
	select {
	// 每次向线程池归还一个连接时，都会往queue中写入，这里就可以读出，那么返回nil，说明不需要等待直接就可以去获取一个连接
	// 如果没有，那么default就是等待超时时间，如果超时就获取失败
	case p.queue <- struct{}{}:
		return nil
	default:
		timer := timers.Get().(*time.Timer)
		// 设置超时时间
		timer.Reset(p.opt.PoolTimeout)

		select {
		case p.queue <- struct{}{}:
			if !timer.Stop() {
				timers.Put(timer)
			}
			return nil
		case <-timer.C: //超时
			timers.Put(timer)
			return ErrPoolTimeout

		}
	}
}

func (p *ConnPool) freeTurn()  {
	<-p.queue
}
func (p *ConnPool) getTurn()  {
	p.queue<- struct{}{}
}

// 弹出一个空闲连接
func (p *ConnPool) popIdle() *Conn {
	if len(p.idleConns) == 0 {
		return nil
	}
	idx := len(p.idleConns) - 1
	cn := p.idleConns[idx]
	p.idleConns = p.idleConns[:idx]
	p.idleConnsLen--
	return cn
}

// 新建一个连接并根据参数决定是否放入连接池
func (p *ConnPool) _NewCoon(pooled bool) (*Conn, error)  {
	cn,err := p.newConn(pooled)
	if err != nil {
		return nil, err
	}
	p.connsMu.Lock()
	p.conns = append(p.conns,cn)
	if pooled {
		if p.poolSize < p.opt.PoolSize {
			p.poolSize++
		} else {
			cn.pooled = false
		}
	}
	p.connsMu.Unlock()
	return cn,nil
}

// 判断连接池是否关闭
func (p *ConnPool) closed() bool  {
	return atomic.LoadUint32(&p._closed) == uint32(1)
}

// 创建一个带有连接池标志的连接
func (p *ConnPool) newConn(pooled bool) (*Conn, error) {
	if p.closed() {	// 如果连接池已经关闭,返回错误
		fmt.Println("ConnPool_newConn err:", ErrorClosed)
		return nil,ErrorClosed
	}
	c, err := p.opt.Dialer()
	if err != nil {
		/*这里可以对连接池做一些新建连接错误处理，比如错我达到一定限制，就另起一个协程一直异步重试*/
		return nil, err
	}
	// 这里仅仅是把原声conn包装一下，以方便记录该conn的使用时间等等，做进一步的处理
	cn := NewConn(c)
	cn.pooled = pooled
	return cn, nil
}


