package gpui

import (
	"slices"
	"testing"
)

func TestSeeds_SeedsFollowIterationsExplicitSeedsAndSeedVariable(t *testing.T) {
	collect := func(iterations uint64, explicitSeeds []uint64, envSeed Option[uint64]) []uint64 {
		seq, _ := seeds(iterations, explicitSeeds, envSeed)
		return slices.Collect(seq)
	}
	cases := []struct {
		got, want []uint64
		msg       string
	}{
		{collect(1, nil, None[uint64]()), []uint64{0}, ""},
		{collect(3, nil, None[uint64]()), []uint64{0, 1, 2}, ""},
		{collect(1, []uint64{10, 20}, None[uint64]()), []uint64{10, 20}, ""},
		{collect(2, []uint64{10}, None[uint64]()), []uint64{0, 1, 10}, ""},
		{collect(1, []uint64{10}, Some[uint64](7)), []uint64{7}, ""},
		{collect(2, []uint64{10}, Some[uint64](7)), []uint64{7, 8}, "`SEED` starts the iterations once, without repeating itself"},
	}
	for i, c := range cases {
		if !slices.Equal(c.got, c.want) {
			t.Errorf("case %d: got %v, want %v %s", i, c.got, c.want, c.msg)
		}
	}
}

func TestSeeds_LazyAndOverflow(t *testing.T) {
	seq, multi := seeds(1<<62, nil, None[uint64]())
	if !multi {
		t.Fatal("expected multiple seeds")
	}
	for s := range seq {
		if s != 0 {
			t.Fatalf("first seed = %d", s)
		}
		break
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected overflow panic")
		}
	}()
	seeds(2, nil, Some[uint64](^uint64(0)))
}
