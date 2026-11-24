//go:build darwin

package ios

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"weak"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/objc"
	"github.com/megascan/gpui-go/internal/apple/uikit"
)

type applicationWindow interface {
	notifyVisibilityChange(visibility gpui.WindowVisibility)
	notifyActiveStatusChange(isActive bool)
	attachToScene(scene uikit.UIWindowScene)
	requestFrame()
}

// IosApplicationState is the UIKit application state one IosPlatform owns. Close is its drop.
type IosApplicationState struct {
	running         bool
	finishLaunching callbackSlot[func()]
	quit            callbackSlot[func()]
	openURLs        callbackSlot[func(urls []string)]
	appLifecycle    callbackSlot[func(phase gpui.AppLifecyclePhase)]
	memoryWarning   callbackSlot[func()]
	scene           uikit.UIWindowScene
	displayLink     uikit.CADisplayLink
	windows         []applicationWindow
}

var (
	currentApplicationMu sync.Mutex
	currentApplicationWP weak.Pointer[IosApplicationState]
)

func currentApplication() *IosApplicationState {
	currentApplicationMu.Lock()
	defer currentApplicationMu.Unlock()
	return currentApplicationWP.Value()
}

func run(application *IosApplicationState, onFinishLaunching func()) {
	if _, ok := objc.NewMainThreadMarker(); !ok {
		panic("UIKit requires the main thread")
	}
	currentApplicationMu.Lock()
	if currentApplicationWP.Value() != nil {
		currentApplicationMu.Unlock()
		panic("the iOS application is already running")
	}
	currentApplicationWP = weak.Make(application)
	currentApplicationMu.Unlock()
	if application.running {
		panic("the iOS application is already running")
	}
	application.running = true
	application.finishLaunching.set(onFinishLaunching)
	sceneDelegateClass()
	delegateClass := objc.NSStringFromGo(appDelegateClassName)
	uikit.UIApplicationMain(objc.Nil, delegateClass)
}

func (s *IosApplicationState) setDisplayLinkPaused(paused bool) {
	if displayLink := s.displayLink; displayLink != 0 {
		displayLink.SetPaused(paused)
	}
}

func (s *IosApplicationState) stopDisplayLink() {
	displayLink := s.displayLink
	s.displayLink = 0
	if displayLink != 0 {
		displayLink.Invalidate()
		objc.Release(displayLink.ID())
	}
}

func (s *IosApplicationState) windowScene() (uikit.UIWindowScene, bool) {
	return s.scene, s.scene != 0
}

func (s *IosApplicationState) registerWindow(window applicationWindow) {
	s.windows = append(s.windows, window)
}

func (s *IosApplicationState) unregisterWindow(window applicationWindow) {
	s.windows = slices.DeleteFunc(s.windows, func(entry applicationWindow) bool { return entry == window })
}

func (s *IosApplicationState) withWindows(callback func(window applicationWindow)) {
	windows := slices.Clone(s.windows)
	for _, window := range windows {
		if slices.Contains(s.windows, window) {
			callback(window)
		}
	}
}

func (s *IosApplicationState) notifyFinishLaunching() {
	if callback, ok := s.finishLaunching.take(); ok {
		callback()
	}
}

func (s *IosApplicationState) notifyMemoryWarning() {
	s.memoryWarning.with(func(callback *func()) { (*callback)() })
}

func (s *IosApplicationState) notifyAppLifecycle(phase gpui.AppLifecyclePhase) {
	switch phase {
	case gpui.AppLifecyclePhaseForeground:
		s.withWindows(func(window applicationWindow) {
			window.notifyVisibilityChange(gpui.WindowVisibilityVisible)
		})
	case gpui.AppLifecyclePhaseActive:
		s.withWindows(func(window applicationWindow) { window.notifyActiveStatusChange(true) })
	case gpui.AppLifecyclePhaseInactive:
		s.withWindows(func(window applicationWindow) { window.notifyActiveStatusChange(false) })
	case gpui.AppLifecyclePhaseBackground, gpui.AppLifecyclePhaseDisconnected:
		s.withWindows(func(window applicationWindow) { window.notifyActiveStatusChange(false) })
		s.withWindows(func(window applicationWindow) {
			window.notifyVisibilityChange(gpui.WindowVisibilityHidden)
		})
	}
	s.appLifecycle.with(func(callback *func(phase gpui.AppLifecyclePhase)) { (*callback)(phase) })
}

func (s *IosApplicationState) openURLContexts(contexts objc.ID) {
	members := uikit.NSSetAllObjects(contexts)
	if len(members) == 0 {
		return
	}
	var urls []string
	for _, context := range members {
		url := objc.Send[objc.ID](uikit.UIOpenURLContextURL(context), objc.Sel("absoluteString"))
		if url != objc.Nil {
			urls = append(urls, objc.GoString(url))
		}
	}
	s.openURLs.with(func(callback *func(urls []string)) { (*callback)(urls) })
}

