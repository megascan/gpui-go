package bus

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/atspi"
	"github.com/megascan/gpui-go/gpuiutil"
)

const (
	statusInterface = "org.a11y.Status"
	statusService   = "org.a11y.Bus"
	statusPath      = "/org/a11y/bus"
)

var (
	appContextOnce sync.Once
	appContext     *atspi.AppContext
	messagesOnce   sync.Once
	messages       *messageQueue
)

func appName() gpuiutil.Option[string] {
	path, err := os.Executable()
	if err != nil {
		return gpuiutil.None[string]()
	}
	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) {
		return gpuiutil.None[string]()
	}
	return gpuiutil.Some(name)
}

func getOrInitAppContext() *atspi.AppContext {
	appContextOnce.Do(func() {
		appContext = atspi.NewAppContext(appName())
	})
	return appContext
}

type messageQueue struct {
	mu     sync.Mutex
	items  []message
	closed bool
	notify chan struct{}
	stop   chan struct{}
	done   chan struct{}
}

func newMessageQueue() *messageQueue {
	return &messageQueue{notify: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
}

func (q *messageQueue) trySend(m message) {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		dropMessage(m)
		return
	}
	q.items = append(q.items, m)
	q.mu.Unlock()
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *messageQueue) next() (message, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil, false
	}
	m := q.items[0]
	q.items[0] = nil
	q.items = q.items[1:]
	return m, true
}

func (q *messageQueue) close() {
	q.mu.Lock()
	q.closed = true
	items := q.items
	q.items = nil
	q.mu.Unlock()
	for _, m := range items {
		dropMessage(m)
	}
}

var connectSessionBus = func() (*dbus.Conn, error) {
	conn, err := dbus.SessionBusPrivateNoAutoStartup()
	if err != nil {
		return nil, err
	}
	if err := conn.Auth(nil); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := conn.Hello(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func getOrInitMessages() *messageQueue {
	messagesOnce.Do(func() {
		q := newMessageQueue()
		messages = q
		go func() {
			defer close(q.done)
			sessionBus, err := connectSessionBus()
			if err != nil {
				q.close()
				return
			}
			var adapters []adapterEntry
			err = runEventLoop(sessionBus, q, &adapters)
			q.close()
			for _, entry := range adapters {
				entry.state.release()
			}
			_ = sessionBus.Close()
			if err != nil && zbusErrorIsUnrecoverable(err) {
				fmt.Fprintf(os.Stderr, "Accessibility event loop failed: %v\n", err)
			}
		}()
	})
	return messages
}

type adapterEntry struct {
	id                  int
	activationHandler   a11y.ActivationHandler
	deactivationHandler a11y.DeactivationHandler
	state               *adapterState
}

func activateAdapter(entry *adapterEntry) {
	state := entry.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.kind != adapterStateInactive {
		return
	}
	if initialState, ok := entry.activationHandler.RequestInitialTree(); ok {
		impl := atspi.NewAdapterWithWrappedActionHandler(
			entry.id,
			getOrInitAppContext(),
			newCallback(),
			initialState,
			state.isWindowFocused,
			state.rootWindowBounds,
			state.actionHandler,
		)
		state.setActiveLocked(impl)
	} else {
		state.kind = adapterStatePending
	}
}

func deactivateAdapter(entry *adapterEntry) {
	state := entry.state
	state.mu.Lock()
	switch state.kind {
	case adapterStateInactive:
		state.mu.Unlock()
	case adapterStatePending:
		state.setInactiveLocked(state.isWindowFocused, state.rootWindowBounds, state.actionHandler)
		state.mu.Unlock()
		entry.deactivationHandler.DeactivateAccessibility()
	default:
		impl := state.impl
		state.setInactiveLocked(impl.IsWindowFocused(), impl.RootWindowBounds(), impl.WrappedActionHandler())
		state.mu.Unlock()
		entry.deactivationHandler.DeactivateAccessibility()
	}
}

type statusChange struct {
	value   bool
	present bool
}

func (c statusChange) get(sessionBus *dbus.Conn) (bool, error) {
	if c.present {
		return c.value, nil
	}
	return getIsEnabled(sessionBus)
}

func getIsEnabled(sessionBus *dbus.Conn) (bool, error) {
	variant, err := sessionBus.Object(statusService, statusPath).GetProperty(statusInterface + ".IsEnabled")
	if err != nil {
		return false, err
	}
	value, ok := variant.Value().(bool)
	if !ok {
		return false, fmt.Errorf("org.a11y.Status.IsEnabled has signature %s, want b", variant.Signature())
	}
	return value, nil
}

func busAfterStatusChange(change statusChange, sessionBus *dbus.Conn) (*Bus, error) {
	enabled, err := change.get(sessionBus)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}
	bus, err := newBus(sessionBus)
	return mapOrIgnoringRecoverableError(bus, err, (*Bus)(nil), func(bus *Bus) *Bus { return bus })
}

func logRecoverable(err error) error {
	if zbusErrorIsUnrecoverable(err) {
		return err
	}
	fmt.Fprintf(os.Stderr, "Accessibility status change failed: %v\n", err)
	return nil
}

func syncAdapters(adapters []adapterEntry, atspiBus *Bus) {
	active := atspiBus != nil
	for n := range adapters {
		if active {
			activateAdapter(&adapters[n])
		} else {
			deactivateAdapter(&adapters[n])
		}
	}
}

type statusStream struct {
	sessionBus *dbus.Conn
	signals    chan *dbus.Signal
	initial    []statusChange
}

func newStatusStream(sessionBus *dbus.Conn) (*statusStream, error) {
	stream := &statusStream{sessionBus: sessionBus, signals: make(chan *dbus.Signal, 16)}
	if err := sessionBus.AddMatchSignal(
		dbus.WithMatchSender(statusService),
		dbus.WithMatchObjectPath(statusPath),
		dbus.WithMatchInterface(propertiesInterface),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchArg(0, statusInterface),
	); err != nil {
		return nil, err
	}
	if err := sessionBus.AddMatchSignal(
		dbus.WithMatchSender("org.freedesktop.DBus"),
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, statusService),
	); err != nil {
		return nil, err
	}
	sessionBus.Signal(stream.signals)
	if value, err := getIsEnabled(sessionBus); err == nil {
		stream.initial = append(stream.initial, statusChange{value: value, present: true})
	}
	return stream, nil
}

