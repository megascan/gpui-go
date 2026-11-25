package gpui

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"time"

	"encoding/json/jsontext"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/gpuiutil"
)

type frameDebugInfo struct {
	viewportSize Size[Pixels]
	scaleFactor  float32
	tabStopCount int
}

type capturedFrame struct {
	renderedAt   string
	frameNumber  uint64
	windowTitle  Option[SharedString]
	nodeCount    int
	tabStopCount int
	viewportSize Size[Pixels]
	scaleFactor  float32
}

type nodeDebugInfo struct {
	synthetic      bool
	view           Option[string]
	elementID      Option[string]
	sourceLocation *SourceLocation
}

type nodeCreator struct {
	view           Option[string]
	elementID      Option[string]
	sourceLocation *SourceLocation
}

type a11yDebug struct {
	lastTreeUpdate       Option[a11y.TreeUpdate]
	lastGpuiFocus        Option[a11y.NodeID]
	lastActiveDescendant Option[a11y.NodeID]
	frameNumber          uint64
	lastFrame            Option[capturedFrame]
	lastNodeInfo         map[a11y.NodeID]nodeDebugInfo
}

func (d *a11yDebug) capture(update *a11y.TreeUpdate, gpuiFocus Option[a11y.NodeID], activeDescendant Option[a11y.NodeID], windowTitle Option[SharedString], frame frameDebugInfo) {
	d.lastTreeUpdate = Some(update.Clone())
	d.lastGpuiFocus = gpuiFocus
	d.lastActiveDescendant = activeDescendant
	d.frameNumber++
	d.lastFrame = Some(capturedFrame{
		renderedAt:   time.Now().Format("2006-01-02T15:04:05.000-07:00"),
		frameNumber:  d.frameNumber,
		windowTitle:  windowTitle,
		nodeCount:    len(update.Nodes),
		tabStopCount: frame.tabStopCount,
		viewportSize: frame.viewportSize,
		scaleFactor:  frame.scaleFactor,
	})
}

func (d *a11yDebug) captureNodeInfo(nodeInfo map[a11y.NodeID]nodeDebugInfo) {
	d.lastNodeInfo = make(map[a11y.NodeID]nodeDebugInfo, len(nodeInfo))
	for id, info := range nodeInfo {
		d.lastNodeInfo[id] = info
	}
}

type debugJSONEntry struct {
	key   string
	value any
}

type debugJSONObject []debugJSONEntry

type debugJSONFloat float64

type debugJSONFloat32 float32

func (o *debugJSONObject) insert(key string, value any) {
	for i := range *o {
		if (*o)[i].key == key {
			(*o)[i].value = value
			return
		}
	}
	*o = append(*o, debugJSONEntry{key: key, value: value})
}

type nodeProvenance struct {
	elementID      Option[string]
	view           Option[string]
	sourceLocation Option[string]
	synthetic      Option[bool]
}

func (d *a11yDebug) toJSON() (string, bool) {
	update, ok := d.lastTreeUpdate.Get()
	if !ok {
		return "", false
	}

	ephemeral := make(map[a11y.NodeID]string, len(update.Nodes))
	for index, entry := range update.Nodes {
		ephemeral[entry.ID] = ephemeralID(index)
	}
	lookup := func(id a11y.NodeID) any {
		if key, ok := ephemeral[id]; ok {
			return key
		}
		return nil
	}

	var nodes debugJSONObject
	for _, entry := range update.Nodes {
		key, ok := ephemeral[entry.ID]
		if !ok {
			key = strconv.FormatUint(uint64(entry.ID), 10)
		}
		var provenance nodeProvenance
		if gpuiutil.DebugAssertions {
			if info, ok := d.lastNodeInfo[entry.ID]; ok {
				provenance.elementID = info.elementID
				provenance.view = info.view
				if info.sourceLocation != nil {
					provenance.sourceLocation = Some(info.sourceLocation.String())
				}
				if info.synthetic {
					provenance.synthetic = Some(true)
				}
			}
		}
		nodes.insert(key, nodeToJSON(entry.ID, &entry.Node, ephemeral, &provenance))
	}

	var frame any
	if f, ok := d.lastFrame.Get(); ok {
		var title any
		if t, ok := f.windowTitle.Get(); ok {
			title = string(t)
		}
		frame = debugJSONObject{
			{"rendered_at", f.renderedAt},
			{"frame_number", f.frameNumber},
			{"window_title", title},
			{"node_count", f.nodeCount},
			{"tab_stop_count", f.tabStopCount},
			{"viewport_size", debugJSONObject{
				{"width", debugJSONFloat32(f.viewportSize.Width)},
				{"height", debugJSONFloat32(f.viewportSize.Height)},
			}},
			{"scale_factor", debugJSONFloat32(f.scaleFactor)},
		}
	}

	var root any
	if tree, ok := update.Tree.Get(); ok {
		root = lookup(tree.Root)
	}
	var gpuiFocus, activeDescendant any
	if id, ok := d.lastGpuiFocus.Get(); ok {
		gpuiFocus = lookup(id)
	}
	if id, ok := d.lastActiveDescendant.Get(); ok {
		activeDescendant = lookup(id)
	}

	value := debugJSONObject{
		{"root", root},
		{"gpui_focus", gpuiFocus},
		{"active_descendant_focus", activeDescendant},
		{"frame", frame},
		{"nodes", nodes},
	}
	var buf bytes.Buffer
	writeDebugJSON(&buf, value, 0)
	return buf.String(), true
}