func (s *IosApplicationState) continueUserActivity(activity objc.ID) {
	activityType := objc.Send[objc.ID](activity, objc.Sel("activityType"))
	if !objc.Send[bool](activityType, objc.Sel("isEqualToString:"), objc.FoundationConstant("NSUserActivityTypeBrowsingWeb")) {
		return
	}
	webpageURL := objc.Send[objc.ID](activity, objc.Sel("webpageURL"))
	if webpageURL == objc.Nil {
		return
	}
	url := objc.Send[objc.ID](webpageURL, objc.Sel("absoluteString"))
	if url == objc.Nil {
		return
	}
	urls := []string{objc.GoString(url)}
	s.openURLs.with(func(callback *func(urls []string)) { (*callback)(urls) })
}

// Close stops the display link; it is the application state's drop.
func (s *IosApplicationState) Close() {
	s.stopDisplayLink()
}

const (
	appDelegateClassName   = "GPUIIosAppDelegate"
	sceneDelegateClassName = "GPUIIosSceneDelegate"
)

var (
	delegateIvarsMu sync.Mutex
	delegateIvars   = map[objc.ID]weak.Pointer[IosApplicationState]{}

	appDelegateClassOnce   sync.Once
	appDelegateClassValue  objc.Class
	sceneDelegateClassOnce sync.Once
	sceneDelegateClassVal  objc.Class
)

func delegateInit(this objc.ID, _ objc.SEL) objc.ID {
	var platform weak.Pointer[IosApplicationState]
	if application := currentApplication(); application != nil {
		platform = weak.Make(application)
	}
	this = objc.SendSuper[objc.ID](this, objc.Sel("init"))
	if this != objc.Nil {
		delegateIvarsMu.Lock()
		delegateIvars[this] = platform
		delegateIvarsMu.Unlock()
	}
	return this
}

func delegateDealloc(this objc.ID, _ objc.SEL) {
	delegateIvarsMu.Lock()
	delete(delegateIvars, this)
	delegateIvarsMu.Unlock()
	objc.SendSuper[objc.ID](this, objc.Sel("dealloc"))
}

func delegatePlatform(this objc.ID) *IosApplicationState {
	delegateIvarsMu.Lock()
	defer delegateIvarsMu.Unlock()
	return delegateIvars[this].Value()
}

func appDelegateClass() objc.Class {
	appDelegateClassOnce.Do(func() {
		if err := uikit.Load(); err != nil {
			panic(err)
		}
		methods := []objc.MethodDef{
			{Cmd: objc.Sel("init"), Fn: delegateInit},
			{Cmd: objc.Sel("dealloc"), Fn: delegateDealloc},
			{Cmd: objc.Sel("application:didFinishLaunchingWithOptions:"), Fn: appDidFinishLaunching},
			{Cmd: objc.Sel("application:configurationForConnectingSceneSession:options:"), Fn: appConfigurationForScene},
			{Cmd: objc.Sel("applicationDidReceiveMemoryWarning:"), Fn: appDidReceiveMemoryWarning},
			{Cmd: objc.Sel("applicationWillTerminate:"), Fn: appWillTerminate},
		}
		class, err := objc.RegisterClass(appDelegateClassName, objc.GetClass("NSObject"), []string{"NSObject", "UIApplicationDelegate"}, nil, methods)
		if err != nil {
			panic(fmt.Sprintf("failed to register %s: %v", appDelegateClassName, err))
		}
		appDelegateClassValue = class
	})
	return appDelegateClassValue
}

func appDidFinishLaunching(_ objc.ID, _ objc.SEL, _ objc.ID, _ objc.ID) bool {
	return true
}

func appConfigurationForScene(_ objc.ID, _ objc.SEL, _ objc.ID, session objc.ID, _ objc.ID) objc.ID {
	name := objc.NSStringFromGo("Default Configuration")
	defer objc.Release(name)
	configuration := uikit.NewUISceneConfiguration(name, uikit.UISceneSession(session).Role())
	configuration.SetDelegateClass(sceneDelegateClass())
	return objc.Autorelease(configuration.ID())
}

func appDidReceiveMemoryWarning(this objc.ID, _ objc.SEL, _ objc.ID) {
	if platform := delegatePlatform(this); platform != nil {
		platform.notifyMemoryWarning()
	}
}

func appWillTerminate(this objc.ID, _ objc.SEL, _ objc.ID) {
	if platform := delegatePlatform(this); platform != nil {
		if callback, ok := platform.quit.take(); ok {
			callback()
		}
	}
}

