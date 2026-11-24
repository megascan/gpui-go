package zeno

// PathData is path data that can be copied into a sink.
type PathData interface {
	CopyTo(sink PathBuilder)
}

// PathBounds computes the bounding box of the path under a fill style.
func PathBounds(data PathData, transform *Transform) Bounds {
	b := NewBoundsBuilder()
	Apply(data, transform, b)
	return b.Build()
}

// Apply applies the transform to the path and emits the result to the sink.
func Apply(data PathData, transform *Transform, sink PathBuilder) {
	if transform != nil {
		data.CopyTo(&TransformSink{Sink: sink, Transform: *transform})
		return
	}
	data.CopyTo(sink)
}
