package lyon

import "github.com/megascan/gpui-go/internal/lyon/geom"

// EventKind names the variant of an Event.
type EventKind uint8

const (
	EventBegin EventKind = iota
	EventLine
	EventQuadratic
	EventCubic
	EventEnd
)

// Event is lyon_path's Event: a path event over endpoint type Ep and control point type Cp.
// Kind selects the variant; Begin uses At, Line From and To, Quadratic From, Ctrl and To,
// Cubic From, Ctrl1, Ctrl2 and To, End Last, First and Close. Other fields stay zero.
type Event[Ep, Cp comparable] struct {
	Kind  EventKind
	At    Ep
	From  Ep
	Ctrl  Cp
	Ctrl1 Cp
	Ctrl2 Cp
	To    Ep
	Last  Ep
	First Ep
	Close bool
}

// PathEvent is lyon_path's PathEvent.
type PathEvent = Event[Point, Point]

// IDEvent is lyon_path's IdEvent.
type IDEvent = Event[EndpointID, ControlPointID]

// IsEdge is Event::is_edge.
func (e Event[Ep, Cp]) IsEdge() bool {
	switch e.Kind {
	case EventLine, EventQuadratic, EventCubic:
		return true
	case EventEnd:
		return e.Close
	}
	return false
}

// FromEndpoint is Event::from.
func (e Event[Ep, Cp]) FromEndpoint() Ep {
	switch e.Kind {
	case EventBegin:
		return e.At
	case EventEnd:
		return e.Last
	}
	return e.From
}

// ToEndpoint is Event::to.
func (e Event[Ep, Cp]) ToEndpoint() Ep {
	switch e.Kind {
	case EventBegin:
		return e.At
	case EventEnd:
		return e.First
	}
	return e.To
}

// PathEventTransformed is PathEvent::transformed.
func PathEventTransformed(e PathEvent, mat geom.Transformation) PathEvent {
	switch e.Kind {
	case EventLine:
		return PathEvent{Kind: EventLine, From: mat.TransformPoint(e.From), To: mat.TransformPoint(e.To)}
	case EventQuadratic:
		return PathEvent{
			Kind: EventQuadratic,
			From: mat.TransformPoint(e.From),
			Ctrl: mat.TransformPoint(e.Ctrl),
			To:   mat.TransformPoint(e.To),
		}
	case EventCubic:
		return PathEvent{
			Kind:  EventCubic,
			From:  mat.TransformPoint(e.From),
			Ctrl1: mat.TransformPoint(e.Ctrl1),
			Ctrl2: mat.TransformPoint(e.Ctrl2),
			To:    mat.TransformPoint(e.To),
		}
	case EventBegin:
		return PathEvent{Kind: EventBegin, At: mat.TransformPoint(e.At)}
	default:
		return PathEvent{
			Kind:  EventEnd,
			Last:  mat.TransformPoint(e.Last),
			First: mat.TransformPoint(e.First),
			Close: e.Close,
		}
	}
}
