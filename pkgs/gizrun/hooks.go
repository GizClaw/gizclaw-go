package gizrun

import "sort"

type initHook struct {
	seq   int
	order int
	fn    func() error
}

type postInitHook struct {
	seq   int
	order int
	fn    func(*RunContext) error
}

type exitHook struct {
	seq   int
	order int
	fn    func(*RunContext) error
}

var initHooks = struct {
	next  int
	hooks []initHook
}{}

var postInitHooks = struct {
	next  int
	hooks []postInitHook
}{}

var exitHooks = struct {
	next  int
	hooks []exitHook
}{}

func InitAt(seq int, fn func() error) {
	if fn == nil {
		return
	}
	initHooks.hooks = append(initHooks.hooks, initHook{seq: seq, order: initHooks.next, fn: fn})
	initHooks.next++
}

func PostInitAt(seq int, fn func(*RunContext) error) {
	if fn == nil {
		return
	}
	postInitHooks.hooks = append(postInitHooks.hooks, postInitHook{seq: seq, order: postInitHooks.next, fn: fn})
	postInitHooks.next++
}

func ExitAt(seq int, fn func(*RunContext) error) {
	if fn == nil {
		return
	}
	exitHooks.hooks = append(exitHooks.hooks, exitHook{seq: seq, order: exitHooks.next, fn: fn})
	exitHooks.next++
}

func runInitHooks(values []initHook) {
	hooks := append([]initHook(nil), values...)
	sort.SliceStable(hooks, func(i, j int) bool {
		if hooks[i].seq != hooks[j].seq {
			return hooks[i].seq < hooks[j].seq
		}
		return hooks[i].order < hooks[j].order
	})
	for _, hook := range hooks {
		if err := hook.fn(); err != nil {
			panic(err)
		}
	}
}

func runExitHooks(ctx *RunContext, values []exitHook) {
	hooks := append([]exitHook(nil), values...)
	sort.SliceStable(hooks, func(i, j int) bool {
		if hooks[i].seq != hooks[j].seq {
			return hooks[i].seq < hooks[j].seq
		}
		return hooks[i].order < hooks[j].order
	})
	for _, hook := range hooks {
		if err := hook.fn(ctx); err != nil {
			panic(err)
		}
	}
}

func runPostInitHooks(ctx *RunContext, values []postInitHook) {
	hooks := append([]postInitHook(nil), values...)
	sort.SliceStable(hooks, func(i, j int) bool {
		if hooks[i].seq != hooks[j].seq {
			return hooks[i].seq < hooks[j].seq
		}
		return hooks[i].order < hooks[j].order
	})
	for _, hook := range hooks {
		if err := hook.fn(ctx); err != nil {
			panic(err)
		}
	}
}
