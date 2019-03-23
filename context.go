package rinchat

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/jmcvetta/randutil"
	"github.com/pkg/errors"
)

const (
	cap       = "cap"
	initevent = "init"
	topicKey  = "session_event"
)

type Context struct {
	Service *Service
	sync.RWMutex
	UID     string
	Vars    map[string]interface{}
	FuncMap map[string]interface{}
	In      string
	Out     string
}

func (ctx *Context) Express(ts ...string) *Context {
	fmt.Println("vars:", ctx.GetAll())
	rs := ctx.Service.GetKnowledge("expression", ts...)
	if len(rs) == 0 {
		fmt.Println("no expressions:", ts)
		ctx.Out = ""
		return ctx
	}
	ret, err := randutil.ChoiceString(rs)
	if err != nil {
		fmt.Println(err)
		ctx.Out = ""
		return ctx
	}
	var err2 error
	ctx.Out, err2 = ctx.Template(ret)
	if err2 != nil {
		fmt.Println(err2)
		ctx.Out = ""
	}
	return ctx
}

func (ctx *Context) Do(in string) *Context {
	fmt.Println("vars:", ctx.GetAll())
	ctx.In = in
	if ctx.Service.InFilterFunc != nil {
		ctx.Service.InFilterFunc(ctx)
	}
	// Scan検索
	ret := ctx.Scan(in)
	var err error
	ctx.Out, err = ctx.Template(ret)
	if err != nil {
		fmt.Println(err)
		ctx.Out = ""
	}
	if ctx.Service.OutFilterFunc != nil {
		ctx.Service.OutFilterFunc(ctx)
	}
	ctx.Service.saveCh <- ctx
	return ctx
}

func (ctx *Context) FlushWithout(prefixes ...string) {
	// 後始末 sessionから始まる変数以外削除
	vs := make(map[string]interface{}, 0)
	ctx.Lock()
	defer ctx.Unlock()
	for k, v := range ctx.GetAll() {
		for _, p := range prefixes {
			if strings.HasPrefix(k, p) {
				vs[k] = v
			}
		}
	}
	ctx.Vars = vs
}

func (ctx *Context) Scan(in string) (ret string) {
	topic, ok := ctx.Get(topicKey)
	if !ok {
		fmt.Println(errors.New("no topic var"))
		topic = initevent
	}
	tstr, ok := topic.(string)
	if !ok {
		fmt.Println(errors.New("topic var is not str"))
		tstr = initevent
	}
	askm, ok := ctx.Service.TopicMap[tstr]
	if !ok {
		ctx.Set(topicKey, initevent)
		askm = ctx.Service.TopicMap[initevent]
	}
	var txts []string
	for _, a := range askm {
		if a.Option.Criteria == "in" {
			if strings.Contains(a.Key, in) {
				txts = a.Texts
				break
			}
		}
		// regex
		r := a.Regex.Copy()
		n1 := r.SubexpNames()
		mp := r.FindAllStringSubmatch(in, -1)
		if len(mp) == 0 {
			continue
		}

		//変数取得
		m := make(map[string]interface{}, len(mp[0]))
		for i, k := range mp[0] {
			key := n1[i]
			if key == "" {
				key = cap + strconv.Itoa(i)
			}
			m[key] = k
		}
		ctx.MSet(m)
		txts = a.Texts
		break
	}

	if len(txts) == 0 {
		ret = ""
		return
	}
	// random
	var err error
	ret, err = randutil.ChoiceString(txts)
	if err != nil {
		fmt.Print(err)
	}
	return
}

func (ctx *Context) Template(cont string) (string, error) {
	tpl, err := template.New("main").Delims("<", ">").Funcs(ctx.FuncMap).Parse(cont)
	if err != nil {
		return "", errors.Wrap(err, "template construction")
	}
	var doc bytes.Buffer
	if err := tpl.Execute(&doc, ctx.GetAll()); err != nil {
		return "", errors.Wrap(err, "template execute")
	}
	return strings.TrimSpace(doc.String()), err
}

func (s *Service) LoadContext(uid string) *Context {
	ctx := &Context{
		UID:     uid,
		Service: s,
		Vars:    make(map[string]interface{}, 0),
	}
	ctx.Set(topicKey, initevent)
	if s.LoadFunc != nil {
		s.LoadFunc(ctx)
	}
	ctx.FuncMap = ctx.BasicFuncs()
	if s.PreservedFuncs != nil {
		for k, f := range s.PreservedFuncs(ctx) {
			ctx.FuncMap[k] = f
		}
	}
	// reload to reflect preserved funcs
	for k, f := range ctx.BasicFuncs() {
		ctx.FuncMap[k] = f
	}
	ctx.Service.saveCh <- ctx
	return ctx
}
