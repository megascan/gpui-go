package bus

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/atspi"
	"github.com/megascan/gpui-go/a11y/consumer"
)

type callback struct {
	messages *messageQueue
}

func newCallback() callback {
	return callback{messages: getOrInitMessages()}
}

func (c callback) sendMessage(m message) {
	c.messages.trySend(m)
}

func (c callback) RegisterInterfaces(adapter *atspi.Adapter, id consumer.NodeID, interfaces atspi.InterfaceSet) {
	node := adapter.PlatformNode(id)
	c.sendMessage(messageRegisterInterfaces{node: node, interfaces: interfaces})
}

func (c callback) UnregisterInterfaces(adapter *atspi.Adapter, id consumer.NodeID, interfaces atspi.InterfaceSet) {
	c.sendMessage(messageUnregisterInterfaces{adapterID: adapter.ID(), nodeID: id, interfaces: interfaces})
}

func (c callback) EmitEvent(adapter *atspi.Adapter, event atspi.Event) {
	if cache, ok := event.(atspi.EventCache); ok {
		switch e := cache.Event.(type) {
		case atspi.CacheEventAdded:
			c.sendMessage(messageEmitCacheAdd{node: adapter.PlatformNode(e.ID)})
			return
		case atspi.CacheEventRemoved:
			c.sendMessage(messageEmitCacheRemove{adapterID: adapter.ID(), nodeID: e.ID})
			return
		}
	}
	c.sendMessage(messageEmitEvent{adapterID: adapter.ID(), event: event})
}

type adapterStateKind uint8

const (
	adapterStateInactive adapterStateKind = iota
	adapterStatePending
	adapterStateActive
)

type adapterState struct {
	mu               sync.Mutex
	kind             adapterStateKind
	isWindowFocused  bool
	rootWindowBounds atspi.WindowBounds
	actionHandler    atspi.ActionHandlerNoMut
	impl             *atspi.Adapter
	refs             atomic.Int32
}

func (s *adapterState) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stringLocked()
}

func (s *adapterState) stringLocked() string {
	switch s.kind {
	case adapterStateInactive:
		return fmt.Sprintf("Inactive { is_window_focused: %v, root_window_bounds: %+v, action_handler: \"ActionHandler\" }", s.isWindowFocused, s.rootWindowBounds)
	case adapterStatePending:
		return fmt.Sprintf("Pending { is_window_focused: %v, root_window_bounds: %+v, action_handler: \"ActionHandler\" }", s.isWindowFocused, s.rootWindowBounds)
	default:
		return fmt.Sprintf("Active(%v)", s.impl)
	}
}

func (s *adapterState) setInactiveLocked(isWindowFocused bool, rootWindowBounds atspi.WindowBounds, actionHandler atspi.ActionHandlerNoMut) {
	impl := s.impl
	s.kind = adapterStateInactive
	s.isWindowFocused = isWindowFocused
	s.rootWindowBounds = rootWindowBounds
	s.actionHandler = actionHandler
	s.impl = nil
	if impl != nil {
		impl.Release()
	}
}

func (s *adapterState) setActiveLocked(impl *atspi.Adapter) {
	s.kind = adapterStateActive
	s.impl = impl
	s.actionHandler = nil
}

func (s *adapterState) release() {
	if s.refs.Add(-1) != 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.impl != nil {
		impl := s.impl
		s.impl = nil
		impl.Release()
	}
}

// Adapter exposes one window's accessibility tree on the AT-SPI bus.
type Adapter struct {
	messages *messageQueue
	id       int
	state    *adapterState
	released bool
}

// String formats the adapter like upstream's Debug impl.
func (a *Adapter) String() string {
	return fmt.Sprintf("Adapter { messages: %p, id: %d, state: %v }", a.messages, a.id, a.state)
}

