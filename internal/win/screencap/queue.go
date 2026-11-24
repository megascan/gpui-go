//go:build windows

package screencap

import (
	"errors"
	"sync"

	gpui "github.com/megascan/gpui-go"
)

var (
	errRecv = errors.New("receiving on a closed channel")
	errSend = errors.New("sending on a closed channel")
)

const frameQueueCap = 8

type frameResult struct {
	frame gpui.ScapFrame
	err   error
}

type frameQueue struct {
	mu     sync.Mutex
	cond   sync.Cond
	items  []frameResult
	closed bool
}

func newFrameQueue() *frameQueue {
	q := &frameQueue{}
	q.cond.L = &q.mu
	return q
}

func (q *frameQueue) send(frame gpui.ScapFrame, err error) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return errSend
	}
	if len(q.items) >= frameQueueCap {
		q.items[0] = frameResult{}
		q.items = q.items[1:]
	}
	q.items = append(q.items, frameResult{frame, err})
	q.cond.Signal()
	return nil
}

func (q *frameQueue) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.items = nil
	q.cond.Broadcast()
}

func (q *frameQueue) finish(err error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	if err != nil {
		q.items = append(q.items, frameResult{err: err})
	}
	q.closed = true
	q.cond.Broadcast()
}

func (q *frameQueue) recv() (frameResult, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}
	if len(q.items) == 0 {
		return frameResult{}, errRecv
	}
	r := q.items[0]
	q.items[0] = frameResult{}
	q.items = q.items[1:]
	return r, nil
}
