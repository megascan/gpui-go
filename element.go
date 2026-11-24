package gpui

import (
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

// Element is implemented by types that participate in laying out and painting the contents of a
// window. RL is the state returned from RequestLayout and PP the state returned from Prepaint.
type Element[RL, PP any] interface {
	// ElementID returns the element's unique identifier among the children of its nearest
	// ancestor with an id, if it has one; it is upstream's Element::id.
	ElementID() (ElementID, bool)
	// SourceLocation returns where the element was constructed, or nil.
	SourceLocation() *SourceLocation
	// RequestLayout requests a layout from taffy and initializes the element's state.
	RequestLayout(id *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, RL)
	// Prepaint commits the element's bounds to the current frame.
	Prepaint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *RL, window *Window, cx *App) PP
	// Paint paints the element to the screen.
	Paint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *RL, prepaint *PP, window *Window, cx *App)
}

// ElementA11y is the optional accessibility hook of an Element. Elements that do not implement
// it, or whose A11yRole reports false, are not included in the accessibility tree.
type ElementA11y interface {
	// A11yRole returns the element's accessible role.
	A11yRole() (a11y.Role, bool)
	// WriteA11yInfo writes accessibility properties to node; it is called only when A11yRole
	// reports true.
	WriteA11yInfo(node *a11y.Node)
}

// ElementA11ySyntheticChildren is the optional hook that adds synthetic accessibility children to
// an element that has an id and an accessible role; PP is the element's prepaint state.
type ElementA11ySyntheticChildren[PP any] interface {
	// ElementA11ySyntheticChildren adds synthetic child nodes with builder.
	ElementA11ySyntheticChildren(prepaint *PP, builder *A11ySubtreeBuilder)
}

// IntoElement is implemented by any type that can be converted into an element.
type IntoElement interface {
	// IntoAnyElement converts the value into a dynamically typed AnyElement.
	IntoAnyElement() AnyElement
}

// Render is implemented by *T when entities of type T can be drawn to the screen as views.
type Render[T any] interface {
	// Render renders the view into an element tree.
	Render(window *Window, cx *Context[T]) IntoElement
}

// Render renders the empty element.
func (Empty) Render(window *Window, cx *Context[Empty]) IntoElement {
	return Empty{}
}

// RenderOnce is implemented by components: plain data rendered once into an element tree.
type RenderOnce interface {
	// Render renders the component into an element tree, consuming it.
	Render(window *Window, cx *App) IntoElement
}

// ParentElement is implemented by elements that accept child elements; embed Parent to get
// Child, Children and ChildText.
type ParentElement interface {
	// Extend extends the element's children with elements.
	Extend(elements []AnyElement)
}

// Parent is the ParentElement mixin; bind it with BindParent.
type Parent[S ParentElement] struct {
	self S
}

// BindParent binds a Parent mixin to the element that embeds it.
func BindParent[S ParentElement](m *Parent[S], self S) {
	m.self = self
}

// Child adds a single child element.
func (p *Parent[S]) Child(child IntoElement) S {
	p.self.Extend([]AnyElement{child.IntoAnyElement()})
	return p.self
}

// Children adds multiple child elements.
func (p *Parent[S]) Children(children ...IntoElement) S {
	elements := make([]AnyElement, 0, len(children))
	for _, child := range children {
		elements = append(elements, child.IntoAnyElement())
	}
	p.self.Extend(elements)
	return p.self
}

// ChildText adds a text child, upstream's child("...").
func (p *Parent[S]) ChildText(text string) S {
	return p.Child(SharedString(text))
}

// GlobalElementID is a globally unique identifier for an element, used to track state across
// frames: the path of element ids from the window root with its hash computed once. Equality
// compares the hash and then the full path; it is not comparable with ==.
type GlobalElementID struct {
	ids  []ElementID
	hash uint64
}

const emptyPathHash uint64 = 0

func extendPathHash(prefix uint64, id ElementID) uint64 {
	hasher := FxHasher{hash: prefix}
	id.Hash(&hasher)
	return hasher.Finish()
}

