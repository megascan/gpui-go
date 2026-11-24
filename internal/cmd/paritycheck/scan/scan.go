// Package scan reimplements the upstream scanner of docs/port/PARITY.md section 9 over _ref/.
package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Item is a public item or a test found in an upstream file.
type Item struct {
	Crate   string
	File    string
	Name    string
	Kind    string
	Line    int
	Cfg     []string
	Test    bool
	Ignored bool
}

// Warning is an upstream construct the section 9 rules do not count.
type Warning struct {
	File string
	Line int
	Msg  string
}

func (w Warning) String() string { return fmt.Sprintf("_ref/%s:%d: %s", w.File, w.Line, w.Msg) }

// Totals is one crate's row of PARITY.md section 2.
type Totals struct {
	Files int
	LOC   int
	Items int
	Tests int
}

// Result is a full scan of _ref/.
type Result struct {
	Items    []Item
	Styles   []StyleMethod
	Warnings []Warning
	Crates   map[string]Totals
	Files    map[string]bool
	Decls    map[string]map[string]bool
}

// StyledFile is the upstream file whose Styled trait invokes the style macros.
const StyledFile = "gpui/src/styled.rs"

const stylesFile = "gpui_macros/src/styles.rs"

// Scan walks every .rs file under refDir.
func Scan(refDir string) (*Result, error) {
	res := &Result{Crates: map[string]Totals{}, Files: map[string]bool{}, Decls: map[string]map[string]bool{}}
	type crateAcc struct {
		files []*fileResult
	}
	crates := map[string]*crateAcc{}
	err := filepath.WalkDir(refDir+string(filepath.Separator), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".rs") {
			return nil
		}
		rel, err := filepath.Rel(refDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		crate, _, _ := strings.Cut(rel, "/")
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		src := string(data)
		t := res.Crates[crate]
		t.Files++
		t.LOC += strings.Count(src, "\n")
		res.Crates[crate] = t
		res.Files[rel] = true
		fr := parseFile(crate, rel, src)
		res.Decls[rel] = fr.decls
		if crates[crate] == nil {
			crates[crate] = &crateAcc{}
		}
		crates[crate].files = append(crates[crate].files, fr)
		if rel == stylesFile {
			styles, err := ExpandStyles(src)
			if err != nil {
				return err
			}
			res.Styles = styles
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", refDir, err)
	}
	for _, acc := range crates {
		pub := map[string]bool{}
		priv := map[string]bool{}
		for _, fr := range acc.files {
			for n := range fr.pubTypes {
				pub[n] = true
			}
			for n := range fr.privTys {
				priv[n] = true
			}
		}
		for _, fr := range acc.files {
			res.Items = append(res.Items, fr.items...)
			for _, ii := range fr.implItems {
				if priv[ii.typ] && !pub[ii.typ] {
					continue
				}
				res.Items = append(res.Items, ii.item)
			}
			res.Warnings = append(res.Warnings, fr.warnings...)
		}
	}
	sort.SliceStable(res.Items, func(i, j int) bool {
		a, b := res.Items[i], res.Items[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	sort.SliceStable(res.Warnings, func(i, j int) bool {
		a, b := res.Warnings[i], res.Warnings[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	for _, it := range res.Items {
		t := res.Crates[it.Crate]
		if it.Test {
			t.Tests++
		} else {
			t.Items++
		}
		res.Crates[it.Crate] = t
	}
	if len(res.Styles) > 0 {
		crate := path.Dir(path.Dir(StyledFile))
		t := res.Crates[crate]
		t.Items += len(res.Styles)
		res.Crates[crate] = t
	}
	return res, nil
}
