package scan

import (
	"fmt"
	"path"
	"strings"
)

type ctxKind uint8

const (
	ctxMod ctxKind = iota
	ctxImpl
	ctxTrait
)

type scope struct {
	kind      ctxKind
	test      bool
	cfg       []string
	implType  string
	baseType  string
	implTrait bool
	counted   bool
	foreign   bool
}

type fileParser struct {
	crate    string
	file     string
	toks     []token
	out      *fileResult
	testFile bool
	macros   bool
}

type fileResult struct {
	items     []Item
	decls     map[string]bool
	pubTypes  map[string]bool
	privTys   map[string]bool
	implItems []implItem
	warnings  []Warning
}

type implItem struct {
	typ  string
	item Item
}

var testAttrs = map[string]bool{
	"test":                true,
	"gpui::test":          true,
	"crate::test":         true,
	"gpui::property_test": true,
	"perf":                true,
	"gpui::bench":         true,
}

func isTestFile(rel string) bool {
	base := path.Base(rel)
	if base == "tests.rs" || strings.HasSuffix(base, "_tests.rs") {
		return true
	}
	for _, seg := range strings.Split(path.Dir(rel), "/") {
		if seg == "tests" || seg == "benches" || seg == "examples" {
			return true
		}
	}
	return false
}

func parseFile(crate, rel, src string) *fileResult {
	p := &fileParser{
		crate:    crate,
		file:     rel,
		toks:     lex(src),
		testFile: isTestFile(rel),
		macros:   crate == "gpui_macros",
		out: &fileResult{
			decls:    map[string]bool{},
			pubTypes: map[string]bool{},
			privTys:  map[string]bool{},
		},
	}
	p.items(0, len(p.toks), &scope{kind: ctxMod, test: p.testFile, counted: true})
	return p.out
}

type attr struct {
	path string
	text string
	toks []token
}

func (p *fileParser) tok(i int) token {
	if i < len(p.toks) {
		return p.toks[i]
	}
	return token{kind: tPunct, text: ""}
}

func (p *fileParser) is(i int, kind tokKind, text string) bool {
	t := p.tok(i)
	return i < len(p.toks) && t.kind == kind && t.text == text
}

func (p *fileParser) punct(i int, text string) bool { return p.is(i, tPunct, text) }
func (p *fileParser) kw(i int, text string) bool    { return p.is(i, tIdent, text) }

