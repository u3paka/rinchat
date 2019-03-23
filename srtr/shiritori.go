package srtr

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"text/template"
	"time"
	"unicode"

	redis "gopkg.in/redis.v5"

	"github.com/jmcvetta/randutil"
	"github.com/k0kubun/pp"
	"github.com/u3paka/jumangok/jmg"
)

const (
	Index_SRTR_Surface = "game:srtr:%s:%s:ls"
	Index_SRTR_Setting = "game:srtr:%s:%s"
	Index_FIXED_DIALOG = "dialog:%s:%s"
)

type Context struct {
	Name         string
	Mode         string
	History      []*jmg.Word
	LenRule      int64
	Flag         Flag
	NextWord     *jmg.Word
	ThisWord     *jmg.Word
	AnsWord      *jmg.Word
	Losecnt      int64
	Wincnt       int64
	Maxcnt       int64
	RoomLen      int64
	Event        string
	Tmpl         string
	LockDuration time.Duration
}

type Flag struct {
	IsNewGame   bool
	IsNewRecord bool
	IsLose      bool
	IsWin       bool
	IsChaos     bool
	IsReverse   bool
	Short       bool
}

func (ctx *Context) Template() string {
	//rtmpl := ctx.Event
	funcMap := template.FuncMap{
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
		"noun": func(s string) {

		},
		"furigana": func(wd *jmg.Word) string {
			switch wd.Sound {
			case "":
				return ""
			case wd.Surface:
				return wd.Surface
			default:
				return wd.Surface + "(" + wd.Sound + ")"
			}
		},
		"head": Head,
	}
	tpl := template.Must(template.New("main").Funcs(funcMap).Parse(ctx.Tmpl))
	var doc bytes.Buffer
	if err := tpl.ExecuteTemplate(&doc, ctx.Event, ctx); err != nil {
		fmt.Println(err)
		return ""
	}
	return strings.TrimSpace(doc.String())
}

func kanatokata(s string) string {
	kanaConv := unicode.SpecialCase{
		// ひらがなをカタカナに変換
		unicode.CaseRange{
			Lo: 0x3041, // Lo: ぁ
			Hi: 0x3093, // Hi: ん
			Delta: [unicode.MaxCase]rune{
				0x30a1 - 0x3041, // UpperCase でカタカナに変換
				0,               // LowerCase では変換しない
				0x30a1 - 0x3041, // TitleCase でカタカナに変換
			},
		},
	}
	return strings.ToUpperSpecial(kanaConv, s)
}

func Head(wd *jmg.Word, reverse bool) string {
	if wd.Sound == "" {
		return ""
	}
	// hiragana -> katakana
	kts := kanatokata(wd.Sound)
	rk := []rune(kts)
	if reverse {
		return string(rk[0])
	}
	for i := 1; i < len(rk)+1; i++ {
		l := string(rk[len(rk)-i])
		switch l {
		case "ー", "ッ", "ャ", "ョ", "ュ", "ァ", "ィ", "ゥ", "ェ", "ォ":
			continue
		default:
			return string(rk[len(rk)-i:])
		}
	}
	return ""
}

func (ctx *Context) Head(w *jmg.Word) string {
	return Head(w, ctx.Flag.IsReverse)
}

