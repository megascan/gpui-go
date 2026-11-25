package taffy

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/megascan/gpui-go/gpuiutil"
)

const xmlFixturesZip = "testdata/xml.zip"

type outputNode struct {
	nodeID          NodeID
	location        Point[float32]
	size            Size[float32]
	scrollSize      Option[Size[float32]]
	resolvedRows    Option[string]
	resolvedColumns Option[string]
	children        []outputNode
}

type trackListToken struct {
	names  []string
	size   float32
	isSize bool
}

func parseTrackList(input string) []trackListToken {
	input = strings.TrimSpace(input)
	if input == "none" {
		return nil
	}
	var tokens []trackListToken
	rest := input
	for rest != "" {
		rest = strings.TrimLeft(rest, " \t\n\r\f\v")
		if rest == "" {
			break
		}
		if afterBracket, ok := strings.CutPrefix(rest, "["); ok {
			end := strings.Index(afterBracket, "]")
			if end < 0 {
				panic(fmt.Sprintf("unterminated line name group in %q", input))
			}
			tokens = append(tokens, trackListToken{names: strings.Fields(afterBracket[:end])})
			rest = afterBracket[end+1:]
		} else {
			end := strings.Index(rest, " ")
			if end < 0 {
				end = len(rest)
			}
			sizeStr, ok := strings.CutSuffix(rest[:end], "px")
			size, err := strconv.ParseFloat(sizeStr, 32)
			if !ok || err != nil {
				panic(fmt.Sprintf("invalid track size %q in %q", rest[:end], input))
			}
			tokens = append(tokens, trackListToken{size: float32(size), isSize: true})
			rest = rest[end:]
		}
	}
	return tokens
}

func trackListsMatch(expected, actual string) bool {
	e := parseTrackList(expected)
	a := parseTrackList(actual)
	if len(e) != len(a) {
		return false
	}
	for i := range e {
		if e[i].isSize && a[i].isSize {
			if !(f32Abs(e[i].size-a[i].size) < 0.1) {
				return false
			}
		} else if e[i].isSize != a[i].isSize || !slices.Equal(e[i].names, a[i].names) {
			return false
		}
	}
	return true
}

func f32Abs(v float32) float32 {
	return float32(math.Abs(float64(v)))
}

func optionalTrackListsMatch(expected, actual Option[string]) bool {
	e, eok := expected.Get()
	a, aok := actual.Get()
	if eok && aok {
		return trackListsMatch(e, a)
	}
	return true
}

func (n *outputNode) eq(other *outputNode) bool {
	scrollSizeMatches := true
	if expected, ok := n.scrollSize.Get(); ok {
		if actual, ok := other.scrollSize.Get(); ok {
			scrollSizeMatches = f32Abs(expected.Width-actual.Width) < 0.1 && f32Abs(expected.Height-actual.Height) < 0.1
		}
	}
	if !(n.nodeID == other.nodeID &&
		f32Abs(n.location.X-other.location.X) < 0.1 &&
		f32Abs(n.location.Y-other.location.Y) < 0.1 &&
		f32Abs(n.size.Width-other.size.Width) < 0.1 &&
		f32Abs(n.size.Height-other.size.Height) < 0.1 &&
		scrollSizeMatches &&
		optionalTrackListsMatch(n.resolvedRows, other.resolvedRows) &&
		optionalTrackListsMatch(n.resolvedColumns, other.resolvedColumns)) {
		return false
	}
	if len(n.children) != len(other.children) {
		return false
	}
	for i := range n.children {
		if !n.children[i].eq(&other.children[i]) {
			return false
		}
	}
	return true
}

func (n *outputNode) writeMetrics(w *strings.Builder) {
	fmt.Fprintf(w, "[x: %-4v y: %-4v w: %-4v h: %-4v]", n.location.X, n.location.Y, n.size.Width, n.size.Height)
	if scrollSize, ok := n.scrollSize.Get(); ok {
		fmt.Fprintf(w, " [scroll_w: %-4v scroll_h: %-4v]", scrollSize.Width, scrollSize.Height)
	}
	if rows, ok := n.resolvedRows.Get(); ok {
		fmt.Fprintf(w, " [rows: %s]", rows)
	}
	if columns, ok := n.resolvedColumns.Get(); ok {
		fmt.Fprintf(w, " [columns: %s]", columns)
	}
}

