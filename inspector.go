package gpui

import (
	"reflect"
	"slices"
)

// InspectorElementID is a unique identifier for an element that can be inspected. Compare ids
// with Equal; Path is shared between ids built for the same path.
type InspectorElementID struct {
	// Path is the stable part of the id.
	Path *InspectorElementPath
	// InstanceID disambiguates elements that have the same path.
	InstanceID int
}

// Equal reports whether id and other have equal paths and instance ids.
func (id InspectorElementID) Equal(other InspectorElementID) bool {
	if id.InstanceID != other.InstanceID {
		return false
	}
	if id.Path == other.Path {
		return true
	}
	if id.Path == nil || other.Path == nil {
		return false
	}
	return id.Path.Equal(*other.Path)
}

// Hash feeds the id to h as upstream's derived Hash does.
func (id InspectorElementID) Hash(h *FxHasher) {
	if id.Path != nil {
		id.Path.Hash(h)
	}
	h.WriteUsize(uint64(id.InstanceID))
}

// InspectorElementPath is a GlobalElementID qualified by the source location of the element's
// construction.
type InspectorElementPath struct {
	// GlobalID is the path to the nearest ancestor element that has an ElementID.
	GlobalID GlobalElementID
	// SourceLocation is where this element was constructed.
	SourceLocation *SourceLocation
}

// Equal reports whether p and other are the same path and source location.
func (p InspectorElementPath) Equal(other InspectorElementPath) bool {
	if !p.GlobalID.Equal(other.GlobalID) {
		return false
	}
	if p.SourceLocation == other.SourceLocation {
		return true
	}
	if p.SourceLocation == nil || other.SourceLocation == nil {
		return false
	}
	return *p.SourceLocation == *other.SourceLocation
}

// Hash feeds the path to h as upstream's derived Hash does.
func (p InspectorElementPath) Hash(h *FxHasher) {
	p.GlobalID.Hash(h)
	if p.SourceLocation != nil {
		p.SourceLocation.Hash(h)
	}
}

// InspectorRenderer renders the inspector UI; it is set on the App.
type InspectorRenderer func(inspector *Inspector, window *Window, cx *Context[Inspector]) AnyElement

// Inspector manages inspector state: which element is currently selected and whether the
// inspector is in picking mode.
type Inspector struct {
	activeElement *inspectedElement
	pickDepth     Option[float32]
	renderers     map[reflect.Type]inspectorElementRenderer
}

type inspectedElement struct {
	id     InspectorElementID
	states inspectorStates
}

type inspectorStates struct {
	order  []reflect.Type
	values map[reflect.Type]any
}

func (s *inspectorStates) remove(t reflect.Type) (any, bool) {
	v, ok := s.values[t]
	if !ok {
		return nil, false
	}
	delete(s.values, t)
	return v, true
}

func (s *inspectorStates) insert(t reflect.Type, v any) {
	if s.values == nil {
		s.values = make(map[reflect.Type]any)
	}
	if !slices.Contains(s.order, t) {
		s.order = append(s.order, t)
	}
	s.values[t] = v
}

func newInspectedElement(id InspectorElementID) *inspectedElement {
	return &inspectedElement{id: id}
}

func newInspector() *Inspector {
	return &Inspector{
		activeElement: nil,
		pickDepth:     Some[float32](0.0),
		renderers:     make(map[reflect.Type]inspectorElementRenderer),
	}
}

func (i *Inspector) selectElement(id InspectorElementID, window *Window) {
	i.setActiveElementID(id, window)
	i.pickDepth = None[float32]()
}

func (i *Inspector) hover(id InspectorElementID, window *Window) {
	if i.IsPicking() {
		changed := i.setActiveElementID(id, window)
		if changed {
			i.pickDepth = Some[float32](0.0)
		}
	}
}

func (i *Inspector) setActiveElementID(id InspectorElementID, window *Window) bool {
	active, ok := i.ActiveElementID()
	changed := !ok || !id.Equal(*active)
	if changed {
		i.activeElement = newInspectedElement(id)
		window.Refresh()
	}
	return changed
}

// ActiveElementID returns the id of the currently hovered or selected element.
func (i *Inspector) ActiveElementID() (*InspectorElementID, bool) {
	if i.activeElement == nil {
		return nil, false
	}
	return &i.activeElement.id, true
}

