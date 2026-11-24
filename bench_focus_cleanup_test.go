package gpui_test

import (
	"slices"
	"testing"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
)

const focusCleanupEntityCount = 10_000

func BenchmarkFocusCleanup_NoDrops(b *testing.B) {
	gpui.RunBenchInputs(b, gpui.BenchOptions{
		InputName: "handles",
		Group:     "focus_cleanup_no_drops_10000_effects",
		Fps:       120,
	}, []int{0, 100, 1_000, 10_000}, func(handleCount int, cx *gpui.BenchAppContext) {
		focusCleanupMeasure(b, handleCount, false, 0, cx)
	})
}

func BenchmarkFocusCleanup_FinalDrops(b *testing.B) {
	gpui.RunBenchInputs(b, gpui.BenchOptions{
		InputName: "handles",
		Group:     "focus_cleanup_final_drops_10000_effects",
		Fps:       120,
	}, []int{10_000}, func(handleCount int, cx *gpui.BenchAppContext) {
		focusCleanupMeasure(b, handleCount, true, 0, cx)
	})
}

func BenchmarkFocusCleanup_FinalDropsWithWindows(b *testing.B) {
	gpui.RunBenchInputs(b, gpui.BenchOptions{
		InputName: "windows",
		Group:     "focus_cleanup_final_drops_with_windows_10000_handles_10000_effects",
		Fps:       120,
	}, []int{1, 10}, func(windowCount int, cx *gpui.BenchAppContext) {
		focusCleanupMeasure(b, 10_000, true, windowCount, cx)
	})
}

type focusCleanupHandles struct {
	handles []gpui.FocusHandle
}

func (h *focusCleanupHandles) Release() {
	for _, handle := range h.handles {
		handle.Release()
	}
	h.handles = nil
}

func focusCleanupMeasure(b *testing.B, handleCount int, finalDrops bool, windowCount int, cx *gpui.BenchAppContext) {
	if gpuiutil.DebugAssertions {
		b.Skip("use -tags gpui_release")
	}
	if windowCount > handleCount {
		panic("assertion failed: window_count <= handle_count")
	}
	windows := make([]gpui.AnyWindowHandle, 0, windowCount)
	for range windowCount {
		windows = append(windows, cx.AddEmptyWindow().WindowHandle())
	}
	observed := &[]gpui.EntityID{}
	subscription := cx.UpdateR(func(app *gpui.App) gpui.Subscription {
		return app.ObserveNew(func(_ *focusCleanupCreatedEntity, _ *gpui.Window, ecx *gpui.Context[focusCleanupCreatedEntity]) {
			*observed = append(*observed, ecx.EntityID())
		})
	})
	defer subscription.Unsubscribe()

	prepare := func(cx *gpui.BenchAppContext) *focusCleanupHandles {
		return cx.UpdateR(func(app *gpui.App) *focusCleanupHandles {
			handles := make([]gpui.FocusHandle, 0, handleCount)
			for range handleCount {
				handles = append(handles, app.FocusHandle())
			}
			for i, window := range windows {
				if i >= len(handles) {
					break
				}
				handle := handles[i]
				err := window.Update(app, func(_ gpui.AnyView, window *gpui.Window, app *gpui.App) {
					window.Focus(handle, app)
					focused, ok := window.Focused(app)
					if !ok || !focused.Equal(handle) {
						panic("assertion failed: window.focused(cx) == Some(handle)")
					}
					focused.Release()
				})
				if err != nil {
					panic("benchmark window missing during focus setup")
				}
			}
			return &focusCleanupHandles{handles: handles}
		})
	}
	run := func(handles []gpui.FocusHandle, cx *gpui.BenchAppContext) *gpui.Task[*focusCleanupCompletedBatch] {
		return cx.ReadR(func(app *gpui.App) *gpui.Task[*focusCleanupCompletedBatch] {
			return app.Spawn(func(acx *gpui.AsyncApp) (*focusCleanupCompletedBatch, error) {
				dropped := make([]gpui.WeakFocusHandle, 0, len(handles))
				for _, handle := range handles {
					dropped = append(dropped, handle.Downgrade())
				}
				entities := acx.UpdateR(func(app *gpui.App) []gpui.Entity[focusCleanupCreatedEntity] {
					for _, handle := range handles {
						handle.Release()
					}
					entities := make([]gpui.Entity[focusCleanupCreatedEntity], 0, focusCleanupEntityCount)
					for range focusCleanupEntityCount {
						entities = append(entities, app.New(func(*gpui.Context[focusCleanupCreatedEntity]) *focusCleanupCreatedEntity {
							return &focusCleanupCreatedEntity{}
						}))
					}
					return entities
				})
				return &focusCleanupCompletedBatch{entities: entities, observed: observed, dropped: dropped}, nil
			})
		})
	}
	if finalDrops {
		cx.BenchBatchedTask(prepare, func(handles **focusCleanupHandles, cx *gpui.BenchAppContext) *gpui.Task[*focusCleanupCompletedBatch] {
			taken := (*handles).handles
			(*handles).handles = nil
			return run(taken, cx)
		})

		for _, window := range windows {
			err := window.Update(cx, func(_ gpui.AnyView, window *gpui.Window, app *gpui.App) {
				if focused, ok := window.Focused(app); ok {
					focused.Release()
					panic("assertion failed: window.focused(cx) == None")
				}
			})
			if err != nil {
				panic("benchmark window missing after focus cleanup")
			}
		}
	} else {
		handles := prepare(cx)
		defer handles.Release()
		cx.BenchTask(func(cx *gpui.BenchAppContext) *gpui.Task[*focusCleanupCompletedBatch] {
			return run(nil, cx)
		})

		for _, handle := range handles.handles {
			upgraded, ok := handle.Downgrade().Upgrade()
			if !ok || !upgraded.Equal(handle) {
				panic("assertion failed: handle.downgrade().upgrade() == Some(handle)")
			}
			upgraded.Release()
		}
	}
}

type focusCleanupCreatedEntity struct{}

type focusCleanupCompletedBatch struct {
	entities []gpui.Entity[focusCleanupCreatedEntity]
	observed *[]gpui.EntityID
	dropped  []gpui.WeakFocusHandle
}

func (c *focusCleanupCompletedBatch) Release() {
	if len(c.entities) != focusCleanupEntityCount {
		panic("assertion failed: entities.len() == ENTITY_COUNT")
	}
	ids := make([]gpui.EntityID, 0, len(c.entities))
	for _, entity := range c.entities {
		ids = append(ids, entity.EntityID())
	}
	if !slices.Equal(*c.observed, ids) {
		panic("assertion failed: observed == entity ids")
	}
	*c.observed = (*c.observed)[:0]
	alive := 0
	for _, handle := range c.dropped {
		if upgraded, ok := handle.Upgrade(); ok {
			alive++
			upgraded.Release()
		}
	}
	for _, entity := range c.entities {
		entity.Release()
	}
	if alive != 0 {
		panic("assertion failed: dropped handles still alive")
	}
}
