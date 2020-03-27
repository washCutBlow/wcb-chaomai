package wcb_redis

import (
	"net"
	"time"
)

type Options struct {
	// 网络协议类型 tcp或者本地域unix
	NetWork string

	// 服务器地址 host:port格式
	Addr string

	// 连接函数
	Dialer  func() (net.Conn,error)

	// 连接超时时间
	DialTimeout time.Duration

	// 从连接池获取连接的等待时间，当连接池中的连接都在使用中，而又不满足新建连接的时候，就阻塞等待直到超时
	PoolTimeout time.Duration

	// 连接池中连接数
	PoolSize int

	// 失败重连次数，默认不进行重连操作
	Maxretries int

	//重连最小时间间隔
	MinRetryBackoff time.Duration

	//重连最大时间间隔
	MaxRetryBackoff time.Duration
}

func (opt *Options) init()  {
	if opt.NetWork == "" {	// 默认是tcp
		opt.NetWork = "tcp"
	}
	if opt.Addr == "" {
		opt.Addr = "localhost:6379"
	}

	if opt.DialTimeout == 0 { // 默认5s的超时时间
		opt.DialTimeout = 5*time.Second
	}

	if opt.PoolSize == 0 { // 默认连接池大小20
		opt.PoolSize = 20
	}

	if opt.PoolTimeout == 0 { //默认从连接池获取连接超时时间5s
		opt.PoolTimeout = 5*time.Second
	}

	if opt.Dialer == nil {
		opt.Dialer = func() (net.Conn, error) {
			netD := &net.Dialer{
				Timeout:opt.DialTimeout,
				KeepAlive:5*time.Minute, //长连接5分钟
			}
			return netD.Dial(opt.NetWork, opt.Addr)
		}
	}
}
