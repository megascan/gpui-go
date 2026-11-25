# M4-ios-window owner checks

- YOU: on macOS with Xcode, run `examples/ios_ios/build-simulator.sh`, install the .app with `xcrun simctl install booted target/gpui-ios-example/GPUIIosExample.app` and launch it; check the dark screen with "GPUI on iOS", that tapping "Taps: N" increments, and that the status bar content is light.
- YOU: in the simulator app, rotate the device and check the layout resizes without stretching (drawable size and scale follow), safe-area insets are respected, and toggling light/dark appearance fires the appearance callback.
- YOU: on iOS with a focused text input, check the software keyboard appears, typing/deleting/emoji/marked (Japanese) text work, a downward swipe of at least 24pt dismisses the keyboard, and trackpad/mouse-wheel scrolling scrolls.
- YOU: on iOS, run the native text-input harness (`go test -c -tags ios ./platform/ios` built for the simulator with CGO_ENABLED=1, then run it in the simulator); it must print "UIKit text input native protocol smoke test passed".
- YOU: on an iOS 26 device or simulator, check `Insets().CornerRadii` reports the display's corner radii.
