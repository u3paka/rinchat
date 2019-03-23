package rinchat

func (tm *Context) Set(key string, s interface{}) *Context {
	tm.Lock()
	defer tm.Unlock()
	tm.Vars[key] = s
	return tm
}

func (ctx *Context) MSet(m map[string]interface{}) *Context {
	ctx.Lock()
	defer ctx.Unlock()
	for k, f := range m {
		ctx.Vars[k] = f
	}
	return ctx
}

func (tm *Context) Get(key string) (interface{}, bool) {
	tm.RLock()
	defer tm.RUnlock()
	s, ok := tm.Vars[key]
	return s, ok
}

func (tm *Context) GetAll() map[string]interface{} {
	return tm.Vars
}

func (tm *Context) Delete(key string) {
	tm.Lock()
	defer tm.Unlock()

	_, ok := tm.Vars[key]
	if ok {
		delete(tm.Vars, key)
	}
	return
}
