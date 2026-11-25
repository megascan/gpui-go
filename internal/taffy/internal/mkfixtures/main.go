// Command mkfixtures packs taffy's tests/xml fixtures into a deterministic zip archive.
package main

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	src := flag.String("src", "", "taffy tests/xml directory")
	out := flag.String("out", "", "output zip file")
	flag.Parse()
	if err := run(*src, *out); err != nil {
		fmt.Fprintln(os.Stderr, "mkfixtures:", err)
		os.Exit(1)
	}
}

func run(src, out string) error {
	if src == "" || out == "" {
		return errors.New("-src and -out are required")
	}
	if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("mkfixtures: %s not found, keeping the committed %s\n", src, out)
		return nil
	}
	var names []string
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".xml") {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", src, err)
	}
	slices.Sort(names)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	w.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.SetMode(0o644)
		fw, err := w.CreateHeader(hdr)
		if err != nil {
			return fmt.Errorf("add %s: %w", name, err)
		}
		if _, err := fw.Write(data); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close zip: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Printf("mkfixtures: wrote %d fixtures to %s\n", len(names), out)
	return nil
}
