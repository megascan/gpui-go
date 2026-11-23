package atspi

// Action describes one action an object exposes over the AT-SPI Action interface.
type Action struct {
	LocalizedName string
	Description   string
	KeyBinding    string
}
