package rinchat

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Answer struct {
	Key    string
	Texts  []string
	Regex  *regexp.Regexp
	Option *Option
}

type Option struct {
	Weight   int
	Criteria string
}

type Service struct {
	Knowledge      map[string]interface{}
	TopicOrder     []string
	TopicMap       map[string][]*Answer
	PreservedFuncs func(*Context) template.FuncMap
	InFilterFunc   func(*Context)
	OutFilterFunc  func(*Context)
	LoadFunc       func(*Context)
	SaveFunc       func(*Context)
	saveCh         chan *Context
}

type Config struct {
	*viper.Viper
}

func NewConfig(fn string, paths ...string) *Config {
	v := &Config{viper.New()}
	v.SetConfigName(fn)  // name of config file (without extension)
	v.AddConfigPath(".") // optionally look for config in the working directory
	for _, path := range paths {
		v.AddConfigPath(path)
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("changed setting file:", e.Name)
	})
	err := v.ReadInConfig() // Find and read the config file
	if err != nil {         // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
	v.AutomaticEnv()
	return v
}

func extractOptions(key string) (*Option, string) {
	const defaultWeight = 10
	const defaultCriteria = "regex"
	const rightParen = "}"
	const leftParen = "{"
	w := defaultWeight
	cr := defaultCriteria
	if strings.HasPrefix(key, "in::") {
		cr = "in"
		key = strings.TrimPrefix(key, "in::")
	}
	if strings.HasSuffix(key, rightParen) {
		key = strings.TrimSuffix(key, rightParen)
		ls := strings.Split(key, leftParen)
		if len(ls) == 2 {
			lls := strings.Split(ls[1], "=")

			k := strings.TrimSpace(lls[0])
			switch k {
			case "weight", "w":
				var err error
				w, err = strconv.Atoi(strings.TrimSpace(lls[1]))
				if err != nil {
					fmt.Println(err)
					w = defaultWeight
				}
			case "criteria", "c":
				cr = lls[1]
			}
		}
		key = ls[0]
	}
	if w == 100 && strings.Contains(key, "(.+)") {
		w = 1
	}
	return &Option{
		Weight:   w,
		Criteria: cr,
	}, key
}

func NewService(cf *Config) *Service {
	s := &Service{
		Knowledge: cf.GetStringMap("knowledge"),
		TopicMap:  make(map[string][]*Answer, 0),
		saveCh:    make(chan *Context, 0),
	}

	ctx := &Context{
		Service: s,
		Vars: map[string]interface{}{
			topicKey: initevent,
		},
	}
	// akka
	go func() {
		for {
			cx := <-s.saveCh
			if s.SaveFunc != nil {
				go s.SaveFunc(cx)
			}
		}
	}()

	rm := cf.GetStringMap("dialog")
	ctx.FuncMap = ctx.BasicFuncs()
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&")

	for t := range rm {
		as := make([]*Answer, 0)
		ms := cf.GetStringMapStringSlice("dialog." + t)
		for k, vs := range ms {
			opt, ck := extractOptions(k)
			tk, err := ctx.Template(ck)
			tk = strings.TrimSpace(tk)
			if err != nil {
				fmt.Println(err)
				continue
			}
			tk = r.Replace(tk)
			as = append(as, &Answer{
				Key:    tk,
				Texts:  vs,
				Regex:  regexp.MustCompile(tk),
				Option: opt,
			})
		}
		s.TopicMap[t] = as
	}
	for _, v := range s.TopicMap {
		rand.Seed(time.Now().UnixNano())
		sort.Slice(v, func(i, j int) bool {
			tw := v[i].Option.Weight + v[j].Option.Weight
			rw := rand.Intn(tw)
			if v[i].Option.Weight < rw {
				return false
			}
			return len(v[i].Key) > len(v[j].Key)
		})
	}
	return s
}

func (s *Service) GetKnowledge(kind string, ts ...string) (synls []string) {
	for _, t := range ts {
		syns, ok := s.Knowledge[kind].(map[string]interface{})[t]
		if !ok {
			continue
		}
		switch vs := syns.(type) {
		case []interface{}:
			buf := make([]string, len(vs))
			for i, syn := range vs {
				buf[i] = syn.(string)
			}
			synls = append(synls, buf...)
		case string:
			synls = append(synls, vs)
		default:
			continue
		}
	}
	return synls
}
