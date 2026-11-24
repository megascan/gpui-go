# M4-ios-app owner checks

- YOU: on an iOS simulator or device (after M4-ios-window), launch the iOS example; check the launch callback runs once the scene connects, background/foreground/active lifecycle callbacks fire, and the display link pauses in background.
- YOU: on iOS, open an https URL via `OpenURL` (Safari view controller appears, `Activate` dismisses it) and a custom-scheme URL; open the app from a universal link and a URL scheme and check `OnOpenURLs` gets the full URL with query and fragment.
- YOU: on iOS, check clipboard read/write, keychain credentials, `Displays`/`PrimaryDisplay` bounds and refresh interval, and that connecting an external display fires `OnDisplaysChanged`.
