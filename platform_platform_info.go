package gpui

// PlatformInfo is the host operating system and the kind of application delivering the UI.
type PlatformInfo struct {
	// Os is the operating system whose input conventions the application follows.
	Os OperatingSystem
	// Target is how the application is delivered, independently of its host OS.
	Target PlatformTarget
}

// OperatingSystem is the host operating system, including when the application runs in a browser.
type OperatingSystem uint8

const (
	// OperatingSystemMac is Apple's desktop operating system.
	OperatingSystemMac OperatingSystem = iota
	// OperatingSystemWindows is Microsoft Windows.
	OperatingSystemWindows
	// OperatingSystemLinux is Linux, including browser hosts running ChromeOS.
	OperatingSystemLinux
	// OperatingSystemFreeBsd is FreeBSD.
	OperatingSystemFreeBsd
	// OperatingSystemIos is iOS and iPadOS.
	OperatingSystemIos
	// OperatingSystemAndroid is Android.
	OperatingSystemAndroid
	// OperatingSystemUnknown means the backend could not identify the host operating system.
	OperatingSystemUnknown
)

// PlatformTarget is the kind of application delivering the UI, independently of its host OS.
type PlatformTarget uint8

const (
	// PlatformTargetNative is a native application, including headless execution.
	PlatformTargetNative PlatformTarget = iota
	// PlatformTargetWeb is an application running in a browser.
	PlatformTargetWeb
)

// IsMac reports whether the host operating system is macOS.
func (i PlatformInfo) IsMac() bool {
	return i.Os == OperatingSystemMac
}
