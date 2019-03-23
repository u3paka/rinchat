package srtr

import (
	"testing"

	"github.com/k0kubun/pp"
	redis "gopkg.in/redis.v5"
)

func TestSrtr(t *testing.T) {

	r := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	scx := LoadContext(map[string]interface{}{}, "session_srtr")
	// Ai(r)(&scx)
	pp.Println(scx)
}