func foldPathHash(ids []ElementID) uint64 {
	hash := emptyPathHash
	for _, id := range ids {
		hash = extendPathHash(hash, id)
	}
	return hash
}

func newGlobalElementID(ids []ElementID) GlobalElementID {
	return globalElementIDWithHash(ids, foldPathHash(ids))
}

func globalElementIDWithHash(ids []ElementID, hash uint64) GlobalElementID {
	if gpuiutil.DebugAssertions {
		if expected := foldPathHash(ids); hash != expected {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", hash, expected))
		}
	}
	return GlobalElementID{ids: slices.Clone(ids), hash: hash}
}

func (g *GlobalElementID) accesskitNodeID() a11y.NodeID {
	return a11y.NodeID(g.hash)
}

// DefaultGlobalElementID returns the id of the empty path; it equals the zero value.
func DefaultGlobalElementID() GlobalElementID {
	return newGlobalElementID(nil)
}

// IDs returns the path of element ids (upstream's Deref to [ElementId]); the slice is read-only.
func (g GlobalElementID) IDs() []ElementID {
	return g.ids
}

// Equal reports whether g and other are the same path.
func (g GlobalElementID) Equal(other GlobalElementID) bool {
	if g.hash != other.hash {
		return false
	}
	if len(g.ids) == len(other.ids) && (len(g.ids) == 0 || unsafe.SliceData(g.ids) == unsafe.SliceData(other.ids)) {
		return true
	}
	return slices.Equal(g.ids, other.ids)
}

// Hash feeds the precomputed path hash to h.
func (g GlobalElementID) Hash(h *FxHasher) {
	h.WriteU64(g.hash)
}

// String joins the ids of the path with ".".
func (g GlobalElementID) String() string {
	var b strings.Builder
	for i, elementID := range g.ids {
		if i > 0 {
			b.WriteString(".")
		}
		b.WriteString(elementID.String())
	}
	return b.String()
}

// SourceLocation is a source position recorded when an element is constructed (upstream's
// &'static panic::Location). Values from CallerSourceLocation are interned per call site.
type SourceLocation struct {
	// File is the source file name.
	File string
	// Line is the 1-based line number.
	Line uint32
	// Column is the 1-based column number, or 0 when unknown.
	Column uint32
}

var sourceLocations sync.Map

// CallerSourceLocation returns a call site, the equivalent of #[track_caller] and
// Location::caller(): skip 0 is the line that calls CallerSourceLocation, 1 the line that called
// that function, and so on. It returns nil when the frame is unknown.
func CallerSourceLocation(skip int) *SourceLocation {
	var pcs [1]uintptr
	if runtime.Callers(skip+2, pcs[:]) == 0 {
		return nil
	}
	if loc, ok := sourceLocations.Load(pcs[0]); ok {
		return loc.(*SourceLocation)
	}
	frame, _ := runtime.CallersFrames(pcs[:]).Next()
	loc := &SourceLocation{File: frame.File, Line: uint32(frame.Line)}
	actual, _ := sourceLocations.LoadOrStore(pcs[0], loc)
	return actual.(*SourceLocation)
}

// String formats the location as file:line:column.
func (l SourceLocation) String() string {
	return l.File + ":" + strconv.FormatUint(uint64(l.Line), 10) + ":" + strconv.FormatUint(uint64(l.Column), 10)
}

// Hash feeds the location to h as Rust's derived Hash for Location does.
func (l SourceLocation) Hash(h *FxHasher) {
	h.WriteString(l.File)
	h.WriteU64(uint64(l.Line))
	h.WriteU64(uint64(l.Column))
}

type elementObject interface {
	innerElement() any
	requestLayout(window *Window, cx *App) LayoutID
	prepaint(window *Window, cx *App)
	paint(window *Window, cx *App)
	layoutAsRoot(availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels]
}

// Drawable is a wrapper around an Element that allows it to be drawn in a window.
type Drawable[RL, PP any] struct {
	// Element is the drawn element.
	Element Element[RL, PP]
	phase   elementDrawPhase[RL, PP]
}

type elementDrawPhaseKind uint8

const (
	elementDrawPhaseStart elementDrawPhaseKind = iota
	elementDrawPhaseRequestLayout
	elementDrawPhaseLayoutComputed
	elementDrawPhasePrepaint
	elementDrawPhasePainted
)

