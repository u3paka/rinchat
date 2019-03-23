package rinchat

import (
	"strconv"
	"testing"
	"time"

	"github.com/k0kubun/pp"
)

func BenchmarkLoadContext(b *testing.B) {
	cf := NewConfig("config")
	s := NewService(cf)
	for n := 0; n < b.N; n++ {
		s.LoadContext("paka")
	}
}

func Test(t *testing.T) {
	cf := NewConfig("config")
	s := NewService(cf)
	pp.Println(s.LoadContext("paka").Scan("ping"))
}

func BenchmarkScan(b *testing.B) {
	cf := NewConfig("config")
	s := NewService(cf)
	cx := s.LoadContext("paka")
	const input = "ping"
	for n := 0; n < b.N; n++ {
		cx.Scan(input)
	}
	pp.Println(cx.Scan(input))
}

func BenchmarkDo(b *testing.B) {
	cf := NewConfig("config")
	s := NewService(cf)
	s.SaveFunc = func(ctx *Context) {
		// heavy
		time.Sleep(time.Millisecond * 20000)
		return
	}
	const input = "re srtr"
	for n := 0; n < b.N; n++ {
		s.LoadContext("paka" + strconv.Itoa(n)).MSet(map[string]interface{}{
			"n": n,
		}).Do(input)
	}
	// pp.Println(cx.Scan(input))
}
