package gpui

//gpui:action name=NoAction no_json
type NoAction struct{}

//gpui:action no_register
type Unbind struct {
	Target string
}
