package gpui

import (
	"fmt"
	"iter"
	"math"
	"os"
	"slices"
	"strconv"
)

// CalculateSeeds returns the seeds a randomized test or benchmark runs with, in order, and
// whether there is more than one.
//
// With $SEED set, it runs $SEED..$SEED+iterations and ignores explicitSeeds. Otherwise it runs
// 0..iterations followed by explicitSeeds, except that the default single iteration is dropped
// when explicit seeds are given. $ITERATIONS overrides iterations. It is meant for RunTest and
// RunBench.
func CalculateSeeds(iterations uint64, explicitSeeds []uint64) (iter.Seq[uint64], bool) {
	if v, ok := os.LookupEnv("ITERATIONS"); ok {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("invalid ITERATIONS variable: %v", err))
		}
		iterations = n
	}

	envSeed := None[uint64]()
	if v, ok := os.LookupEnv("SEED"); ok {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("invalid SEED variable as integer: %v", err))
		}
		envSeed = Some(n)
	}

	return seeds(iterations, explicitSeeds, envSeed)
}

func seeds(iterations uint64, explicitSeeds []uint64, envSeed Option[uint64]) (iter.Seq[uint64], bool) {
	var start, end uint64
	if s, ok := envSeed.Get(); ok {
		if iterations > math.MaxUint64-s {
			panic("attempt to add with overflow")
		}
		start, end = s, s+iterations
		explicitSeeds = nil
	} else if iterations == 1 && len(explicitSeeds) > 0 {
		start, end = 0, 0
	} else {
		start, end = 0, iterations
	}
	explicitSeeds = slices.Clone(explicitSeeds)
	seq := func(yield func(uint64) bool) {
		for s := start; s < end; s++ {
			if !yield(s) {
				return
			}
		}
		for _, s := range explicitSeeds {
			if !yield(s) {
				return
			}
		}
	}
	n := end - start
	return seq, n > 1 || (n == 1 && len(explicitSeeds) > 0) || len(explicitSeeds) > 1
}
