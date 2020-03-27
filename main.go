package main

import (
	"fmt"
	"github.com/washCutBlow/wcb-chaomai/chaomai"
	"github.com/washCutBlow/wcb-chaomai/wcb_redis"
	"net/http"
	"os"
)

func handlechaomai(w http.ResponseWriter, r *http.Request)  {
	myRedis := &chaomai.Redis{
		Host:"xx",
		Port:"10578",
		PassWord:"xx",
	}

	err := myRedis.GetConn()
	if err != nil {
		fmt.Println(err.Error())
	}
	defer myRedis.Close()

	//设置初始库存
	storeNum := 95
	//设置总库存
	limitStoreNum := 100
	//设置key
	redisKey := "huawei_p30_num_100"
	if !myRedis.KeyExists(redisKey){
		//实现分布式锁
		myRedis.SetNX(redisKey,storeNum)
	}
	//递增完再判断
	num := myRedis.IncrBy(redisKey,1)
	if num >limitStoreNum{
		fmt.Println("writeDb Error,storeNum is ",num)
	}else{
		fmt.Println("writeDb SUCCESS,storeNum is ",num)
	}
}
func main()  {
/*
	http.HandleFunc("/", handlechaomai)

	err := http.ListenAndServe("0.0.0.0:9876",nil)
	if(err != nil){
		fmt.Println("start Http Error,err is ",err)
	}*/

	client := wcb_redis.NewClient(&wcb_redis.Options{
		Addr:"localhost:6379",
	})

	if client == nil {
		fmt.Printf("client create eror")
		os.Exit(0)
	}
	err := client.SendCommand("SET yzqsb world")
	//err = client.SendCommand("SET zldd world")

	if err != nil {
		fmt.Println(err)
	}

}
