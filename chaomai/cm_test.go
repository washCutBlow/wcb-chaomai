package chaomai_test

import (
	"github.com/washCutBlow/wcb-chaomai/chaomai"
	"testing"
)

func BenchmarkRedis_GetConn(b *testing.B) {

	MyRed := chaomai.Redis{
		Host:"xxx",
		Port:"10578",
		PassWord:"xxx",
	}

	b.ResetTimer()
	for i:=0; i < b.N;i++ {
		MyRed.GetConn()
	}
}


