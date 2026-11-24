package gpui

import (
	"fmt"
	"reflect"

	"github.com/megascan/gpui-go/scheduler"
)

// PromptResponse is the event a prompt emits when one of its options is selected: the index of
// the option among the actions passed to the prompt.
type PromptResponse int

// Prompt is a view that can be rendered in the window as a prompt: it emits PromptResponse and
// is Focusable.
type Prompt interface {
	Focusable
}

// PromptHandle is a handle to a prompt, used to resolve it.
type PromptHandle struct {
	sender *scheduler.OneshotSender[int]
}

func newPromptHandle(sender *scheduler.OneshotSender[int]) PromptHandle {
	return PromptHandle{sender: sender}
}

// WithView builds a renderable prompt from view; *V must implement Prompt and Render[V]. The
// prompt resolves with the first PromptResponse view emits, and focus returns to what was focused
// before.
func (h PromptHandle) WithView[V any](view Entity[V], window *Window, cx *App) RenderablePromptHandle {
	if _, ok := any((*V)(nil)).(Render[V]); !ok {
		panic(fmt.Sprintf("gpui: %s does not implement Render", typeNameOf(reflect.TypeFor[V]())))
	}
	sender := h.sender
	previousFocus, hasPreviousFocus := window.Focused(cx)
	windowHandle := window.WindowHandle()
	cx.Subscribe(view, func(_ Entity[V], e *PromptResponse, cx *App) {
		if sender != nil {
			s := sender
			sender = nil
			_ = s.Send(int(*e))
			_ = windowHandle.Update(cx, func(_ AnyView, window *Window, cx *App) {
				if prompt, ok := window.prompt.Take().Get(); ok {
					prompt.release()
				}
				if hasPreviousFocus {
					window.Focus(previousFocus, cx)
				}
			})
			if hasPreviousFocus {
				previousFocus.Release()
			}
		}
	}).Detach()
	cx.ObserveRelease(view, func(_ *V, _ *App) {
		if sender != nil {
			s := sender
			sender = nil
			s.Close()
			if hasPreviousFocus {
				previousFocus.Release()
			}
		}
	}).Detach()

	focusHandle := view.FocusHandle(cx)
	window.Focus(focusHandle, cx)
	focusHandle.Release()

	return RenderablePromptHandle{view: entityPromptView[V]{entity: view}}
}

// RenderablePromptHandle is a prompt that can be rendered in a window.
type RenderablePromptHandle struct {
	view promptViewHandle
}

func (h RenderablePromptHandle) release() {
	if h.view != nil {
		h.view.release()
	}
}

// NewFallbackPromptRenderer renders a prompt with FallbackPromptRenderer; pass it to
// App.SetPromptBuilder to always use the fallback renderer.
func NewFallbackPromptRenderer(level PromptLevel, message string, detail Option[string], actions []PromptButton, handle PromptHandle, window *Window, cx *App) RenderablePromptHandle {
	renderer := cx.New(func(cx *Context[FallbackPromptRenderer]) *FallbackPromptRenderer {
		return &FallbackPromptRenderer{
			level:   level,
			message: message,
			detail:  detail,
			actions: append([]PromptButton(nil), actions...),
			focus:   cx.FocusHandle(),
		}
	})

	return handle.WithView(renderer, window, cx)
}

// FallbackPromptRenderer is GPUI's default prompt view, used when the platform has no native
// prompts.
type FallbackPromptRenderer struct {
	level   PromptLevel
	message string
	detail  Option[string]
	actions []PromptButton
	focus   FocusHandle
}

// Render renders the prompt over a translucent backdrop.
func (r *FallbackPromptRenderer) Render(window *Window, cx *Context[FallbackPromptRenderer]) IntoElement {
	prompt := NewDiv().
		CursorDefault().
		TrackFocus(r.focus).
		W72().
		Bg(White()).
		RoundedLg().
		OverflowHidden().
		P3().
		Child(
			NewDiv().
				WFull().
				Flex().
				FlexRow().
				JustifyAround().
				Child(NewDiv().OverflowHidden().ChildText(r.message)),
		)
	if detail, ok := r.detail.Get(); ok {
		prompt = prompt.Child(
			NewDiv().
				WFull().
				Flex().
				FlexRow().
				JustifyAround().
				TextSm().
				Mb2().
				Child(NewDiv().ChildText(detail)),
		)
	}
	for ix, action := range r.actions {
		prompt = prompt.Child(
			NewDiv().
				Flex().
				FlexRow().
				JustifyAround().
				Border1().
				BorderColor(OpaqueGrey(0.2, 0.5)).
				Mt1().
				RoundedXs().
				CursorPointer().
				TextSm().
				Child(action.Label()).
				ID(ix).
				OnClick(cx.Listener(func(_ *FallbackPromptRenderer, _ *ClickEvent, _ *Window, cx *Context[FallbackPromptRenderer]) {
					cx.Emit(PromptResponse(ix))
					cx.StopPropagation()
				})),
		)
	}

	return NewDiv().
		SizeFull().
		Child(
			NewDiv().
				SizeFull().
				Bg(OpaqueGrey(0.5, 0.6)).
				Absolute().
				Top0().
				Left0(),
		).
		Child(
			NewDiv().
				SizeFull().
				Absolute().
				Top0().
				Left0().
				Flex().
				FlexCol().
				JustifyAround().
				Child(
					NewDiv().
						WFull().
						Flex().
						FlexRow().
						JustifyAround().
						Child(prompt),
				),
		)
}

// FocusHandle returns the prompt's focus handle; the caller releases it.
func (r *FallbackPromptRenderer) FocusHandle(cx *App) FocusHandle {
	return r.focus.Clone()
}

// Drop releases the prompt's focus handle when the entity is released.
func (r *FallbackPromptRenderer) Drop(cx *App) {
	r.focus.Release()
}

type promptViewHandle interface {
	anyView() AnyView
	release()
}

type entityPromptView[V any] struct {
	entity Entity[V]
}

func (v entityPromptView[V]) anyView() AnyView {
	return AnyViewFromEntity(v.entity.Clone())
}

func (v entityPromptView[V]) release() {
	v.entity.Release()
}

type promptBuilderFunc func(level PromptLevel, message string, detail Option[string], actions []PromptButton, handle PromptHandle, window *Window, cx *App) RenderablePromptHandle

type promptBuilder interface {
	isPromptBuilder()
}

type promptBuilderDefault struct{}

func (promptBuilderDefault) isPromptBuilder() {}

type promptBuilderCustom struct {
	renderer func(level PromptLevel, message string, detail Option[string], actions []PromptButton, handle PromptHandle, window *Window, cx *App) RenderablePromptHandle
}

func (promptBuilderCustom) isPromptBuilder() {}

func promptBuilderRenderer(builder promptBuilder) promptBuilderFunc {
	switch builder := builder.(type) {
	case promptBuilderCustom:
		return builder.renderer
	default:
		return NewFallbackPromptRenderer
	}
}
