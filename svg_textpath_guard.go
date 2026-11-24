package gpui

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	okagepath "github.com/xo/okage/path"
	"github.com/xo/okage/roxmltree"
	"github.com/xo/okage/svg"
)

const (
	svgTextPathMaxArclenRatio float64 = 1 << 24
	svgXMLNamespace                   = "http://www.w3.org/2000/svg"
	svgXlinkNamespace                 = "http://www.w3.org/1999/xlink"
)

var (
	errSvgTextPathTooLong    = errors.New("SVG text path exceeds the arc length precision limit")
	errSvgTextPathUnmeasured = errors.New("SVG text path cannot be measured safely")
)

var svgTextPathShapes = []string{"path", "rect", "circle", "ellipse", "line", "polyline", "polygon"}

type svgTextPathEdit struct {
	pos    int
	remove int
	insert string
}

type svgTextPathProbe struct {
	id       string
	textID   string
	inline   bool
	hasChars bool
}

type svgTextPathPlan struct {
	edits  []svgTextPathEdit
	probes []svgTextPathProbe
}

type svgTextPathMeasure struct {
	probes     map[string]bool
	inline     map[string]bool
	texts      map[string]bool
	textScale  map[string]float64
	probeFound map[string]bool
	probeLen   map[string]float64
}

func (r SvgRenderer) checkTextPaths(data []byte) error {
	if bytes.HasPrefix(data, []byte{0x1f, 0x8b}) {
		decoded, err := svg.DecompressSVGZ(data)
		if err != nil {
			return nil
		}
		data = decoded
	}
	if !bytes.Contains(data, []byte("textPath")) && !bytes.Contains(data, []byte("<!ENTITY")) {
		return nil
	}
	if !utf8.Valid(data) {
		return nil
	}
	source := string(data)
	opt := roxmltree.DefaultParsingOptions()
	opt.AllowDTD = true
	doc, err := roxmltree.ParseWithOptions(source, opt)
	if err != nil {
		return nil
	}
	plan, err := planSvgTextPathProbe(source, doc)
	if err != nil || len(plan.probes) == 0 {
		return err
	}
	tree, err := svg.FromStr(applySvgTextPathEdits(source, plan.edits), r.textPathProbeOptions()...)
	if err != nil {
		return errSvgTextPathUnmeasured
	}
	return plan.check(tree)
}

func (r SvgRenderer) textPathProbeOptions() []svg.Option {
	return append(slices.Clip(r.usvgOptions), svg.WithImageHrefResolver(svg.ImageHrefResolver{
		ResolveData: func(string, []byte) (svg.ImageKind, bool) {
			return nil, false
		},
		ResolveString: func(string) (svg.ImageKind, bool) {
			return nil, false
		},
	}))
}

func isSvgXMLElement(n roxmltree.Node, local string) bool {
	if !n.IsElement() || n.TagName().Name() != local {
		return false
	}
	ns, ok := n.TagName().Namespace()
	return !ok || ns == svgXMLNamespace
}

func svgXMLAttribute(n roxmltree.Node, local string) (string, bool) {
	for a := range n.Attributes() {
		if _, hasNS := a.Namespace(); a.Name() == local && !hasNS {
			return a.Value(), true
		}
	}
	return "", false
}

func svgXMLHrefs(n roxmltree.Node) []string {
	var hrefs []string
	for a := range n.Attributes() {
		ns, hasNS := a.Namespace()
		if a.Name() == "href" && (!hasNS || ns == svgXlinkNamespace) {
			hrefs = append(hrefs, a.Value())
		}
	}
	return hrefs
}

func svgHrefID(href string) (string, bool) {
	i := strings.LastIndexByte(href, '#')
	if i < 0 {
		return "", false
	}
	id := strings.TrimSpace(href[i+1:])
	return id, id != ""
}

func svgXMLStartTagName(source string, n roxmltree.Node, local string) (int, string, bool) {
	start := n.Range().Start
	if start < 0 || start >= len(source) || source[start] != '<' {
		return 0, "", false
	}
	end := start + 1
	for end < len(source) && !strings.ContainsRune(" \t\r\n/>", rune(source[end])) {
		end++
	}
	qname := source[start+1 : end]
	if qname != local && !strings.HasSuffix(qname, ":"+local) {
		return 0, "", false
	}
	return end, qname, true
}

func svgStyleSelectorsUnsafe(css string) bool {
	for {
		i := strings.Index(css, "/*")
		if i < 0 {
			break
		}
		j := strings.Index(css[i+2:], "*/")
		if j < 0 {
			css = css[:i]
			break
		}
		css = css[:i] + " " + css[i+2+j+2:]
	}
	segment := 0
	for i := 0; i < len(css); i++ {
		switch css[i] {
		case '{':
			if strings.ContainsAny(css[segment:i], ":+~[") {
				return true
			}
			segment = i + 1
		case '}', ';':
			segment = i + 1
		}
	}
	return false
}

