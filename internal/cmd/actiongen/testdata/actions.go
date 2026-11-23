package editor

import "time"

//gpui:action namespace=editor
type MoveUp struct{}

// Selects the next match.
//
// Replaces the newest selection when set.
//
//gpui:action namespace=editor name=SelectNext
type SelectNextMatch struct {
	ReplaceNewest bool `json:"replace_newest"`
	Count         int  `json:"count"`
	When          time.Duration
}

//gpui:action no_json
type Opaque struct {
	Handle uint32
}

//gpui:action no_register
type Local struct {
	Line int
}

//gpui:action deprecated_aliases=editor::Old,editor::Older deprecated="use editor::Move instead"
type MoveDown struct {
	Steps int
}

//gpui:action
type Bare struct{ X string }

type Plain struct {
	A int
}