func (p *fileParser) match(i int) int {
	depth := 0
	for j := i; j < len(p.toks); j++ {
		t := p.toks[j]
		if t.kind != tPunct {
			continue
		}
		switch t.text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return len(p.toks) - 1
}

func (p *fileParser) skipTo(i, end int, stops ...string) int {
	for i < end {
		t := p.toks[i]
		if t.kind == tPunct {
			for _, s := range stops {
				if t.text == s {
					return i
				}
			}
			if t.text == "(" || t.text == "[" || t.text == "{" {
				i = p.match(i) + 1
				continue
			}
		}
		i++
	}
	return end
}

func (p *fileParser) skipToBodyOrSemi(i, end int) int {
	for i < end {
		t := p.toks[i]
		if t.kind == tPunct {
			switch t.text {
			case "{", ";":
				return i
			case "(", "[":
				i = p.match(i) + 1
				continue
			}
		}
		i++
	}
	return end
}

func (p *fileParser) readAttr(i int) (attr, int) {
	j := p.match(i)
	inner := p.toks[i+1 : j]
	var pathParts []string
	k := 0
	for k < len(inner) {
		t := inner[k]
		if t.kind == tIdent {
			pathParts = append(pathParts, t.text)
			k++
			if k+1 < len(inner) && inner[k].text == ":" && inner[k+1].text == ":" {
				k += 2
				continue
			}
		}
		break
	}
	var b strings.Builder
	for idx, t := range inner {
		if idx > 0 && needsSpace(inner[idx-1], t) {
			b.WriteByte(' ')
		}
		if t.kind == tStr {
			b.WriteString("\"" + t.text + "\"")
		} else {
			b.WriteString(t.text)
		}
	}
	return attr{path: strings.Join(pathParts, "::"), text: b.String(), toks: inner}, j + 1
}

func needsSpace(a, b token) bool {
	word := func(t token) bool { return t.kind != tPunct }
	if word(a) && word(b) {
		return true
	}
	if a.kind == tPunct && (a.text == "," || a.text == "=") {
		return true
	}
	if b.kind == tPunct && b.text == "=" {
		return true
	}
	return false
}

func cfgOf(attrs []attr) []string {
	var out []string
	for _, a := range attrs {
		if a.path != "cfg" && a.path != "cfg_attr" {
			continue
		}
		toks := a.toks
		for k := 0; k+2 < len(toks); k++ {
			if toks[k].kind == tIdent && toks[k+1].text == "=" && toks[k+2].kind == tStr {
				switch toks[k].text {
				case "target_os", "target_family", "feature":
					out = mergeCfg(out, []string{toks[k+2].text})
				}
			}
		}
	}
	return out
}

func isCfgTest(a attr) bool {
	if a.path != "cfg" {
		return false
	}
	if len(a.toks) == 4 && a.toks[2].text == "test" {
		return true
	}
	if len(a.toks) > 4 && a.toks[2].text == "all" {
		depth := 0
		for _, t := range a.toks[3:] {
			switch {
			case t.text == "(":
				depth++
			case t.text == ")":
				depth--
			case depth == 1 && t.kind == tIdent && t.text == "test":
				return true
			}
		}
	}
	return false
}

func hasPath(attrs []attr, p string) bool {
	for _, a := range attrs {
		if a.path == p {
			return true
		}
	}
	return false
}

func mergeCfg(a, b []string) []string {
	out := append([]string(nil), a...)
	for _, s := range b {
		dup := false
		for _, o := range out {
			if o == s {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, s)
		}
	}
	return out
}

func (p *fileParser) record(it Item) {
	it.Crate = p.crate
	it.File = p.file
	p.out.items = append(p.out.items, it)
}

func (p *fileParser) items(i, end int, sc *scope) {
	for i < end {
		var attrs []attr
		for p.punct(i, "#") {
			if p.punct(i+1, "!") && p.punct(i+2, "[") {
				a, j := p.readAttr(i + 2)
				if isCfgTest(a) {
					sc.test = true
				}
				sc.cfg = mergeCfg(sc.cfg, cfgOf([]attr{a}))
				i = j
				continue
			}
			if p.punct(i+1, "[") {
				a, j := p.readAttr(i + 1)
				attrs = append(attrs, a)
				i = j
				continue
			}
			break
		}
		if i >= end {
			return
		}
		if p.punct(i, ";") || p.punct(i, ",") {
			i++
			continue
		}
		start := i
		vis := ""
		if p.kw(i, "pub") {
			vis = "pub"
			i++
			if p.punct(i, "(") {
				vis = "restricted"
				i = p.match(i) + 1
			}
		}
		for {
			switch {
			case p.kw(i, "default") && !p.punct(i+1, "!") && !p.punct(i+1, "(") && !p.punct(i+1, ":") && !p.punct(i+1, "="):
				i++
				continue
			case p.kw(i, "async"), p.kw(i, "unsafe"), p.kw(i, "auto"),
				p.kw(i, "safe") && (p.kw(i+1, "fn") || p.kw(i+1, "static")):
				i++
				continue
			case p.kw(i, "const") && (p.kw(i+1, "fn") || p.kw(i+1, "unsafe") || p.kw(i+1, "async") || p.kw(i+1, "extern")):
				i++
				continue
			case p.kw(i, "extern") && p.tok(i+1).kind == tStr && (p.kw(i+2, "fn") || p.kw(i+2, "unsafe")):
				i += 2
				continue
			case p.kw(i, "extern") && p.kw(i+1, "fn"):
				i++
				continue
			}
			break
		}
		testCtx := sc.test
		itemCfg := mergeCfg(sc.cfg, cfgOf(attrs))
		for _, a := range attrs {
			if isCfgTest(a) {
				testCtx = true
			}
		}
		t := p.tok(i)
		if t.kind != tIdent {
			if p.punct(i, "{") {
				i = p.match(i) + 1
				continue
			}
			i = start + 1
			if i <= start {
				i = start + 1
			}
			continue
		}
		switch t.text {
		case "fn":
			name := p.tok(i + 1).text
			line := t.line
			j := p.skipToBodyOrSemi(i+2, end)
			isMethod := p.hasSelf(i+2, j)
			if j < end && p.punct(j, "{") {
				j = p.match(j)
			}
			p.onFn(sc, attrs, vis, name, line, itemCfg, testCtx, isMethod)
			i = j + 1
		case "struct", "enum", "union":
			if t.text == "union" && p.tok(i+1).kind != tIdent {
				i = p.skipTo(i, end, ";") + 1
				continue
			}
			name := p.tok(i + 1).text
			j := p.skipToBodyOrSemi(i+2, end)
			if j < end && p.punct(j, "{") {
				j = p.match(j)
			}
			p.onType(sc, vis, t.text, name, t.line, itemCfg, testCtx)
			i = j + 1
		case "type", "const", "static":
			k := i + 1
			if p.kw(k, "mut") {
				k++
			}
			name := p.tok(k).text
			j := p.skipTo(i+1, end, ";")
			if sc.kind == ctxMod {
				if t.text == "type" {
					p.onType(sc, vis, "type", name, t.line, itemCfg, testCtx)
				} else {
					p.onValue(sc, vis, t.text, name, t.line, itemCfg, testCtx)
				}
			} else if t.text == "const" {
				p.onAssocConst(sc, vis, name, t.line, itemCfg, testCtx)
			}
			i = j + 1
		case "use":
			i = p.skipTo(i, end, ";") + 1
		case "extern":
			if p.kw(i+1, "crate") {
				i = p.skipTo(i, end, ";") + 1
				continue
			}
			j := p.skipToBodyOrSemi(i+1, end)
			if j < end && p.punct(j, "{") {
				body := p.match(j)
				if sc.kind == ctxMod {
					inner := &scope{kind: ctxMod, test: testCtx, cfg: itemCfg, counted: sc.counted, foreign: true}
					p.items(j+1, body, inner)
				}
				j = body
			}
			i = j + 1
		case "trait":
			name := p.tok(i + 1).text
			j := p.skipToBodyOrSemi(i+2, end)
			p.onType(sc, vis, "trait", name, t.line, itemCfg, testCtx)
			if j < end && p.punct(j, "{") {
				body := p.match(j)
				inner := &scope{kind: ctxTrait, test: testCtx, cfg: itemCfg, implType: name, counted: vis == "pub" && sc.counted}
				p.items(j+1, body, inner)
				j = body
			}
			i = j + 1
		case "impl":
			j := p.skipToBodyOrSemi(i+1, end)
			typ, display, isTrait := p.implHeader(i+1, j)
			if j < end && p.punct(j, "{") {
				body := p.match(j)
				inner := &scope{kind: ctxImpl, test: testCtx, cfg: itemCfg, implType: display, baseType: typ, implTrait: isTrait, counted: !isTrait}
				p.items(j+1, body, inner)
				j = body
			}
			i = j + 1
		case "mod":
			name := p.tok(i + 1).text
			j := p.skipToBodyOrSemi(i+2, end)
			if j < end && p.punct(j, "{") {
				body := p.match(j)
				inner := &scope{kind: ctxMod, test: testCtx || name == "tests", cfg: itemCfg, counted: true}
				p.items(j+1, body, inner)
				j = body
			}
			i = j + 1
		case "macro_rules":
			if !p.punct(i+1, "!") {
				i++
				continue
			}
			name := p.tok(i + 2).text
			j := i + 3
			if j < end && (p.punct(j, "{") || p.punct(j, "(") || p.punct(j, "[")) {
				p.macroBody(j+1, p.match(j), "macro_rules! "+name, testCtx, itemCfg, sc)
				j = p.match(j)
			}
			if hasPath(attrs, "macro_export") && !testCtx && sc.kind == ctxMod {
				if !p.macros {
					p.record(Item{Name: name, Kind: "macro", Line: t.line, Cfg: itemCfg})
				}
			}
			p.out.decls[name] = true
			i = j + 1
		default:
			k := i
			for p.tok(k).kind == tIdent && p.punct(k+1, ":") && p.punct(k+2, ":") {
				k += 3
			}
			if p.tok(k).kind == tIdent && p.punct(k+1, "!") && (p.punct(k+2, "{") || p.punct(k+2, "(") || p.punct(k+2, "[")) {
				body := p.match(k + 2)
				p.onMacroCall(sc, p.tok(k).text, k+3, body, testCtx, itemCfg)
				i = body + 1
				continue
			}
			i = p.skipTo(i, end, ";") + 1
		}
	}
}

func (p *fileParser) hasSelf(i, j int) bool {
	if !p.punct(i, "<") && !p.punct(i, "(") {
		return false
	}
	k := i
	if p.punct(k, "<") {
		depth := 0
		for ; k < j; k++ {
			if p.punct(k, "<") {
				depth++
			} else if p.punct(k, ">") && !p.punct(k-1, "-") {
				depth--
				if depth == 0 {
					k++
					break
				}
			}
		}
	}
	if !p.punct(k, "(") {
		return false
	}
	k++
	for p.punct(k, "&") || p.tok(k).kind == tLifetime || p.kw(k, "mut") {
		k++
	}
	return p.kw(k, "self")
}

func (p *fileParser) implHeader(i, j int) (string, string, bool) {
	k := i
	generic := false
	if p.punct(k, "<") {
		k = p.skipAngle(k, j)
		generic = true
	}
	if p.punct(k, "!") {
		k++
	}
	isTrait := false
	for m := k; m < j; m++ {
		if p.punct(m, "<") {
			m = p.skipAngle(m, j) - 1
			continue
		}
		if p.kw(m, "where") {
			break
		}
		if p.kw(m, "for") && !p.punct(m+1, "<") {
			isTrait = true
			k = m + 1
			break
		}
	}
	for p.punct(k, "&") || p.kw(k, "dyn") || p.kw(k, "mut") || p.tok(k).kind == tLifetime {
		k++
	}
	name := ""
	for k < j && p.tok(k).kind == tIdent {
		name = p.tok(k).text
		if p.punct(k+1, ":") && p.punct(k+2, ":") {
			k += 3
			continue
		}
		break
	}
	display := name
	if !generic && p.punct(k+1, "<") && p.plainArgs(k+2, p.skipAngle(k+1, j)-1) {
		e := p.skipAngle(k+1, j)
		var b strings.Builder
		b.WriteString(name)
		for m := k + 1; m < e; m++ {
			t := p.tok(m)
			if t.kind != tPunct && m > k+1 && p.tok(m-1).kind != tPunct {
				b.WriteByte(' ')
			}
			b.WriteString(t.text)
			if t.text == "," {
				b.WriteByte(' ')
			}
		}
		display = b.String()
	}
	return name, display, isTrait
}

func (p *fileParser) plainArgs(i, e int) bool {
	if i >= e {
		return false
	}
	for m := i; m < e; m++ {
		if p.tok(m).kind != tIdent && !p.punct(m, ",") {
			return false
		}
	}
	return true
}

func (p *fileParser) skipAngle(k, j int) int {
	depth := 0
	for ; k < j; k++ {
		if p.punct(k, "<") {
			depth++
		} else if p.punct(k, ">") && !p.punct(k-1, "-") && !p.punct(k-1, "=") {
			depth--
			if depth == 0 {
				return k + 1
			}
		}
	}
	return j
}

func (p *fileParser) testAttr(attrs []attr) (attr, bool) {
	for _, a := range attrs {
		if testAttrs[a.path] {
			return a, true
		}
	}
	return attr{}, false
}

func (p *fileParser) onFn(sc *scope, attrs []attr, vis, name string, line int, cfg []string, testCtx, isMethod bool) {
	if a, ok := p.testAttr(attrs); ok {
		attrText := a.text
		ignored := hasPath(attrs, "ignore")
		p.record(Item{Name: name, Kind: attrText, Line: line, Cfg: cfg, Test: true, Ignored: ignored})
		p.out.decls[name] = true
		return
	}
	switch sc.kind {
	case ctxMod:
		p.out.decls[name] = true
		if testCtx || sc.foreign {
			return
		}
		for _, a := range attrs {
			kind := ""
			switch a.path {
			case "proc_macro":
				kind = "proc-macro"
			case "proc_macro_derive":
				kind = "proc-macro-derive"
			case "proc_macro_attribute":
				kind = "proc-macro-attribute"
			}
			if kind == "" {
				continue
			}
			n := name
			if kind == "proc-macro-derive" && len(a.toks) > 2 && a.toks[2].kind == tIdent {
				n = a.toks[2].text
				p.out.decls[n] = true
			}
			p.record(Item{Name: n, Kind: kind, Line: line, Cfg: cfg})
			return
		}
		if p.macros {
			return
		}
		if vis == "pub" {
			p.record(Item{Name: name, Kind: "fn", Line: line, Cfg: cfg})
		}
	case ctxImpl:
		full := sc.implType + "::" + name
		p.out.decls[full] = true
		p.out.decls[sc.baseType+"::"+name] = true
		p.out.decls[name] = true
		if testCtx || p.macros || sc.implTrait || vis != "pub" {
			return
		}
		kind := "assoc-fn"
		if isMethod {
			kind = "method"
		}
		p.pendImpl(sc.baseType, Item{Name: full, Kind: kind, Line: line, Cfg: cfg})
	case ctxTrait:
		full := sc.implType + "::" + name
		p.out.decls[full] = true
		p.out.decls[name] = true
		if testCtx || p.macros || !sc.counted {
			return
		}
		p.record(Item{Name: full, Kind: "trait-method", Line: line, Cfg: cfg})
	}
}

func (p *fileParser) pendImpl(typ string, it Item) {
	it.Crate = p.crate
	it.File = p.file
	p.out.implItems = append(p.out.implItems, implItem{typ: typ, item: it})
}

func (p *fileParser) warn(line int, format string, args ...any) {
	p.out.warnings = append(p.out.warnings, Warning{File: p.file, Line: line, Msg: fmt.Sprintf(format, args...)})
}

func (p *fileParser) onType(sc *scope, vis, kind, name string, line int, cfg []string, testCtx bool) {
	p.out.decls[name] = true
	if sc.kind != ctxMod || testCtx || sc.foreign {
		return
	}
	if vis == "pub" {
		p.out.pubTypes[name] = true
	} else {
		p.out.privTys[name] = true
	}
	if p.macros || vis != "pub" {
		return
	}
	p.record(Item{Name: name, Kind: kind, Line: line, Cfg: cfg})
}

func (p *fileParser) onValue(sc *scope, vis, kind, name string, line int, cfg []string, testCtx bool) {
	p.out.decls[name] = true
	if testCtx || sc.foreign || p.macros || vis != "pub" || name == "_" {
		return
	}
	p.record(Item{Name: name, Kind: kind, Line: line, Cfg: cfg})
}

func (p *fileParser) onAssocConst(sc *scope, vis, name string, line int, cfg []string, testCtx bool) {
	full := sc.implType + "::" + name
	p.out.decls[full] = true
	p.out.decls[name] = true
	if sc.kind != ctxImpl || sc.implTrait || testCtx || p.macros || vis != "pub" {
		return
	}
	p.pendImpl(sc.baseType, Item{Name: full, Kind: "assoc-const", Line: line, Cfg: cfg})
}

func (p *fileParser) onMacroCall(sc *scope, name string, i, end int, testCtx bool, cfg []string) {
	switch name {
	case "new_key_type":
		for k := i; k < end; k++ {
			if p.kw(k, "struct") && p.tok(k+1).kind == tIdent {
				pub := p.kw(k-1, "pub")
				n := p.tok(k + 1).text
				p.out.decls[n] = true
				if pub && !testCtx && !p.macros && sc.kind == ctxMod {
					p.out.pubTypes[n] = true
					p.record(Item{Name: n, Kind: "struct", Line: p.tok(k).line, Cfg: cfg})
				}
			}
		}
		return
	case "bitflags":
		for k := i; k < end; k++ {
			if p.kw(k, "struct") && p.tok(k+1).kind == tIdent {
				n := p.tok(k + 1).text
				p.out.decls[n] = true
				p.warn(p.tok(k).line, "bitflags! declares %s (pub=%t), not counted by the section 9 scan", n, p.kw(k-1, "pub"))
			}
			if p.kw(k, "const") && p.tok(k+1).kind == tIdent {
				p.out.decls[p.tok(k+1).text] = true
			}
		}
	case "actions":
		var names []string
		for k := i; k < end; k++ {
			if p.punct(k, "[") {
				e := p.match(k)
				for m := k + 1; m < e; m++ {
					if p.tok(m).kind == tIdent {
						names = append(names, p.tok(m).text)
						p.out.decls[p.tok(m).text] = true
					}
				}
				break
			}
		}
		p.warn(p.tok(i).line, "actions! declares %s, not counted by the section 9 scan", strings.Join(names, ", "))
	}
	p.macroBody(i, end, name+"!", testCtx, cfg, sc)
}

func (p *fileParser) macroBody(i, end int, where string, testCtx bool, cfg []string, sc *scope) {
	for k := i; k+2 < end; k++ {
		if p.punct(k, "#") && p.punct(k+1, "[") {
			a, j := p.readAttr(k + 1)
			if testAttrs[a.path] {
				m := j
				for m < end && !p.kw(m, "fn") && !p.punct(m, "{") {
					m++
				}
				if p.kw(m, "fn") {
					p.warn(p.tok(k).line, "test %s inside %s, not counted by the section 9 scan", p.tok(m+1).text, where)
				}
			}
		}
	}
}
