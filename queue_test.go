package gpui

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func TestQueue_BlockedReceiversEndWhenTheSenderDrops(t *testing.T) {
	sender, receiver := NewPriorityQueueReceiver[uint32]()
	done := make(chan []uint32, 2)
	var workers sync.WaitGroup
	for range 2 {
		receiver := receiver.Clone()
		workers.Go(func() {
			var received []uint32
			for item := range receiver.Iter() {
				received = append(received, item)
			}
			done <- received
		})
	}
	receiver.Close()

	if err := sender.Send(scheduler.PriorityMedium, 7); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("receivers keep waiting while the sender is alive")
	default:
	}

	sender.Close()
	var received []uint32
	for range 2 {
		select {
		case r := <-done:
			received = append(received, r...)
		case <-time.After(5 * time.Second):
			t.Fatal("a waiting receiver should end once the sender is dropped")
		}
	}
	if !slices.Equal(received, []uint32{7}) {
		t.Fatalf("received %v, want [7]", received)
	}
	workers.Wait()
}

func TestQueue_AllTasksGetYielded(t *testing.T) {
	tx, rx := NewPriorityQueueReceiver[int]()
	for _, s := range []struct {
		p Priority
		v int
	}{
		{scheduler.PriorityMedium, 20},
		{scheduler.PriorityHigh, 30},
		{scheduler.PriorityLow, 10},
		{scheduler.PriorityMedium, 21},
		{scheduler.PriorityHigh, 31},
	} {
		if err := tx.Send(s.p, s.v); err != nil {
			t.Fatal(err)
		}
	}

	tx.Close()

	got := slices.Sorted(rx.Iter())
	if want := []int{10, 20, 21, 30, 31}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestQueue_NewHighPrioTaskGetScheduledQuickly(t *testing.T) {
	tx, rx := NewPriorityQueueReceiver[int]()
	for range 100 {
		if err := tx.Send(scheduler.PriorityLow, 1); err != nil {
			t.Fatal(err)
		}
	}

	pop := func() int {
		v, err := rx.Pop()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if v := pop(); v != 1 {
		t.Fatalf("got %d, want 1", v)
	}
	if err := tx.Send(scheduler.PriorityHigh, 3); err != nil {
		t.Fatal(err)
	}
	if v := pop(); v != 3 {
		t.Fatalf("got %d, want 3", v)
	}
	if v := pop(); v != 1 {
		t.Fatalf("got %d, want 1", v)
	}
}

func TestQueue_SmallRngMatchesXoshiro256PlusPlus(t *testing.T) {
	rng := smallRng{s: [4]uint64{1, 2, 3, 4}}
	expected := []uint64{
		41943041,
		58720359,
		3588806011781223,
		3591011842654386,
		9228616714210784205,
		9973669472204895162,
		14011001112246962877,
		12406186145184390807,
		15849039046786891736,
		10450023813501588000,
	}
	for _, e := range expected {
		if got := rng.nextU64(); got != e {
			t.Fatalf("got %d, want %d", got, e)
		}
	}
}

func TestQueue_SmallRngSeedFromU64MatchesRand(t *testing.T) {
	rng := smallRngSeedFromU64(0)
	expected := []uint64{
		5987356902031041503,
		7051070477665621255,
		6633766593972829180,
		211316841551650330,
		9136120204379184874,
		379361710973160858,
		15813423377499357806,
		15596884590815070553,
		5439680534584881407,
		1369371744833522710,
	}
	for _, e := range expected {
		if got := rng.nextU64(); got != e {
			t.Fatalf("got %d, want %d", got, e)
		}
	}
}

func TestQueue_CloseIsIdempotentAndEndsReceivers(t *testing.T) {
	tx, rx := NewPriorityQueueReceiver[int]()
	tx.Close()
	tx.Close()
	if _, err := rx.Pop(); err == nil {
		t.Fatal("Pop after the last sender closed must fail")
	}
	rx.Close()
	rx.Close()
	tx2, rx2 := NewPriorityQueueReceiver[int]()
	rx2.Close()
	if err := tx2.Send(scheduler.PriorityLow, 1); err == nil {
		t.Fatal("Send without receivers must fail")
	}
}
