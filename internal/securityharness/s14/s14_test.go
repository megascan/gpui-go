//go:build securityharness

package s14

import (
	"fmt"
	"math"
	"math/rand/v2"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/etagere"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
)

func etagereRun(seed uint64, ops int, gpuiUsage bool) (known int, err error) {
	rng := rand.New(rand.NewPCG(seed, seed+1))
	sizes := []int32{0, -1, -100, 1, 2, 3, 7, 16, 255, 256, 257, 1023, 1024, 2047, 2048, 4095, 4096, 65534, 65535, 65536, math.MaxInt32, math.MinInt32}
	pick := func() int32 {
		if rng.IntN(3) == 0 {
			return sizes[rng.IntN(len(sizes))]
		}
		return int32(rng.IntN(600)) - 4
	}
	for round := 0; round < 200; round++ {
		w := int32(256 << rng.IntN(5))
		opts := etagere.DefaultAllocatorOptions()
		if !gpuiUsage {
			opts.VerticalShelves = rng.IntN(2) == 0
			opts.NumColumns = int32(1 + rng.IntN(3))
		}
		a := etagere.BucketedAtlasAllocatorWithOptions(etagere.Size2(w, w), opts)
		var live []etagere.AllocID
		for i := 0; i < ops; i++ {
			var panicked any
			var stack string
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked, stack = r, string(debug.Stack())
					}
				}()
				switch rng.IntN(10) {
				case 0, 1, 2, 3, 4, 5:
					if al, ok := a.Allocate(etagere.Size2(pick(), pick())); ok {
						live = append(live, al.ID)
					}
				case 6, 7:
					if len(live) > 0 {
						j := rng.IntN(len(live))
						a.Deallocate(live[j])
						live = append(live[:j], live[j+1:]...)
					}
				case 8:
					if gpuiUsage {
						return
					}
					s := a.Size()
					nw := min(s.Width+int32(rng.IntN(512)), 65534)
					nh := min(s.Height+int32(rng.IntN(512)), 65534)
					a.Grow(etagere.Size2(nw, nh))
				default:
					if rng.IntN(20) == 0 {
						a.Clear()
						live = live[:0]
					}
				}
			}()
			if panicked != nil {
				if strings.Contains(stack, "coalesceShelves") {
					known++
					break
				}
				fmt.Printf("PANIC seed=%d round=%d op=%d: %v\n%s\n", seed, round, i, panicked, stack)
				return known, fmt.Errorf("unknown panic %v", panicked)
			}
		}
	}
	return known, nil
}

func TestS14_01_EtagereFuzz(t *testing.T) {
	seed := uint64(0x5e1401)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 55*time.Second, "", func(string) (string, error) {
		known, err := etagereRun(seed, 3000, true)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("known E-COALESCE-OOB hits=%d, no other panic", known), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "etagere fuzz as gpui atlases use it: default options, Allocate/Deallocate/Clear, negative/zero/huge sizes")
	}
}

func TestS14_03_EtagereFuzzFullAPI(t *testing.T) {
	seed := uint64(0x5e1403)
	t.Logf("seed=%d", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 55*time.Second, "", func(string) (string, error) {
		known, err := etagereRun(seed, 3000, false)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("known E-COALESCE-OOB hits=%d, no other panic", known), nil
	})
	if parent {
		t.Logf("INFO log-only (Grow/vertical shelves/columns are not used by gpui atlases): %s", o)
	}
}

func hostilePaths() map[string]func(b *gpui.PathBuilder) {
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))
	p := func(x, y float32) gpui.Point[gpui.Pixels] { return gpui.NewPoint(gpui.Px(x), gpui.Px(y)) }
	return map[string]func(b *gpui.PathBuilder){
		"nan": func(b *gpui.PathBuilder) {
			b.MoveTo(p(0, 0))
			b.LineTo(p(nan, 10))
			b.LineTo(p(10, 10))
			b.Close()
		},
		"inf": func(b *gpui.PathBuilder) {
			b.MoveTo(p(0, 0))
			b.LineTo(p(inf, 10))
			b.LineTo(p(10, -inf))
			b.Close()
		},
		"huge": func(b *gpui.PathBuilder) {
			b.MoveTo(p(-3e38, -3e38))
			b.LineTo(p(3e38, -3e38))
			b.LineTo(p(3e38, 3e38))
			b.Close()
		},
		"wide": func(b *gpui.PathBuilder) {
			b.MoveTo(p(0, 0))
			b.LineTo(p(2e9, 1))
			b.LineTo(p(0, 2e9))
			b.Close()
		},
		"manycurves": func(b *gpui.PathBuilder) {
			b.MoveTo(p(0, 0))
			for i := range 200000 {
				b.CubicBezierTo(p(float32(i%4000), float32(i/4000)), p(1e6, -1e6), p(-1e6, 1e6))
			}
			b.Close()
		},
		"arc": func(b *gpui.PathBuilder) {
			b.MoveTo(p(0, 0))
			b.ArcTo(p(1e-30, 1e30), gpui.Px(nan), true, true, p(1e20, 1e20))
			b.Close()
		},
	}
}

func TestS14_02_PathBinHostileCoords(t *testing.T) {
	for name, build := range hostilePaths() {
		for _, mode := range []string{"fill", "stroke"} {
			t.Run(name+"_"+mode, func(t *testing.T) {
				o, parent := bombchild.Run(t, bombchild.DefaultLimit, 40*time.Second, "", func(string) (string, error) {
					var b *gpui.PathBuilder
					if mode == "fill" {
						b = gpui.PathBuilderFill()
					} else {
						b = gpui.PathBuilderStroke(gpui.Px(3))
					}
					build(b)
					start := time.Now()
					path, err := b.Build()
					if err != nil {
						return "", err
					}
					mask := gpui.ContentMask[gpui.Pixels]{Bounds: gpui.NewBounds(gpui.NewPoint(gpui.Px(0), gpui.Px(0)), gpui.NewSize(gpui.Px(4096), gpui.Px(4096)))}
					for _, scale := range []float32{1, 2, 1e6} {
						gpui.NewPaintedPath(path, scale, mask, gpui.SolidBackground(gpui.Rgba{R: 1, A: 1}))
					}
					return fmt.Sprintf("built+binned in %s", time.Since(start).Round(time.Millisecond)), nil
				})
				if parent {
					if name == "manycurves" {
						t.Logf("INFO log-only (200k app-authored curves): %s", o)
						return
					}
					bombchild.ExpectSafe(t, o, "PathBuilder "+name+" "+mode+" -> NewPaintedPath")
				}
			})
		}
	}
}
