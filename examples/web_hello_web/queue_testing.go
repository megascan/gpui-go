//go:build js && wasm

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall/js"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

type queueItem struct {
	value int
	Stop  bool
}

type QueueStress struct {
	sender   *gpui.PriorityQueueSender[queueItem]
	receiver *gpui.PriorityQueueReceiver[int]
	ready    atomic.Int64
	stopped  atomic.Int64
	workers  int
	seen     []bool
	sent     int
	received int
	wg       sync.WaitGroup
}

func Priority(item int) scheduler.Priority {
	switch item % 3 {
	case 0:
		return scheduler.PriorityHigh
	case 1:
		return scheduler.PriorityMedium
	default:
		return scheduler.PriorityLow
	}
}

func NewQueueStress(workerCount, itemCount int) (*QueueStress, error) {
	if workerCount <= 0 || itemCount <= 0 {
		return nil, fmt.Errorf("worker and item counts must be positive")
	}
	sender, receiver := gpui.NewPriorityQueueReceiver[queueItem]()
	replySender, replyReceiver := gpui.NewPriorityQueueReceiver[int]()
	stress := &QueueStress{
		sender:   sender,
		receiver: replyReceiver,
		workers:  workerCount,
		seen:     make([]bool, itemCount),
	}
	for range workerCount {
		worker := receiver.Clone()
		stress.wg.Add(1)
		go func() {
			defer stress.wg.Done()
			stress.ready.Add(1)
			for {
				item, err := worker.Pop()
				if err != nil || item.Stop {
					break
				}
				_ = replySender.Send(Priority(item.value), item.value)
			}
			stress.stopped.Add(1)
		}()
	}
	return stress, nil
}

func (s *QueueStress) Ready() bool {
	return s.ready.Load() == int64(s.workers)
}

func (s *QueueStress) SendBatch(count int) error {
	end := min(s.sent+count, len(s.seen))
	for item := s.sent; item < end; item++ {
		if err := s.sender.SpinSend(Priority(item), queueItem{value: item}); err != nil {
			return fmt.Errorf("send work from browser main thread: %w", err)
		}
	}
	s.sent = end
	return nil
}

func (s *QueueStress) Drain() (int, error) {
	for {
		item, ok, err := s.receiver.SpinTryPop()
		if err != nil {
			return 0, fmt.Errorf("receive reply on browser main thread: %w", err)
		}
		if !ok {
			return s.received, nil
		}
		if item >= len(s.seen) {
			return 0, fmt.Errorf("unexpected reply")
		}
		if s.seen[item] {
			return 0, fmt.Errorf("duplicate reply")
		}
		s.seen[item] = true
		s.received++
	}
}

func (s *QueueStress) Stop() error {
	if s.sent != len(s.seen) || s.received != s.sent {
		return fmt.Errorf("stop before every item was received")
	}
	for range s.workers {
		if err := s.sender.SpinSend(scheduler.PriorityLow, queueItem{Stop: true}); err != nil {
			return fmt.Errorf("stop worker: %w", err)
		}
	}
	return nil
}

func (s *QueueStress) Stopped() bool {
	return s.stopped.Load() == int64(s.workers)
}

func registerQueueTest() {
	exports := js.Global().Get("Object").New()
	exports.Set("new_queue_stress", js.FuncOf(func(_ js.Value, args []js.Value) any {
		stress, err := NewQueueStress(args[0].Int(), args[1].Int())
		if err != nil {
			return js.ValueOf(err.Error())
		}
		return queueStressObject(stress)
	}))
	js.Global().Set("gpuiWebTests", exports)
}

func queueStressObject(stress *QueueStress) js.Value {
	object := js.Global().Get("Object").New()
	object.Set("ready", js.FuncOf(func(js.Value, []js.Value) any { return stress.Ready() }))
	object.Set("send_batch", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if err := stress.SendBatch(args[0].Int()); err != nil {
			return js.ValueOf(err.Error())
		}
		return js.Undefined()
	}))
	object.Set("drain", js.FuncOf(func(js.Value, []js.Value) any {
		received, err := stress.Drain()
		if err != nil {
			return js.ValueOf(err.Error())
		}
		return received
	}))
	object.Set("stop", js.FuncOf(func(js.Value, []js.Value) any {
		if err := stress.Stop(); err != nil {
			return js.ValueOf(err.Error())
		}
		return js.Undefined()
	}))
	object.Set("stopped", js.FuncOf(func(js.Value, []js.Value) any { return stress.Stopped() }))
	return object
}
