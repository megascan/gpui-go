package lyon

// Polygon is lyon_path's Polygon, a view over a sequence of endpoints forming a polygon.
type Polygon[T any] struct {
	Points []T
	Closed bool
}