type elementDrawPhase[RL, PP any] struct {
	kind           elementDrawPhaseKind
	layoutID       LayoutID
	nodeID         dispatchNodeID
	globalID       *GlobalElementID
	inspectorID    *InspectorElementID
	availableSpace SizeOf[AvailableSpace]
	bounds         Bounds[Pixels]
	requestLayout  RL
	prepaint       PP
}

func newDrawable[RL, PP any](element Element[RL, PP]) *Drawable[RL, PP] {
	return &Drawable[RL, PP]{
		Element: element,
		phase:   elementDrawPhase[RL, PP]{kind: elementDrawPhaseStart},
	}
}

func (d *Drawable[RL, PP]) takePhase() elementDrawPhase[RL, PP] {
	phase := d.phase
	d.phase = elementDrawPhase[RL, PP]{}
	return phase
}

func (d *Drawable[RL, PP]) requestLayout(window *Window, cx *App) LayoutID {
	phase := d.takePhase()
	if phase.kind != elementDrawPhaseStart {
		panic("must call request_layout only once")
	}

	var globalID *GlobalElementID
	if elementID, ok := d.Element.ElementID(); ok {
		id := prepareElementID(elementID, window)
		globalID = &id
	}

	var inspectorID *InspectorElementID
	if window.inspectorEnabled() {
		if source := d.Element.SourceLocation(); source != nil {
			id := prepareInspectorID(source, window)
			inspectorID = &id
		}
	}

	layoutID, requestLayout := d.Element.RequestLayout(globalID, inspectorID, window, cx)

	if globalID != nil {
		window.elementIDStack.pop()
	}

	d.phase = elementDrawPhase[RL, PP]{
		kind:          elementDrawPhaseRequestLayout,
		layoutID:      layoutID,
		globalID:      globalID,
		inspectorID:   inspectorID,
		requestLayout: requestLayout,
	}
	return layoutID
}

func debugAssertIDStack(globalID *GlobalElementID, window *Window) {
	if !gpuiutil.DebugAssertions {
		return
	}
	if globalID == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	if !slices.Equal(globalID.ids, window.elementIDStack.ids) {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %#v\n right: %#v", globalID.ids, window.elementIDStack.ids))
	}
}

func (d *Drawable[RL, PP]) prepaint(window *Window, cx *App) {
	phase := d.takePhase()
	if phase.kind != elementDrawPhaseRequestLayout && phase.kind != elementDrawPhaseLayoutComputed {
		panic("must call request_layout before prepaint")
	}
	layoutID := phase.layoutID
	globalID := phase.globalID
	inspectorID := phase.inspectorID
	requestLayout := phase.requestLayout

	if elementID, ok := d.Element.ElementID(); ok {
		window.elementIDStack.push(elementID)
		debugAssertIDStack(globalID, window)
	}

	bounds := window.LayoutBounds(layoutID)
	pushedA11yNode := false
	if window.a11y.isActive() {
		if globalID != nil {
			if withA11y, ok := any(d.Element).(ElementA11y); ok {
				if role, ok := withA11y.A11yRole(); ok {
					nodeID := globalID.accesskitNodeID()
					node := a11y.NewNode(role)
					scale := window.ScaleFactor()
					node.SetBounds(a11y.Rect{
						X0: float64(float32(bounds.Origin.X) * scale),
						Y0: float64(float32(bounds.Origin.Y) * scale),
						X1: float64((float32(bounds.Origin.X) + float32(bounds.Size.Width)) * scale),
						Y1: float64((float32(bounds.Origin.Y) + float32(bounds.Size.Height)) * scale),
					})
					withA11y.WriteA11yInfo(&node)
					window.a11y.nodeBounds[nodeID] = bounds
					pushedA11yNode = window.a11y.nodes.push(nodeID, node)
					if gpuiutil.DebugAssertions && pushedA11yNode {
						window.a11y.nodes.recordNodeInfo(nodeID, nodeDebugInfo{
							synthetic:      false,
							view:           a11yViewTypeName(window),
							elementID:      lastElementIDDebug(globalID),
							sourceLocation: d.Element.SourceLocation(),
						})
					}
				}
			}
		}
	}

	nodeID := window.nextFrame.dispatchTree.pushNode()
	prepaint := d.Element.Prepaint(globalID, inspectorID, bounds, &requestLayout, window, cx)
	window.nextFrame.dispatchTree.popNode()

	if pushedA11yNode {
		if globalID != nil {
			builder := newA11ySubtreeBuilder(globalID.accesskitNodeID(), &window.a11y.nodes)
			if gpuiutil.DebugAssertions {
				builder = builder.withCreator(nodeCreator{
					view:           a11yViewTypeName(window),
					elementID:      lastElementIDDebug(globalID),
					sourceLocation: d.Element.SourceLocation(),
				})
			}
			if synthetic, ok := any(d.Element).(ElementA11ySyntheticChildren[PP]); ok {
				synthetic.ElementA11ySyntheticChildren(&prepaint, &builder)
			}
		}
		window.a11y.nodes.pop()
	}

	if globalID != nil {
		window.elementIDStack.pop()
	}

	d.phase = elementDrawPhase[RL, PP]{
		kind:          elementDrawPhasePrepaint,
		nodeID:        nodeID,
		globalID:      globalID,
		inspectorID:   inspectorID,
		bounds:        bounds,
		requestLayout: requestLayout,
		prepaint:      prepaint,
	}
}

