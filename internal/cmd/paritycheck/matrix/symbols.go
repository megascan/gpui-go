package matrix

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type symset struct {
	decls   map[string]bool
	members map[string]bool
}

type dirIndex struct {
	exists bool
	prod   symset
	all    symset
}

type Index struct {
	root string
	dirs map[string]*dirIndex
	fset *token.FileSet
}

func NewIndex(root string) *Index {
	return &Index{root: root, dirs: map[string]*dirIndex{}, fset: token.NewFileSet()}
}

func newSymset() symset {
	return symset{decls: map[string]bool{}, members: map[string]bool{}}
}

func (x *Index) dir(rel string) *dirIndex {
	if d, ok := x.dirs[rel]; ok {
		return d
	}
	d := &dirIndex{prod: newSymset(), all: newSymset()}
	x.dirs[rel] = d
	abs := filepath.Join(x.root, filepath.FromSlash(rel))
	entries, err := os.ReadDir(abs)
	if err != nil {
		return d
	}
	d.exists = true
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		f, err := parser.ParseFile(x.fset, filepath.Join(abs, name), nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		isTest := strings.HasSuffix(name, "_test.go")
		collect(f, d.all)
		if !isTest {
			collect(f, d.prod)
		}
	}
	return d
}

func collect(f *ast.File, s symset) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil || len(d.Recv.List) == 0 {
				s.decls[d.Name.Name] = true
				continue
			}
			s.members[baseName(d.Recv.List[0].Type)+"."+d.Name.Name] = true
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch sp := spec.(type) {
				case *ast.TypeSpec:
					s.decls[sp.Name.Name] = true
					switch t := sp.Type.(type) {
					case *ast.InterfaceType:
						for _, m := range t.Methods.List {
							for _, n := range m.Names {
								s.members[sp.Name.Name+"."+n.Name] = true
							}
						}
					case *ast.StructType:
						for _, fld := range t.Fields.List {
							for _, n := range fld.Names {
								s.members[sp.Name.Name+"."+n.Name] = true
							}
						}
					}
				case *ast.ValueSpec:
					for _, n := range sp.Names {
						s.decls[n.Name] = true
					}
				}
			}
		}
	}
}

func baseName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return baseName(t.X)
	case *ast.IndexExpr:
		return baseName(t.X)
	case *ast.IndexListExpr:
		return baseName(t.X)
	case *ast.ParenExpr:
		return baseName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

func (x *Index) Exists(rel string) bool {
	return x.dir(rel).exists
}

func (x *Index) HasSymbol(rel, sym string, test bool) bool {
	d := x.dir(rel)
	set := d.prod
	if test {
		set = d.all
	}
	return symbolIn(set, sym)
}

func symbolIn(set symset, raw string) bool {
	s := strings.TrimSpace(strings.ReplaceAll(raw, "`", ""))
	switch {
	case s == "" || s == "-":
		return false
	case strings.HasPrefix(s, "("):
		end := strings.Index(s, ")")
		if end < 0 {
			return false
		}
		recv := strings.TrimPrefix(s[1:end], "*")
		if i := strings.Index(recv, "["); i >= 0 {
			recv = recv[:i]
		}
		name := strings.TrimPrefix(s[end+1:], ".")
		return set.members[recv+"."+name]
	case strings.Contains(s, "."):
		i := strings.Index(s, ".")
		return set.members[s[:i]+"."+s[i+1:]]
	}
	if i := strings.Index(s, "["); i >= 0 {
		s = s[:i]
	}
	return set.decls[s]
}