func nodeToJSON(id a11y.NodeID, node *a11y.Node, ephemeral map[a11y.NodeID]string, provenance *nodeProvenance) debugJSONObject {
	var m debugJSONObject
	m.insert("accesskit_id", strconv.FormatUint(uint64(id), 10))

	var children []string
	for _, child := range node.Children() {
		key, ok := ephemeral[child]
		if !ok {
			key = strconv.FormatUint(uint64(child), 10)
		}
		children = append(children, key)
	}
	if len(children) > 0 {
		m.insert("children", children)
	}

	if v, ok := provenance.elementID.Get(); ok {
		m.insert("element_id", v)
	}
	if v, ok := provenance.view.Get(); ok {
		m.insert("view", v)
	}
	if v, ok := provenance.sourceLocation.Get(); ok {
		m.insert("source_location", v)
	}
	if v, ok := provenance.synthetic.Get(); ok {
		m.insert("synthetic", v)
	}

	var aria debugJSONObject
	aria.insert("role", node.Role().String())

	var onAction []string
	for action := a11y.Action(0); a11yActionExists(action); action++ {
		if node.SupportsAction(action) {
			onAction = append(onAction, action.String())
		}
	}
	if len(onAction) > 0 {
		aria.insert("on_action", onAction)
	}

	insertString := func(key string, get func() (string, bool)) {
		if v, ok := get(); ok {
			aria.insert(key, v)
		}
	}
	insertString("label", node.Label)
	insertString("description", node.Description)
	insertString("value", node.Value)
	insertString("keyboard_shortcut", node.KeyboardShortcut)
	insertString("access_key", node.AccessKey)
	insertString("placeholder", node.Placeholder)
	insertString("tooltip", node.Tooltip)
	insertString("role_description", node.RoleDescription)

	if v, ok := node.IsSelected(); ok {
		aria.insert("selected", v)
	}
	if v, ok := node.IsExpanded(); ok {
		aria.insert("expanded", v)
	}
	if v, ok := node.Toggled(); ok {
		aria.insert("toggled", v.String())
	}
	if v, ok := node.Orientation(); ok {
		aria.insert("orientation", v.String())
	}

	insertFloat := func(key string, get func() (float64, bool)) {
		if v, ok := get(); ok {
			aria.insert(key, debugJSONFloat(v))
		}
	}
	insertFloat("numeric_value", node.NumericValue)
	insertFloat("min_numeric_value", node.MinNumericValue)
	insertFloat("max_numeric_value", node.MaxNumericValue)
	insertFloat("numeric_value_step", node.NumericValueStep)

	insertInt := func(key string, get func() (int, bool)) {
		if v, ok := get(); ok {
			aria.insert(key, v)
		}
	}
	insertInt("level", node.Level)
	insertInt("position_in_set", node.PositionInSet)
	insertInt("size_of_set", node.SizeOfSet)
	insertInt("row_index", node.RowIndex)
	insertInt("column_index", node.ColumnIndex)
	insertInt("row_count", node.RowCount)
	insertInt("column_count", node.ColumnCount)

	m.insert("aria", aria)
	return m
}

func a11yActionExists(action a11y.Action) bool {
	return action <= a11y.ActionShowContextMenu
}

func ephemeralID(index int) string {
	var b []byte
	for {
		b = append(b, byte('a'+index%26))
		if index < 26 {
			break
		}
		index = index/26 - 1
	}
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

func writeDebugJSON(buf *bytes.Buffer, value any, indent int) {
	switch v := value.(type) {
	case nil:
		buf.WriteString("null")
	case string:
		quoted, _ := jsontext.AppendQuote(nil, v)
		buf.Write(quoted)
	case bool:
		buf.WriteString(strconv.FormatBool(v))
	case int:
		buf.WriteString(strconv.Itoa(v))
	case uint64:
		buf.WriteString(strconv.FormatUint(v, 10))
	case debugJSONFloat:
		buf.WriteString(formatSerdeFloat(float64(v), 64))
	case debugJSONFloat32:
		buf.WriteString(formatSerdeFloat(float64(v), 32))
	case []string:
		if len(v) == 0 {
			buf.WriteString("[]")
			return
		}
		buf.WriteString("[\n")
		for i, item := range v {
			buf.WriteString(strings.Repeat("  ", indent+1))
			writeDebugJSON(buf, item, indent+1)
			if i+1 < len(v) {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(strings.Repeat("  ", indent))
		buf.WriteByte(']')
	case debugJSONObject:
		if len(v) == 0 {
			buf.WriteString("{}")
			return
		}
		buf.WriteString("{\n")
		for i, entry := range v {
			buf.WriteString(strings.Repeat("  ", indent+1))
			writeDebugJSON(buf, entry.key, indent+1)
			buf.WriteString(": ")
			writeDebugJSON(buf, entry.value, indent+1)
			if i+1 < len(v) {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(strings.Repeat("  ", indent))
		buf.WriteByte('}')
	default:
		panic("gpui: unsupported debug JSON value")
	}
}

func formatSerdeFloat(v float64, bitSize int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "null"
	}
	if v == 0 {
		if math.Signbit(v) {
			return "-0.0"
		}
		return "0.0"
	}
	exp := math.Floor(math.Log10(math.Abs(v)))
	if exp < -5 || exp >= 16 {
		s := strconv.FormatFloat(v, 'e', -1, bitSize)
		mantissa, exponent, _ := strings.Cut(s, "e")
		sign := ""
		if rest, ok := strings.CutPrefix(exponent, "-"); ok {
			sign, exponent = "-", rest
		}
		return mantissa + "e" + sign + strings.TrimLeft(strings.TrimPrefix(exponent, "+"), "0")
	}
	s := strconv.FormatFloat(v, 'f', -1, bitSize)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
