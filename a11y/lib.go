// Package a11y is a pure-Go copy of the AccessKit 0.24 data model shared by elements, windows and platform adapters.
package a11y

//go:generate go run github.com/megascan/gpui-go/a11y/internal/gen

import (
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/jsonschema"
)

// Uuid is a 128-bit universally unique identifier.
type Uuid [16]byte

// String formats u in the hyphenated lowercase form.
func (u Uuid) String() string {
	var b [36]byte
	hex.Encode(b[0:8], u[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], u[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], u[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], u[8:10])
	b[23] = '-'
	hex.Encode(b[24:], u[10:])
	return string(b[:])
}

// MarshalText encodes u in the hyphenated form.
func (u Uuid) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// UnmarshalText parses the simple, hyphenated, braced and urn forms.
func (u *Uuid) UnmarshalText(text []byte) error {
	s := string(text)
	switch {
	case len(s) == 45 && strings.EqualFold(s[:9], "urn:uuid:"):
		s = s[9:]
	case len(s) == 38 && s[0] == '{' && s[37] == '}':
		s = s[1:37]
	}
	if len(s) == 36 {
		if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
			return fmt.Errorf("a11y: invalid uuid %q", text)
		}
		s = s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:]
	}
	if len(s) != 32 {
		return fmt.Errorf("a11y: invalid uuid %q", text)
	}
	var v Uuid
	if _, err := hex.Decode(v[:], []byte(s)); err != nil {
		return fmt.Errorf("a11y: invalid uuid %q: %w", text, err)
	}
	*u = v
	return nil
}

// NodeIDContent is the integer behind a NodeID.
type NodeIDContent = uint64

// NodeID is the stable identity of a Node, unique within the node's tree.
type NodeID NodeIDContent

// String formats id as Rust's Debug does, "#<n>".
func (id NodeID) String() string {
	return "#" + strconv.FormatUint(uint64(id), 10)
}

// TreeID is the stable identity of a Tree.
type TreeID Uuid

// TreeIDRoot returns the reserved tree ID of the root tree, the nil UUID.
func TreeIDRoot() TreeID {
	return TreeID(Uuid{})
}

// String formats id as Rust's Debug does, "TreeId(<uuid>)".
func (id TreeID) String() string {
	return "TreeId(" + Uuid(id).String() + ")"
}

// MarshalText encodes id as its UUID string.
func (id TreeID) MarshalText() ([]byte, error) {
	return Uuid(id).MarshalText()
}

// UnmarshalText decodes a UUID string.
func (id *TreeID) UnmarshalText(text []byte) error {
	return (*Uuid)(id).UnmarshalText(text)
}

func enumString(names []string, v uint8, typ string) string {
	if int(v) < len(names) {
		return names[v]
	}
	return typ + "(" + strconv.Itoa(int(v)) + ")"
}

func enumMarshal(names []string, v uint8, typ string) ([]byte, error) {
	if int(v) < len(names) {
		return []byte(names[v]), nil
	}
	return nil, fmt.Errorf("a11y: invalid %s value %d", typ, v)
}

func enumUnmarshal(names []string, text []byte, typ string) (uint8, error) {
	if i := slices.Index(names, string(text)); i >= 0 {
		return uint8(i), nil
	}
	return 0, fmt.Errorf("a11y: unknown variant `%s` of %s", text, typ)
}

func (a Action) mask() uint32 {
	return 1 << uint8(a)
}

func actionN(value uint8) (Action, bool) {
	if int(value) < len(actionNames) {
		return Action(value), true
	}
	return 0, false
}

func actionMaskToActionVec(mask uint32) []Action {
	actions := []Action{}
	var i uint8
	for {
		variant, ok := actionN(i)
		if !ok {
			break
		}
		if mask&variant.mask() != 0 {
			actions = append(actions, variant)
		}
		i++
	}
	return actions
}

func (f flag) mask() uint32 {
	return 1 << uint8(f)
}

