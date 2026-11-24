package gpui

import (
	"fmt"
	"maps"
	"math"
	"reflect"

	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/scheduler"
)

// AnyView is a dynamically typed view handle that can be downcast to a specific Entity[V]. It
// holds a counted entity handle: copying the value aliases it, Clone makes a new one and Release
// drops it.
type AnyView struct {
	entity       AnyEntity
	render       func(view *AnyView, window *Window, cx *App) AnyElement
	viewTypeName string
}

// AnyViewFromEntity converts an entity whose *V implements Render[V] into an AnyView, taking
// over the entity's handle.
func AnyViewFromEntity[V any](value Entity[V]) AnyView {
	mustRender[V]()
	return AnyView{
		entity:       value.IntoAny(),
		render:       anyViewRender[V],
		viewTypeName: typeNameOf(reflect.TypeFor[V]()),
	}
}

func anyViewFromEntity[V any](entity Entity[V]) AnyView {
	return AnyViewFromEntity(entity)
}

func mustRender[V any]() {
	if _, ok := any((*V)(nil)).(Render[V]); !ok {
		panic(fmt.Sprintf("gpui: *%s does not implement Render", reflect.TypeFor[V]()))
	}
}

func typeNameOf(t reflect.Type) string {
	if t.Name() != "" && t.PkgPath() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	return t.String()
}

// Cached embeds this view as a cached ViewElement laid out at style. The rendered subtree is
// recycled from the previous frame unless the backing entity was notified since it was rendered,
// or the window is refreshed.
func (v AnyView) Cached(style StyleRefinement) *ViewElement {
	return v.intoElement().cached(style)
}

// Downgrade converts this to a weak handle.
func (v AnyView) Downgrade() AnyWeakView {
	return AnyWeakView{
		entity:       v.entity.Downgrade(),
		render:       v.render,
		viewTypeName: v.viewTypeName,
	}
}

// Downcast converts this to an Entity[T] aliasing the same handle. It reports false, leaving the
// handle with the caller, when the view is not a T.
func (v AnyView) Downcast[T any]() (Entity[T], bool) {
	return v.entity.Downcast[T]()
}

func anyViewDowncast[V any](view AnyView) (Entity[V], bool) {
	return view.Downcast[V]()
}

// EntityType returns the type of the underlying view's state.
func (v AnyView) EntityType() reflect.Type {
	return v.entity.EntityType()
}

// EntityID returns the id of this view's entity.
func (v AnyView) EntityID() EntityID {
	return v.entity.EntityID()
}

// Clone returns a new counted handle to the same view.
func (v AnyView) Clone() AnyView {
	return AnyView{entity: v.entity.Clone(), render: v.render, viewTypeName: v.viewTypeName}
}

// Release drops this handle.
func (v AnyView) Release() {
	v.entity.Release()
}

// Equal reports whether both handles refer to the same entity.
func (v AnyView) Equal(other AnyView) bool {
	return v.entity.EntityID() == other.entity.EntityID()
}

// IntoAnyElement embeds the view as an element, moving the handle into it.
func (v AnyView) IntoAnyElement() AnyElement {
	return v.intoElement().IntoAnyElement()
}

func (v AnyView) intoElement() *ViewElement {
	return newViewElementWithTypeName(anyViewView{view: v}, v.viewTypeName, v.viewTypeName, CallerSourceLocation(2))
}

type anyViewView struct {
	view AnyView
}

func (v anyViewView) EntityID() (EntityID, bool) {
	return v.view.entity.EntityID(), true
}

func (v anyViewView) Render(window *Window, cx *App) IntoElement {
	defer v.view.Release()
	return v.view.render(&v.view, window, cx)
}

func (v anyViewView) releaseView() {
	v.view.Release()
}

// IntoAnyElement embeds the entity as a view element, moving the handle into it; *T must
// implement Render[T].
func (e Entity[T]) IntoAnyElement() AnyElement {
	mustRender[T]()
	name := typeNameOf(reflect.TypeFor[T]())
	return newViewElementWithTypeName(entityView[T]{entity: e}, name, name, CallerSourceLocation(1)).IntoAnyElement()
}