func (n *outputNode) writeTree(out *strings.Builder) {
	out.WriteString("TREE\n")
	var printNode func(node *outputNode, hasSibling bool, linesString string)
	printNode = func(node *outputNode, hasSibling bool, linesString string) {
		forkString := "└── "
		bar := "    "
		if hasSibling {
			forkString = "├── "
			bar = "│   "
		}
		fmt.Fprintf(out, "%s%s %v ", linesString, forkString, node.nodeID)
		node.writeMetrics(out)
		out.WriteString("\n")
		for index := range node.children {
			printNode(&node.children[index], index < len(node.children)-1, linesString+bar)
		}
	}
	printNode(n, false, "")
}

func (n *outputNode) treeString() string {
	var s strings.Builder
	n.writeTree(&s)
	return s.String()
}

type xmlNode struct {
	name     string
	attrs    map[string]string
	children []*xmlNode
	text     Option[string]
}

func (n *xmlNode) attribute(name string) Option[string] {
	if v, ok := n.attrs[name]; ok {
		return some(v)
	}
	return none[string]()
}

func (n *xmlNode) child(name string) *xmlNode {
	for _, c := range n.children {
		if c.name == name {
			return c
		}
	}
	panic("called `Option::unwrap()` on a `None` value")
}

func (n *xmlNode) firstElementChild() *xmlNode {
	if len(n.children) == 0 {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return n.children[0]
}

func parseXMLDocument(raw []byte) (*xmlNode, error) {
	dec := xml.NewDecoder(strings.NewReader(string(raw)))
	var stack []*xmlNode
	var root *xmlNode
	firstChild := map[*xmlNode]bool{}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse xml: %w", err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			node := &xmlNode{name: tok.Name.Local, attrs: map[string]string{}}
			for _, a := range tok.Attr {
				node.attrs[a.Name.Local] = a.Value
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, node)
				firstChild[parent] = true
			} else {
				root = node
			}
			stack = append(stack, node)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				if !firstChild[parent] {
					parent.text = some(string(tok))
					firstChild[parent] = true
				}
			}
		}
	}
	if root == nil {
		return nil, errors.New("parse xml: no root element")
	}
	return root, nil
}

func parseF32(input string) (float32, error) {
	v, err := strconv.ParseFloat(input, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid float literal %q: %w", input, err)
	}
	return float32(v), nil
}