func strictUnmarshal[T any](dec *jsontext.Decoder, v *T) error {
	raw, err := dec.ReadValue()
	if err != nil {
		return err
	}
	if raw.Kind() == '{' {
		var members map[string]jsontext.Value
		if err := json.Unmarshal(raw, &members); err != nil {
			return err
		}
		t := reflect.TypeFor[T]()
		for i := range t.NumField() {
			f := t.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "" || name == "-" || isOptionType(f.Type) {
				continue
			}
			if _, ok := members[name]; !ok {
				return fmt.Errorf("a11y: missing field `%s`", name)
			}
		}
	}
	return json.Unmarshal(raw, v, json.RejectUnknownMembers(true))
}

func isOptionType(t reflect.Type) bool {
	return t.PkgPath() == "github.com/megascan/gpui-go/gpuiutil" && strings.HasPrefix(t.Name(), "Option[")
}

// CustomAction defines a custom action for a UI element.
type CustomAction struct {
	ID          int32  `json:"id"`
	Description string `json:"description"`
}

// String formats c as Rust's derived Debug does.
func (c CustomAction) String() string {
	return "CustomAction { id: " + strconv.Itoa(int(c.ID)) + ", description: " + debugStr(c.Description) + " }"
}

// UnmarshalJSONFrom decodes c, rejecting unknown fields.
func (c *CustomAction) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain CustomAction
	return strictUnmarshal(dec, (*plain)(c))
}

// TextPosition is a position within the text of a TextRun node.
type TextPosition struct {
	Node           NodeID `json:"node"`
	CharacterIndex int    `json:"characterIndex"`
}

// String formats p as Rust's derived Debug does.
func (p TextPosition) String() string {
	return "TextPosition { node: " + p.Node.String() + ", character_index: " + strconv.Itoa(p.CharacterIndex) + " }"
}

// UnmarshalJSONFrom decodes p, rejecting unknown fields.
func (p *TextPosition) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain TextPosition
	return strictUnmarshal(dec, (*plain)(p))
}

// TextSelection is a text selection, or a caret when Anchor equals Focus.
type TextSelection struct {
	Anchor TextPosition `json:"anchor"`
	Focus  TextPosition `json:"focus"`
}

// String formats s as Rust's derived Debug does.
func (s TextSelection) String() string {
	return "TextSelection { anchor: " + s.Anchor.String() + ", focus: " + s.Focus.String() + " }"
}

// UnmarshalJSONFrom decodes s, rejecting unknown fields.
func (s *TextSelection) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain TextSelection
	return strictUnmarshal(dec, (*plain)(s))
}

// Color is a color represented in 8-bit sRGB plus alpha.
type Color struct {
	Red   uint8 `json:"red"`
	Green uint8 `json:"green"`
	Blue  uint8 `json:"blue"`
	Alpha uint8 `json:"alpha"`
}

// String formats c as Rust's derived Debug does.
func (c Color) String() string {
	return fmt.Sprintf("Color { red: %d, green: %d, blue: %d, alpha: %d }", c.Red, c.Green, c.Blue, c.Alpha)
}

// UnmarshalJSONFrom decodes c, rejecting unknown fields.
func (c *Color) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain Color
	return strictUnmarshal(dec, (*plain)(c))
}

// TextDecoration is the style and color for a type of text decoration.
type TextDecoration struct {
	Style TextDecorationStyle `json:"style"`
	Color Color               `json:"color"`
}

// String formats d as Rust's derived Debug does.
func (d TextDecoration) String() string {
	return "TextDecoration { style: " + d.Style.String() + ", color: " + d.Color.String() + " }"
}

// UnmarshalJSONFrom decodes d, rejecting unknown fields.
func (d *TextDecoration) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain TextDecoration
	return strictUnmarshal(dec, (*plain)(d))
}

type propertyValue interface{ isPropertyValue() }

type (
	pvNone            struct{}
	pvNodeIDVec       []NodeID
	pvNodeID          NodeID
	pvString          string
	pvF64             float64
	pvF32             float32
	pvUsize           int
	pvColor           Color
	pvTextDecoration  TextDecoration
	pvLengthSlice     []uint8
	pvCoordSlice      []float32
	pvBool            bool
	pvAffine          Affine
	pvRect            Rect
	pvTextSelection   TextSelection
	pvCustomActionVec []CustomAction
	pvTreeID          TreeID
)

