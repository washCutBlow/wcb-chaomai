package chaomai

import (
	"fmt"
)
import "github.com/gomodule/redigo/redis"


type Redis struct {
	Host string
	Port string
	PassWord string
	Conn redis.Conn
	Expire uint64

}

/*connect*/
func (r *Redis) GetConn() error {
	c,err := redis.Dial("tcp", r.Host+":"+string(r.Port))
	if err != nil {
		fmt.Println("GetConn err",err.Error())
		return err
	}

	if _,err := c.Do("AUTH",r.PassWord); err != nil {
		fmt.Println("auth failed", err.Error())
		return err
	}
	r.Conn = c
	return nil
}

/*close connection*/
func (r *Redis) Close()  {
	err := r.Conn.Close()
	if err != nil {
		fmt.Println("Close err:", err.Error())
	}
}

/* check key exists*/
func (r *Redis) KeyExists(key string) bool  {
	exist,err := redis.Bool(r.Conn.Do("EXISTS",key))

	if err != nil {
		fmt.Println("KeyExists err", err.Error())
	}
	return exist
}

/*set key*/
func (r *Redis) Set(key string, value interface{}) error {
	_,err := r.Conn.Do("SET",key,value)
	if err != nil {
		fmt.Println("Set err:", err.Error())
	}
	return nil
}

/*get key*/
func (r *Redis) Get(key string) string   {
	v, err := redis.String(r.Conn.Do("GET", key))

	if err != nil {
		fmt.Println("Get err:", err.Error())
	}

	return v
}

/*setnx*/
func (r *Redis) SetNX(k string,v interface{}) bool   {
	b,err := redis.Bool(r.Conn.Do("SETNX",k,v))

	if err != nil {
		fmt.Println("SetNX err:",err.Error())
	}

	return b
}

/*set key expire time*/
func (r *Redis) SetExpire(key string, sec uint64) bool  {
	b,err := redis.Bool(r.Conn.Do("EXPIRE",key,sec))

	if err != nil {
		fmt.Println("SetExpire err:",err.Error())
	}
	return b
}


func (r *Redis) Ttl(key string) int64  {
	n,err := redis.Int64(r.Conn.Do("TTL",key))

	if err != nil {
		fmt.Println("Ttl err:",err.Error())
	}
	return n
}


func (r *Redis) IncrBy(key string, value interface{}) int {
	n,err := redis.Int(r.Conn.Do("INCRBY",key,value))

	if err != nil {
		fmt.Println("IncrBy err",err.Error())
	}

	return n

}


func (r *Redis) DecrBy(key string, value interface{}) int {
	n,err := redis.Int(r.Conn.Do("DECRBY",key,value))

	if err != nil {
		fmt.Println("DecrBy err",err.Error())
	}

	return n
}