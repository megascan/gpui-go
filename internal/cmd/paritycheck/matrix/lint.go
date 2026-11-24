package matrix

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var knownGOOS = []string{"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos"}

var mixinTypes = map[string]bool{"Styled": true, "Interactive": true, "StatefulInteractive": true, "Parent": true, "Fluent": true}

type osRule struct {
	prefix  string
	allowed []string
}

var constrainedDirs = []osRule{
	{"internal/linux", []string{"linux"}},
	{"platform/linux", []string{"linux"}},
	{"internal/apple", []string{"darwin", "ios"}},
	{"platform/apple", []string{"darwin", "ios"}},
	{"platform/macos", []string{"darwin"}},
	{"platform/ios", []string{"ios", "darwin"}},
	{"internal/win", []string{"windows"}},
	{"platform/windows", []string{"windows"}},
	{"platform/web", []string{"js"}},
}

var importRules = []osRule{
	{"internal/linux", []string{"linux"}},
	{"internal/apple", []string{"darwin", "ios"}},
	{"internal/win", []string{"windows"}},
}

var rootAllowedInternal = []string{"a11y", "scheduler", "sumtree", "collections", "refineable", "gpuiutil", "internal/taffy", "internal/etagere", "internal/lyon", "internal/jsonschema"}

var rootForbiddenThirdParty = []string{"purego", "webgpu", "godbus", "gen2brain"}

type goFile struct {
	rel  string
	abs  string
	name string
	ast  *ast.File
}

func Lint(root string) ([]Problem, error) {
	var probs []Problem
	module, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	dirs, dirProblems, err := walkDirs(root)
	if err != nil {
		return nil, err
	}
	probs = append(probs, dirProblems...)
	fset := token.NewFileSet()
	byDir := map[string][]*goFile{}
	for _, d := range dirs {
		files, fileProbs := parseDir(fset, root, d)
		probs = append(probs, fileProbs...)
		byDir[d] = files
	}
	mixinMethods, err := mixinMethodSet(root, byDir)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		for _, f := range byDir[d] {
			probs = append(probs, lintFile(root, module, d, f, fset)...)
		}
		probs = append(probs, lintDirConstraint(root, d, byDir[d])...)
		probs = append(probs, lintMixins(d, byDir[d], mixinMethods)...)
	}
	sort.SliceStable(probs, func(i, j int) bool {
		if probs[i].File != probs[j].File {
			return probs[i].File < probs[j].File
		}
		return probs[i].Line < probs[j].Line
	})
	return probs, nil
}

func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("%s/go.mod has no module line", root)
}

func walkDirs(root string) ([]string, []Problem, error) {
	var dirs []string
	var probs []Problem
	err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			return nil
		}
		name := e.Name()
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if p != root && (strings.HasPrefix(name, ".") || name == "_ref" || name == "_ext" || name == "node_modules") {
			return filepath.SkipDir
		}
		if name != "docs" && strings.EqualFold(name, "docs") {
			probs = append(probs, Problem{File: rel, Msg: "directory name differs only by case from docs/; use lowercase docs/"})
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return err
		}
		for _, en := range entries {
			if !en.IsDir() && strings.HasSuffix(en.Name(), ".go") {
				dirs = append(dirs, rel)
				break
			}
		}
		return nil
	})
	return dirs, probs, err
}

func parseDir(fset *token.FileSet, root, rel string) ([]*goFile, []Problem) {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, []Problem{{File: rel, Msg: err.Error()}}
	}
	var out []*goFile
	var probs []Problem
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		fp := filepath.Join(abs, e.Name())
		f, err := parser.ParseFile(fset, fp, nil, parser.ParseComments)
		relFile := filepath.ToSlash(filepath.Join(rel, e.Name()))
		if err != nil {
			probs = append(probs, Problem{File: relFile, Msg: "parse error: " + err.Error()})
			continue
		}
		out = append(out, &goFile{rel: rel, abs: fp, name: e.Name(), ast: f})
	}
	return out, probs
}

func archFor(goos string) string {
	switch goos {
	case "js", "wasip1":
		return "wasm"
	case "ios":
		return "arm64"
	}
	return "amd64"
}