// Cached embeds this entity as a cached ViewElement laid out at style; *T must implement
// Render[T]. The rendered subtree is reused until the entity is notified or the cached bounds or
// text style change. A cached view is laid out from style and is not measured from its contents.
func (e Entity[T]) Cached(style StyleRefinement) *ViewElement {
	mustRender[T]()
	name := typeNameOf(reflect.TypeFor[T]())
	return newViewElementWithTypeName(entityView[T]{entity: e}, name, name, CallerSourceLocation(1)).cached(style)
}

type entityView[T any] struct {
	entity Entity[T]
}

func (v entityView[T]) EntityID() (EntityID, bool) {
	return v.entity.EntityID(), true
}

func (v entityView[T]) Render(window *Window, cx *App) IntoElement {
	defer v.entity.Release()
	return v.entity.UpdateR(cx, func(this *T, cx *Context[T]) AnyElement {
		return any(this).(Render[T]).Render(window, cx).IntoAnyElement()
	})
}

func (v entityView[T]) releaseView() {
	v.entity.Release()
}

// AnyWeakView is a weak, dynamically typed view handle. It is a plain value.
type AnyWeakView struct {
	entity       AnyWeakEntity
	render       func(view *AnyView, window *Window, cx *App) AnyElement
	viewTypeName string
}

// Upgrade returns a strong AnyView handle, or false if the view has been released.
func (v AnyWeakView) Upgrade() (AnyView, bool) {
	entity, ok := v.entity.Upgrade()
	if !ok {
		return AnyView{}, false
	}
	return AnyView{entity: entity, render: v.render, viewTypeName: v.viewTypeName}, true
}

// AnyWeakViewFromWeakEntity converts a weak entity whose *V implements Render[V] into an
// AnyWeakView.
func AnyWeakViewFromWeakEntity[V any](view WeakEntity[V]) AnyWeakView {
	mustRender[V]()
	return AnyWeakView{
		entity:       view.AnyWeakEntity,
		render:       anyViewRender[V],
		viewTypeName: typeNameOf(reflect.TypeFor[V]()),
	}
}

// Equal reports whether both handles refer to the same entity.
func (v AnyWeakView) Equal(other AnyWeakView) bool {
	return v.entity.EntityID() == other.entity.EntityID()
}

// String formats the handle like upstream's Debug impl.
func (v AnyWeakView) String() string {
	return fmt.Sprintf("AnyWeakView { entity_id: %v, .. }", v.entity.EntityID())
}

func anyViewRender[V any](view *AnyView, window *Window, cx *App) AnyElement {
	entity, ok := view.Clone().Downcast[V]()
	if !ok {
		panic("called `Result::unwrap()` on an `Err` value")
	}
	defer entity.Release()
	if gpuiutil.DebugAssertions {
		window.a11y.viewTypeNames[entity.EntityID()] = typeNameOf(reflect.TypeFor[V]())
	}
	return entity.UpdateR(cx, func(this *V, cx *Context[V]) AnyElement {
		return any(this).(Render[V]).Render(window, cx).IntoAnyElement()
	})
}

// View is a renderable that participates in GPUI's reactive graph. When EntityID reports true,
// that id becomes the view's identity: it gets its own element id space and notifying that
// entity re-renders only this view's subtree. Entities and components are wrapped as views by
// IntoAnyElement and NewComponent; implement View by hand only when a component needs both
// parent-supplied props and a backing entity for identity.
type View interface {
	// EntityID returns this view's identity, if it has one.
	EntityID() (EntityID, bool)
	// Render renders this view into an element tree, consuming it.
	Render(window *Window, cx *App) IntoElement
}

type viewReleaser interface {
	releaseView()
}

type componentView[C RenderOnce] struct {
	component C
}

func (componentView[C]) EntityID() (EntityID, bool) {
	return 0, false
}

func (v componentView[C]) Render(window *Window, cx *App) IntoElement {
	return v.component.Render(window, cx)
}

