package matrix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StyledArea is the area that owns the generated Styled methods listed in styled-methods.tsv.
const StyledArea = "core-style"

type Options struct {
	Areas        []string
	Done         []string
	AllDone      bool
	AllowBlocked bool
}

type Report struct {
	Problems []Problem
	Warnings []Problem
	Rows     int
}

func (r *Report) fail(row Row, format string, args ...any) {
	r.Problems = append(r.Problems, Problem{File: row.File, Line: row.Line, Msg: fmt.Sprintf(format, args...)})
}

func (r *Report) failf(format string, args ...any) {
	r.Problems = append(r.Problems, Problem{Msg: fmt.Sprintf(format, args...)})
}

func stageDone(root, stage string) bool {
	_, err := os.Stat(filepath.Join(root, "docs", "port", "status", stage+".done"))
	return err == nil
}

func Check(m *Matrix, opt Options) (Report, error) {
	var rep Report
	scope, err := selectAreas(m, opt.Areas)
	if err != nil {
		return rep, err
	}
	idx := NewIndex(m.Root)
	for _, a := range scope {
		for _, r := range a.Rows {
			rep.Rows++
			checkRow(m, idx, opt, r, &rep)
		}
	}
	for _, prefix := range opt.Done {
		matched := false
		for _, a := range scope {
			for _, r := range a.Rows {
				if !strings.HasPrefix(r.Heading, prefix) {
					continue
				}
				matched = true
				if r.Status == "todo" {
					rep.fail(r, "todo row under -done %q: %s", prefix, r.Name)
				}
			}
		}
		if !matched {
			rep.failf("-done %q matches no upstream file in scope", prefix)
		}
	}
	checkCounts(m, scope, opt, &rep)
	return rep, nil
}

func selectAreas(m *Matrix, names []string) ([]*Area, error) {
	if len(names) == 0 {
		return m.Areas, nil
	}
	var out []*Area
	for _, n := range names {
		a := m.Area(n)
		if a == nil {
			return nil, fmt.Errorf("unknown area %q", n)
		}
		out = append(out, a)
	}
	return out, nil
}

func checkRow(m *Matrix, idx *Index, opt Options, r Row, rep *Report) {
	kind, arg, _ := strings.Cut(r.Status, ":")
	switch kind {
	case "dev", "n/a":
		if !m.IsRegistered(arg) {
			rep.fail(r, "deviation id %q is not registered (PARITY.md section 5 or the area's Deviations table)", arg)
		}
	case "blocked":
		if arg == "owner" {
			if opt.AllDone {
				rep.fail(r, "blocked:owner is not allowed: %s", r.Name)
			} else {
				rep.Warnings = append(rep.Warnings, Problem{File: r.File, Line: r.Line, Msg: "blocked:owner row: " + r.Name})
			}
		}
	}
	if kind == "done" {
		checkDoneSymbol(idx, r, rep)
	}
	if opt.AllDone {
		switch {
		case r.Status == "todo":
			rep.fail(r, "todo row under -all-done: %s", r.Name)
		case kind == "blocked" && arg != "owner":
			if !opt.AllowBlocked {
				rep.fail(r, "blocked row under -all-done: %s (%s)", r.Name, r.Status)
			} else if stageDone(m.Root, arg) {
				rep.fail(r, "blocked:%s row remains after stage %s is done: %s", arg, arg, r.Name)
			}
		}
	}
}

func checkDoneSymbol(idx *Index, r Row, rep *Report) {
	if r.Target == "" {
		rep.fail(r, "done row has no Go target: %s", r.Name)
		return
	}
	if !idx.Exists(r.Target) {
		rep.fail(r, "done row's Go target directory %s does not exist", r.Target)
		return
	}
	if !idx.HasSymbol(r.Target, r.Symbol, r.Test) {
		rep.fail(r, "done row's Go symbol %q not found in %s", r.Symbol, r.Target)
	}
}

func checkCounts(m *Matrix, scope []*Area, opt Options, rep *Report) {
	if len(opt.Areas) == 0 {
		for name := range m.Expect {
			if m.Area(name) == nil {
				rep.failf("section 2 lists area %q but %s/%s.md does not exist", name, ParityDir, name)
			}
		}
	}
	for _, a := range scope {
		want, ok := m.Expect[a.Name]
		if !ok {
			continue
		}
		var items, tests int
		for _, r := range a.Rows {
			if r.Test {
				tests++
			} else {
				items++
			}
		}
		if a.Name == StyledArea {
			items += len(m.Styled)
		}
		if items < want.Items {
			rep.failf("area %s has %d item rows, PARITY.md section 2 needs %d", a.Name, items, want.Items)
		}
		if tests < want.Tests {
			rep.failf("area %s has %d test rows, PARITY.md section 2 needs %d", a.Name, tests, want.Tests)
		}
	}
}
