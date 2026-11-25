package matrix

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	ParityFile = "docs/port/PARITY.md"
	ParityDir  = "docs/port/parity"
	StyledTSV  = "docs/port/parity/styled-methods.tsv"
)

var idPattern = regexp.MustCompile(`^[LD]-[A-Z0-9][A-Z0-9-]*$`)

func Load(root string) (*Matrix, error) {
	m := &Matrix{Root: root, Expect: map[string]Expect{}, Registered: map[string]int{}, Styled: map[string]bool{}}
	parity, err := readLines(filepath.Join(root, ParityFile))
	if err != nil {
		return nil, err
	}
	if err := parseParity(m, parity); err != nil {
		return nil, fmt.Errorf("%s: %w", ParityFile, err)
	}
	entries, err := os.ReadDir(filepath.Join(root, ParityDir))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, "_") {
			continue
		}
		a, err := loadArea(filepath.Join(root, ParityDir, name))
		if err != nil {
			return nil, err
		}
		m.Areas = append(m.Areas, a)
	}
	sort.Slice(m.Areas, func(i, j int) bool { return m.Areas[i].Name < m.Areas[j].Name })
	if err := loadStyled(m, filepath.Join(root, StyledTSV)); err != nil {
		return nil, err
	}
	return m, nil
}

func readLines(p string) ([]string, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}

func linkTarget(cell string) string {
	open := strings.LastIndex(cell, "](")
	if open < 0 {
		return ""
	}
	rest := cell[open+2:]
	end := strings.Index(rest, ")")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func parseParity(m *Matrix, lines []string) error {
	section := ""
	var header []string
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			section = strings.TrimPrefix(line, "## ")
			header = nil
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			header = nil
			continue
		}
		cells := cellsOf(line)
		switch {
		case strings.HasPrefix(section, "2. "):
			if header == nil {
				if len(cells) > 0 && cells[0] == "Area file" {
					if indexOf(cells, "Public items") < 0 || indexOf(cells, "Tests") < 0 {
						return fmt.Errorf("line %d: section 2 header lacks Public items or Tests", i+1)
					}
					header = cells
				}
				continue
			}
			if isSeparator(cells) {
				continue
			}
			if len(cells) != len(header) {
				return fmt.Errorf("line %d: section 2 row has %d cells, want %d", i+1, len(cells), len(header))
			}
			file := linkTarget(cells[0])
			if file == "" {
				continue
			}
			items, err := atoiCell(cells[indexOf(header, "Public items")])
			if err != nil {
				return fmt.Errorf("line %d: %w", i+1, err)
			}
			tests, err := atoiCell(cells[indexOf(header, "Tests")])
			if err != nil {
				return fmt.Errorf("line %d: %w", i+1, err)
			}
			name := strings.TrimSuffix(path.Base(file), ".md")
			m.Expect[name] = Expect{Items: items, Tests: tests}
		case strings.HasPrefix(section, "5. "):
			if isSeparator(cells) || len(cells) == 0 {
				continue
			}
			id := unticks(cells[0])
			if idPattern.MatchString(id) {
				m.Registered[id] = i + 1
			}
		}
	}
	if len(m.Expect) == 0 {
		return errors.New("section 2 area table not found")
	}
	return nil
}

func atoiCell(s string) (int, error) {
	n, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	if err != nil {
		return 0, fmt.Errorf("bad count %q", s)
	}
	return n, nil
}

func parseTarget(line string) string {
	tok := firstTicked(strings.TrimPrefix(line, "Go target:"))
	switch {
	case tok == "" || tok == "-":
		return ""
	case strings.HasSuffix(tok, "/"):
		return path.Clean(tok)
	case strings.HasSuffix(tok, ".go"):
		return path.Dir(tok)
	}
	return ""
}

func loadArea(p string) (*Area, error) {
	name := strings.TrimSuffix(filepath.Base(p), ".md")
	lines, err := readLines(p)
	if err != nil {
		return nil, err
	}
	a := &Area{Name: name, File: filepath.ToSlash(filepath.Join(ParityDir, filepath.Base(p))), Deviations: map[string]int{}}
	var heading, target string
	inDev := false
	var header []string
	kind := 0
	const (
		none = iota
		items
		tests
		devs
	)
	for i, line := range lines {
		lineNo := i + 1
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimPrefix(line, "## ")
			inDev = title == "Deviations registered here"
			heading, target = "", ""
			if !inDev {
				heading = title
			}
			header, kind = nil, none
			continue
		}
		if strings.HasPrefix(line, "Go target:") {
			target = parseTarget(line)
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			header, kind = nil, none
			continue
		}
		cells := cellsOf(line)
		if kind == none {
			switch {
			case len(cells) > 0 && (cells[0] == "Upstream item" || cells[0] == "Upstream test"):
				kind = items
				if cells[0] == "Upstream test" {
					kind = tests
				}
				header = cells
			case inDev && len(cells) > 0 && cells[0] == "Id":
				kind = devs
				header = cells
			}
			continue
		}
		if isSeparator(cells) {
			continue
		}
		if len(cells) != len(header) {
			return nil, fmt.Errorf("%s:%d: row has %d cells, want %d", a.File, lineNo, len(cells), len(header))
		}
		if kind == devs {
			if id := unticks(cells[0]); id != "" {
				a.Deviations[id] = lineNo
			}
			continue
		}
		if heading == "" {
			return nil, fmt.Errorf("%s:%d: row outside an upstream heading", a.File, lineNo)
		}
		kindCol := firstIndex(header, "Kind", "Attr")
		symCol := firstIndex(header, "Go symbol", "Go test")
		statusCol := indexOf(header, "Status")
		if kindCol < 0 || symCol < 0 || statusCol < 0 {
			return nil, fmt.Errorf("%s:%d: table header lacks Kind/Attr, Go symbol/Go test or Status", a.File, lineNo)
		}
		r := Row{
			Area:    name,
			File:    a.File,
			Line:    lineNo,
			Heading: heading,
			Target:  target,
			Name:    unticks(cells[0]),
			Kind:    unticks(cells[kindCol]),
			Test:    kind == tests,
			Symbol:  unticks(cells[symCol]),
			Status:  unticks(cells[statusCol]),
		}
		if !validStatus(r.Status) {
			return nil, fmt.Errorf("%s:%d: bad status %q", a.File, lineNo, r.Status)
		}
		a.Rows = append(a.Rows, r)
	}
	return a, nil
}

func loadStyled(m *Matrix, p string) error {
	lines, err := readLines(p)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) < 2 {
			return fmt.Errorf("%s: want 2 tab-separated columns: %q", p, line)
		}
		m.Styled[strings.TrimSpace(cols[1])] = true
	}
	return nil
}