// NewComponent embeds a RenderOnce component as a stateless view element; it replaces
// #[derive(IntoElement)].
func NewComponent[C RenderOnce](component C) AnyElement {
	name := typeNameOf(reflect.TypeFor[C]())
	return newViewElementWithTypeName(componentView[C]{component: component}, name, name, CallerSourceLocation(1)).IntoAnyElement()
}

// ViewElement is the element type for View implementations: it wraps a View and hooks it into
// layout, prepaint and paint. Construct it with NewViewElement.
type ViewElement struct {
	view           View
	entityID       Option[EntityID]
	cachedStyle    Option[StyleRefinement]
	viewTypeName   string
	componentName  string
	source         *SourceLocation
	dropRegistered bool
}

// NewViewElement wraps a View as an element.
func NewViewElement(view View) *ViewElement {
	name := typeNameOf(reflect.TypeOf(view))
	return newViewElementWithTypeName(view, name, name, CallerSourceLocation(1))
}

func newViewElementWithTypeName(view View, viewTypeName, componentName string, source *SourceLocation) *ViewElement {
	entityID := None[EntityID]()
	if id, ok := view.EntityID(); ok {
		entityID = Some(id)
	}
	return &ViewElement{
		entityID:      entityID,
		cachedStyle:   None[StyleRefinement](),
		viewTypeName:  viewTypeName,
		componentName: componentName,
		view:          view,
		source:        source,
	}
}

func (e *ViewElement) cached(style StyleRefinement) *ViewElement {
	e.cachedStyle = Some(style)
	return e
}

// IntoAnyElement converts the element into an AnyElement.
func (e *ViewElement) IntoAnyElement() AnyElement {
	return NewAnyElement[Option[AnyElement], Option[AnyElement]](e)
}

func (e *ViewElement) takeView() View {
	view := e.view
	if view == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	e.view = nil
	return view
}

func (e *ViewElement) registerDrop(window *Window) {
	if e.dropRegistered {
		return
	}
	if _, ok := e.view.(viewReleaser); !ok {
		return
	}
	e.dropRegistered = true
	window.deferArenaDrop(func() {
		if r, ok := e.view.(viewReleaser); ok {
			e.view = nil
			r.releaseView()
		}
	})
}

type viewElementState struct {
	prepaintRange    prepaintStateRange
	paintRange       paintRange
	cacheKey         viewElementCacheKey
	accessedEntities collections.HashSet[EntityID]
}

type prepaintStateRange struct {
	start, end prepaintStateIndex
}

type paintRange struct {
	start, end paintIndex
}

type viewElementCacheKey struct {
	bounds      Bounds[Pixels]
	contentMask ContentMask[Pixels]
	textStyle   TextStyle
	opacity     float32
}

func renderView(view View, window *Window, cx *App) AnyElement {
	return view.Render(window, cx).IntoAnyElement()
}

// ElementID returns the view's entity id as an ElementIDView, if it has one.
func (e *ViewElement) ElementID() (ElementID, bool) {
	if id, ok := e.entityID.Get(); ok {
		return ElementIDView(id), true
	}
	return ElementID{}, false
}

// SourceLocation returns where the view element was constructed in debug builds.
func (e *ViewElement) SourceLocation() *SourceLocation {
	if gpuiutil.DebugAssertions {
		return e.source
	}
	return nil
}

// RequestLayout renders the view (or reuses its cache) and requests its layout.
func (e *ViewElement) RequestLayout(id *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, Option[AnyElement]) {
	e.registerDrop(window)
	if entityID, ok := e.entityID.Get(); ok {
		var start scheduler.Instant
		if profilerFeatureEnabled {
			start = window.windowProfiler.BeginView()
		}
		layoutID, element := requestLayoutView(entityID, e.cachedStyle, window, cx, func(window *Window, cx *App) AnyElement {
			return renderView(e.takeView(), window, cx)
		})
		if profilerFeatureEnabled {
			window.windowProfiler.EndView(start, e.viewTypeName, ViewPhaseRequestLayout)
		}
		return layoutID, element
	}
	return requestLayoutComponent(e.componentName, window, cx, func(window *Window, cx *App) AnyElement {
		return renderView(e.takeView(), window, cx)
	})
}