func (pvNone) isPropertyValue()            {}
func (pvNodeIDVec) isPropertyValue()       {}
func (pvNodeID) isPropertyValue()          {}
func (pvString) isPropertyValue()          {}
func (pvF64) isPropertyValue()             {}
func (pvF32) isPropertyValue()             {}
func (pvUsize) isPropertyValue()           {}
func (pvColor) isPropertyValue()           {}
func (pvTextDecoration) isPropertyValue()  {}
func (pvLengthSlice) isPropertyValue()     {}
func (pvCoordSlice) isPropertyValue()      {}
func (pvBool) isPropertyValue()            {}
func (pvAffine) isPropertyValue()          {}
func (pvRect) isPropertyValue()            {}
func (pvTextSelection) isPropertyValue()   {}
func (pvCustomActionVec) isPropertyValue() {}
func (pvTreeID) isPropertyValue()          {}

func propertyValueEqual(a, b propertyValue) bool {
	switch x := a.(type) {
	case pvNodeIDVec:
		y, ok := b.(pvNodeIDVec)
		return ok && slices.Equal(x, y)
	case pvLengthSlice:
		y, ok := b.(pvLengthSlice)
		return ok && slices.Equal(x, y)
	case pvCoordSlice:
		y, ok := b.(pvCoordSlice)
		return ok && slices.Equal(x, y)
	case pvCustomActionVec:
		y, ok := b.(pvCustomActionVec)
		return ok && slices.Equal(x, y)
	}
	return a == b
}

func clonePropertyValue(v propertyValue) propertyValue {
	switch x := v.(type) {
	case pvNodeIDVec:
		return pvNodeIDVec(cloneSlice(x))
	case pvLengthSlice:
		return pvLengthSlice(cloneSlice(x))
	case pvCoordSlice:
		return pvCoordSlice(cloneSlice(x))
	case pvCustomActionVec:
		return pvCustomActionVec(cloneSlice(x))
	}
	return v
}

func cloneSlice[S ~[]E, E any](s S) S {
	if s == nil {
		return S{}
	}
	return slices.Clone(s)
}

type propertyIndices [propertyIDUnset]uint8

func (p *propertyIndices) get(values []propertyValue, id propertyID) propertyValue {
	index := p[id]
	if index == 0 {
		return pvNone{}
	}
	return values[index-1]
}

type properties struct {
	indices propertyIndices
	values  []propertyValue
}

func (p *properties) getMut(id propertyID, def propertyValue) int {
	index := p.indices[id]
	if index == 0 {
		p.values = append(slices.Clip(p.values), def)
		p.indices[id] = uint8(len(p.values))
		return len(p.values) - 1
	}
	p.values = slices.Clone(p.values)
	return int(index) - 1
}

func (p *properties) set(id propertyID, value propertyValue) {
	index := p.indices[id]
	if index == 0 {
		p.values = append(slices.Clip(p.values), value)
		p.indices[id] = uint8(len(p.values))
	} else {
		p.values = slices.Clone(p.values)
		p.values[index-1] = value
	}
}

func (p *properties) clear(id propertyID) {
	index := p.indices[id]
	if index != 0 {
		p.values = slices.Clone(p.values)
		p.values[index-1] = pvNone{}
	}
}

func (p *properties) equal(o *properties) bool {
	if p.indices != o.indices || len(p.values) != len(o.values) {
		return false
	}
	for i := range p.values {
		if !propertyValueEqual(p.values[i], o.values[i]) {
			return false
		}
	}
	return true
}

