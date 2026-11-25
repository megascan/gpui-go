package taffy

// Float controls whether a box is floated.
type Float uint8

const (
	FloatLeft Float = iota
	FloatRight
	FloatNone
)

// FloatDefault is the Default value of Float.
const FloatDefault = FloatNone

// FloatDirection is the direction of a floated box (Float without None).
type FloatDirection uint8

const (
	FloatDirectionLeft  FloatDirection = 0
	FloatDirectionRight FloatDirection = 1
)

// IsFloated reports whether the box is floated.
func (f Float) IsFloated() bool {
	return f == FloatLeft || f == FloatRight
}

// FloatDirection returns the float direction, or None if the box is not floated.
func (f Float) FloatDirection() Option[FloatDirection] {
	switch f {
	case FloatLeft:
		return some(FloatDirectionLeft)
	case FloatRight:
		return some(FloatDirectionRight)
	}
	return none[FloatDirection]()
}

// Clear controls whether a box clears previous floats.
type Clear uint8

const (
	ClearLeft Clear = iota
	ClearRight
	ClearBoth
	ClearNone
)

// ClearDefault is the Default value of Clear.
const ClearDefault = ClearNone
