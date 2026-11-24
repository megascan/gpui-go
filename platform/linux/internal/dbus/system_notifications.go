//go:build linux

package dbus

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	godbus "github.com/godbus/dbus/v5"

	gpui "github.com/megascan/gpui-go"
)

const defaultAction = "default"

const (
	notificationsService   = "org.freedesktop.Notifications"
	notificationsPath      = godbus.ObjectPath("/org/freedesktop/Notifications")
	notificationsInterface = "org.freedesktop.Notifications"
)

// SystemNotificationState shows XDG notifications and forwards their activations.
type SystemNotificationState struct {
	conn            *godbus.Conn
	responses       chan gpui.SystemNotificationResponse
	callback        *func(gpui.SystemNotificationResponse)
	startedResponse sync.Once
}

// NewSystemNotificationState returns notification state that talks over conn.
func NewSystemNotificationState(conn *godbus.Conn) *SystemNotificationState {
	return &SystemNotificationState{
		conn:      conn,
		responses: make(chan gpui.SystemNotificationResponse, 16),
		callback:  new(func(gpui.SystemNotificationResponse)),
	}
}

// Show sends notification to the notification server and waits for its action on a goroutine.
func (s *SystemNotificationState) Show(appName gpui.Option[string], notification gpui.SystemNotification) {
	name, ok := appName.Get()
	if !ok {
		name = executableName()
	}
	actions := []string{defaultAction, defaultAction}
	actionIDs := make(map[string]gpui.SharedString, len(notification.Actions))
	for index, action := range notification.Actions {
		transportID := fmt.Sprintf("gpui-action-%d", index)
		actions = append(actions, transportID, string(action.Label))
		actionIDs[transportID] = action.ID
	}
	tag := notification.Tag
	go s.showAndWait(name, string(notification.Title), string(notification.Body), actions, func(action string) {
		if action == "__closed" {
			return
		}
		actionID, ok := responseActionID(action, actionIDs)
		if !ok {
			slog.Warn(fmt.Sprintf("system notification returned unknown action %q", action))
			return
		}
		s.responses <- gpui.SystemNotificationResponse{Tag: tag, ActionID: actionID}
	})
}

func (s *SystemNotificationState) showAndWait(appName, summary, body string, actions []string, onAction func(string)) {
	signals := make(chan *godbus.Signal, 16)
	s.conn.Signal(signals)
	defer s.conn.RemoveSignal(signals)
	match := []godbus.MatchOption{
		godbus.WithMatchObjectPath(notificationsPath),
		godbus.WithMatchInterface(notificationsInterface),
	}
	if err := s.conn.AddMatchSignal(match...); err != nil {
		slog.Warn("failed to show system notification: " + err.Error())
		return
	}
	defer func() { _ = s.conn.RemoveMatchSignal(match...) }()
	var id uint32
	err := s.conn.Object(notificationsService, notificationsPath).Call(notificationsInterface+".Notify", 0,
		appName, uint32(0), "", summary, body, actions, map[string]godbus.Variant{}, int32(-1)).Store(&id)
	if err != nil {
		slog.Warn("failed to show system notification: " + err.Error())
		return
	}
	for sig := range signals {
		if sig.Path != notificationsPath || len(sig.Body) < 2 {
			continue
		}
		if sigID, ok := sig.Body[0].(uint32); !ok || sigID != id {
			continue
		}
		switch sig.Name {
		case notificationsInterface + ".ActionInvoked":
			if action, ok := sig.Body[1].(string); ok {
				onAction(action)
				return
			}
		case notificationsInterface + ".NotificationClosed":
			onAction("__closed")
			return
		}
	}
}

// Dismiss is a no-op: the XDG protocol closes only by the server-assigned id of a live handle.
func (s *SystemNotificationState) Dismiss(tag string) {}

// OnResponse registers callback; responses are handed to dispatch, which must
// run the given function on the main thread.
func (s *SystemNotificationState) OnResponse(dispatch func(func()), callback func(gpui.SystemNotificationResponse)) {
	*s.callback = callback
	s.startedResponse.Do(func() {
		slot := s.callback
		go func() {
			for response := range s.responses {
				dispatch(func() {
					taken := *slot
					*slot = nil
					if taken != nil {
						taken(response)
						if *slot == nil {
							*slot = taken
						}
					}
				})
			}
		}()
	})
}

func responseActionID(transportID string, actionIDs map[string]gpui.SharedString) (gpui.Option[gpui.SharedString], bool) {
	if transportID == defaultAction {
		return gpui.None[gpui.SharedString](), true
	}
	id, ok := actionIDs[transportID]
	if !ok {
		return gpui.None[gpui.SharedString](), false
	}
	return gpui.Some(id), true
}

func executableName() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Base(exe)
}
