package atspi

// Error is an error returned by the platform node layer.
type Error uint8

// Error variants.
const (
	ErrorDefunct Error = iota
	ErrorUnsupportedInterface
	ErrorTooManyChildren
	ErrorIndexOutOfRange
	ErrorTooManyCharacters
	ErrorUnsupportedTextGranularity
)

// Error returns upstream's Display text.
func (e Error) Error() string {
	switch e {
	case ErrorDefunct:
		return "defunct"
	case ErrorUnsupportedInterface:
		return "unsupported interface"
	case ErrorTooManyChildren:
		return "too many children"
	case ErrorIndexOutOfRange:
		return "index out of range"
	case ErrorTooManyCharacters:
		return "too many characters"
	default:
		return "unsupported text granularity"
	}
}
