package gpui

import (
	"math/rand/v2"
	"testing"
)

func TestBoundsTree_Insert(t *testing.T) {
	var tree boundsTree[float32]
	quad, shadow := primitiveKindQuad, primitiveKindShadow
	square := func(x, y float32) Bounds[float32] {
		return Bounds[float32]{Origin: Point[float32]{X: x, Y: y}, Size: Size[float32]{Width: 10.0, Height: 10.0}}
	}
	bounds1 := square(0.0, 0.0)
	bounds2 := square(5.0, 5.0)
	bounds3 := square(10.0, 10.0)

	for i, tc := range []struct {
		bounds Bounds[float32]
		want   uint32
	}{{bounds1, 1}, {bounds2, 2}, {bounds3, 3}} {
		if got := tree.insert(tc.bounds, quad, shadow); got != tc.want {
			t.Fatalf("insert %d: got ordering %d, want %d", i+1, got, tc.want)
		}
	}

	bounds4 := square(20.0, 20.0)
	bounds5 := square(40.0, 40.0)
	bounds6 := square(25.0, 25.0)
	for i, tc := range []struct {
		bounds Bounds[float32]
		want   uint32
	}{{bounds4, 1}, {bounds5, 1}, {bounds6, 2}} {
		if got := tree.insert(tc.bounds, quad, shadow); got != tc.want {
			t.Fatalf("insert %d: got ordering %d, want %d", i+4, got, tc.want)
		}
	}
}

func TestBoundsTree_RandomIterations(t *testing.T) {
	const maxBounds = 100
	kinds := [...]primitiveKind{primitiveKindShadow, primitiveKindQuad, primitiveKindPath, primitiveKindUnderline}
	type expectedQuad struct {
		bounds   Bounds[float32]
		ordering uint32
		kind     primitiveKind
	}
	for seed := uint64(1); seed <= 1000; seed++ {
		var tree boundsTree[float32]
		rng := rand.New(rand.NewPCG(seed, 0))
		var expectedQuads []expectedQuad

		numBounds := 1 + rng.IntN(maxBounds)
		for range numBounds {
			minX := rng.Float32()*200.0 - 100.0
			minY := rng.Float32()*200.0 - 100.0
			width := rng.Float32() * 50.0
			height := rng.Float32() * 50.0
			bounds := Bounds[float32]{Origin: Point[float32]{X: minX, Y: minY}, Size: Size[float32]{Width: width, Height: height}}
			kind := kinds[rng.IntN(len(kinds))]
			maxSharedKind := kinds[rng.IntN(len(kinds))]

			found := false
			var maxKey boundsTreeKey
			for _, quad := range expectedQuads {
				if !quad.bounds.Intersects(bounds) {
					continue
				}
				key := boundsTreeKey{ordering: quad.ordering, kind: quad.kind}
				if !found || key.cmp(maxKey) > 0 {
					maxKey = key
					found = true
				}
			}
			var expectedOrdering uint32
			switch {
			case !found:
				expectedOrdering = 1
			case maxKey.kind <= maxSharedKind:
				expectedOrdering = maxKey.ordering
			default:
				expectedOrdering = maxKey.ordering + 1
			}
			expectedQuads = append(expectedQuads, expectedQuad{bounds: bounds, ordering: expectedOrdering, kind: kind})

			if actual := tree.insert(bounds, kind, maxSharedKind); actual != expectedOrdering {
				t.Fatalf("seed %d: got ordering %d, want %d", seed, actual, expectedOrdering)
			}
		}
	}
}