func parseBool(input string) (bool, error) {
	switch input {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("provided string was not `true` or `false`: %q", input)
}

func cssParse[T any](f func(*cssParser) (T, error)) func(string) (T, error) {
	return func(input string) (T, error) {
		return parseCSSStrEntirely(input, f)
	}
}

func parseOr[T any](input Option[string], fallback T, parse func(string) (T, error)) T {
	s, ok := input.Get()
	if !ok {
		return fallback
	}
	v, err := parse(s)
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
	return v
}

func parseSelfAlignment(input Option[string]) Option[AlignSelf] {
	s, ok := input.Get()
	if !ok || strings.EqualFold(strings.TrimSpace(s), "auto") {
		return none[AlignSelf]()
	}
	return some(parseOr(input, AlignItemsNormal, cssParse(cssAlignItems)))
}

func maybeParse[T any](input Option[string], parse func(string) (T, error)) Option[T] {
	s, ok := input.Get()
	if !ok {
		return none[T]()
	}
	v, err := parse(s)
	if err != nil {
		return none[T]()
	}
	return some(v)
}

func requireF32(n *xmlNode, name string) float32 {
	return parseOr(some(n.attribute(name).Unwrap()), 0, parseF32)
}

func buildExpectations(xnode *xmlNode, nodeID NodeID) outputNode {
	scrollSize := none[Size[float32]]()
	width := maybeParse(xnode.attribute("scroll_width"), parseF32)
	height := maybeParse(xnode.attribute("scroll_height"), parseF32)
	if w, ok := width.Get(); ok {
		if h, ok := height.Get(); ok {
			scrollSize = some(Size[float32]{Width: w, Height: h})
		}
	}
	return outputNode{
		nodeID:          nodeID,
		location:        Point[float32]{X: requireF32(xnode, "x"), Y: requireF32(xnode, "y")},
		size:            Size[float32]{Width: requireF32(xnode, "width"), Height: requireF32(xnode, "height")},
		scrollSize:      scrollSize,
		resolvedRows:    xnode.attribute("resolved-rows"),
		resolvedColumns: xnode.attribute("resolved-columns"),
	}
}

func buildStyle(xnode *xmlNode) Style[defaultCheapStr] {
	attr := xnode.attribute
	gridTemplateRows := parseOr(attr("grid-template-rows"), GridTemplateTracksDefault[defaultCheapStr, GridTemplateComponent[defaultCheapStr]](), cssParse(func(p *cssParser) (GridTemplateTracks[defaultCheapStr, GridTemplateComponent[defaultCheapStr]], error) {
		return cssGridTemplateTracks(p, cssGridTemplateComponent)
	}))
	gridTemplateColumns := parseOr(attr("grid-template-columns"), GridTemplateTracksDefault[defaultCheapStr, GridTemplateComponent[defaultCheapStr]](), cssParse(func(p *cssParser) (GridTemplateTracks[defaultCheapStr, GridTemplateComponent[defaultCheapStr]], error) {
		return cssGridTemplateTracks(p, cssGridTemplateComponent)
	}))
	lpa := cssParse(cssLengthPercentageAuto)
	lp := cssParse(cssLengthPercentage)
	dim := cssParse(cssDimension)
	placement := cssParse(cssGridPlacement)
	autoPlacement := GridPlacementDefault[defaultCheapStr]()

	style := StyleDefault[defaultCheapStr]()
	style.Display = parseOr(attr("display"), DisplayDefault, cssParse(cssDisplay))
	style.Direction = parseOr(attr("direction"), DirectionLtr, cssParse(cssDirection))
	style.ItemIsTable = false
	style.ItemIsReplaced = false
	style.ItemIsCompressibleReplaced = false
	style.BoxSizing = parseOr(attr("box-sizing"), BoxSizingBorderBox, cssParse(cssBoxSizing))
	style.Overflow = Point[Overflow]{
		X: parseOr(attr("overflow-x"), OverflowVisible, cssParse(cssOverflow)),
		Y: parseOr(attr("overflow-y"), OverflowVisible, cssParse(cssOverflow)),
	}
	style.ScrollbarWidth = parseOr(attr("scrollbar-width"), 0, parseF32)
	style.Contain = parseOr(attr("contain"), ContainDefault, cssParse(cssContain))
	style.Float = parseOr(attr("float"), FloatDefault, cssParse(cssFloat))
	style.Clear = parseOr(attr("clear"), ClearDefault, cssParse(cssClear))
	style.Position = parseOr(attr("position"), PositionStatic, cssParse(cssPosition))

	style.Size = Size[Dimension]{
		Width:  parseOr(attr("width"), DimensionAuto(), dim),
		Height: parseOr(attr("height"), DimensionAuto(), dim),
	}
	style.MinSize = Size[LengthPercentageAuto]{
		Width:  parseOr(attr("min-width"), LengthPercentageAutoAuto(), lpa),
		Height: parseOr(attr("min-height"), LengthPercentageAutoAuto(), lpa),
	}
	style.MaxSize = Size[LengthPercentageAuto]{
		Width:  parseOr(attr("max-width"), LengthPercentageAutoAuto(), lpa),
		Height: parseOr(attr("max-height"), LengthPercentageAutoAuto(), lpa),
	}
	style.Inset = Rect[LengthPercentageAuto]{
		Top:    parseOr(attr("top"), LengthPercentageAutoAuto(), lpa),
		Left:   parseOr(attr("left"), LengthPercentageAutoAuto(), lpa),
		Bottom: parseOr(attr("bottom"), LengthPercentageAutoAuto(), lpa),
		Right:  parseOr(attr("right"), LengthPercentageAutoAuto(), lpa),
	}
	style.Margin = Rect[LengthPercentageAuto]{
		Top:    parseOr(attr("margin-top"), LengthPercentageAutoZero, lpa),
		Left:   parseOr(attr("margin-left"), LengthPercentageAutoZero, lpa),
		Bottom: parseOr(attr("margin-bottom"), LengthPercentageAutoZero, lpa),
		Right:  parseOr(attr("margin-right"), LengthPercentageAutoZero, lpa),
	}
	style.Padding = Rect[LengthPercentage]{
		Top:    parseOr(attr("padding-top"), LengthPercentageZero, lp),
		Left:   parseOr(attr("padding-left"), LengthPercentageZero, lp),
		Bottom: parseOr(attr("padding-bottom"), LengthPercentageZero, lp),
		Right:  parseOr(attr("padding-right"), LengthPercentageZero, lp),
	}
	style.Border = Rect[LengthPercentage]{
		Top:    parseOr(attr("border-top"), LengthPercentageZero, lp),
		Left:   parseOr(attr("border-left"), LengthPercentageZero, lp),
		Bottom: parseOr(attr("border-bottom"), LengthPercentageZero, lp),
		Right:  parseOr(attr("border-right"), LengthPercentageZero, lp),
	}
	style.Gap = Size[LengthPercentage]{
		Width:  parseOr(attr("column-gap"), LengthPercentageZero, lp),
		Height: parseOr(attr("row-gap"), LengthPercentageZero, lp),
	}

	style.AspectRatio = maybeParse(attr("aspect-ratio"), parseF32)
	style.AlignItems = parseOr(attr("align-items"), AlignItemsNormal, cssParse(cssAlignItems))
	style.AlignSelf = parseSelfAlignment(attr("align-self"))
	style.JustifyItems = parseOr(attr("justify-items"), AlignItemsNormal, cssParse(cssAlignItems))
	style.JustifySelf = parseSelfAlignment(attr("justify-self"))
	style.AlignContent = parseOr(attr("align-content"), AlignContentNormal, cssParse(cssAlignContent))
	style.JustifyContent = parseOr(attr("justify-content"), AlignContentNormal, cssParse(cssAlignContent))

	style.TextAlign = parseOr(attr("text-align"), TextAlignDefault, cssParse(cssTextAlign))
	style.FlexDirection = parseOr(attr("flex-direction"), FlexDirectionDefault, cssParse(cssFlexDirection))
	style.FlexWrap = parseOr(attr("flex-wrap"), FlexWrapDefault, cssParse(cssFlexWrap))
	style.FlexGrow = parseOr(attr("flex-grow"), 0.0, parseF32)
	style.FlexShrink = parseOr(attr("flex-shrink"), 1.0, parseF32)
	style.FlexBasis = parseOr(attr("flex-basis"), DimensionAuto(), dim)

	style.GridAutoFlow = parseOr(attr("grid-auto-flow"), GridAutoFlowRow, cssParse(cssGridAutoFlow))

	style.GridTemplateRows = gridTemplateRows.Tracks
	style.GridTemplateRowNames = gridTemplateRows.LineNames

	style.GridTemplateColumns = gridTemplateColumns.Tracks
	style.GridTemplateColumnNames = gridTemplateColumns.LineNames

	style.GridAutoRows = parseOr(attr("grid-auto-rows"), GridAutoTracks(nil), cssParse(cssGridAutoTracks))
	style.GridAutoColumns = parseOr(attr("grid-auto-columns"), GridAutoTracks(nil), cssParse(cssGridAutoTracks))
	style.GridTemplateAreas = none[GridTemplateAreas[defaultCheapStr]]()

	style.GridRow = Line[GridPlacement[defaultCheapStr]]{
		Start: parseOr(attr("grid-row-start"), autoPlacement, placement),
		End:   parseOr(attr("grid-row-end"), autoPlacement, placement),
	}
	style.GridColumn = Line[GridPlacement[defaultCheapStr]]{
		Start: parseOr(attr("grid-column-start"), autoPlacement, placement),
		End:   parseOr(attr("grid-column-end"), autoPlacement, placement),
	}
	return style
}

func mustTree[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
	}
	return v
}