// Prepaint prepaints the rendered subtree or replays the cached one.
func (e *ViewElement) Prepaint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], element *Option[AnyElement], window *Window, cx *App) Option[AnyElement] {
	if entityID, ok := e.entityID.Get(); ok {
		var start scheduler.Instant
		if profilerFeatureEnabled {
			start = window.windowProfiler.BeginView()
		}
		prepainted := prepaintView(entityID, globalID, bounds, element, window, cx, func(window *Window, cx *App) AnyElement {
			return renderView(e.takeView(), window, cx)
		})
		if profilerFeatureEnabled {
			window.windowProfiler.EndView(start, e.viewTypeName, ViewPhasePrepaint)
		}
		return prepainted
	}
	return prepaintComponent(e.componentName, element, window, cx)
}

// Paint paints the rendered subtree or replays the cached one.
func (e *ViewElement) Paint(globalID *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *Option[AnyElement], element *Option[AnyElement], window *Window, cx *App) {
	if entityID, ok := e.entityID.Get(); ok {
		var start scheduler.Instant
		if profilerFeatureEnabled {
			start = window.windowProfiler.BeginView()
		}
		paintView(entityID, e.cachedStyle.IsSome(), globalID, element, window, cx)
		if profilerFeatureEnabled {
			window.windowProfiler.EndView(start, e.viewTypeName, ViewPhasePaint)
		}
		return
	}
	paintComponent(e.componentName, element, window, cx)
}

// EmptyView is a view that renders nothing.
type EmptyView struct{}

// Render renders the empty element.
func (*EmptyView) Render(window *Window, cx *Context[EmptyView]) IntoElement {
	return Empty{}
}

func requestLayoutView(entityID EntityID, cachedStyle Option[StyleRefinement], window *Window, cx *App, render func(window *Window, cx *App) AnyElement) (LayoutID, Option[AnyElement]) {
	type result struct {
		layoutID LayoutID
		element  Option[AnyElement]
	}
	r := window.withRenderedView(entityID, func(window *Window) result {
		cachingDisabled := window.IsInspectorPicking(cx)
		if style, ok := cachedStyle.Get(); ok && !cachingDisabled {
			rootStyle := DefaultStyle()
			rootStyle.Refine(&style)
			layoutID := window.RequestLayout(rootStyle, nil, cx)
			return result{layoutID, None[AnyElement]()}
		}
		element := render(window, cx)
		layoutID := element.RequestLayout(window, cx)
		return result{layoutID, Some(element)}
	})
	return r.layoutID, r.element
}

func requestLayoutComponent(name string, window *Window, cx *App, render func(window *Window, cx *App) AnyElement) (LayoutID, Option[AnyElement]) {
	type result struct {
		layoutID LayoutID
		element  Option[AnyElement]
	}
	r := window.WithID(NewElementID(name), func(window *Window) result {
		element := render(window, cx)
		layoutID := element.RequestLayout(window, cx)
		return result{layoutID, Some(element)}
	})
	return r.layoutID, r.element
}