func matchedGOOS(f *goFile) []string {
	var out []string
	for _, g := range knownGOOS {
		ctx := build.Default
		ctx.GOOS = g
		ctx.GOARCH = archFor(g)
		ctx.CgoEnabled = true
		ok, err := ctx.MatchFile(filepath.Dir(f.abs), f.name)
		if err == nil && ok {
			out = append(out, g)
		}
	}
	return out
}

func allowedWithImplied(allowed []string) map[string]bool {
	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
		if a == "linux" {
			ok["android"] = true
		}
	}
	return ok
}

func disallowed(f *goFile, allowed []string) []string {
	ok := allowedWithImplied(allowed)
	var bad []string
	for _, g := range matchedGOOS(f) {
		if !ok[g] {
			bad = append(bad, g)
		}
	}
	return bad
}

func lintFile(root, module, dir string, f *goFile, fset *token.FileSet) []Problem {
	var probs []Problem
	relFile := filepath.ToSlash(filepath.Join(dir, f.name))
	add := func(pos token.Pos, msg string) {
		p := fset.Position(pos)
		probs = append(probs, Problem{File: relFile, Line: p.Line, Msg: msg})
	}
	for _, imp := range f.ast.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		switch {
		case p == "C":
			if !strings.HasPrefix(dir, "render/wgpu") && !strings.HasPrefix(dir, "internal/cmd/wlspike") && !strings.HasPrefix(dir, "examples/ios_ios") {
				add(imp.Pos(), `cgo ("C") is only allowed under render/wgpu, internal/cmd/wlspike and examples/ios_ios`)
			}
		case strings.HasPrefix(p, module+"/"):
			target := strings.TrimPrefix(p, module+"/")
			if dir == "." && !rootMayImport(target) {
				add(imp.Pos(), fmt.Sprintf("root package imports %s; only internal/{taffy,etagere,lyon,jsonschema} and the core packages are allowed", target))
			}
			for _, r := range importRules {
				if under(target, r.prefix) {
					if bad := disallowed(f, r.allowed); len(bad) > 0 {
						add(imp.Pos(), fmt.Sprintf("%s is imported only from packages built for %v; this file builds for %v", target, r.allowed, bad))
					}
				}
			}
		default:
			if dir == "." {
				for _, bad := range rootForbiddenThirdParty {
					if strings.Contains(p, bad) {
						add(imp.Pos(), fmt.Sprintf("root package imports %s (platform, renderer, cgo or purego code)", p))
					}
				}
			}
		}
	}
	probs = append(probs, lintLockThread(dir, f, fset, relFile)...)
	probs = append(probs, lintComments(f.ast, fset, relFile)...)
	return probs
}

func rootMayImport(target string) bool {
	for _, a := range rootAllowedInternal {
		if target == a || strings.HasPrefix(target, a+"/") {
			return true
		}
	}
	return false
}

func under(rel, prefix string) bool {
	return rel == prefix || strings.HasPrefix(rel, prefix+"/")
}

func lintDirConstraint(root, dir string, files []*goFile) []Problem {
	var rule *osRule
	for i := range constrainedDirs {
		if under(dir, constrainedDirs[i].prefix) {
			rule = &constrainedDirs[i]
			break
		}
	}
	if rule == nil {
		return nil
	}
	var probs []Problem
	for _, f := range files {
		if bad := disallowed(f, rule.allowed); len(bad) > 0 {
			probs = append(probs, Problem{File: filepath.ToSlash(filepath.Join(dir, f.name)), Msg: fmt.Sprintf("%s/** must build only for %v; this file also builds for %v", rule.prefix, rule.allowed, bad)})
		}
	}
	return probs
}

var lockThreadAllowed = map[string]bool{
	"internal/linux/wl/display.go":           true,
	"platform/macos/system_notifications.go": true,
}

func lintLockThread(dir string, f *goFile, fset *token.FileSet, relFile string) []Problem {
	if dir == "platform" || strings.HasSuffix(relFile, "_test.go") || lockThreadAllowed[relFile] {
		return nil
	}
	var probs []Problem
	ast.Inspect(f.ast, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "LockOSThread" {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "runtime" {
			probs = append(probs, Problem{File: relFile, Line: fset.Position(sel.Pos()).Line, Msg: "runtime.LockOSThread is called only from package platform"})
		}
		return true
	})
	return probs
}

var directivePrefixes = []string{"//go:", "//gpui:", "//export "}

