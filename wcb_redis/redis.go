package wcb_redis

import (
	"fmt"
	"github.com/washCutBlow/wcb-chaomai/wcb_pool"
)

type Client struct {
	opt *Options
	connPool wcb_pool.Pooler
}

//  新建一个redis客户端，该客户端附带连接池
func NewClient(opt *Options) *Client  {
	opt.init()
	c := &Client{
		opt: opt,
		connPool: newConnPool(opt),
	}

	return c
}

//  新建一个连接池
func newConnPool(opt *Options) *wcb_pool.ConnPool {
	return wcb_pool.NewConnPool(&wcb_pool.Options{
		Dialer: opt.Dialer,
		PoolSize:opt.PoolSize,
	})
}

// 发送命令
func (c *Client) SendCommand(cmd string) error  {
	//  获取一个可用连接
	cn,err := c.getConn()
	if err != nil {
		fmt.Println("Client_SendCommand获取可用连接错误:",err)
		return err
	}
	// 执行命令（这里直接使用redis的 resp原装协议）
	cn.WithWrite(cmd)
	// 释放连接，即把连接归还给连接池
	c.releaseConn(cn)
	return nil
}

// 获取一个可用连接
func (c *Client) getConn() (*wcb_pool.Conn, error){
	cn,err := c.connPool.Get()
	if err != nil {
		fmt.Println("Client_getConn获取可用连接错误",err)
		return nil,err
	}
	return cn,nil
}

// 释放连接,归还给连接池
func (c *Client) releaseConn(conn *wcb_pool.Conn)  {
	c.connPool.Put(conn)
}