func (i *Inspector) withActiveElementState[T, R any](window *Window, f func(state *Option[T], window *Window) R) R {
	activeElement := i.activeElement
	if activeElement == nil {
		state := None[T]()
		return f(&state, window)
	}

	typeID := reflect.TypeFor[T]()
	inspectorState := None[T]()
	if state, ok := activeElement.states.remove(typeID); ok {
		inspectorState = Some(*state.(*T))
	}

	result := f(&inspectorState, window)

	if state, ok := inspectorState.Get(); ok {
		activeElement.states.insert(typeID, &state)
	}

	return result
}

// StartPicking starts element picking mode, allowing the user to select elements by clicking.
func (i *Inspector) StartPicking() {
	i.pickDepth = Some[float32](0.0)
}

// IsPicking reports whether the inspector is in picking mode.
func (i *Inspector) IsPicking() bool {
	return i.pickDepth.IsSome()
}

// RenderInspectorStates renders elements for all registered inspector states of the active
// inspector element, in the order the states were first stored.
func (i *Inspector) RenderInspectorStates(window *Window, cx *Context[Inspector]) []AnyElement {
	var elements []AnyElement
	if activeElement := i.activeElement; activeElement != nil {
		i.activeElement = nil
		for _, typeID := range activeElement.states.order {
			state, ok := activeElement.states.values[typeID]
			if !ok {
				continue
			}
			renderer, ok := i.renderers[typeID]
			if !ok {
				factory, ok := cx.inspectorElementRegistry.factoriesByTypeID[typeID]
				if !ok {
					continue
				}
				delete(cx.inspectorElementRegistry.factoriesByTypeID, typeID)
				renderer = factory(window, cx.App)
				cx.inspectorElementRegistry.factoriesByTypeID[typeID] = factory
				i.renderers[typeID] = renderer
			}
			elements = append(elements, renderer(activeElement.id, state, window, cx.App))
		}

		i.activeElement = activeElement
	}

	return elements
}

// Render renders the inspector with the App's InspectorRenderer, or nothing when none is set.
func (i *Inspector) Render(window *Window, cx *Context[Inspector]) IntoElement {
	if inspectorRenderer, ok := cx.inspectorRenderer.Take().Get(); ok {
		result := inspectorRenderer(i, window, cx)
		cx.inspectorRenderer = Some(inspectorRenderer)
		return result
	}
	return Empty{}.IntoAnyElement()
}

type inspectorElementRegistry struct {
	factoriesByTypeID map[reflect.Type]func(window *Window, cx *App) inspectorElementRenderer
}

func defaultInspectorElementRegistry() inspectorElementRegistry {
	return inspectorElementRegistry{
		factoriesByTypeID: make(map[reflect.Type]func(window *Window, cx *App) inspectorElementRenderer),
	}
}

func (r *inspectorElementRegistry) register[T any, R IntoElement](factory func(window *Window, cx *App) func(id InspectorElementID, state *T, window *Window, cx *App) R) {
	if r.factoriesByTypeID == nil {
		r.factoriesByTypeID = make(map[reflect.Type]func(window *Window, cx *App) inspectorElementRenderer)
	}
	r.factoriesByTypeID[reflect.TypeFor[T]()] = func(window *Window, cx *App) inspectorElementRenderer {
		renderer := factory(window, cx)
		return func(id InspectorElementID, value any, window *Window, cx *App) AnyElement {
			state, ok := value.(*T)
			if !ok {
				panic("registered inspector state type")
			}
			return renderer(id, state, window, cx).IntoAnyElement()
		}
	}
}

type inspectorElementRenderer func(id InspectorElementID, value any, window *Window, cx *App) AnyElement

// FunctionReflection is the reification of a function with the signature func(T) T: its name,
// documentation and the ability to invoke it.
type FunctionReflection[T any] struct {
	// Name is the name of the function.
	Name string
	// Function is the method.
	Function func(value any) any
	// Documentation is the documentation for the function.
	Documentation Option[string]
}

// Invoke invokes this method on value and returns the result.
func (f FunctionReflection[T]) Invoke(value T) T {
	result, ok := f.Function(value).(T)
	if !ok {
		panic("Type mismatch in reflection invoke: Any { .. }")
	}
	return result
}