func constructTree(input, expectedX *xmlNode, tree *TaffyTree[TestNodeContext], parent Option[NodeID]) outputNode {
	if len(input.children) > 0 {
		tnode := mustTree(tree.NewWithChildren(buildStyle(input), nil))
		expected := buildExpectations(expectedX, tnode)

		if p, ok := parent.Get(); ok {
			mustTree(struct{}{}, tree.AddChild(p, tnode))
		}

		for i := 0; i < len(input.children) && i < len(expectedX.children); i++ {
			expected.children = append(expected.children, constructTree(input.children[i], expectedX.children[i], tree, some(tnode)))
		}
		return expected
	}
	writingMode := parseOr(input.attribute("writing-mode"), WritingModeHorizontal, func(s string) (WritingMode, error) {
		if m, ok := WritingModeFromStr(s); ok {
			return m, nil
		}
		return 0, errors.New("()")
	})

	tnode := mustTree(tree.NewLeaf(buildStyle(input)))
	context := none[TestNodeContext]()
	if text, ok := input.text.Get(); ok {
		context = some(testNodeContextAhemText(strings.TrimSpace(text), writingMode))
	}
	mustTree(struct{}{}, tree.SetNodeContext(tnode, context))

	if p, ok := parent.Get(); ok {
		mustTree(struct{}{}, tree.AddChild(p, tnode))
	}

	return buildExpectations(expectedX, tnode)
}