func svgXMLHasChars(n roxmltree.Node) bool {
	for d := range n.Descendants() {
		if !d.IsText() {
			continue
		}
		if text, ok := d.Text(); ok && text != "" {
			return true
		}
	}
	return false
}

func newSvgTextPathID() string {
	return "gpui-tp-" + rand.Text()
}

func planSvgTextPathProbe(source string, doc *roxmltree.Document) (svgTextPathPlan, error) {
	var plan svgTextPathPlan
	root := doc.RootElement()
	rootRange := root.Range()
	inRoot := func(n roxmltree.Node) bool {
		rng := n.Range()
		return rng.Start >= rootRange.Start && rng.End <= rootRange.End && rng.Start < rng.End
	}

	ids := map[string][]roxmltree.Node{}
	var texts []roxmltree.Node
	textPaths := map[int][]roxmltree.Node{}
	var styles []string
	var uses []roxmltree.Node
	for n := range doc.Descendants() {
		if !n.IsElement() {
			continue
		}
		if id, ok := svgXMLAttribute(n, "id"); ok {
			ids[id] = append(ids[id], n)
		}
		switch {
		case n.TagName().Name() == "style":
			var b strings.Builder
			for c := range n.Children() {
				if !c.IsText() {
					continue
				}
				if text, ok := c.Text(); ok {
					b.WriteString(text)
				}
			}
			styles = append(styles, b.String())
		case isSvgXMLElement(n, "use"):
			uses = append(uses, n)
		case isSvgXMLElement(n, "textPath"):
			parent, ok := n.ParentElement()
			if !ok || !isSvgXMLElement(parent, "text") {
				continue
			}
			if !inRoot(n) || !inRoot(parent) {
				return plan, errSvgTextPathUnmeasured
			}
			key := parent.Range().Start
			if _, seen := textPaths[key]; !seen {
				texts = append(texts, parent)
			}
			textPaths[key] = append(textPaths[key], n)
		}
	}
	if len(texts) == 0 {
		return plan, nil
	}
	for _, css := range styles {
		if svgStyleSelectorsUnsafe(css) {
			return plan, errSvgTextPathUnmeasured
		}
	}

	containers := map[int]bool{}
	for _, text := range texts {
		containers[text.Range().Start] = true
		for a := range text.Ancestors() {
			if a.IsElement() {
				containers[a.Range().Start] = true
			}
		}
	}
	for _, use := range uses {
		for _, href := range svgXMLHrefs(use) {
			id, ok := svgHrefID(href)
			if !ok {
				continue
			}
			for _, target := range ids[id] {
				if containers[target.Range().Start] {
					return plan, errSvgTextPathUnmeasured
				}
			}
		}
	}

	seen := map[int]bool{}
	for _, text := range texts {
		nameEnd, qname, ok := svgXMLStartTagName(source, text, "text")
		textRange := text.Range()
		if !ok || source[textRange.End-1] != '>' {
			return plan, errSvgTextPathUnmeasured
		}
		prefix := qname[:len(qname)-len("text")]
		textID, hasID := svgXMLAttribute(text, "id")
		switch {
		case !hasID:
			textID = newSvgTextPathID()
			plan.edits = append(plan.edits, svgTextPathEdit{pos: nameEnd, insert: ` id="` + textID + `"`})
		case len(ids[textID]) != 1:
			return plan, errSvgTextPathUnmeasured
		}
		var probes strings.Builder
		for _, tp := range textPaths[textRange.Start] {
			rng := tp.Range()
			if seen[rng.Start] || rng.Start < textRange.Start || rng.End > textRange.End {
				return plan, errSvgTextPathUnmeasured
			}
			seen[rng.Start] = true
			tagEnd, tpName, ok := svgXMLStartTagName(source, tp, "textPath")
			if !ok {
				return plan, errSvgTextPathUnmeasured
			}
			plan.edits = append(plan.edits, svgTextPathEdit{pos: tagEnd - len("textPath"), remove: len("textPath"), insert: "tspan"})
			if !strings.HasSuffix(source[:rng.End], "/>") {
				closeAt := strings.LastIndex(source[rng.Start:rng.End], "</")
				if closeAt < 0 || !strings.HasPrefix(source[rng.Start+closeAt+2:], tpName) {
					return plan, errSvgTextPathUnmeasured
				}
				nameStart := rng.Start + closeAt + 2
				plan.edits = append(plan.edits, svgTextPathEdit{pos: nameStart + len(tpName) - len("textPath"), remove: len("textPath"), insert: "tspan"})
			}
			hasChars := svgXMLHasChars(tp)
			if d, ok := svgXMLAttribute(tp, "path"); ok && strings.TrimSpace(d) != "" {
				probe := svgTextPathProbe{id: newSvgTextPathID(), textID: textID, inline: true, hasChars: hasChars}
				plan.probes = append(plan.probes, probe)
				probes.WriteString("<" + prefix + `path id="` + probe.id + `" d="`)
				pushXMLEscaped(&probes, d)
				probes.WriteString(`"/>`)
			}
			for _, href := range svgXMLHrefs(tp) {
				id, ok := svgHrefID(href)
				if !ok || !slices.ContainsFunc(ids[id], func(n roxmltree.Node) bool {
					return slices.ContainsFunc(svgTextPathShapes, func(shape string) bool { return isSvgXMLElement(n, shape) })
				}) {
					continue
				}
				probe := svgTextPathProbe{id: newSvgTextPathID(), textID: textID, hasChars: hasChars}
				plan.probes = append(plan.probes, probe)
				probes.WriteString("<" + prefix + `use id="` + probe.id + `" href="#`)
				pushXMLEscaped(&probes, id)
				probes.WriteString(`"/>`)
			}
		}
		plan.edits = append(plan.edits, svgTextPathEdit{pos: textRange.End, insert: probes.String()})
	}
	return plan, nil
}

