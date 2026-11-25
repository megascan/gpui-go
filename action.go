package gpui

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"iter"
	"reflect"
	"sync"

	"github.com/megascan/gpui-go/jsonschema"
)

// Action is a value that can be bound to keystrokes and dispatched through the element tree.
// Declare one with a //gpui:action directive (internal/cmd/actiongen) or implement the three
// methods by hand and call RegisterAction from init. The static parts of upstream's trait
// (name_for_type, build, action_json_schema, deprecated_aliases, deprecation_message,
// documentation) are the fields of ActionData.
type Action interface {
	// Name returns the action's name, for display in UI.
	Name() string
	// BoxedClone clones the action.
	BoxedClone() Action
	// PartialEq reports whether other is an equal action.
	PartialEq(other Action) bool
}

// ActionData is the registration record of an action type (upstream's MacroActionData).
type ActionData struct {
	// Name is the action's qualified name.
	Name string
	// Type is the action's Go type.
	Type reflect.Type
	// Build builds the action from its JSON value; `{}` is passed for actions without parameters.
	Build func(value jsontext.Value) (Action, error)
	// JSONSchema returns the JSON schema of the action's value; nil means none.
	JSONSchema func() *jsonschema.Schema
	// DeprecatedAliases are other names the action can be built from.
	DeprecatedAliases []string
	// DeprecationMessage is shown when the action is used; empty means none.
	DeprecationMessage string
	// Documentation is the action's documentation; empty means none.
	Documentation string
}

var actionInventory struct {
	mu      sync.Mutex
	actions []ActionData
}

// RegisterAction registers an action type; call it from init. Duplicate names panic when an
// App loads the registry, as upstream's do at App creation.
func RegisterAction(data ActionData) {
	actionInventory.mu.Lock()
	defer actionInventory.mu.Unlock()
	actionInventory.actions = append(actionInventory.actions, data)
}

func registeredActionData() []ActionData {
	actionInventory.mu.Lock()
	defer actionInventory.mu.Unlock()
	return append([]ActionData(nil), actionInventory.actions...)
}

// ActionBuildError is the error returned when an action cannot be built: ActionBuildErrorNotFound
// or ActionBuildErrorBuildError.
type ActionBuildError interface {
	error
	isActionBuildError()
}

// ActionBuildErrorNotFound means no action with the name has been registered.
type ActionBuildErrorNotFound struct {
	// Name is the name of the action that was not found.
	Name string
}

// ActionBuildErrorBuildError means building the action failed, typically while decoding its JSON.
type ActionBuildErrorBuildError struct {
	// Name is the name of the action being built.
	Name string
	// Err is the error that occurred.
	Err error
}

func (*ActionBuildErrorNotFound) isActionBuildError()   {}
func (*ActionBuildErrorBuildError) isActionBuildError() {}

// Error formats the error.
func (e *ActionBuildErrorNotFound) Error() string {
	return fmt.Sprintf("Didn't find an action named \"%s\"", e.Name)
}

// Error formats the error.
func (e *ActionBuildErrorBuildError) Error() string {
	return fmt.Sprintf("Error while building action \"%s\": %v", e.Name, e.Err)
}

// Unwrap returns the source of the build error, as upstream's Error::source.
func (e *ActionBuildErrorBuildError) Unwrap() error {
	return e.Err
}

type registeredAction struct {
	build      func(value jsontext.Value) (Action, error)
	jsonSchema func() *jsonschema.Schema
}

type actionRegistry struct {
	byName              map[string]registeredAction
	namesByType         map[reflect.Type]string
	allNames            []string
	deprecatedAliases   map[string]string
	deprecationMessages map[string]string
	documentation       map[string]string
}

func newActionRegistry() *actionRegistry {
	r := &actionRegistry{
		byName:              map[string]registeredAction{},
		namesByType:         map[reflect.Type]string{},
		documentation:       map[string]string{},
		deprecatedAliases:   map[string]string{},
		deprecationMessages: map[string]string{},
	}
	r.loadActions()
	return r
}

func (r *actionRegistry) loadActions() {
	for _, action := range registeredActionData() {
		r.insertAction(action)
	}
}

func (r *actionRegistry) insertAction(action ActionData) {
	name := action.Name
	if _, ok := r.byName[name]; ok {
		panic(fmt.Sprintf("Action with name `%s` already registered "+
			"(might be registered in `#[action(deprecated_aliases = [...])]`.", name))
	}
	r.byName[name] = registeredAction{build: action.Build, jsonSchema: action.JSONSchema}
	for _, alias := range action.DeprecatedAliases {
		if _, ok := r.byName[alias]; ok {
			panic(fmt.Sprintf("Action with name `%s` already registered. "+
				"`%s` is specified in `#[action(deprecated_aliases = [...])]` for action `%s`.", alias, alias, name))
		}
		r.byName[alias] = registeredAction{build: action.Build, jsonSchema: action.JSONSchema}
		r.deprecatedAliases[alias] = name
		r.allNames = append(r.allNames, alias)
	}
	r.namesByType[action.Type] = name
	r.allNames = append(r.allNames, name)
	if action.DeprecationMessage != "" {
		r.deprecationMessages[name] = action.DeprecationMessage
	}
	if action.Documentation != "" {
		r.documentation[name] = action.Documentation
	}
}

