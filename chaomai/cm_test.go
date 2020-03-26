package chaomai_test

import (
	"github.com/washCutBlow/wcb-chaomai/chaomai"
	"testing"
)

func BenchmarkRedis_GetConn(b *testing.B) {

	MyRed := chaomai.Redis{
		Host:"m10578.mars.test.redis.ljnode.com",
		Port:"10578",
		PassWord:"FafBf4bec4",
	}

	b.ResetTimer()
	for i:=0; i < b.N;i++ {
		MyRed.GetConn()
	}
}


