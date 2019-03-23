package rinchat

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func (ctx *Context) BasicFuncs() template.FuncMap {
	return template.FuncMap{
		"go": func(e string) string {
			ctx.Set(topicKey, e)
			return ""
		},

		"init": func() string {
			ctx.Set(topicKey, "common")
			return ""
		},

		"set": func(k string, v interface{}) string {
			ctx.Set(k, v)
			return ""
		},

		"del": func(k string) string {
			ctx.Delete(k)
			return ""
		},

		"get": func(k string) string {
			v, ok := ctx.Get(k)
			if !ok {
				return ""
			}
			return v.(string)
		},

		"atoi": func(v string) int {
			n, err := strconv.Atoi(v)
			if err != nil {
				fmt.Println(err)
				return 0
			}
			return n
		},

		"incrby": func(k string, b int) string {
			v, ok := ctx.Get(k)
			if !ok {
				v = "0"
			}
			vint, ok := v.(int)
			if !ok {
				fmt.Println(v, "is not int")
				return ""
			}
			ctx.Set(k, strconv.Itoa(vint+b))
			return ""
		},

		"rand": func(cs ...string) string {
			rand.Seed(time.Now().Unix())
			return cs[rand.Intn(len(cs))]
		},

		"join": func(cs ...string) string {
			if cs[0] == "" {
				return ""
			}
			return strings.Join(cs, "")
		},

		"re": func(t string) string {
			ret := ctx.Scan(t)
			var err error
			ctx.Out, err = ctx.Template(ret)
			if err != nil {
				fmt.Println(err)
			}
			return ctx.Out
		},

		"syn": func(ts ...string) string {
			l := ctx.Service.GetKnowledge("synonyms", ts...)
			return fmt.Sprintf("(%s)", strings.Join(l, "|"))
		},
		"getsyn": func(ts ...string) string {
			l := ctx.Service.GetKnowledge("synonyms", ts...)
			rand.Seed(time.Now().Unix())
			return l[rand.Intn(len(l))]
		},
		"reg": func(ts ...string) (ret string) {
			l := ctx.Service.GetKnowledge("regex", ts...)
			return fmt.Sprintf("(%s)", strings.Join(l, "|"))
		},
		"cap": func(ns ...string) string {
			switch len(ns) {
			case 0:
				return "(.+)"
			case 1:
				return fmt.Sprintf("(?P<%s>.+)", ns[0])
			case 2:
				return fmt.Sprintf("(?P<%s>%s)", ns[0], ns[1])
			default:
				return ""
			}
		},
		"replace": func(t string, ns ...string) string {
			if len(ns)%2 == 1 {
				ns = ns[:len(ns)-1]
			}
			r := strings.NewReplacer(ns...)
			return r.Replace(t)
		},
		"iscontain": func(t string, n string) bool {
			return strings.Contains(t, n)
		},
	}
}
