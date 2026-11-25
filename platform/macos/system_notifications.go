//go:build darwin && !ios

package macos

import (
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"sync"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/scheduler"
)

const userNotificationsPath = "/System/Library/Frameworks/UserNotifications.framework/UserNotifications"

const (
	unAuthorizationOptionSound             uint = 1 << 1
	unAuthorizationOptionAlert             uint = 1 << 2
	unNotificationPresentationOptionList   uint = 1 << 3
	unNotificationPresentationOptionBanner uint = 1 << 4
)

type notificationCommand interface {
	isNotificationCommand()
}

type notificationCommandShow struct {
	notification gpui.SystemNotification
}

type notificationCommandDismiss struct {
	tag gpui.SharedString
}

func (notificationCommandShow) isNotificationCommand()    {}
func (notificationCommandDismiss) isNotificationCommand() {}

type notificationCommandEntry struct {
	command   notificationCommand
	completed *scheduler.OneshotSender[struct{}]
}

type responseCallback struct {
	callback func(response gpui.SystemNotificationResponse)
}

type systemNotificationState struct {
	initialized  bool
	commands     *scheduler.ChanSender[notificationCommandEntry]
	callback     *responseCallback
	delegate     *objc.Retained
	responseTask *gpui.Task[struct{}]
}

func newSystemNotificationState() *systemNotificationState {
	return &systemNotificationState{callback: &responseCallback{}}
}

func (s *systemNotificationState) send(foregroundExecutor gpui.ForegroundExecutor, backgroundExecutor gpui.BackgroundExecutor, command notificationCommand) *gpui.Task[struct{}] {
	s.initialize(foregroundExecutor)
	if s.commands == nil {
		return scheduler.ReadyTask(struct{}{})
	}
	completed, completion := scheduler.Oneshot[struct{}]()
	if err := s.commands.TrySend(notificationCommandEntry{command: command, completed: completed}); err != nil {
		slog.Error(fmt.Sprintf("system notification worker disconnected: %v", err))
		return scheduler.ReadyTask(struct{}{})
	}
	return backgroundExecutor.Spawn(func(sc *scheduler.Scope) (struct{}, error) {
		if _, err := completion.Recv(sc); err != nil {
			slog.Error(fmt.Sprintf("system notification worker stopped before completing a command: %v", err))
		}
		return struct{}{}, nil
	})
}

func (s *systemNotificationState) onResponse(executor gpui.ForegroundExecutor, callback func(response gpui.SystemNotificationResponse)) {
	s.initialize(executor)
	s.callback.callback = callback
}

func (s *systemNotificationState) initialize(executor gpui.ForegroundExecutor) {
	if s.initialized {
		return
	}
	s.initialized = true

	if err := loadUserNotifications(); err != nil {
		slog.Error(fmt.Sprintf("system notifications disabled: %v", err))
		return
	}
	if objc.Send[objc.ID](mainBundle(), objc.Sel("bundleIdentifier")) == objc.Nil {
		slog.Info("system notifications disabled: not running from an app bundle")
		return
	}

	sender, receiver := scheduler.Chan[gpui.SystemNotificationResponse](0)
	objc.AutoreleasePool(func() {
		delegate := newNotificationResponseDelegate(sender)
		objc.SendVoid(currentNotificationCenter(), objc.Sel("setDelegate:"), delegate.ID())
		s.delegate = delegate
	})

	commands, pendingCommands := scheduler.Chan[notificationCommandEntry](0)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		var center *notificationCenter
		objc.AutoreleasePool(func() { center = newNotificationCenter() })
		for {
			entry, ok := pendingCommands.RecvBlocking()
			if !ok {
				return
			}
			objc.AutoreleasePool(func() {
				switch command := entry.command.(type) {
				case notificationCommandShow:
					center.show(command.notification)
				case notificationCommandDismiss:
					center.dismiss(string(command.tag))
				}
			})
			_ = entry.completed.Send(struct{}{})
		}
	}()
	s.commands = commands

	callback := s.callback
	s.responseTask = executor.Spawn(func(sc *scheduler.Scope) (struct{}, error) {
		for {
			response, ok := receiver.Recv(sc)
			if !ok {
				return struct{}{}, nil
			}
			taken := callback.callback
			callback.callback = nil
			if taken != nil {
				taken(response)
				if callback.callback == nil {
					callback.callback = taken
				}
			}
		}
	})
}