func a11yViewTypeName(window *Window) Option[string] {
	if name, ok := window.a11y.viewTypeNames[window.CurrentView()]; ok {
		return Some(name)
	}
	return None[string]()
}

func lastElementIDDebug(globalID *GlobalElementID) Option[string] {
	if len(globalID.ids) == 0 {
		return None[string]()
	}
	return Some(fmt.Sprintf("%#v", globalID.ids[len(globalID.ids)-1]))
}

func (d *Drawable[RL, PP]) paint(window *Window, cx *App) (RL, PP) {
	phase := d.takePhase()
	if phase.kind != elementDrawPhasePrepaint {
		panic("must call prepaint before paint")
	}
	globalID := phase.globalID
	requestLayout := phase.requestLayout
	prepaint := phase.prepaint

	if elementID, ok := d.Element.ElementID(); ok {
		window.elementIDStack.push(elementID)
		debugAssertIDStack(globalID, window)
	}

	window.nextFrame.dispatchTree.setActiveNode(phase.nodeID)
	d.Element.Paint(globalID, phase.inspectorID, phase.bounds, &requestLayout, &prepaint, window, cx)

	if globalID != nil {
		window.elementIDStack.pop()
	}

	d.phase = elementDrawPhase[RL, PP]{kind: elementDrawPhasePainted}
	return requestLayout, prepaint
}

func (d *Drawable[RL, PP]) layoutAsRoot(availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels] {
	if d.phase.kind == elementDrawPhaseStart {
		d.requestLayout(window, cx)
	}

	phase := d.takePhase()
	var layoutID LayoutID
	switch phase.kind {
	case elementDrawPhaseRequestLayout:
		layoutID = phase.layoutID
		window.ComputeLayout(layoutID, availableSpace, cx)
	case elementDrawPhaseLayoutComputed:
		layoutID = phase.layoutID
		if availableSpace != phase.availableSpace {
			window.ComputeLayout(layoutID, availableSpace, cx)
		}
	default:
		panic("cannot measure after painting")
	}
	d.phase = elementDrawPhase[RL, PP]{
		kind:           elementDrawPhaseLayoutComputed,
		layoutID:       layoutID,
		globalID:       phase.globalID,
		inspectorID:    phase.inspectorID,
		availableSpace: availableSpace,
		requestLayout:  phase.requestLayout,
	}

	return window.LayoutBounds(layoutID).Size
}

type drawableObject[RL, PP any] struct {
	*Drawable[RL, PP]
}

func (d drawableObject[RL, PP]) innerElement() any {
	return d.Element
}

func (d drawableObject[RL, PP]) paint(window *Window, cx *App) {
	d.Drawable.paint(window, cx)
}