func lintComments(f *ast.File, fset *token.FileSet, relFile string) []Problem {
	allowed := map[*ast.CommentGroup]bool{}
	if f.Doc != nil {
		allowed[f.Doc] = true
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.FuncDecl:
			if d.Doc != nil && d.Name.IsExported() {
				allowed[d.Doc] = true
			}
		case *ast.GenDecl:
			if d.Doc != nil && genDeclExported(d) {
				allowed[d.Doc] = true
			}
			for _, spec := range d.Specs {
				switch sp := spec.(type) {
				case *ast.TypeSpec:
					if sp.Doc != nil && sp.Name.IsExported() {
						allowed[sp.Doc] = true
					}
				case *ast.ValueSpec:
					if sp.Doc != nil && anyExported(sp.Names) {
						allowed[sp.Doc] = true
					}
				case *ast.ImportSpec:
					if sp.Path.Value == `"C"` && d.Doc != nil {
						allowed[d.Doc] = true
					}
				}
			}
		case *ast.Field:
			if d.Doc != nil && anyExported(d.Names) {
				allowed[d.Doc] = true
			}
		}
		return true
	})
	var probs []Problem
	for _, cg := range f.Comments {
		if allowed[cg] || allDirectives(cg) || isGeneratedHeader(cg) || isCgoPreamble(f, cg, fset) {
			continue
		}
		probs = append(probs, Problem{File: relFile, Line: fset.Position(cg.Pos()).Line, Msg: "comment not allowed: only godoc on exported API, //go: and //gpui: directives, and generated headers"})
	}
	return probs
}

func genDeclExported(d *ast.GenDecl) bool {
	for _, spec := range d.Specs {
		switch sp := spec.(type) {
		case *ast.TypeSpec:
			if sp.Name.IsExported() {
				return true
			}
		case *ast.ValueSpec:
			if anyExported(sp.Names) {
				return true
			}
		}
	}
	return false
}

func anyExported(names []*ast.Ident) bool {
	for _, n := range names {
		if n.IsExported() {
			return true
		}
	}
	return false
}

func allDirectives(cg *ast.CommentGroup) bool {
	for _, c := range cg.List {
		ok := false
		for _, p := range directivePrefixes {
			if strings.HasPrefix(c.Text, p) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func isGeneratedHeader(cg *ast.CommentGroup) bool {
	text := cg.Text()
	return strings.HasPrefix(text, "Code generated ") && strings.Contains(text, "DO NOT EDIT")
}

func isCgoPreamble(f *ast.File, cg *ast.CommentGroup, fset *token.FileSet) bool {
	for _, imp := range f.Imports {
		if imp.Path.Value != `"C"` {
			continue
		}
		if fset.Position(cg.End()).Line == fset.Position(imp.Pos()).Line-1 {
			return true
		}
	}
	return false
}

func mixinMethodSet(root string, byDir map[string][]*goFile) (map[string]bool, error) {
	set := map[string]bool{}
	tsv, err := readLines(filepath.Join(root, StyledTSV))
	if err != nil {
		return nil, err
	}
	for _, line := range tsv {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if cols := strings.Split(line, "\t"); len(cols) >= 2 {
			set[strings.TrimSpace(cols[1])] = true
		}
	}
	for _, files := range byDir {
		for _, f := range files {
			for _, decl := range f.ast.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 {
					continue
				}
				if mixinTypes[baseName(fd.Recv.List[0].Type)] {
					set[fd.Name.Name] = true
				}
			}
		}
	}
	return set, nil
}

func lintMixins(dir string, files []*goFile, methods map[string]bool) []Problem {
	embedders := map[string]bool{}
	for _, f := range files {
		for _, decl := range f.ast.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || mixinTypes[ts.Name.Name] {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, fld := range st.Fields.List {
					if len(fld.Names) == 0 && mixinTypes[baseName(fld.Type)] {
						embedders[ts.Name.Name] = true
					}
				}
			}
		}
	}
	var probs []Problem
	for _, f := range files {
		for _, decl := range f.ast.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 {
				continue
			}
			recv := baseName(fd.Recv.List[0].Type)
			if embedders[recv] && methods[fd.Name.Name] {
				probs = append(probs, Problem{File: filepath.ToSlash(filepath.Join(dir, f.name)), Msg: fmt.Sprintf("%s declares %s, which a mixin provides (shadowing)", recv, fd.Name.Name)})
			}
		}
	}
	return probs
}