func (p properties) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for id, index := range p.indices {
		if index == 0 {
			continue
		}
		value := p.values[index-1]
		if _, ok := value.(pvNone); ok {
			continue
		}
		if err := enc.WriteToken(jsontext.String(propertyIDJSONNames[id])); err != nil {
			return err
		}
		if err := marshalPropertyValue(enc, value); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

func (p *properties) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	*p = properties{}
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '{' {
		return errors.New("a11y: invalid type, expected property map")
	}
	for dec.PeekKind() != '}' {
		key, err := dec.ReadToken()
		if err != nil {
			return err
		}
		i := slices.Index(propertyIDJSONNames[:], key.String())
		if i < 0 {
			return fmt.Errorf("a11y: unknown variant `%s` of PropertyId", key.String())
		}
		id := propertyID(i)
		if id == propertyIDUnset {
			if err := dec.SkipValue(); err != nil {
				return err
			}
			continue
		}
		value, err := unmarshalPropertyValue(dec, id)
		if err != nil {
			return err
		}
		p.set(id, value)
	}
	_, err = dec.ReadToken()
	return err
}

// JSONSchema describes the property map, one camelCase key per property, as accesskit's hand-written JsonSchema impl does.
func (p *properties) JSONSchema(g *jsonschema.Generator) *jsonschema.Schema {
	props := map[string]*jsonschema.Schema{}
	for _, e := range propertySchemaTypes {
		var sub *jsonschema.Schema
		if e.t == reflect.TypeFor[[]uint8]() {
			sub = &jsonschema.Schema{Type: jsonschema.Types{"array"}, Items: g.SchemaFor(reflect.TypeFor[uint8]())}
		} else {
			sub = g.SchemaFor(e.t)
		}
		props[propertyIDJSONNames[e.id]] = sub
	}
	return &jsonschema.Schema{Type: jsonschema.Types{"object"}, Properties: props}
}

// JSONSchema describes the serde shape of Node.
func (n *Node) JSONSchema(g *jsonschema.Generator) *jsonschema.Schema {
	u32 := g.SchemaFor(reflect.TypeFor[uint32]())
	return &jsonschema.Schema{
		Type: jsonschema.Types{"object"},
		Properties: map[string]*jsonschema.Schema{
			"role":         g.SchemaFor(reflect.TypeFor[Role]()),
			"actions":      u32,
			"childActions": u32,
			"flags":        u32,
			"properties":   g.SchemaFor(reflect.TypeFor[properties]()),
		},
		Required:             []string{"role", "actions", "childActions", "flags", "properties"},
		AdditionalProperties: false,
	}
}

// JSONSchema describes u as a UUID string.
func (u *Uuid) JSONSchema(g *jsonschema.Generator) *jsonschema.Schema {
	return &jsonschema.Schema{Type: jsonschema.Types{"string"}, Format: "uuid"}
}

// JSONSchema describes id as a UUID string.
func (id *TreeID) JSONSchema(g *jsonschema.Generator) *jsonschema.Schema {
	return g.SchemaFor(reflect.TypeFor[Uuid]())
}

func enumSchema(names []string) *jsonschema.Schema {
	enum := make([]any, len(names))
	for i, n := range names {
		enum[i] = n
	}
	return &jsonschema.Schema{Type: jsonschema.Types{"string"}, Enum: enum}
}

func marshalLengthSlice(enc *jsontext.Encoder, v []uint8) error {
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, x := range v {
		if err := enc.WriteToken(jsontext.Uint(uint64(x))); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndArray)
}

func unmarshalLengthSlice(dec *jsontext.Decoder) ([]uint8, error) {
	var raw []int
	if err := json.UnmarshalDecode(dec, &raw); err != nil {
		return nil, err
	}
	out := make([]uint8, len(raw))
	for i, x := range raw {
		if x < 0 || x > 255 {
			return nil, fmt.Errorf("a11y: invalid value %d, expected u8", x)
		}
		out[i] = uint8(x)
	}
	return out, nil
}

// Node is a single accessible object. A complete UI is represented as a tree of these.
type Node struct {
	role         Role
	actions      uint32
	childActions uint32
	flags        uint32
	properties   properties
}

// NewNode returns a node with the given role and no other properties.
func NewNode(role Role) Node {
	return Node{role: role}
}

