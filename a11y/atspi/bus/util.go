package bus

import (
	"errors"

	"github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go/a11y/atspi"
)

func mapError(source objectID, err error) *dbus.Error {
	var internal atspi.Error
	if !errors.As(err, &internal) {
		return fdoFailed(err.Error())
	}
	switch internal {
	case atspi.ErrorDefunct, atspi.ErrorUnsupportedInterface:
		return fdoUnknownObject(string(source.path()))
	case atspi.ErrorTooManyChildren:
		return fdoFailed("Too many children.")
	case atspi.ErrorIndexOutOfRange:
		return fdoFailed("Index is too big.")
	case atspi.ErrorTooManyCharacters:
		return fdoFailed("Too many characters.")
	default:
		return fdoFailed("Unsupported text granularity.")
	}
}

func mapErrorFromNode(source atspi.PlatformNode, err error) *dbus.Error {
	return mapError(objectIDFromPlatformNode(source), err)
}
