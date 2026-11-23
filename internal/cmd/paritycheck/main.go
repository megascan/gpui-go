// Command paritycheck validates the parity matrix in docs/port and enforces the port's import rules.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/megascan/gpui-go/internal/cmd/paritycheck/matrix"
)

const usage = `usage: paritycheck [-root dir] [-area a,b] [-done prefix,...] [-all-done [-allow-blocked]] [-lint] [-rescan]

With no flags, every area file is checked. -lint alone runs only the import and comment lint.`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("paritycheck", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root")
	area := fs.String("area", "", "comma-separated area names to check")
	done := fs.String("done", "", "comma-separated upstream path prefixes that must have no todo rows")
	allDone := fs.Bool("all-done", false, "fail on any todo or blocked row")
	allowBlocked := fs.Bool("allow-blocked", false, "with -all-done, tolerate blocked:<stage> while docs/port/status/<stage>.done is absent")
	lint := fs.Bool("lint", false, "check DESIGN section 4 import rules, cgo confinement, comments and mixin shadowing")
	rescan := fs.Bool("rescan", false, "compare the matrix with a fresh scan of _ref/")
	fs.Usage = func() { fmt.Fprintln(stderr, usage); fs.PrintDefaults() }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	onlyLint := *lint && *area == "" && *done == "" && !*allDone && !*rescan
	exit := 0
	if !onlyLint {
		m, err := matrix.Load(absRoot)
		if err != nil {
			fmt.Fprintln(stderr, "paritycheck:", err)
			return 2
		}
		opt := matrix.Options{Areas: splitList(*area), Done: splitList(*done), AllDone: *allDone, AllowBlocked: *allowBlocked}
		if *allowBlocked && !*allDone {
			fmt.Fprintln(stderr, "paritycheck: -allow-blocked needs -all-done")
			return 2
		}
		rep, err := matrix.Check(m, opt)
		if err != nil {
			fmt.Fprintln(stderr, "paritycheck:", err)
			return 2
		}
		for _, w := range rep.Warnings {
			fmt.Fprintln(stderr, "warning:", w)
		}
		for _, p := range rep.Problems {
			fmt.Fprintln(stderr, p)
		}
		if len(rep.Problems) > 0 {
			exit = 1
		}
		if exit == 0 {
			fmt.Fprintf(stdout, "paritycheck: %d rows checked\n", rep.Rows)
		}
		if *rescan {
			if code := runRescan(absRoot, m, stdout, stderr); code != 0 {
				exit = code
			}
		}
	}
	if *lint {
		probs, err := matrix.Lint(absRoot)
		if err != nil {
			fmt.Fprintln(stderr, "paritycheck:", err)
			return 2
		}
		for _, p := range probs {
			fmt.Fprintln(stderr, p)
		}
		if len(probs) > 0 {
			exit = 1
		}
		if len(probs) == 0 && exit == 0 {
			fmt.Fprintln(stdout, "paritycheck: lint ok")
		}
	}
	return exit
}

func runRescan(root string, m *matrix.Matrix, stdout, stderr io.Writer) int {
	if _, err := os.Stat(filepath.Join(root, "_ref")); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(stdout, "paritycheck: rescan skipped, _ref/ is absent")
		return 0
	}
	if matrix.Rescanner == nil {
		fmt.Fprintln(stderr, "paritycheck: rescan: no scanner registered")
		return 1
	}
	if err := matrix.Rescanner(root, m, stdout); err != nil {
		fmt.Fprintln(stderr, "paritycheck: rescan:", err)
		return 1
	}
	return 0
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
