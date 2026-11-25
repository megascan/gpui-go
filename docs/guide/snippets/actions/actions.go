// Package actions holds the snippets of docs/guide/actions-keymaps-focus.md.
package actions

//go:generate go run github.com/megascan/gpui-go/internal/cmd/actiongen

//gpui:action namespace=menu
type MoveUp struct{}

//gpui:action namespace=menu
type MoveDown struct{}

//gpui:action namespace=menu
type Confirm struct{}

//gpui:action namespace=menu
type Jump struct {
	Index int  `json:"index"`
	Wrap  bool `json:"wrap"`
}