func (s *statusStream) changes(signal *dbus.Signal) []statusChange {
	switch signal.Name {
	case propertiesInterface + ".PropertiesChanged":
		if string(signal.Path) != statusPath || len(signal.Body) < 3 {
			return nil
		}
		if iface, _ := signal.Body[0].(string); iface != statusInterface {
			return nil
		}
		if changed, ok := signal.Body[1].(map[string]dbus.Variant); ok {
			if value, ok := changed["IsEnabled"]; ok {
				enabled, isBool := value.Value().(bool)
				return []statusChange{{value: enabled, present: isBool}}
			}
		}
		if invalidated, ok := signal.Body[2].([]string); ok && slices.Contains(invalidated, "IsEnabled") {
			return []statusChange{{}}
		}
	case "org.freedesktop.DBus.NameOwnerChanged":
		if len(signal.Body) < 3 {
			return nil
		}
		if name, _ := signal.Body[0].(string); name != statusService {
			return nil
		}
		owner, _ := signal.Body[2].(string)
		if owner == "" {
			return []statusChange{{value: false, present: true}}
		}
		if value, err := getIsEnabled(s.sessionBus); err == nil {
			return []statusChange{{value: value, present: true}}
		}
	}
	return nil
}

func runEventLoop(sessionBus *dbus.Conn, rx *messageQueue, adapters *[]adapterEntry) error {
	status, err := newStatusStream(sessionBus)
	if err != nil {
		return err
	}
	defer sessionBus.RemoveSignal(status.signals)

	var atspiBus *Bus
	defer func() {
		if atspiBus != nil {
			atspiBus.close()
		}
	}()

	handleChange := func(change statusChange) error {
		enabled, err := change.get(sessionBus)
		if err != nil {
			return logRecoverable(err)
		}
		if enabled && atspiBus != nil {
			return nil
		}
		newBus, err := busAfterStatusChange(statusChange{value: enabled, present: true}, sessionBus)
		old := atspiBus
		atspiBus = newBus
		if old != nil {
			old.close()
		}
		if err != nil {
			return logRecoverable(err)
		}
		syncAdapters(*adapters, atspiBus)
		return nil
	}

	for _, change := range status.initial {
		if err := handleChange(change); err != nil {
			return err
		}
	}

	for {
		select {
		case <-rx.stop:
			return nil
		case signal, ok := <-status.signals:
			if !ok {
				return nil
			}
			for _, change := range status.changes(signal) {
				if err := handleChange(change); err != nil {
					return err
				}
			}
		case <-rx.notify:
			for {
				m, ok := rx.next()
				if !ok {
					break
				}
				if err := processAdapterMessage(atspiBus, adapters, m); err != nil {
					return err
				}
			}
		}
	}
}

func processAdapterMessage(atspiBus *Bus, adapters *[]adapterEntry, m message) error {
	switch m := m.(type) {
	case messageAddAdapter:
		*adapters = append(*adapters, adapterEntry{
			id:                  m.id,
			activationHandler:   m.activationHandler,
			deactivationHandler: m.deactivationHandler,
			state:               m.state,
		})
		if atspiBus != nil {
			activateAdapter(&(*adapters)[len(*adapters)-1])
		}
	case messageRemoveAdapter:
		if index := slices.IndexFunc(*adapters, func(entry adapterEntry) bool { return entry.id == m.id }); index >= 0 {
			entry := (*adapters)[index]
			*adapters = slices.Delete(*adapters, index, index+1)
			entry.state.release()
		}
	case messageRegisterInterfaces:
		if atspiBus != nil {
			return atspiBus.registerInterfaces(m.node, m.interfaces)
		}
	case messageUnregisterInterfaces:
		if atspiBus != nil {
			return atspiBus.unregisterInterfaces(m.adapterID, m.nodeID, m.interfaces)
		}
	case messageEmitEvent:
		switch event := m.event.(type) {
		case atspi.EventObject:
			if atspiBus != nil {
				return atspiBus.emitObjectEvent(m.adapterID, event.Target, event.Event)
			}
		case atspi.EventWindow:
			if atspiBus != nil {
				return atspiBus.emitWindowEvent(m.adapterID, event.Target, event.Name, event.Event)
			}
		case atspi.EventCache:
			panic("cache events are sent as EmitCacheAdd/EmitCacheRemove")
		}
	case messageEmitCacheAdd:
		if atspiBus != nil {
			return atspiBus.emitCacheAdd(m.node)
		}
	case messageEmitCacheRemove:
		if atspiBus != nil {
			return atspiBus.emitCacheRemove(m.adapterID, m.nodeID)
		}
	}
	return nil
}

func stopEventLoop() {
	if messages == nil {
		return
	}
	select {
	case <-messages.stop:
	default:
		close(messages.stop)
	}
	<-messages.done
}
