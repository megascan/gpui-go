//go:build darwin

package appkit

import "github.com/megascan/gpui-go/internal/apple/objc"

// NSApplicationActivationPolicy is an application's activation policy.
type NSApplicationActivationPolicy int

// NSApplicationActivationPolicy values.
const (
	NSApplicationActivationPolicyRegular    NSApplicationActivationPolicy = 0
	NSApplicationActivationPolicyAccessory  NSApplicationActivationPolicy = 1
	NSApplicationActivationPolicyProhibited NSApplicationActivationPolicy = 2
)

// NSApplicationOcclusionState is an application's occlusion state.
type NSApplicationOcclusionState uint

// NSApplicationOcclusionStateVisible is set when any part of the application is visible.
const NSApplicationOcclusionStateVisible NSApplicationOcclusionState = 1 << 1

// NSRequestUserAttentionType is the kind of attention request.
type NSRequestUserAttentionType uint

// NSRequestUserAttentionType values.
const (
	NSCriticalRequest      NSRequestUserAttentionType = 0
	NSInformationalRequest NSRequestUserAttentionType = 10
)

// NSApplication wraps an NSApplication object.
type NSApplication objc.ID

// ID returns the underlying object.
func (a NSApplication) ID() objc.ID { return objc.ID(a) }

// NSApplicationSharedApplication returns the shared application, creating it when needed.
func NSApplicationSharedApplication() NSApplication {
	return NSApplication(sendClass[objc.ID]("NSApplication", "sharedApplication"))
}

// NSApp returns the shared application.
func NSApp() NSApplication { return NSApplicationSharedApplication() }

// SetActivationPolicy sets the activation policy.
func (a NSApplication) SetActivationPolicy(policy NSApplicationActivationPolicy) bool {
	return send[bool](a.ID(), "setActivationPolicy:", policy)
}

// SetDelegate sets the application delegate.
func (a NSApplication) SetDelegate(delegate objc.ID) { sendVoid(a.ID(), "setDelegate:", delegate) }

// Delegate returns the application delegate.
func (a NSApplication) Delegate() objc.ID { return send[objc.ID](a.ID(), "delegate") }

// Run starts the main event loop.
func (a NSApplication) Run() { sendVoid(a.ID(), "run") }

// Stop stops the main event loop.
func (a NSApplication) Stop(sender objc.ID) { sendVoid(a.ID(), "stop:", sender) }

// Terminate terminates the application.
func (a NSApplication) Terminate(sender objc.ID) { sendVoid(a.ID(), "terminate:", sender) }

// ActivateIgnoringOtherApps activates the application.
func (a NSApplication) ActivateIgnoringOtherApps(flag bool) {
	sendVoid(a.ID(), "activateIgnoringOtherApps:", flag)
}

// Hide hides the application.
func (a NSApplication) Hide(sender objc.ID) { sendVoid(a.ID(), "hide:", sender) }

// HideOtherApplications hides every other application.
func (a NSApplication) HideOtherApplications(sender objc.ID) {
	sendVoid(a.ID(), "hideOtherApplications:", sender)
}

// UnhideAllApplications unhides every application.
func (a NSApplication) UnhideAllApplications(sender objc.ID) {
	sendVoid(a.ID(), "unhideAllApplications:", sender)
}

// IsActive reports whether the application is active.
func (a NSApplication) IsActive() bool { return send[bool](a.ID(), "isActive") }

// KeyWindow returns the key window or nil.
func (a NSApplication) KeyWindow() NSWindow { return NSWindow(send[objc.ID](a.ID(), "keyWindow")) }

// MainWindow returns the main window or nil.
func (a NSApplication) MainWindow() NSWindow { return NSWindow(send[objc.ID](a.ID(), "mainWindow")) }

// Windows returns the application's windows.
func (a NSApplication) Windows() []NSWindow {
	ids := objc.NSArrayToIDs(send[objc.ID](a.ID(), "windows"))
	out := make([]NSWindow, len(ids))
	for i, id := range ids {
		out[i] = NSWindow(id)
	}
	return out
}

// OrderedWindows returns the application's windows in front-to-back order.
func (a NSApplication) OrderedWindows() []NSWindow {
	ids := objc.NSArrayToIDs(send[objc.ID](a.ID(), "orderedWindows"))
	out := make([]NSWindow, len(ids))
	for i, id := range ids {
		out[i] = NSWindow(id)
	}
	return out
}

// SetMainMenu sets the main menu.
func (a NSApplication) SetMainMenu(menu NSMenu) { sendVoid(a.ID(), "setMainMenu:", menu.ID()) }

// MainMenu returns the main menu.
func (a NSApplication) MainMenu() NSMenu { return NSMenu(send[objc.ID](a.ID(), "mainMenu")) }

// SetServicesMenu sets the services menu.
func (a NSApplication) SetServicesMenu(menu NSMenu) {
	sendVoid(a.ID(), "setServicesMenu:", menu.ID())
}

// SetWindowsMenu sets the windows menu.
func (a NSApplication) SetWindowsMenu(menu NSMenu) {
	sendVoid(a.ID(), "setWindowsMenu:", menu.ID())
}

// CurrentEvent returns the event being dispatched or nil.
func (a NSApplication) CurrentEvent() NSEvent { return NSEvent(send[objc.ID](a.ID(), "currentEvent")) }

// SendEvent dispatches an event.
func (a NSApplication) SendEvent(event NSEvent) { sendVoid(a.ID(), "sendEvent:", event.ID()) }

// PostEventAtStart posts an event to the event queue.
func (a NSApplication) PostEventAtStart(event NSEvent, atStart bool) {
	sendVoid(a.ID(), "postEvent:atStart:", event.ID(), atStart)
}

// RequestUserAttention bounces the dock icon and returns the request id.
func (a NSApplication) RequestUserAttention(kind NSRequestUserAttentionType) int {
	return send[int](a.ID(), "requestUserAttention:", kind)
}

// OcclusionState returns the application's occlusion state.
func (a NSApplication) OcclusionState() NSApplicationOcclusionState {
	return send[NSApplicationOcclusionState](a.ID(), "occlusionState")
}

// EffectiveAppearance returns the application's effective appearance.
func (a NSApplication) EffectiveAppearance() NSAppearance {
	return NSAppearance(send[objc.ID](a.ID(), "effectiveAppearance"))
}

// SetAppearance sets the application's appearance; nil follows the system.
func (a NSApplication) SetAppearance(appearance NSAppearance) {
	sendVoid(a.ID(), "setAppearance:", appearance.ID())
}

// FinishLaunching completes launch when the run loop is not started with Run.
func (a NSApplication) FinishLaunching() { sendVoid(a.ID(), "finishLaunching") }
