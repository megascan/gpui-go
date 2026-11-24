package zeno

// Verb is a path command without its points.
type Verb uint8

const (
	// VerbMoveTo begins a subpath.
	VerbMoveTo Verb = iota
	// VerbLineTo adds a line.
	VerbLineTo
	// VerbCurveTo adds a cubic curve.
	VerbCurveTo
	// VerbQuadTo adds a quadratic curve.
	VerbQuadTo
	// VerbClose closes the subpath.
	VerbClose
)

// PointsCommands is path data stored as parallel point and verb lists.
type PointsCommands struct {
	Points []Point
	Verbs  []Verb
}

// CopyTo copies the path data into the sink.
func (c PointsCommands) CopyTo(sink PathBuilder) {
	i := 0
	for _, verb := range c.Verbs {
		switch verb {
		case VerbMoveTo:
			if i >= len(c.Points) {
				return
			}
			sink.MoveTo(c.Points[i])
			i++
		case VerbLineTo:
			if i >= len(c.Points) {
				return
			}
			sink.LineTo(c.Points[i])
			i++
		case VerbQuadTo:
			if i+1 >= len(c.Points) {
				return
			}
			sink.QuadTo(c.Points[i], c.Points[i+1])
			i += 2
		case VerbCurveTo:
			if i+2 >= len(c.Points) {
				return
			}
			sink.CurveTo(c.Points[i], c.Points[i+1], c.Points[i+2])
			i += 3
		case VerbClose:
			sink.Close()
		}
	}
}
