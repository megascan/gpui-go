// Command wlgen generates typed Go proxies over internal/linux/wl from Wayland protocol XML.
//
// Usage: wlgen [-o dir] [-pkg name] <xml file or directory>...
//
// Every protocol is parsed first so interface and enum references across files resolve;
// each <stem>.xml becomes <stem>_gen.go with dashes turned into underscores.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	fs := flag.NewFlagSet("wlgen", flag.ContinueOnError)
	out := fs.String("o", ".", "output directory")
	pkg := fs.String("pkg", "", "package name (default: base name of the output directory)")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	files, err := collect(fs.Args())
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("wlgen: no protocol XML given")
	}
	name := *pkg
	if name == "" {
		abs, err := filepath.Abs(*out)
		if err != nil {
			return fmt.Errorf("wlgen: %w", err)
		}
		name = filepath.Base(abs)
	}
	outputs, err := generateAll(files, name)
	if err != nil {
		return err
	}
	for _, o := range outputs {
		if err := os.WriteFile(filepath.Join(*out, o.name), o.data, 0o644); err != nil {
			return fmt.Errorf("wlgen: %w", err)
		}
	}
	return nil
}

type output struct {
	name string
	data []byte
}

func collect(args []string) ([]string, error) {
	var files []string
	for _, a := range args {
		st, err := os.Stat(a)
		if err != nil {
			return nil, fmt.Errorf("wlgen: %w", err)
		}
		if !st.IsDir() {
			files = append(files, a)
			continue
		}
		matches, err := filepath.Glob(filepath.Join(a, "*.xml"))
		if err != nil {
			return nil, fmt.Errorf("wlgen: %w", err)
		}
		sort.Strings(matches)
		files = append(files, matches...)
	}
	return files, nil
}

func outputName(path string) string {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return strings.ReplaceAll(stem, "-", "_") + "_gen.go"
}

func generateAll(files []string, pkg string) ([]output, error) {
	var ps []*protocol
	for _, f := range files {
		p, err := parseProtocol(f)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	u, err := newUniverse(ps)
	if err != nil {
		return nil, err
	}
	idents := map[string]string{}
	var outs []output
	for _, p := range ps {
		data, err := generate(p, u, pkg, idents)
		if err != nil {
			return nil, err
		}
		outs = append(outs, output{name: outputName(p.file), data: data})
	}
	return outs, nil
}