func prepaintView(entityID EntityID, globalID *GlobalElementID, bounds Bounds[Pixels], element *Option[AnyElement], window *Window, cx *App, render func(window *Window, cx *App) AnyElement) Option[AnyElement] {
	window.SetViewID(entityID)
	return window.withRenderedView(entityID, func(window *Window) Option[AnyElement] {
		if el, ok := element.Take().Get(); ok {
			el.Prepaint(window, cx)
			return Some(el)
		}

		if globalID == nil {
			panic("called `Option::unwrap()` on a `None` value")
		}
		return window.WithElementState[viewElementState, Option[AnyElement]](globalID, func(elementState Option[viewElementState], window *Window) (Option[AnyElement], viewElementState) {
			contentMask := window.ContentMask()
			textStyle := window.TextStyle()
			opacity := window.elementOpacity()

			if state, ok := elementState.Get(); ok {
				_, dirty := window.dirtyViews[entityID]
				if state.cacheKey.bounds == bounds &&
					state.cacheKey.contentMask == contentMask &&
					state.cacheKey.textStyle == textStyle &&
					math.Float32bits(state.cacheKey.opacity) == math.Float32bits(opacity) &&
					!dirty &&
					!window.refreshing {
					prepaintStart := window.prepaintIndex()
					window.reusePrepaint(state.prepaintRange.start, state.prepaintRange.end)
					cx.entities.extendAccessed(entityIDsOf(state.accessedEntities))
					prepaintEnd := window.prepaintIndex()
					state.prepaintRange = prepaintStateRange{prepaintStart, prepaintEnd}

					return None[AnyElement](), state
				}
			}

			refreshing := window.refreshing
			window.refreshing = true
			prepaintStart := window.prepaintIndex()
			el, accessedEntities := cx.detectAccessedEntities(func(cx *App) AnyElement {
				el := render(window, cx)
				el.LayoutAsRoot(SizeAvailableSpaceFromPixels(bounds.Size), window, cx)
				el.PrepaintAt(bounds.Origin, window, cx)
				return el
			})

			prepaintEnd := window.prepaintIndex()
			window.refreshing = refreshing

			return Some(el), viewElementState{
				accessedEntities: accessedEntities,
				prepaintRange:    prepaintStateRange{prepaintStart, prepaintEnd},
				paintRange:       paintRange{},
				cacheKey: viewElementCacheKey{
					bounds:      bounds,
					contentMask: contentMask,
					textStyle:   textStyle,
					opacity:     opacity,
				},
			}
		})
	})
}

func entityIDsOf(set collections.HashSet[EntityID]) []EntityID {
	ids := make([]EntityID, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

func prepaintComponent(name string, element *Option[AnyElement], window *Window, cx *App) Option[AnyElement] {
	window.WithID(NewElementID(name), func(window *Window) struct{} {
		optionUnwrapMut(element).Prepaint(window, cx)
		return struct{}{}
	})
	return Some(element.Take().Unwrap())
}

func paintView(entityID EntityID, cached bool, globalID *GlobalElementID, element *Option[AnyElement], window *Window, cx *App) {
	window.withRenderedView(entityID, func(window *Window) struct{} {
		cachingDisabled := window.IsInspectorPicking(cx)
		if cached && !cachingDisabled {
			if globalID == nil {
				panic("called `Option::unwrap()` on a `None` value")
			}
			window.WithElementState[viewElementState, struct{}](globalID, func(elementState Option[viewElementState], window *Window) (struct{}, viewElementState) {
				state := elementState.Unwrap()

				paintStart := window.paintIndex()
				opacity := window.elementOpacity()

				if el := element.AsMut(); el != nil {
					refreshing := window.refreshing
					window.refreshing = true
					_, accessedEntities := cx.detectAccessedEntities(func(cx *App) struct{} {
						el.Paint(window, cx)
						return struct{}{}
					})
					if state.accessedEntities == nil {
						state.accessedEntities = make(collections.HashSet[EntityID], len(accessedEntities))
					}
					maps.Copy(state.accessedEntities, accessedEntities)
					window.refreshing = refreshing
					state.cacheKey.opacity = opacity
				} else {
					window.reusePaint(state.paintRange.start, state.paintRange.end)
					if math.Float32bits(state.cacheKey.opacity) != math.Float32bits(opacity) {
						window.RequestAnimationFrame()
					}
				}

				paintEnd := window.paintIndex()
				state.paintRange = paintRange{paintStart, paintEnd}

				return struct{}{}, state
			})
		} else {
			optionUnwrapMut(element).Paint(window, cx)
		}
		return struct{}{}
	})
}

func paintComponent(name string, element *Option[AnyElement], window *Window, cx *App) {
	window.WithID(ElementIDName(SharedString(name)), func(window *Window) struct{} {
		optionUnwrapMut(element).Paint(window, cx)
		return struct{}{}
	})
}

func optionUnwrapMut[T any](o *Option[T]) *T {
	v := o.AsMut()
	if v == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return v
}