var (
	userNotificationsOnce sync.Once
	userNotificationsErr  error
	userNotificationsLib  uintptr
)

func loadUserNotifications() error {
	userNotificationsOnce.Do(func() {
		if err := objc.Load(); err != nil {
			userNotificationsErr = err
			return
		}
		lib, err := purego.Dlopen(userNotificationsPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			userNotificationsErr = fmt.Errorf("macos: open %s: %w", userNotificationsPath, err)
			return
		}
		userNotificationsLib = lib
	})
	return userNotificationsErr
}

func currentNotificationCenter() objc.ID {
	return objc.SendClass[objc.ID](objc.GetClass("UNUserNotificationCenter"), objc.Sel("currentNotificationCenter"))
}

type notificationCategory struct {
	actions    []gpui.SystemNotificationAction
	identifier gpui.SharedString
	category   *objc.Retained
}

type notificationCenter struct {
	center                 *objc.Retained
	categories             []notificationCategory
	authorizationRequested bool
}

func newNotificationCenter() *notificationCenter {
	return &notificationCenter{center: objc.RetainObject(currentNotificationCenter())}
}

func (c *notificationCenter) requestAuthorization() {
	if c.authorizationRequested {
		return
	}
	c.authorizationRequested = true
	completion := objc.NewBlock(func(_ objc.Block, granted uintptr, nsErr objc.ID) {
		if err := objc.NSErrorFromID(nsErr); err != nil {
			slog.Warn("system notification authorization failed: " + err.LocalizedDescription)
		} else if granted&0xff == 0 {
			slog.Info("system notification authorization denied")
		}
	})
	objc.SendVoid(c.center.ID(), objc.Sel("requestAuthorizationWithOptions:completionHandler:"), unAuthorizationOptionAlert|unAuthorizationOptionSound, completion)
	completion.Release()
}

func (c *notificationCenter) show(notification gpui.SystemNotification) {
	c.requestAuthorization()

	content := objc.AllocInit(objc.GetClass("UNMutableNotificationContent"))
	defer objc.Release(content)
	objc.SendVoid(content, objc.Sel("setTitle:"), nsString(string(notification.Title)))
	objc.SendVoid(content, objc.Sel("setBody:"), nsString(string(notification.Body)))
	if len(notification.Actions) > 0 {
		categoryIdentifier := c.registerCategory(notification.Actions)
		objc.SendVoid(content, objc.Sel("setCategoryIdentifier:"), nsString(string(categoryIdentifier)))
	}

	request := objc.SendClass[objc.ID](objc.GetClass("UNNotificationRequest"), objc.Sel("requestWithIdentifier:content:trigger:"), nsString(string(notification.Tag)), content, objc.Nil)
	completion := objc.NewBlock(func(_ objc.Block, nsErr objc.ID) {
		if err := objc.NSErrorFromID(nsErr); err != nil {
			slog.Warn("failed to deliver system notification: " + err.LocalizedDescription)
		}
	})
	objc.SendVoid(c.center.ID(), objc.Sel("addNotificationRequest:withCompletionHandler:"), request, completion)
	completion.Release()
}

func (c *notificationCenter) registerCategory(actions []gpui.SystemNotificationAction) gpui.SharedString {
	for _, category := range c.categories {
		if slices.Equal(category.actions, actions) {
			return category.identifier
		}
	}

	identifier := gpui.SharedString(fmt.Sprintf("gpui-system-notification-%d", len(c.categories)))
	platformActions := make([]objc.ID, 0, len(actions))
	for _, action := range actions {
		platformActions = append(platformActions, objc.SendClass[objc.ID](
			objc.GetClass("UNNotificationAction"),
			objc.Sel("actionWithIdentifier:title:options:"),
			nsString(string(action.ID)),
			nsString(string(action.Label)),
			uint(0),
		))
	}
	actionArray := objc.NSArrayFromIDs(platformActions)
	defer objc.Release(actionArray)
	intentIdentifiers := objc.NSArrayFromIDs(nil)
	defer objc.Release(intentIdentifiers)
	category := objc.SendClass[objc.ID](
		objc.GetClass("UNNotificationCategory"),
		objc.Sel("categoryWithIdentifier:actions:intentIdentifiers:options:"),
		nsString(string(identifier)),
		actionArray,
		intentIdentifiers,
		uint(0),
	)

	c.categories = append(c.categories, notificationCategory{
		actions:    slices.Clone(actions),
		identifier: identifier,
		category:   objc.RetainObject(category),
	})
	all := make([]objc.ID, 0, len(c.categories))
	for _, category := range c.categories {
		all = append(all, category.category.ID())
	}
	allArray := objc.NSArrayFromIDs(all)
	defer objc.Release(allArray)
	set := objc.SendClass[objc.ID](objc.GetClass("NSSet"), objc.Sel("setWithArray:"), allArray)
	objc.SendVoid(c.center.ID(), objc.Sel("setNotificationCategories:"), set)
	return identifier
}

