//go:build linux

package common

import (
	"testing"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

func TestDispatcher_CalloopWorks(t *testing.T) {
	eventLoop, err := NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	defer eventLoop.Close()
	handle := eventLoop.Handle()

	tx, rx := NewPriorityQueueCalloopReceiver[struct{}]()

	type data struct {
		gotMsg    bool
		gotClosed bool
	}
	var d data

	if _, err := InsertSource(handle, rx, func(evt ChannelEvent[struct{}]) {
		if evt.Closed {
			d.gotClosed = true
		} else {
			d.gotMsg = true
		}
	}); err != nil {
		t.Fatal(err)
	}

	if err := eventLoop.Dispatch(gpui.Some(time.Duration(0))); err != nil {
		t.Fatal(err)
	}
	if d.gotMsg || d.gotClosed {
		t.Fatalf("got %+v before anything was sent", d)
	}

	if err := tx.Send(scheduler.PriorityMedium, struct{}{}); err != nil {
		t.Fatal(err)
	}
	if err := eventLoop.Dispatch(gpui.Some(time.Duration(0))); err != nil {
		t.Fatal(err)
	}
	if !d.gotMsg || d.gotClosed {
		t.Fatalf("got %+v after a send", d)
	}

	tx.Close()
	if err := eventLoop.Dispatch(gpui.Some(time.Duration(0))); err != nil {
		t.Fatal(err)
	}
	if !d.gotMsg || !d.gotClosed {
		t.Fatalf("got %+v after the sender was dropped", d)
	}
}