// Role returns the node's role.
func (n *Node) Role() Role {
	return n.role
}

// SetRole sets the node's role.
func (n *Node) SetRole(value Role) {
	n.role = value
}

// SupportsAction reports whether the node supports action.
func (n *Node) SupportsAction(action Action) bool {
	return n.actions&action.mask() != 0
}

// AddAction marks action as supported.
func (n *Node) AddAction(action Action) {
	n.actions |= action.mask()
}

// RemoveAction marks action as unsupported.
func (n *Node) RemoveAction(action Action) {
	n.actions &^= action.mask()
}

// ClearActions removes every supported action.
func (n *Node) ClearActions() {
	n.actions = 0
}

// ChildSupportsAction reports whether the node's children support action.
func (n *Node) ChildSupportsAction(action Action) bool {
	return n.childActions&action.mask() != 0
}

// AddChildAction marks action as supported by the node's children.
func (n *Node) AddChildAction(action Action) {
	n.childActions |= action.mask()
}

// RemoveChildAction marks action as unsupported by the node's children.
func (n *Node) RemoveChildAction(action Action) {
	n.childActions &^= action.mask()
}

// ClearChildActions removes every action supported by the node's children.
func (n *Node) ClearChildActions() {
	n.childActions = 0
}

// Clone returns a deep copy of n.
func (n *Node) Clone() Node {
	c := *n
	if n.properties.values != nil {
		c.properties.values = make([]propertyValue, len(n.properties.values))
		for i, v := range n.properties.values {
			c.properties.values[i] = clonePropertyValue(v)
		}
	}
	return c
}

// Equal reports whether n and o are equal as Rust's derived PartialEq compares them.
func (n *Node) Equal(o *Node) bool {
	return n.role == o.role && n.actions == o.actions && n.childActions == o.childActions &&
		n.flags == o.flags && n.properties.equal(&o.properties)
}

// String formats n as accesskit's Debug implementation does.
func (n *Node) String() string {
	d := debugStruct{name: "Node"}
	d.field("role", n.Role().String())
	if supported := actionMaskToActionVec(n.actions); len(supported) != 0 {
		d.field("actions", debugList(supported))
	}
	if supported := actionMaskToActionVec(n.childActions); len(supported) != 0 {
		d.field("child_actions", debugList(supported))
	}
	n.debugFlagProperties(&d)
	n.debugNodeIDVecProperties(&d)
	n.debugNodeIDProperties(&d)
	n.debugStringProperties(&d)
	n.debugF64Properties(&d)
	n.debugF32Properties(&d)
	n.debugUsizeProperties(&d)
	n.debugColorProperties(&d)
	n.debugTextDecorationProperties(&d)
	n.debugLengthSliceProperties(&d)
	n.debugCoordSliceProperties(&d)
	n.debugBoolProperties(&d)
	n.debugUniqueEnumProperties(&d)
	n.debugOptionProperties(&d)
	if customActions := n.CustomActions(); len(customActions) != 0 {
		d.field("custom_actions", debugCustomActions(customActions))
	}
	return d.finish()
}

// MarshalJSONTo encodes n as accesskit's serde derive does.
func (n Node) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("role")); err != nil {
		return err
	}
	if err := json.MarshalEncode(enc, n.role); err != nil {
		return err
	}
	for _, f := range []struct {
		name  string
		value uint32
	}{{"actions", n.actions}, {"childActions", n.childActions}, {"flags", n.flags}} {
		if err := enc.WriteToken(jsontext.String(f.name)); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.Uint(uint64(f.value))); err != nil {
			return err
		}
	}
	if err := enc.WriteToken(jsontext.String("properties")); err != nil {
		return err
	}
	if err := n.properties.MarshalJSONTo(enc); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSONFrom decodes n, rejecting unknown and missing fields.