func RandomWord(sws []jmg.Word, cxt *Context) jmg.Word {
	weight := 1
	choices := make([]randutil.Choice, 0, 2)
	for _, w := range sws {
		switch {
		case len([]rune(w.Surface)) < int(cxt.LenRule):
			continue
		case strings.Contains(w.Sound, "*"):
			continue
		case !cxt.Flag.IsLose && !cxt.Flag.IsReverse && strings.HasSuffix(w.Sound, "ン"):
			continue
		case !cxt.Flag.IsChaos:
			weight = int(w.Meta.TFscore)
			fallthrough
		default:
			choices = append(choices, randutil.Choice{Weight: weight, Item: w})
		}
	}
	var resw jmg.Word
	for i := 0; i < 3; i++ {
		result, _ := randutil.WeightedChoice(choices)
		if result.Item == nil {
			break
		}
		resw = result.Item.(jmg.Word)
		if func() bool {
			for _, v := range cxt.History {
				if v == &resw {
					return false
				}
			}
			return true
		}() {
			break
		}
	}
	return resw
}
func LoadContext(vars map[string]interface{}, skey string) *Context {
	sx, ok := vars[skey]
	if ok {
		sx2, ok2 := sx.(Context)
		if ok2 {
			sx2.Flag.IsNewGame = false
			sx2.Mode = ""
			return &sx2
		}
	}
	s := &jmg.Word{
		Surface: "しりとり",
		Lemma:   "しりとり",
		Sound:   "しりとり",
	}

	hst := make([]*jmg.Word, 1)
	hst[0] = s
	pp.Println("new ctx")
	return &Context{
		AnsWord: s,
		History: hst,
		LenRule: 2,
		Flag: Flag{
			IsNewGame:   true,
			IsNewRecord: false,
			IsLose:      false,
			IsReverse:   false,
			Short:       false,
		},
	}

}
func Ai(r *redis.Client) func(*Context, *jmg.Word) string {
	return func(sctx *Context, n *jmg.Word) (ret string) {

		sctx.ThisWord = n
		// 字数チェック
		if len([]rune(n.Sound)) < int(sctx.LenRule) {
			sctx.Mode = "ErrTooShort"
			return
		}

		// historyから同じものがないかチェック
		for _, v := range sctx.History {
			if v.Lemma == sctx.ThisWord.Lemma {
				sctx.Mode = "ErrHasAlready"
				return
			}
		}
		//今回の頭
		atama := Head(n, !sctx.Flag.IsReverse)
		// 今回の尻
		head := Head(n, sctx.Flag.IsReverse)

		//前の尻
		reqhead := Head(sctx.History[len(sctx.History)-1], sctx.Flag.IsReverse)

		//来たものを覚える。
		go func() {
			pipe := r.Pipeline()
			pipe.SAdd(fmt.Sprintf("pre:%s", atama), jmg.WtoCSV(n)).Result()
			pipe.SAdd(fmt.Sprintf("suf:%s", head), jmg.WtoCSV(n)).Result()
			pipe.Exec()
		}()

		switch {
		case !sctx.Flag.IsReverse && !strings.HasPrefix(atama, reqhead),
			sctx.Flag.IsReverse && !strings.HasSuffix(atama, reqhead):
			sctx.Mode = "ErrNotMatch"
			return
		}

		// ン終わりの判定
		if head == "ン" {
			sctx.Mode = "RestrictHead"
			return
		}

		// 回数に応じて自分から負ける
		if len(sctx.History) > 20 {
			sctx.Flag.IsLose = true
			sctx.Mode = "AILoseN"
			return
		}

		var k string
		if sctx.Flag.IsReverse {
			k = fmt.Sprintf("suf:%s", head)
		} else {
			k = fmt.Sprintf("pre:%s", head)
		}
		randget := func(k string) (w *jmg.Word) {
			csv, err := r.SRandMember(k).Result()
			if err != nil {
				fmt.Println(err)
				return
			}
			w, err = jmg.CSVtoW(csv)
			if err != nil {
				fmt.Println(err)
				return
			}
			return
		}
		for i := 0; i < 3; i++ {
			nn := randget(k)
			if nn == nil {
				sctx.Mode = "AILoseNoExist"
				return
			}
			// 字数チェック
			if len([]rune(nn.Sound)) < int(sctx.LenRule) {
				continue
			}
			// historyから同じものがないかチェック
			for _, v := range sctx.History {
				if v.Lemma == nn.Lemma {
					continue
				}
			}
			ah := Head(nn, sctx.Flag.IsReverse)
			if ah == "ン" {
				if sctx.Flag.IsLose {
					sctx.AnsWord = nn
					sctx.Mode = "AILoseRestrictHead"
					return
				}
				continue
			}
			sctx.AnsWord = nn
			goto success
		}
		sctx.Mode = "AILoseNoExist"
		return
	success:
		sctx.History = append(sctx.History, n, sctx.AnsWord)
		return
	}
}
