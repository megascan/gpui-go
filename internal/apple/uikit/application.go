//go:build darwin

package uikit

import "github.com/megascan/gpui-go/internal/apple/objc"

// UIApplication wraps a UIApplication object.
type UIApplication objc.ID

// ID returns the underlying object.
func (a UIApplication) ID() objc.ID { return objc.ID(a) }

// UIApplicationSharedApplication returns the singleton application.
func UIApplicationSharedApplication() UIApplication {
	return UIApplication(sendClass[objc.ID]("UIApplication", "sharedApplication"))
}

// KeyWindow returns the legacy key window, or nil.
func (a UIApplication) KeyWindow() UIWindow { return UIWindow(send[objc.ID](a.ID(), "keyWindow")) }

// OpenURLOptionsCompletionHandler asks the system to open url with options and an optional
// completion block.
func (a UIApplication) OpenURLOptionsCompletionHandler(url objc.ID, options objc.ID, completion objc.ID) {
	sendVoid(a.ID(), "openURL:options:completionHandler:", url, options, completion)
}

// UIScene wraps a UIScene object.
type UIScene objc.ID

// ID returns the underlying object.
func (s UIScene) ID() objc.ID { return objc.ID(s) }

// AsWindowScene returns the scene as a UIWindowScene when it is one.
func (s UIScene) AsWindowScene() (UIWindowScene, bool) {
	if IsKindOfClass(s.ID(), "UIWindowScene") {
		return UIWindowScene(s), true
	}
	return 0, false
}

// UIWindowScene wraps a UIWindowScene object.
type UIWindowScene objc.ID

// ID returns the underlying object.
func (s UIWindowScene) ID() objc.ID { return objc.ID(s) }

// Windows returns the scene's windows.
func (s UIWindowScene) Windows() []UIWindow {
	ids := objc.NSArrayToIDs(send[objc.ID](s.ID(), "windows"))
	out := make([]UIWindow, len(ids))
	for i, id := range ids {
		out[i] = UIWindow(id)
	}
	return out
}

// UISceneSession wraps a UISceneSession object.
type UISceneSession objc.ID

// ID returns the underlying object.
func (s UISceneSession) ID() objc.ID { return objc.ID(s) }

// Role returns the session's role.
func (s UISceneSession) Role() objc.ID { return send[objc.ID](s.ID(), "role") }

// UISceneConfiguration wraps a UISceneConfiguration object.
type UISceneConfiguration objc.ID

// ID returns the underlying object.
func (c UISceneConfiguration) ID() objc.ID { return objc.ID(c) }

// NewUISceneConfiguration returns a +1 configuration with name and role.
func NewUISceneConfiguration(name objc.ID, role objc.ID) UISceneConfiguration {
	alloc := objc.Alloc(class("UISceneConfiguration"))
	return UISceneConfiguration(send[objc.ID](alloc, "initWithName:sessionRole:", name, role))
}

// SetDelegateClass sets the class UIKit instantiates as the scene delegate.
func (c UISceneConfiguration) SetDelegateClass(delegateClass objc.Class) {
	sendVoid(c.ID(), "setDelegateClass:", objc.ID(delegateClass))
}

// UIOpenURLContextURL returns the URL of a UIOpenURLContext.
func UIOpenURLContextURL(context objc.ID) objc.ID { return send[objc.ID](context, "URL") }

// UISceneConnectionOptionsURLContexts returns the options' URL contexts set, which iOS 18 can
// return as nil.
func UISceneConnectionOptionsURLContexts(options objc.ID) objc.ID {
	return send[objc.ID](options, "URLContexts")
}

// UISceneConnectionOptionsUserActivities returns the options' user activities set, or nil.
func UISceneConnectionOptionsUserActivities(options objc.ID) objc.ID {
	return send[objc.ID](options, "userActivities")
}

// NSSetAllObjects returns the members of an NSSet.
func NSSetAllObjects(set objc.ID) []objc.ID {
	if set == objc.Nil {
		return nil
	}
	return objc.NSArrayToIDs(objc.Send[objc.ID](set, objc.Sel("allObjects")))
}

// CADisplayLink wraps a CADisplayLink object.
type CADisplayLink objc.ID

// ID returns the underlying object.
func (l CADisplayLink) ID() objc.ID { return objc.ID(l) }

// CADisplayLinkWithTargetSelector returns an autoreleased display link calling selector on
// target every frame.
func CADisplayLinkWithTargetSelector(target objc.ID, selector objc.SEL) CADisplayLink {
	return CADisplayLink(sendClass[objc.ID]("CADisplayLink", "displayLinkWithTarget:selector:", target, selector))
}

// SetPaused pauses or resumes the link.
func (l CADisplayLink) SetPaused(paused bool) { sendVoid(l.ID(), "setPaused:", paused) }

// AddToRunLoopForMode schedules the link on runLoop in mode.
func (l CADisplayLink) AddToRunLoopForMode(runLoop objc.ID, mode objc.ID) {
	sendVoid(l.ID(), "addToRunLoop:forMode:", runLoop, mode)
}

// Invalidate removes the link from every run loop and releases its target.
func (l CADisplayLink) Invalidate() { sendVoid(l.ID(), "invalidate") }