func (n *Node) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	*n = Node{}
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '{' {
		return errors.New("a11y: invalid type, expected struct Node")
	}
	names := []string{"role", "actions", "childActions", "flags", "properties"}
	var seen [5]bool
	for dec.PeekKind() != '}' {
		key, err := dec.ReadToken()
		if err != nil {
			return err
		}
		i := slices.Index(names, key.String())
		if i >= 0 && seen[i] {
			return fmt.Errorf("a11y: duplicate field `%s`", names[i])
		}
		switch i {
		case 0:
			err = json.UnmarshalDecode(dec, &n.role)
		case 1:
			err = json.UnmarshalDecode(dec, &n.actions)
		case 2:
			err = json.UnmarshalDecode(dec, &n.childActions)
		case 3:
			err = json.UnmarshalDecode(dec, &n.flags)
		case 4:
			err = n.properties.UnmarshalJSONFrom(dec)
		default:
			return fmt.Errorf("a11y: unknown field `%s`", key.String())
		}
		if err != nil {
			return err
		}
		seen[i] = true
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	for i, ok := range seen {
		if !ok {
			return fmt.Errorf("a11y: missing field `%s`", names[i])
		}
	}
	return nil
}

// Tree is the data associated with an accessibility tree that is not associated with a specific node.
type Tree struct {
	Root           NodeID                  `json:"root"`
	ToolkitName    gpuiutil.Option[string] `json:"toolkitName"`
	ToolkitVersion gpuiutil.Option[string] `json:"toolkitVersion"`
}

// NewTree returns a tree with the given root and no toolkit information.
func NewTree(root NodeID) Tree {
	return Tree{Root: root}
}

// UnmarshalJSONFrom decodes t, rejecting unknown fields.
func (t *Tree) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain Tree
	return strictUnmarshal(dec, (*plain)(t))
}

// NodeEntry is one (NodeID, Node) pair of a TreeUpdate.
type NodeEntry struct {
	ID   NodeID
	Node Node
}

// MarshalJSONTo encodes e as the two-element array serde uses for tuples.
func (e NodeEntry) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Uint(uint64(e.ID))); err != nil {
		return err
	}
	if err := e.Node.MarshalJSONTo(enc); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndArray)
}

// UnmarshalJSONFrom decodes a two-element [id, node] array.
func (e *NodeEntry) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '[' {
		return errors.New("a11y: invalid type, expected a tuple of size 2")
	}
	if err := json.UnmarshalDecode(dec, &e.ID); err != nil {
		return err
	}
	if err := e.Node.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	tok, err = dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != ']' {
		return errors.New("a11y: invalid length, expected a tuple of size 2")
	}
	return nil
}

// TreeUpdate is a serializable representation of an atomic change to a Tree.
type TreeUpdate struct {
	Nodes  []NodeEntry           `json:"nodes"`
	Tree   gpuiutil.Option[Tree] `json:"tree"`
	TreeID TreeID                `json:"treeId"`
	Focus  NodeID                `json:"focus"`
}

// UnmarshalJSONFrom decodes u, rejecting unknown fields.
func (u *TreeUpdate) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain TreeUpdate
	return strictUnmarshal(dec, (*plain)(u))
}

// Clone returns a deep copy of u.
func (u *TreeUpdate) Clone() TreeUpdate {
	c := *u
	if u.Nodes != nil {
		c.Nodes = make([]NodeEntry, len(u.Nodes))
		for i, e := range u.Nodes {
			c.Nodes[i] = NodeEntry{ID: e.ID, Node: e.Node.Clone()}
		}
	}
	return c
}

// Equal reports whether u and o are equal as Rust's derived PartialEq compares them.
func (u *TreeUpdate) Equal(o *TreeUpdate) bool {
	if u.Tree != o.Tree || u.TreeID != o.TreeID || u.Focus != o.Focus || len(u.Nodes) != len(o.Nodes) {
		return false
	}
	for i := range u.Nodes {
		if u.Nodes[i].ID != o.Nodes[i].ID || !u.Nodes[i].Node.Equal(&o.Nodes[i].Node) {
			return false
		}
	}
	return true
}

// ActionData is the optional payload of an ActionRequest.
type ActionData interface{ isActionData() }