func (r *actionRegistry) buildActionType(actionType reflect.Type) (Action, error) {
	name, ok := r.namesByType[actionType]
	if !ok {
		return nil, fmt.Errorf("no action type registered for %v", actionType)
	}
	action, err := r.buildAction(name, None[jsontext.Value]())
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (r *actionRegistry) tryResolveAction(actionType reflect.Type) (string, bool) {
	name, ok := r.namesByType[actionType]
	return name, ok
}

func (r *actionRegistry) buildAction(name string, params Option[jsontext.Value]) (Action, error) {
	data, ok := r.byName[name]
	if !ok {
		return nil, &ActionBuildErrorNotFound{Name: name}
	}
	action, err := data.build(params.UnwrapOr(jsontext.Value("{}")))
	if err != nil {
		return nil, &ActionBuildErrorBuildError{Name: name, Err: err}
	}
	return action, nil
}

func (r *actionRegistry) allActionNames() []string {
	return r.allNames
}

type actionSchema struct {
	name   string
	schema *jsonschema.Schema
}

func callActionSchema(f func() *jsonschema.Schema) *jsonschema.Schema {
	if f == nil {
		return nil
	}
	return f()
}

func (r *actionRegistry) actionSchemas() []actionSchema {
	out := make([]actionSchema, 0, len(r.allNames))
	for _, name := range r.allNames {
		data, ok := r.byName[name]
		if !ok {
			panic("All actions in all_names should be registered")
		}
		out = append(out, actionSchema{name: name, schema: callActionSchema(data.jsonSchema)})
	}
	return out
}

func (r *actionRegistry) actionSchemaByName(name string) (*jsonschema.Schema, bool) {
	data, ok := r.byName[name]
	if !ok {
		return nil, false
	}
	return callActionSchema(data.jsonSchema), true
}

func (r *actionRegistry) deprecatedAliasesMap() map[string]string {
	return r.deprecatedAliases
}

func (r *actionRegistry) deprecationMessagesMap() map[string]string {
	return r.deprecationMessages
}

func (r *actionRegistry) documentationMap() map[string]string {
	return r.documentation
}

// GenerateListOfAllRegisteredActions returns every registered action, for static analysis such
// as validating keymaps or generating documentation.
func GenerateListOfAllRegisteredActions() iter.Seq[ActionData] {
	actions := registeredActionData()
	return func(yield func(ActionData) bool) {
		for _, a := range actions {
			if !yield(a) {
				return
			}
		}
	}
}

// NoAction is the action with special handling that unbinds the keybinding it is associated
// with, if it is the highest precedence match.
type NoAction struct{}

// Name returns "zed::NoAction".
func (NoAction) Name() string {
	return "zed::NoAction"
}

// BoxedClone clones the action.
func (a NoAction) BoxedClone() Action {
	return a
}

// PartialEq reports whether other is a NoAction.
func (a NoAction) PartialEq(other Action) bool {
	o, ok := other.(NoAction)
	return ok && a == o
}

// Unbind is the action with special handling that unbinds later bindings for the same
// keystrokes when they dispatch the named action, regardless of that action's context. In
// keymap JSON it is written as `["zed::Unbind", "editor::NewLine"]`.
type Unbind SharedString

// Name returns "zed::Unbind".
func (Unbind) Name() string {
	return "zed::Unbind"
}

// BoxedClone clones the action.
func (a Unbind) BoxedClone() Action {
	return a
}

// PartialEq reports whether other is an equal Unbind.
func (a Unbind) PartialEq(other Action) bool {
	o, ok := other.(Unbind)
	return ok && a == o
}

func init() {
	RegisterAction(ActionData{
		Name: "zed::NoAction",
		Type: reflect.TypeFor[NoAction](),
		Build: func(jsontext.Value) (Action, error) {
			return NoAction{}, nil
		},
		Documentation: "Action with special handling which unbinds the keybinding this is associated with,\nif it is the highest precedence match.",
	})
	RegisterAction(ActionData{
		Name: "zed::Unbind",
		Type: reflect.TypeFor[Unbind](),
		Build: func(value jsontext.Value) (Action, error) {
			var a Unbind
			if err := json.Unmarshal(value, &a); err != nil {
				return nil, err
			}
			return a, nil
		},
		JSONSchema: func() *jsonschema.Schema {
			return jsonschema.NewGenerator().Reflect(reflect.TypeFor[Unbind]())
		},
		Documentation: "Action with special handling which unbinds later bindings for the same keystrokes when they\n" +
			"dispatch the named action, regardless of that action's context.\n\n" +
			"In keymap JSON this is written as:\n\n" +
			"`[\"zed::Unbind\", \"editor::NewLine\"]`",
	})
}

// IsNoAction reports whether the action represents a removed key binding.
func IsNoAction(action Action) bool {
	_, ok := action.(NoAction)
	return ok
}

// IsUnbind reports whether the action represents an unbind marker.
func IsUnbind(action Action) bool {
	_, ok := action.(Unbind)
	return ok
}