func sceneDelegateClass() objc.Class {
	sceneDelegateClassOnce.Do(func() {
		if err := uikit.Load(); err != nil {
			panic(err)
		}
		lifecycle := func(fn func(platform *IosApplicationState)) func(objc.ID, objc.SEL, objc.ID) {
			return func(this objc.ID, _ objc.SEL, scene objc.ID) {
				if platform := platformForScene(this, scene); platform != nil {
					fn(platform)
				}
			}
		}
		methods := []objc.MethodDef{
			{Cmd: objc.Sel("init"), Fn: delegateInit},
			{Cmd: objc.Sel("dealloc"), Fn: delegateDealloc},
			{Cmd: objc.Sel("renderFrame:"), Fn: sceneRenderFrame},
			{Cmd: objc.Sel("scene:willConnectToSession:options:"), Fn: sceneWillConnect},
			{Cmd: objc.Sel("sceneDidDisconnect:"), Fn: lifecycle(func(platform *IosApplicationState) {
				platform.stopDisplayLink()
				scene := platform.scene
				platform.scene = 0
				objc.Release(scene.ID())
				platform.notifyAppLifecycle(gpui.AppLifecyclePhaseDisconnected)
			})},
			{Cmd: objc.Sel("sceneWillEnterForeground:"), Fn: lifecycle(func(platform *IosApplicationState) {
				platform.setDisplayLinkPaused(false)
				platform.notifyAppLifecycle(gpui.AppLifecyclePhaseForeground)
			})},
			{Cmd: objc.Sel("sceneDidBecomeActive:"), Fn: lifecycle(func(platform *IosApplicationState) {
				platform.notifyAppLifecycle(gpui.AppLifecyclePhaseActive)
			})},
			{Cmd: objc.Sel("sceneWillResignActive:"), Fn: lifecycle(func(platform *IosApplicationState) {
				platform.notifyAppLifecycle(gpui.AppLifecyclePhaseInactive)
			})},
			{Cmd: objc.Sel("sceneDidEnterBackground:"), Fn: lifecycle(func(platform *IosApplicationState) {
				platform.setDisplayLinkPaused(true)
				platform.notifyAppLifecycle(gpui.AppLifecyclePhaseBackground)
			})},
			{Cmd: objc.Sel("scene:openURLContexts:"), Fn: sceneOpenURLContexts},
			{Cmd: objc.Sel("scene:continueUserActivity:"), Fn: sceneContinueUserActivity},
		}
		class, err := objc.RegisterClass(sceneDelegateClassName, objc.GetClass("NSObject"), []string{"NSObject", "UISceneDelegate", "UIWindowSceneDelegate"}, nil, methods)
		if err != nil {
			panic(fmt.Sprintf("failed to register %s: %v", sceneDelegateClassName, err))
		}
		sceneDelegateClassVal = class
	})
	return sceneDelegateClassVal
}

func sceneRenderFrame(this objc.ID, _ objc.SEL, _ objc.ID) {
	if platform := delegatePlatform(this); platform != nil {
		platform.withWindows(func(window applicationWindow) { window.requestFrame() })
	}
}

func sceneWillConnect(this objc.ID, _ objc.SEL, sceneID objc.ID, _ objc.ID, options objc.ID) {
	scene, ok := uikit.UIScene(sceneID).AsWindowScene()
	if !ok {
		return
	}
	application := delegatePlatform(this)
	if application == nil {
		return
	}
	if _, ok := application.windowScene(); ok {
		slog.Error("GPUI iOS currently supports one connected window scene")
		return
	}
	application.scene = uikit.UIWindowScene(objc.Retain(scene.ID()))
	application.notifyFinishLaunching()
	application.withWindows(func(window applicationWindow) { window.attachToScene(scene) })
	if contexts := uikit.UISceneConnectionOptionsURLContexts(options); contexts != objc.Nil {
		application.openURLContexts(contexts)
	}
	if activities := uikit.UISceneConnectionOptionsUserActivities(options); activities != objc.Nil {
		for _, activity := range uikit.NSSetAllObjects(activities) {
			application.continueUserActivity(activity)
		}
	}

	application.stopDisplayLink()
	displayLink := uikit.CADisplayLinkWithTargetSelector(this, objc.Sel("renderFrame:"))
	displayLink.SetPaused(true)
	mainRunLoop := objc.SendClass[objc.ID](objc.GetClass("NSRunLoop"), objc.Sel("mainRunLoop"))
	displayLink.AddToRunLoopForMode(mainRunLoop, objc.FoundationConstant("NSRunLoopCommonModes"))
	application.displayLink = uikit.CADisplayLink(objc.Retain(displayLink.ID()))
}

func sceneOpenURLContexts(this objc.ID, _ objc.SEL, scene objc.ID, contexts objc.ID) {
	if platform := platformForScene(this, scene); platform != nil {
		platform.openURLContexts(contexts)
	}
}

func sceneContinueUserActivity(this objc.ID, _ objc.SEL, scene objc.ID, activity objc.ID) {
	if platform := platformForScene(this, scene); platform != nil {
		platform.continueUserActivity(activity)
	}
}

func platformForScene(this objc.ID, scene objc.ID) *IosApplicationState {
	platform := delegatePlatform(this)
	if platform == nil {
		return nil
	}
	if current, ok := platform.windowScene(); ok && current.ID() == scene {
		return platform
	}
	return nil
}