// AnyElement is a dynamically typed element that can be used to store any element type. Copying
// the value aliases the same element, as a Rust move does.
type AnyElement struct {
	element elementObject
}

// NewAnyElement boxes element into an AnyElement, erasing its concrete type.
func NewAnyElement[RL, PP any](element Element[RL, PP]) AnyElement {
	return AnyElement{element: drawableObject[RL, PP]{newDrawable(element)}}
}

// IntoAnyElement returns the element itself.
func (e AnyElement) IntoAnyElement() AnyElement {
	return e
}

// DowncastMut returns the boxed element as a *T when the element stored in e is a *T.
func (e *AnyElement) DowncastMut[T any]() (*T, bool) {
	inner, ok := e.element.innerElement().(*T)
	return inner, ok
}

// RequestLayout requests the layout id of the element stored in this AnyElement; parents use it
// to lay out their children.
func (e *AnyElement) RequestLayout(window *Window, cx *App) LayoutID {
	return e.element.requestLayout(window, cx)
}

// Prepaint prepares the element to be painted by storing its bounds, giving it a chance to draw
// hitboxes and request autoscroll before the final paint pass is confirmed.
func (e *AnyElement) Prepaint(window *Window, cx *App) {
	e.element.prepaint(window, cx)
}

// Paint paints the element stored in this AnyElement.
func (e *AnyElement) Paint(window *Window, cx *App) {
	e.element.paint(window, cx)
}

// LayoutAsRoot performs layout for this element within availableSpace and returns its size.
func (e *AnyElement) LayoutAsRoot(availableSpace SizeOf[AvailableSpace], window *Window, cx *App) Size[Pixels] {
	return e.element.layoutAsRoot(availableSpace, window, cx)
}

// PrepaintAt prepaints this element at the given absolute origin.
func (e *AnyElement) PrepaintAt(origin Point[Pixels], window *Window, cx *App) {
	window.WithAbsoluteElementOffset(origin, func(window *Window) struct{} {
		e.Prepaint(window, cx)
		return struct{}{}
	})
}

// PrepaintAsRoot performs layout on this element in availableSpace, then prepaints it at the
// given absolute origin.
func (e *AnyElement) PrepaintAsRoot(origin Point[Pixels], availableSpace SizeOf[AvailableSpace], window *Window, cx *App) {
	e.LayoutAsRoot(availableSpace, window, cx)
	window.WithAbsoluteElementOffset(origin, func(window *Window) struct{} {
		e.Prepaint(window, cx)
		return struct{}{}
	})
}

// Empty is the empty element, which renders nothing.
type Empty struct{}

// ElementID reports that Empty has no id.
func (Empty) ElementID() (ElementID, bool) {
	return ElementID{}, false
}

// SourceLocation returns nil.
func (Empty) SourceLocation() *SourceLocation {
	return nil
}

// RequestLayout requests a layout node with display none.
func (Empty) RequestLayout(id *GlobalElementID, inspectorID *InspectorElementID, window *Window, cx *App) (LayoutID, struct{}) {
	style := DefaultStyle()
	style.Display = DisplayNone
	return window.RequestLayout(style, nil, cx), struct{}{}
}

// Prepaint does nothing.
func (Empty) Prepaint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], state *struct{}, window *Window, cx *App) struct{} {
	return struct{}{}
}

// Paint does nothing.
func (Empty) Paint(id *GlobalElementID, inspectorID *InspectorElementID, bounds Bounds[Pixels], requestLayout *struct{}, prepaint *struct{}, window *Window, cx *App) {
}

// IntoAnyElement converts the element into an AnyElement.
func (e Empty) IntoAnyElement() AnyElement {
	return NewAnyElement[struct{}, struct{}](e)
}

func prepareElementID(elementID ElementID, window *Window) GlobalElementID {
	window.elementIDStack.push(elementID)
	return window.elementIDStack.globalID()
}

func prepareInspectorID(source *SourceLocation, window *Window) InspectorElementID {
	path := InspectorElementPath{
		GlobalID:       window.elementIDStack.globalID(),
		SourceLocation: source,
	}
	return window.buildInspectorElementID(path)
}