type (
	// ActionDataCustomAction carries the ID of a CustomAction.
	ActionDataCustomAction int32
	// ActionDataValue carries a text value.
	ActionDataValue string
	// ActionDataNumericValue carries a numeric value.
	ActionDataNumericValue float64
	// ActionDataScrollUnit carries the unit of a scroll action.
	ActionDataScrollUnit ScrollUnit
	// ActionDataScrollHint carries where ScrollIntoView should place the node.
	ActionDataScrollHint ScrollHint
	// ActionDataScrollToPoint carries the target point in the tree's container coordinate space.
	ActionDataScrollToPoint Point
	// ActionDataSetScrollOffset carries the scroll offset to set.
	ActionDataSetScrollOffset Point
	// ActionDataSetTextSelection carries the text selection to set.
	ActionDataSetTextSelection TextSelection
)

func (ActionDataCustomAction) isActionData()     {}
func (ActionDataValue) isActionData()            {}
func (ActionDataNumericValue) isActionData()     {}
func (ActionDataScrollUnit) isActionData()       {}
func (ActionDataScrollHint) isActionData()       {}
func (ActionDataScrollToPoint) isActionData()    {}
func (ActionDataSetScrollOffset) isActionData()  {}
func (ActionDataSetTextSelection) isActionData() {}