func applySvgTextPathEdits(source string, edits []svgTextPathEdit) string {
	slices.SortStableFunc(edits, func(a, b svgTextPathEdit) int { return a.pos - b.pos })
	var b strings.Builder
	last := 0
	for _, e := range edits {
		b.WriteString(source[last:e.pos])
		b.WriteString(e.insert)
		last = e.pos + e.remove
	}
	b.WriteString(source[last:])
	return b.String()
}

func svgPolylineLength(p *okagepath.Path, ts svg.Transform) float64 {
	points := slices.Clone(p.Points())
	ts.MapPoints(points)
	var length float64
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X) - float64(points[i-1].X)
		dy := float64(points[i].Y) - float64(points[i-1].Y)
		length += math.Hypot(dx, dy)
	}
	return length
}

func svgGroupPolylineLength(g *svg.Group, ts svg.Transform) float64 {
	var length float64
	for _, child := range g.Children() {
		switch c := child.(type) {
		case *svg.Group:
			length += svgGroupPolylineLength(c, ts.PreConcat(c.Transform()))
		case *svg.Path:
			length += svgPolylineLength(c.Data(), ts)
		}
	}
	return length
}

func (m *svgTextPathMeasure) visit(g *svg.Group) {
	if id := g.ID(); m.probes[id] && !m.inline[id] {
		m.probeFound[id] = true
		m.probeLen[id] = max(m.probeLen[id], svgGroupPolylineLength(g, okagepath.DefaultTransform()))
	}
	g.Subroots(m.visit)
	for _, child := range g.Children() {
		switch c := child.(type) {
		case *svg.Group:
			m.visit(c)
		case *svg.Path:
			if id := c.ID(); m.inline[id] {
				m.probeFound[id] = true
				m.probeLen[id] = max(m.probeLen[id], svgPolylineLength(c.Data(), okagepath.DefaultTransform()))
			}
			c.Subroots(m.visit)
		case *svg.Text:
			if id := c.ID(); m.texts[id] {
				sx, sy := c.AbsTransform().GetScale()
				scale := math.Sqrt(float64(sx) * float64(sy))
				if math.IsNaN(scale) {
					scale = math.Inf(1)
				}
				m.textScale[id] = max(m.textScale[id], scale)
			}
		default:
			c.Subroots(m.visit)
		}
	}
}

func (p svgTextPathPlan) check(tree *svg.Tree) error {
	m := &svgTextPathMeasure{
		probes:     map[string]bool{},
		inline:     map[string]bool{},
		texts:      map[string]bool{},
		textScale:  map[string]float64{},
		probeFound: map[string]bool{},
		probeLen:   map[string]float64{},
	}
	for _, probe := range p.probes {
		m.probes[probe.id] = true
		m.inline[probe.id] = probe.inline
		m.texts[probe.textID] = true
	}
	m.visit(tree.Root())
	for _, probe := range p.probes {
		scale, textFound := m.textScale[probe.textID]
		if !textFound {
			if probe.hasChars {
				return errSvgTextPathUnmeasured
			}
			continue
		}
		if !m.probeFound[probe.id] {
			return errSvgTextPathUnmeasured
		}
		if ratio := m.probeLen[probe.id] * max(scale, 1) * 2; !(ratio <= svgTextPathMaxArclenRatio) {
			return errSvgTextPathTooLong
		}
	}
	return nil
}
