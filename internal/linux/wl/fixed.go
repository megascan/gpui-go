//go:build linux

package wl

// Fixed is wl_fixed_t, a signed 24.8 fixed-point number.
type Fixed int32

// FixedFromFloat64 converts f to Fixed, truncating as wayland-backend does.
func FixedFromFloat64(f float64) Fixed {
	return Fixed(int32(f * 256))
}

// FixedFromInt converts i to Fixed.
func FixedFromInt(i int32) Fixed { return Fixed(i * 256) }

// Float64 converts the value to float64 the way wl_fixed_to_double does.
func (f Fixed) Float64() float64 { return float64(f) / 256 }

// Int truncates the value toward zero.
func (f Fixed) Int() int32 { return int32(f) / 256 }
