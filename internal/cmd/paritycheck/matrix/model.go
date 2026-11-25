package matrix

import (
	"fmt"
	"strings"
)

type Row struct {
	Area    string
	File    string
	Line    int
	Heading string
	Target  string
	Name    string
	Kind    string
	Test    bool
	Symbol  string
	Status  string
}

type Area struct {
	Name       string
	File       string
	Rows       []Row
	Deviations map[string]int
}

type Expect struct {
	Items int
	Tests int
}

type Matrix struct {
	Root       string
	Areas      []*Area
	Expect     map[string]Expect
	Registered map[string]int
	Styled     map[string]bool
}

type Problem struct {
	File string
	Line int
	Msg  string
}

func (p Problem) String() string {
	switch {
	case p.File != "" && p.Line > 0:
		return fmt.Sprintf("%s:%d: %s", p.File, p.Line, p.Msg)
	case p.File != "":
		return fmt.Sprintf("%s: %s", p.File, p.Msg)
	default:
		return p.Msg
	}
}

func (m *Matrix) Area(name string) *Area {
	for _, a := range m.Areas {
		if a.Name == name {
			return a
		}
	}
	return nil
}

func (m *Matrix) IsRegistered(id string) bool {
	_, ok := m.Registered[id]
	if ok {
		return true
	}
	for _, a := range m.Areas {
		if _, ok := a.Deviations[id]; ok {
			return true
		}
	}
	return false
}

func cellsOf(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	parts := strings.Split(t, "|")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if c == "" {
			return false
		}
		for _, r := range c {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

func unticks(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "`", ""))
}

func firstTicked(s string) string {
	i := strings.Index(s, "`")
	if i < 0 {
		return ""
	}
	j := strings.Index(s[i+1:], "`")
	if j < 0 {
		return ""
	}
	return s[i+1 : i+1+j]
}

func firstIndex(cells []string, names ...string) int {
	for _, n := range names {
		if i := indexOf(cells, n); i >= 0 {
			return i
		}
	}
	return -1
}

func indexOf(cells []string, want string) int {
	for i, c := range cells {
		if c == want {
			return i
		}
	}
	return -1
}

func validStatus(s string) bool {
	switch {
	case s == "todo" || s == "done":
		return true
	case strings.HasPrefix(s, "dev:"), strings.HasPrefix(s, "n/a:"):
		return len(s) > strings.Index(s, ":")+1
	case strings.HasPrefix(s, "blocked:"):
		return len(s) > len("blocked:")
	}
	return false
}
