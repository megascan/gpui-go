package scan

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/megascan/gpui-go/internal/cmd/paritycheck/matrix"
)

func init() {
	matrix.Rescanner = Rescan
}

var scannedKinds = map[string]bool{
	"struct": true, "enum": true, "union": true, "type": true, "const": true, "static": true, "fn": true,
	"trait": true, "trait-method": true, "method": true, "assoc-fn": true, "assoc-const": true,
	"macro": true, "proc-macro": true, "proc-macro-derive": true, "proc-macro-attribute": true,
}

// Report is the comparison of a scan with the matrix.
type Report struct {
	Missing  []Item
	Gone     []matrix.Row
	NoFile   []matrix.Row
	Styles   []string
	Problems []string
}

type key struct {
	file string
	name string
	test bool
}

// Compare checks the matrix rows of the selected areas (all when areas is empty) against a scan.
func Compare(res *Result, m *matrix.Matrix, styled map[string]bool, areas []string) Report {
	want := map[string]bool{}
	for _, a := range areas {
		want[a] = true
	}
	headings := map[string]bool{}
	rows := map[key]int{}
	var rep Report
	for _, a := range m.Areas {
		if strings.HasPrefix(a.Name, "ext-") {
			continue
		}
		inScope := len(want) == 0 || want[a.Name]
		for _, r := range a.Rows {
			if inScope {
				headings[r.Heading] = true
			}
			rows[key{r.Heading, r.Name, r.Test}]++
			if !inScope {
				continue
			}
			if !res.Files[r.Heading] {
				if strings.HasSuffix(r.Heading, ".rs") {
					rep.NoFile = append(rep.NoFile, r)
				}
				continue
			}
			if !r.Test && !scannedKinds[r.Kind] {
				continue
			}
			if !declared(res.Decls[r.Heading], r.Name) {
				rep.Gone = append(rep.Gone, r)
			}
		}
	}
	seen := map[key]int{}
	for _, it := range res.Items {
		if len(want) > 0 && !headings[it.File] {
			continue
		}
		k := key{it.File, it.Name, it.Test}
		seen[k]++
		if seen[k] > rows[k] {
			rep.Missing = append(rep.Missing, it)
		}
	}
	if (len(want) == 0 || headings[StyledFile]) && styled != nil {
		for _, s := range res.Styles {
			if !styled[s.Name] {
				rep.Styles = append(rep.Styles, s.Name)
			}
		}
	}
	return rep
}

func declared(decls map[string]bool, name string) bool {
	if decls[name] {
		return true
	}
	if i := strings.LastIndex(name, "::"); i >= 0 {
		return decls[name[i+2:]]
	}
	return false
}

// Rescan scans root/_ref, prints totals and warnings to w, and fails when the matrix misses scanned items or
// lists items that are gone upstream.
func Rescan(root string, m *matrix.Matrix, w io.Writer) error {
	res, err := Scan(filepath.Join(root, "_ref"))
	if err != nil {
		return err
	}
	return report(root, res, m, areasFromArgs(os.Args[1:]), w)
}

func areasFromArgs(args []string) []string {
	var val string
	for i := 0; i < len(args); i++ {
		a := strings.TrimLeft(args[i], "-")
		if !strings.HasPrefix(args[i], "-") {
			continue
		}
		if a == "area" && i+1 < len(args) {
			val = args[i+1]
			i++
		} else if v, ok := strings.CutPrefix(a, "area="); ok {
			val = v
		}
	}
	var out []string
	for _, p := range strings.Split(val, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func report(root string, res *Result, m *matrix.Matrix, areas []string, w io.Writer) error {
	expect, err := crateTotals(filepath.Join(root, matrix.ParityFile))
	if err != nil {
		return err
	}
	var crates []string
	for c := range res.Crates {
		crates = append(crates, c)
	}
	sort.Strings(crates)
	var sum, want Totals
	for _, c := range crates {
		got := res.Crates[c]
		exp := expect[c]
		sum = add(sum, got)
		want = add(want, exp)
		if got != exp {
			fmt.Fprintf(w, "rescan: %s: files %d LOC %d items %d tests %d; section 2 has %d %d %d %d (see docs/port/parity/_rescan.md)\n",
				c, got.Files, got.LOC, got.Items, got.Tests, exp.Files, exp.LOC, exp.Items, exp.Tests)
		}
	}
	fmt.Fprintf(w, "rescan: %d files, %d LOC, %d public items, %d tests (section 2: %d, %d, %d, %d)\n",
		sum.Files, sum.LOC, sum.Items, sum.Tests, want.Files, want.LOC, want.Items, want.Tests)
	for _, wn := range res.Warnings {
		fmt.Fprintln(w, "warning:", wn)
	}
	styled, err := styledNames(filepath.Join(root, matrix.StyledTSV))
	if err != nil {
		return err
	}
	rep := Compare(res, m, styled, areas)
	for _, it := range rep.Missing {
		rep.Problems = append(rep.Problems, fmt.Sprintf("_ref/%s:%d: %s %s `%s` has no matrix row", it.File, it.Line, it.Crate, kindLabel(it), it.Name))
	}
	for _, r := range rep.Gone {
		rep.Problems = append(rep.Problems, fmt.Sprintf("%s:%d: row `%s` (%s) is not declared in _ref/%s", r.File, r.Line, r.Name, r.Kind, r.Heading))
	}
	for _, r := range rep.NoFile {
		rep.Problems = append(rep.Problems, fmt.Sprintf("%s:%d: heading %s does not exist under _ref/", r.File, r.Line, r.Heading))
	}
	for _, s := range rep.Styles {
		rep.Problems = append(rep.Problems, fmt.Sprintf("%s: style method %s is not in %s", StyledFile, s, matrix.StyledTSV))
	}
	for _, p := range rep.Problems {
		fmt.Fprintln(w, p)
	}
	if len(rep.Problems) > 0 {
		return fmt.Errorf("%d problems", len(rep.Problems))
	}
	fmt.Fprintln(w, "paritycheck: rescan ok")
	return nil
}

func kindLabel(it Item) string {
	if it.Test {
		return "test"
	}
	return it.Kind
}

func add(a, b Totals) Totals {
	return Totals{a.Files + b.Files, a.LOC + b.LOC, a.Items + b.Items, a.Tests + b.Tests}
}

func crateTotals(p string) (map[string]Totals, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]Totals{}
	inTable := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "| Crate |") {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		if !strings.HasPrefix(line, "|") {
			break
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) != 5 {
			continue
		}
		name := strings.TrimSpace(cells[0])
		var nums [4]int
		ok := true
		for i := range 4 {
			n, err := strconv.Atoi(strings.TrimSpace(cells[i+1]))
			if err != nil {
				ok = false
				break
			}
			nums[i] = n
		}
		if !ok || name == "Total" {
			continue
		}
		out[name] = Totals{nums[0], nums[1], nums[2], nums[3]}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: section 2 crate table not found", p)
	}
	return out, nil
}

func styledNames(p string) (map[string]bool, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, _, _ := strings.Cut(line, "\t")
		out[strings.TrimSpace(name)] = true
	}
	return out, nil
}