func getComputedExpectations(tree *TaffyTree[TestNodeContext], nodeID NodeID) outputNode {
	layout := tree.GetFinalLayout(nodeID)
	output := outputNode{
		nodeID:          nodeID,
		location:        layout.Location,
		size:            layout.Size,
		scrollSize:      some(Size[float32]{Width: layout.ScrollWidth(), Height: layout.ScrollHeight()}),
		resolvedRows:    none[string](),
		resolvedColumns: none[string](),
	}
	for _, childID := range mustTree(tree.Children(nodeID)) {
		output.children = append(output.children, getComputedExpectations(tree, childID))
	}
	return output
}

func xmlFixtureUsesUnportedFeature(raw string) string {
	if strings.Contains(raw, "balance") {
		return "flex-wrap: balance needs the flexbox_balance feature (n/a:D-TAFFY-FEATURE)"
	}
	return ""
}

func runXMLTest(t *testing.T, rawXML []byte) {
	t.Helper()
	if reason := xmlFixtureUsesUnportedFeature(string(rawXML)); reason != "" {
		t.Skip(reason)
	}
	var failure any
	func() {
		defer func() { failure = recover() }()
		runXMLTestInner(t, rawXML)
	}()
	if msg, ok := failure.(string); ok && strings.HasPrefix(msg, "taffy: unimplemented ") {
		t.Skipf("blocked on a later stage: %s", msg)
	}
	if failure != nil {
		t.Fatalf("INPUT\n%s\npanic: %v", rawXML, failure)
	}
}

func runXMLTestInner(t *testing.T, rawXML []byte) {
	root, err := parseXMLDocument(rawXML)
	if err != nil {
		panic(err)
	}
	useRounding := parseOr(root.attribute("use-rounding"), true, parseBool)
	viewport := root.child("viewport")
	input := root.child("input")
	expectations := root.child("expectations")

	availableSpace := Size[AvailableSpace]{
		Width:  parseOr(viewport.attribute("width"), AvailableSpaceMaxContent, cssParse(cssAvailableSpace)),
		Height: parseOr(viewport.attribute("height"), AvailableSpaceMaxContent, cssParse(cssAvailableSpace)),
	}

	tree := newTestTree()
	if useRounding {
		tree.EnableRounding()
	} else {
		tree.DisableRounding()
	}
	expectedOutput := constructTree(input.firstElementChild(), expectations.firstElementChild(), tree, none[NodeID]())
	rootNodeID := expectedOutput.nodeID

	cacheModeChangeEvictions := CacheModeChangeEvictions()

	mustTree(struct{}{}, tree.ComputeLayoutWithMeasure(rootNodeID, availableSpace, testMeasureFunction))

	if gpuiutil.DebugAssertions {
		if got := CacheModeChangeEvictions(); got != cacheModeChangeEvictions {
			panic(fmt.Sprintf("assertion `left == right` failed: a node was laid out with more than one cache mode in a single layout pass\n  left: %d\n right: %d", got, cacheModeChangeEvictions))
		}
	}

	actualOutput := getComputedExpectations(tree, rootNodeID)

	if !expectedOutput.eq(&actualOutput) {
		t.Errorf("\nINPUT\n%s\n\nEXPECTED\n%s\nACTUAL\n%s", rawXML, expectedOutput.treeString(), actualOutput.treeString())
	}
}

type xmlFixture struct {
	category string
	name     string
	data     []byte
}

var loadXMLFixtures = sync.OnceValues(func() ([]xmlFixture, error) {
	r, err := zip.OpenReader(xmlFixturesZip)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", xmlFixturesZip, err)
	}
	defer r.Close()
	var fixtures []xmlFixture
	for _, f := range r.File {
		category, file := path.Split(f.Name)
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.Name, err)
		}
		fixtures = append(fixtures, xmlFixture{
			category: strings.TrimSuffix(category, "/"),
			name:     strings.TrimSuffix(file, ".xml"),
			data:     data,
		})
	}
	return fixtures, nil
})

func TestXml_GetDir(t *testing.T) {
	t.Log(xmlFixturesZip)
}

func TestXml(t *testing.T) {
	fixtures, err := loadXMLFixtures()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		t.Run(f.category+"/"+f.name, func(t *testing.T) {
			if !slices.Contains(xmlFixturesEnabled, f.category) {
				t.Skipf("category %s is not enabled in fixtures_enabled_test.go", f.category)
			}
			runXMLTest(t, f.data)
		})
	}
}
