//go:build linux

// Package xcursor ports the xcursor crate (0.3.11: theme lookup and the Xcursor file
// parser) and x11rb's cursor module (0.13.2: loading theme cursors over the X connection).
package xcursor

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type cursorFormat uint8

const (
	cursorFormatXCursor cursorFormat = iota
	cursorFormatScalable
)

func (f cursorFormat) subdir() string {
	if f == cursorFormatScalable {
		return "cursors_scalable"
	}
	return "cursors"
}

func (f cursorFormat) matches(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if f == cursorFormatScalable {
		return info.IsDir()
	}
	return info.Mode().IsRegular()
}

// CursorTheme is a cursor theme with its search paths.
type CursorTheme struct {
	theme       cursorThemeImpl
	searchPaths []string
}

// LoadCursorTheme loads the theme name from the standard search paths.
func LoadCursorTheme(name string) CursorTheme {
	searchPaths := themeSearchPaths(getSearchPathsEnvironment())
	return CursorTheme{theme: loadCursorThemeImpl(name, searchPaths), searchPaths: searchPaths}
}

// LoadIcon returns the path of the Xcursor file for iconName.
func (t CursorTheme) LoadIcon(iconName string) (string, bool) {
	walked := map[string]bool{}
	path, _, ok := t.theme.loadIconWithDepth(iconName, cursorFormatXCursor, t.searchPaths, walked)
	return path, ok
}

type themeDir struct {
	dir      string
	inherits string
	hasInh   bool
}

type cursorThemeImpl struct {
	name string
	data []themeDir
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func loadCursorThemeImpl(name string, searchPaths []string) cursorThemeImpl {
	var data []themeDir
	for _, base := range searchPaths {
		path := filepath.Join(base, name)
		if !isDir(path) {
			continue
		}
		d := themeDir{dir: path}
		if inherits, ok := themeInherits(filepath.Join(path, "index.theme")); ok {
			d.inherits, d.hasInh = inherits, true
		} else if name != "default" {
			d.inherits, d.hasInh = "default", true
		}
		data = append(data, d)
	}
	return cursorThemeImpl{name: name, data: data}
}

func (t cursorThemeImpl) loadIconWithDepth(iconName string, format cursorFormat, searchPaths []string, walked map[string]bool) (string, int, bool) {
	for _, d := range t.data {
		iconPath := filepath.Join(d.dir, format.subdir(), iconName)
		if format.matches(iconPath) {
			return iconPath, 0, true
		}
	}
	walked[t.name] = true
	for _, d := range t.data {
		if !d.hasInh || walked[d.inherits] {
			continue
		}
		inherited := loadCursorThemeImpl(d.inherits, searchPaths)
		if path, depth, ok := inherited.loadIconWithDepth(iconName, format, searchPaths, walked); ok {
			return path, depth + 1, true
		}
	}
	return "", 0, false
}

type searchPathsEnvironment struct {
	home, xcursorPath, xdgDataHome, xdgDataDirs string
}

func nonEmptyEnv(name string) string { return os.Getenv(name) }

func getSearchPathsEnvironment() searchPathsEnvironment {
	return searchPathsEnvironment{
		home:        nonEmptyEnv("HOME"),
		xcursorPath: nonEmptyEnv("XCURSOR_PATH"),
		xdgDataHome: nonEmptyEnv("XDG_DATA_HOME"),
		xdgDataDirs: nonEmptyEnv("XDG_DATA_DIRS"),
	}
}

func themeSearchPaths(env searchPathsEnvironment) []string {
	if env.xcursorPath != "" {
		var paths []string
		for _, entry := range strings.Split(env.xcursorPath, ":") {
			if entry == "" {
				continue
			}
			if p, ok := expandHomeDir(entry, env.home); ok {
				paths = append(paths, p)
			}
		}
		return paths
	}
	var paths []string
	if env.xdgDataHome != "" {
		if p, ok := expandHomeDir(env.xdgDataHome, env.home); ok {
			paths = append(paths, p)
		}
	} else if env.home != "" {
		paths = append(paths, filepath.Join(env.home, ".local/share/icons"))
	}
	if env.home != "" {
		paths = append(paths, filepath.Join(env.home, ".icons"))
	}
	if env.xdgDataDirs != "" {
		for _, entry := range strings.Split(env.xdgDataDirs, ":") {
			if entry == "" {
				continue
			}
			if p, ok := expandHomeDir(entry, env.home); ok {
				paths = append(paths, filepath.Join(p, "icons"))
			}
		}
	} else {
		paths = append(paths, "/usr/local/share/icons", "/usr/share/icons")
	}
	paths = append(paths, "/usr/share/pixmaps")
	if env.home != "" {
		paths = append(paths, filepath.Join(env.home, ".cursors"))
	}
	return append(paths, "/usr/share/cursors/xorg-x11")
}

func expandHomeDir(path, home string) (string, bool) {
	first, rest, _ := strings.Cut(path, "/")
	if first != "~" {
		return path, true
	}
	if home == "" {
		return "", false
	}
	return filepath.Join(home, rest), true
}

func themeInherits(path string) (string, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return parseTheme(string(content))
}

func parseTheme(content string) (string, bool) {
	const pattern = "Inherits"
	sep := func(r rune) bool { return unicode.IsSpace(r) || r == ';' || r == ',' }
	for line := range strings.Lines(content) {
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		rest, ok := strings.CutPrefix(line, pattern)
		if !ok {
			continue
		}
		rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
		rest, ok = strings.CutPrefix(rest, "=")
		if !ok {
			continue
		}
		rest = strings.TrimLeftFunc(rest, sep)
		if i := strings.IndexFunc(rest, sep); i >= 0 {
			rest = rest[:i]
		}
		if rest != "" {
			return rest, true
		}
	}
	return "", false
}
