package wcb_pool

import (
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

type Conn struct {
	netConn net.Conn
	createdAt time.Time
	// 是否属于连接池的连接
	pooled bool
	// 使用时间(原子更新)
	usedAt atomic.Value
}

func NewConn(netConn net.Conn) *Conn  {
	cn := &Conn{
		netConn: netConn,
		createdAt: time.Now(),
	}

	return cn
}

func (c *Conn) SetUsedAt(tm time.Time)  {
	c.usedAt.Store(tm)
}

func (c *Conn) UsedAt() time.Time  {
	return c.usedAt.Load().(time.Time)
}

// 因为要更新连接的使用时间，所以这里要重写Write接口
func (c *Conn) Write(b []byte) (int, error)  {
	now := time.Now()
	// 更新使用时间
	c.SetUsedAt(now)
	return c.netConn.Write(b)
}

func (c *Conn) RemoveAddr() net.Addr  {
	return c.netConn.RemoteAddr()
}

func (c *Conn) Close() error  {
	return c.netConn.Close()
}

func (c *Conn) WithWrite(cmd string) {
	cmdArgv := strings.Fields(cmd) //空格分开
	// 拼接处redis协议格式
	protocolCmd := fmt.Sprintf("*%d\r\n", len(cmdArgv))

	for _,arg := range cmdArgv {
		protocolCmd += fmt.Sprintf("$%d\r\n",len(arg))
		protocolCmd += arg
		protocolCmd += "\r\n"
	}

	//打印出来redis服务器接受的命令协议
	fmt.Printf("%q\n", protocolCmd)

	n,err := c.netConn.Write([]byte(protocolCmd))
	fmt.Println(n)

	if err != nil {
		fmt.Println(err)
	}
}

