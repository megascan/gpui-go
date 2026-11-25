package gpuiutil

import (
	tz "time"

	"example.com/geom"
)

//gpui:refineable Debug
type Pair[K comparable, V any] struct {
	Key   K
	Value Option[V]
	//gpui:refineable
	Bounds geom.Bounds[float32]
	//gpui:refineable
	Inner InnerRefinement
	When  tz.Time
}

type Unmarked struct {
	A int
}
