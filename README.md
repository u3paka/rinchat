# Rinchat
## Description
rinchat is a Re-configurable INstant chat framework (pure-Go).

this framework enables you to reconfigure how your bots reply to your input by editing config.yml file.

## Installation and Usages
### install with go command:

    go get github.com/u3paka/rinchat

## Usage as a golang package
### import
```golang
import "github.com/u3paka/rinchat"
```

### Example
```golang
cf := rinchat.NewConfig("config", "../dirpath/of/config_file")
s := rinchat.NewService(cf)
input := "hello"
out := s.LoadContext("username").MSet(map[string]interface{}{
    "txt": "you can set any interface{} variables",
}).Do(input)

fmt.Println(out.Out) // こんにちはです！
```

## configuration[config.yml]

### dialog
```yml
dialog:
    init: 
        ping:
            - pong
            - pong!
            - pang!?
        hello:
            - hi!
        you (.+) regex:
            - yeah!
        go to topic2:
            - <go "topic2"> OK! changed topic...
    topic2:
        hello:
            - hello topic2!
        hoge:
            - fuga
        in::sushi:
            - I think so!
```

random reaction

    >> ping
    bot: pong!
    >> ping
    bot: pang?
    >> hello
    bot: hi!

you can use regex style.

    >> you are to use regex:
    bot: yeah!
    >> you may understand regex:
    bot: yeah!

you can change topic freely.

    >> go to topic2
    bot: OK! changed topic...
    >> hello
    bot: hello topic2!
    
prefix ```in::``` works if the sentence contains the words. ```in::sushi``` works much faster than the following regex ```(.+)sushi(.+)``` because ```in::sushi``` internally uses ```strings.Contain```.

    >> I love sushi!
    bot: I think so!

### knowledge.expression
```yml
dialog:
    init: 
        you are smart:
            - Wow, <ex "thanks">
knowledge:
    expression:
        thanks:
            - thank you!!
            - ありがとー
            - danke!
            - 謝謝
```

cmd    

    >> you are smart
    bot: Wow, thank you!!
    >> you are smart
    bot: Wow, 謝謝


### knowledge.synonyms
you can set synonyms and call ```<syn "title">```.

```yml
dialog:
    init: 
        Rin <syn "namesuf">:
            - Nya!
        (.+):
            - ha?
knowledge:
    synonyms:
        namesuf:
            - san
            - chan
            - kun
```

cmd    

    >> Rin chan
    bot: Nya!
    >> Rin san
    bot: Nya!
    >> Rin kun
    bot: Nya!
    >> Rin senpai
    bot: ha?

## Customize

you can customize middleware fuctions such as ```(*Service).LoadFunc (*Service).SaveFunc (*Service).PreservedFuncs ...``` and call the interconnected other services.

redis session example.
```golang

cf := rinchat.NewConfig("config")
rin := rinchat.NewService(cf)
r := redis.NewClient(&redis.Options{Addr: "localhost:6379"}

rin.LoadFunc = func(ctx *rinchat.Context) {
	d, err := r.Get(fmt.Sprintf("sess:%s", ctx.Uid)).Result()
	if err != nil {
		fmt.Println(err)
		return
	}
	dec := gob.NewDecoder(bytes.NewBuffer([]byte(d)))
	v := make(map[string]interface{}, 0)
	dec.Decode(&v)
	ctx.Lock()
	defer ctx.Unlock()
	ctx.Vars = v
	return
}

rin.SaveFunc = func(ctx *rinchat.Context) {
	ctx.FlushWithout("session")
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	ctx.RLock()
	defer ctx.RUnlock()
	err := enc.Encode(ctx.Vars)
	if err != nil {
		fmt.Println(err)
		return
	}
	key := fmt.Sprintf("sess:%s", ctx.Uid)
	r.Set(key, buf.Bytes(), cf.GetDuration("ttl.session"))
	return
}

// now you can use these funcs such as <sismember ""> in the dialog system.
rin.PreservedFuncs = func(ctx *rinchat.Context) template.FuncMap {
    "sismember": func(k, t string) bool {
		return r.SIsMember(k, t).Val()
	},
	"smembers": func(k string) []string {
		return r.SMembers(k).Val()
	},
	"srand": func(k string) string {
		return r.SRandMember(k).Val()
	},
    // しりとり plugin
    "srtrctx": func() *srtr.Context {
	    const skey = "session_srtr"
	    vs := ctx.GetAll()
	    f := srtr.LoadContext(vs, skey)
	    ctx.Set(skey, f)
	    // pp.Println(ctx.Vars)
	    return f
	},
	"srtrai": srtr.Ai(r),
}

```

## Author
u3paka

## Licence
MIT