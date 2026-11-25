// Package lyon is the subset of lyon_path 1.0.19 that gpui's path builder uses.
package lyon

import (
	"math"

	"github.com/megascan/gpui-go/internal/lyon/geom"
)

// ArcFlags is lyon_path's re-export of lyon_geom::ArcFlags.
type ArcFlags = geom.ArcFlags

// Point is lyon_path::math::Point.
type Point = geom.Point

// Vector is lyon_path::math::Vector.
type Vector = geom.Vector

// Transform is lyon_path::math::Transform.
type Transform = geom.Transform

// Rotation is lyon_path::math::Rotation.
type Rotation = geom.Rotation

// Angle is lyon_path::math::Angle.
type Angle = geom.Angle

// NewVector is lyon_path::math::vector.
func NewVector(x, y float32) Vector { return geom.NewVector(x, y) }

// NewPoint is lyon_path::math::point.
func NewPoint(x, y float32) Point { return geom.NewPoint(x, y) }

// LineCap is lyon_path's LineCap, the shape at the end of open sub-paths.
type LineCap uint8

const (
	LineCapButt LineCap = iota
	LineCapSquare
	LineCapRound
)

// LineJoin is lyon_path's LineJoin, the shape where two stroke segments meet.
type LineJoin uint8

const (
	LineJoinMiter LineJoin = iota
	LineJoinMiterClip
	LineJoinRound
	LineJoinBevel
)

// Side is lyon_path's Side, the left or right of a path.
type Side uint8

const (
	SidePositive Side = iota
	SideNegative
)

// Opposite is Side::opposite.
func (s Side) Opposite() Side {
	if s == SidePositive {
		return SideNegative
	}
	return SidePositive
}

// IsPositive is Side::is_positive.
func (s Side) IsPositive() bool { return s == SidePositive }

// IsNegative is Side::is_negative.
func (s Side) IsNegative() bool { return s == SideNegative }

// ToF32 is Side::to_f32.
func (s Side) ToF32() float32 {
	if s == SidePositive {
		return 1.0
	}
	return -1.0
}

// FillRule is lyon_path's FillRule.
type FillRule uint8

const (
	FillRuleEvenOdd FillRule = iota
	FillRuleNonZero
)

// IsIn is FillRule::is_in.
func (f FillRule) IsIn(windingNumber int16) bool {
	if f == FillRuleEvenOdd {
		return windingNumber%2 != 0
	}
	return windingNumber != 0
}

// IsOut is FillRule::is_out.
func (f FillRule) IsOut(windingNumber int16) bool { return !f.IsIn(windingNumber) }

// Winding is lyon_path's Winding, the orientation of a shape.
type Winding uint8

const (
	WindingPositive Winding = iota
	WindingNegative
)

// ControlPointID is lyon_path's ControlPointId.
type ControlPointID uint32

// ControlPointIDInvalid is ControlPointId::INVALID.
const ControlPointIDInvalid ControlPointID = math.MaxUint32

// Offset is ControlPointId::offset.
func (id ControlPointID) Offset() int { return int(id) }

// ToUsize is ControlPointId::to_usize.
func (id ControlPointID) ToUsize() int { return int(id) }

// ControlPointIDFromUsize is ControlPointId::from_usize.
func ControlPointIDFromUsize(val int) ControlPointID { return ControlPointID(uint32(val)) }

// EndpointID is lyon_path's EndpointId.
type EndpointID uint32

// EndpointIDInvalid is EndpointId::INVALID.
const EndpointIDInvalid EndpointID = math.MaxUint32

// Offset is EndpointId::offset.
func (id EndpointID) Offset() int { return int(id) }

// ToUsize is EndpointId::to_usize.
func (id EndpointID) ToUsize() int { return int(id) }

// EndpointIDFromUsize is EndpointId::from_usize.
func EndpointIDFromUsize(val int) EndpointID { return EndpointID(uint32(val)) }

// EventID is lyon_path's EventId.
type EventID uint32

// EventIDInvalid is EventId::INVALID.
const EventIDInvalid EventID = math.MaxUint32

// ToUsize is EventId::to_usize.
func (id EventID) ToUsize() int { return int(id) }

// Attributes is lyon_path's Attributes; nil is NO_ATTRIBUTES.
type Attributes = []float32

func float32Nan() float32 { return float32(math.NaN()) }
