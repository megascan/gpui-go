package scheduler_test

import (
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

type asyncContext struct {
	*scheduler.Scope
	label string
}

type inlineScheduler struct {
	queue []*scheduler.Runnable
}

func (s *inlineScheduler) dispatch(r *scheduler.Runnable) {
	s.queue = append(s.queue, r)
}

func (s *inlineScheduler) run() {
	for len(s.queue) > 0 {
		r := s.queue[0]
		s.queue = s.queue[1:]
		r.Run()
	}
}

func TestAwaiter_EmbeddedScope(t *testing.T) {
	s := &inlineScheduler{}
	e := scheduler.NewLocalExecutor(0, nil, s.dispatch)
	inner := e.Spawn(func(*scheduler.Scope) (int, error) { return 21, nil })
	outer := e.Spawn(func(s *scheduler.Scope) (string, error) {
		cx := &asyncContext{Scope: s, label: "x"}
		v, err := inner.Await(cx)
		if err != nil {
			return "", err
		}
		return cx.label + string(rune('0'+v/10)), nil
	})
	s.run()
	r := scheduler.BlockOn[scheduler.Result[string]](outer)
	if r.Value != "x2" || r.Err != nil {
		t.Fatalf("got %v", r)
	}
}