func (c *notificationCenter) dismiss(tag string) {
	identifiers := objc.NSArrayFromIDs([]objc.ID{nsString(tag)})
	defer objc.Release(identifiers)
	objc.SendVoid(c.center.ID(), objc.Sel("removePendingNotificationRequestsWithIdentifiers:"), identifiers)
	objc.SendVoid(c.center.ID(), objc.Sel("removeDeliveredNotificationsWithIdentifiers:"), identifiers)
}

var (
	notificationDelegateClassOnce sync.Once
	notificationDelegateClass     objc.Class
	notificationDelegateErr       error

	notificationDelegateSendersMu sync.Mutex
	notificationDelegateSenders   = map[objc.ID]*scheduler.ChanSender[gpui.SystemNotificationResponse]{}
)

func notificationResponseDelegateClass() (objc.Class, error) {
	notificationDelegateClassOnce.Do(func() {
		notificationDelegateClass, notificationDelegateErr = objc.RegisterClass(
			"GPUINotificationResponseDelegate",
			objc.GetClass("NSObject"),
			[]string{"NSObject", "UNUserNotificationCenterDelegate"},
			nil,
			[]objc.MethodDef{
				{
					Cmd: objc.Sel("userNotificationCenter:didReceiveNotificationResponse:withCompletionHandler:"),
					Fn:  didReceiveNotificationResponse,
				},
				{
					Cmd: objc.Sel("userNotificationCenter:willPresentNotification:withCompletionHandler:"),
					Fn:  willPresentNotification,
				},
			},
		)
	})
	return notificationDelegateClass, notificationDelegateErr
}

func newNotificationResponseDelegate(sender *scheduler.ChanSender[gpui.SystemNotificationResponse]) *objc.Retained {
	class, err := notificationResponseDelegateClass()
	if err != nil {
		panic(err)
	}
	delegate := objc.AllocInit(class)
	notificationDelegateSendersMu.Lock()
	notificationDelegateSenders[delegate] = sender
	notificationDelegateSendersMu.Unlock()
	return objc.Own(delegate)
}

func didReceiveNotificationResponse(this objc.ID, _ objc.SEL, _ objc.ID, response objc.ID, completionHandler objc.ID) {
	notification := objc.Send[objc.ID](response, objc.Sel("notification"))
	request := objc.Send[objc.ID](notification, objc.Sel("request"))
	tag := nsStringToStr(objc.Send[objc.ID](request, objc.Sel("identifier")))
	action := objc.Send[objc.ID](response, objc.Sel("actionIdentifier"))
	actionID := gpui.None[gpui.SharedString]()
	defaultAction, err := objc.Constant(userNotificationsLib, "UNNotificationDefaultActionIdentifier")
	if err != nil || !objc.Send[bool](action, objc.Sel("isEqualToString:"), defaultAction) {
		actionID = gpui.Some(gpui.SharedString(nsStringToStr(action)))
	}
	notificationDelegateSendersMu.Lock()
	sender := notificationDelegateSenders[this]
	notificationDelegateSendersMu.Unlock()
	if sender != nil {
		_ = sender.TrySend(gpui.SystemNotificationResponse{Tag: gpui.SharedString(tag), ActionID: actionID})
	}
	invokeForeignBlock(completionHandler)
}

func willPresentNotification(_ objc.ID, _ objc.SEL, _ objc.ID, _ objc.ID, completionHandler objc.ID) {
	invokeForeignBlock(completionHandler, uintptr(unNotificationPresentationOptionBanner|unNotificationPresentationOptionList))
}

const blockInvokeOffset = 16

func invokeForeignBlock(block objc.ID, args ...uintptr) {
	if block == objc.Nil {
		return
	}
	invoke := objc.LoadPointer(uintptr(block) + blockInvokeOffset)
	purego.SyscallN(invoke, append([]uintptr{uintptr(block)}, args...)...)
}
