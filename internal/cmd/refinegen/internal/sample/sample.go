package sample

import (
	"time"

	"github.com/megascan/gpui-go/gpuiutil"
)

//go:generate go run github.com/megascan/gpui-go/internal/cmd/refinegen

// Number is the constraint of the generic sample types.
type Number interface {
	~int | ~float32
}

// Point is a generic refineable pair.
//
//gpui:refineable Debug PartialEq Serialize
type Point[T Number] struct {
	X T `json:"x"`
	Y T
}

// Edges holds four optional-capable values.
//
//gpui:refineable Debug PartialEq Serialize Deserialize JsonSchema
type Edges[T any] struct {
	Top    T
	Bottom T
}

// Style mixes plain, optional and nested fields.
//
//gpui:refineable Debug PartialEq Serialize
type Style struct {
	Visible bool
	//gpui:refineable
	Origin      Point[float32]
	AspectRatio gpuiutil.Option[float32]
	Margin      Edges[int] //gpui:refineable
	Tags        []string
	Delay       time.Duration
	BorderWidth int
}
