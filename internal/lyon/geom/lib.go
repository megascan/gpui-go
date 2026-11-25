// Package geom is the subset of lyon_geom 1.0.20 that gpui's path builder uses, at f32.
package geom

// Epsilon is Scalar::EPSILON for f32.
const Epsilon float32 = 1e-4

// EpsilonFor is Scalar::epsilon_for for f32.
func EpsilonFor(reference float32) float32 {
	magnitude := asI32(abs(reference))
	switch {
	case magnitude >= 0 && magnitude <= 7:
		return 1e-5
	case magnitude >= 8 && magnitude <= 1023:
		return 1e-3
	case magnitude >= 1024 && magnitude <= 4095:
		return 1e-2
	case magnitude >= 4096 && magnitude <= 65535:
		return 1e-1
	case magnitude >= 65536 && magnitude <= 8_388_607:
		return 0.5
	default:
		return 1.0
	}
}

// NewVector is lyon_geom::vector.
func NewVector(x, y float32) Vector { return Vector{x, y} }

// NewPoint is lyon_geom::point.
func NewPoint(x, y float32) Point { return Point{x, y} }

// Transformation is lyon_geom::traits::Transformation<f32>.
type Transformation interface {
	TransformPoint(p Point) Point
	TransformVector(v Vector) Vector
}

var (
	_ Transformation = Transform{}
	_ Transformation = Rotation{}
)