func marshalActionData(enc *jsontext.Encoder, data ActionData) error {
	var name string
	var value any
	switch v := data.(type) {
	case ActionDataCustomAction:
		name, value = "customAction", int32(v)
	case ActionDataValue:
		name, value = "value", string(v)
	case ActionDataNumericValue:
		name, value = "numericValue", float64(v)
	case ActionDataScrollUnit:
		name, value = "scrollUnit", ScrollUnit(v)
	case ActionDataScrollHint:
		name, value = "scrollHint", ScrollHint(v)
	case ActionDataScrollToPoint:
		name, value = "scrollToPoint", Point(v)
	case ActionDataSetScrollOffset:
		name, value = "setScrollOffset", Point(v)
	case ActionDataSetTextSelection:
		name, value = "setTextSelection", TextSelection(v)
	default:
		return fmt.Errorf("a11y: unknown ActionData %T", data)
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(name)); err != nil {
		return err
	}
	if err := json.MarshalEncode(enc, value); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

func unmarshalActionData(dec *jsontext.Decoder) (ActionData, error) {
	if dec.PeekKind() == '"' {
		tok, err := dec.ReadToken()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("a11y: invalid type: unit variant `%s`, expected newtype variant of ActionData", tok.String())
	}
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	if tok.Kind() != '{' {
		return nil, errors.New("a11y: invalid type, expected enum ActionData")
	}
	key, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	var data ActionData
	switch key.String() {
	case "customAction":
		var v int32
		err = json.UnmarshalDecode(dec, &v)
		data = ActionDataCustomAction(v)
	case "value":
		var v string
		err = json.UnmarshalDecode(dec, &v)
		data = ActionDataValue(v)
	case "numericValue":
		var v float64
		err = json.UnmarshalDecode(dec, &v)
		data = ActionDataNumericValue(v)
	case "scrollUnit":
		var v ScrollUnit
		err = json.UnmarshalDecode(dec, &v)
		data = ActionDataScrollUnit(v)
	case "scrollHint":
		var v ScrollHint
		err = json.UnmarshalDecode(dec, &v)
		data = ActionDataScrollHint(v)
	case "scrollToPoint":
		var v Point
		err = json.UnmarshalDecode(dec, &v)
		data = ActionDataScrollToPoint(v)
	case "setScrollOffset":
		var v Point
		err = json.UnmarshalDecode(dec, &v)
		data = ActionDataSetScrollOffset(v)
	case "setTextSelection":
		var v TextSelection
		err = json.UnmarshalDecode(dec, &v)
		data = ActionDataSetTextSelection(v)
	default:
		return nil, fmt.Errorf("a11y: unknown variant `%s` of ActionData", key.String())
	}
	if err != nil {
		return nil, err
	}
	tok, err = dec.ReadToken()
	if err != nil {
		return nil, err
	}
	if tok.Kind() != '}' {
		return nil, errors.New("a11y: invalid ActionData, expected a single variant")
	}
	return data, nil
}

// ActionRequest is a request to perform an action on a node.
type ActionRequest struct {
	Action     Action
	TargetTree TreeID
	TargetNode NodeID
	Data       ActionData
}

// MarshalJSONTo encodes r as accesskit's serde derive does; a nil Data is null.
func (r ActionRequest) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for _, f := range []struct {
		name  string
		value any
	}{{"action", r.Action}, {"targetTree", r.TargetTree}, {"targetNode", r.TargetNode}} {
		if err := enc.WriteToken(jsontext.String(f.name)); err != nil {
			return err
		}
		if err := json.MarshalEncode(enc, f.value); err != nil {
			return err
		}
	}
	if err := enc.WriteToken(jsontext.String("data")); err != nil {
		return err
	}
	if r.Data == nil {
		if err := enc.WriteToken(jsontext.Null); err != nil {
			return err
		}
	} else if err := marshalActionData(enc, r.Data); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSONFrom decodes r, rejecting unknown and missing fields.
func (r *ActionRequest) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	*r = ActionRequest{}
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '{' {
		return errors.New("a11y: invalid type, expected struct ActionRequest")
	}
	names := []string{"action", "targetTree", "targetNode", "data"}
	var seen [4]bool
	for dec.PeekKind() != '}' {
		key, err := dec.ReadToken()
		if err != nil {
			return err
		}
		i := slices.Index(names, key.String())
		if i >= 0 && seen[i] {
			return fmt.Errorf("a11y: duplicate field `%s`", names[i])
		}
		switch i {
		case 0:
			err = json.UnmarshalDecode(dec, &r.Action)
		case 1:
			err = json.UnmarshalDecode(dec, &r.TargetTree)
		case 2:
			err = json.UnmarshalDecode(dec, &r.TargetNode)
		case 3:
			if dec.PeekKind() == 'n' {
				_, err = dec.ReadToken()
			} else {
				r.Data, err = unmarshalActionData(dec)
			}
		default:
			return fmt.Errorf("a11y: unknown field `%s`", key.String())
		}
		if err != nil {
			return err
		}
		seen[i] = true
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	for i, ok := range seen[:3] {
		if !ok {
			return fmt.Errorf("a11y: missing field `%s`", names[i])
		}
	}
	return nil
}

// ActivationHandler handles requests from assistive technologies or other clients to activate accessibility.
type ActivationHandler interface {
	RequestInitialTree() (TreeUpdate, bool)
}

// ActionHandler handles requests from assistive technologies or other clients to perform actions.
type ActionHandler interface {
	DoAction(request ActionRequest)
}

// DeactivationHandler handles requests from assistive technologies or other clients to deactivate accessibility.
type DeactivationHandler interface {
	DeactivateAccessibility()
}

type debugStruct struct {
	name   string
	fields []string
}

func (d *debugStruct) field(name, value string) {
	d.fields = append(d.fields, name+": "+value)
}

func (d *debugStruct) finish() string {
	if len(d.fields) == 0 {
		return d.name
	}
	return d.name + " { " + strings.Join(d.fields, ", ") + " }"
}

func debugList[T fmt.Stringer](items []T) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = it.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func debugNodeIDs(v []NodeID) string {
	return debugList(v)
}

func debugCustomActions(v []CustomAction) string {
	return debugList(v)
}

func debugLengths(v []uint8) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = strconv.Itoa(int(x))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func debugCoords(v []float32) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = debugF32(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func debugStr(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case 0:
			b.WriteString(`\0`)
		default:
			if unicode.IsGraphic(r) && !(b.Len() == 1 && isGraphemeExtend(r)) {
				b.WriteRune(r)
			} else {
				b.WriteString(`\u{` + strconv.FormatInt(int64(r), 16) + `}`)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func isGraphemeExtend(r rune) bool {
	return unicode.In(r, unicode.Mn, unicode.Me, unicode.Other_Grapheme_Extend)
}
