package zeno

// Scratch is reusable memory for rasterization.
type Scratch struct {
	render HeapStorage
}

// NewScratch creates a scratch memory context.
func NewScratch() *Scratch {
	return &Scratch{}
}

// Apply applies the transform to the path and emits the result to the sink.
func (s *Scratch) Apply(data PathData, transform *Transform, sink PathBuilder) {
	Apply(data, transform, sink)
}

// Bounds computes the bounding box of the path.
func (s *Scratch) Bounds(data PathData, transform *Transform) Bounds {
	b := NewBoundsBuilder()
	s.Apply(data, transform, b)
	return b.Build()
}