// NewAdapter creates a Unix adapter. All of the handlers are always called
// from another goroutine.
func NewAdapter(activationHandler a11y.ActivationHandler, actionHandler a11y.ActionHandler, deactivationHandler a11y.DeactivationHandler) *Adapter {
	id := atspi.NextAdapterID()
	messages := getOrInitMessages()
	state := &adapterState{
		kind:          adapterStateInactive,
		actionHandler: atspi.NewActionHandlerWrapper(actionHandler),
	}
	state.refs.Store(2)
	adapter := &Adapter{id: id, messages: messages, state: state}
	adapter.sendMessage(messageAddAdapter{
		id:                  id,
		activationHandler:   activationHandler,
		deactivationHandler: deactivationHandler,
		state:               state,
	})
	return adapter
}

func (a *Adapter) sendMessage(m message) {
	a.messages.trySend(m)
}

func (a *Adapter) checkLive() {
	if a.released {
		panic("bus: Adapter used after Release")
	}
}

// SetRootWindowBounds sets the bounds of the top-level window. The outer
// bounds contain any window decoration and borders. Since an application can
// not get the position of its window under Wayland, this only makes sense
// under X11.
func (a *Adapter) SetRootWindowBounds(outer, inner a11y.Rect) {
	a.checkLive()
	newBounds := atspi.NewWindowBounds(outer, inner)
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	switch a.state.kind {
	case adapterStateInactive, adapterStatePending:
		a.state.rootWindowBounds = newBounds
	default:
		a.state.impl.SetRootWindowBounds(newBounds)
	}
}

// UpdateIfActive calls updateFactory and applies the resulting update if and
// only if the tree has been initialized. If the activation handler initially
// returned no tree, the update must contain a full tree.
func (a *Adapter) UpdateIfActive(updateFactory func() a11y.TreeUpdate) {
	a.checkLive()
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	switch a.state.kind {
	case adapterStateInactive:
	case adapterStatePending:
		initialState := updateFactory()
		impl := atspi.NewAdapterWithWrappedActionHandler(
			a.id,
			getOrInitAppContext(),
			newCallback(),
			initialState,
			a.state.isWindowFocused,
			a.state.rootWindowBounds,
			a.state.actionHandler,
		)
		a.state.setActiveLocked(impl)
	default:
		a.state.impl.Update(updateFactory())
	}
}

// UpdateWindowFocusState updates the tree state based on whether the window is focused.
func (a *Adapter) UpdateWindowFocusState(isFocused bool) {
	a.checkLive()
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	switch a.state.kind {
	case adapterStateInactive, adapterStatePending:
		a.state.isWindowFocused = isFocused
	default:
		a.state.impl.UpdateWindowFocusState(isFocused)
	}
}

// Release is upstream's Drop: it removes the adapter from the event loop. It is idempotent.
func (a *Adapter) Release() {
	if a.released {
		return
	}
	a.released = true
	a.sendMessage(messageRemoveAdapter{id: a.id})
	a.state.release()
}

type message interface{ isMessage() }

type messageAddAdapter struct {
	id                  int
	activationHandler   a11y.ActivationHandler
	deactivationHandler a11y.DeactivationHandler
	state               *adapterState
}

type messageRemoveAdapter struct {
	id int
}

type messageRegisterInterfaces struct {
	node       atspi.PlatformNode
	interfaces atspi.InterfaceSet
}

type messageUnregisterInterfaces struct {
	adapterID  int
	nodeID     consumer.NodeID
	interfaces atspi.InterfaceSet
}

type messageEmitEvent struct {
	adapterID int
	event     atspi.Event
}

type messageEmitCacheAdd struct {
	node atspi.PlatformNode
}

type messageEmitCacheRemove struct {
	adapterID int
	nodeID    consumer.NodeID
}

func (messageAddAdapter) isMessage()           {}
func (messageRemoveAdapter) isMessage()        {}
func (messageRegisterInterfaces) isMessage()   {}
func (messageUnregisterInterfaces) isMessage() {}
func (messageEmitEvent) isMessage()            {}
func (messageEmitCacheAdd) isMessage()         {}
func (messageEmitCacheRemove) isMessage()      {}

func dropMessage(m message) {
	if add, ok := m.(messageAddAdapter); ok {
		add.state.release()
	}
}
