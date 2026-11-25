//go:build linux

// Package xrm is a port of x11rb's X resource manager database
// (x11rb-protocol 0.13.2 resource_manager and x11rb's resource_manager module).
package xrm

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"
)

const maxInclusionDepth = 100

const maxIncludeFiles = 64

const maxIncludeFileBytes = 1 << 20

// AtomResourceManager is the RESOURCE_MANAGER atom.
const AtomResourceManager = 23

// AtomString is the STRING atom.
const AtomString = 31

// GetResourceDatabaseLongLength is the long_length of Database::GET_RESOURCE_DATABASE.
const GetResourceDatabaseLongLength = 100_000_000

type binding uint8

const (
	bindingTight binding = iota
	bindingLoose
)

type component struct {
	wildcard bool
	name     string
}

type bound struct {
	binding   binding
	component component
}

type entry struct {
	components []bound
	value      []byte
}

// Database is a parsed X resource database.
type Database struct {
	entries []entry
}

// NewFromDefault builds the database the way x11rb's Database::new_from_default does:
// RESOURCE_MANAGER (format and value of the root window's property), else ~/.Xresources or
// ~/.Xdefaults, combined with $XENVIRONMENT or ~/.Xdefaults-<hostname>.
func NewFromDefault(format uint8, value []byte, hostname string) *Database {
	var entries []entry
	if db, ok := NewFromGetPropertyReply(format, value); ok {
		entries = db.entries
	} else if home, ok := os.LookupEnv("HOME"); ok {
		data, err := os.ReadFile(filepath.Join(home, ".Xresources"))
		if err == nil {
			parseDataWithBaseDirectory(&entries, data, home)
		} else if data, err := os.ReadFile(filepath.Join(home, ".Xdefaults")); err == nil {
			parseDataWithBaseDirectory(&entries, data, home)
		}
	}
	if xenv, ok := os.LookupEnv("XENVIRONMENT"); ok {
		if data, err := os.ReadFile(xenv); err == nil {
			parseDataWithBaseDirectory(&entries, data, parentOr(xenv))
		}
	} else {
		path := filepath.Join(os.Getenv("HOME"), ".Xdefaults-"+hostname)
		if data, err := os.ReadFile(path); err == nil {
			parseDataWithBaseDirectory(&entries, data, parentOr(path))
		}
	}
	return &Database{entries: entries}
}

func parentOr(path string) string {
	dir := filepath.Dir(path)
	if dir == "" {
		return "."
	}
	return dir
}

// NewFromGetPropertyReply parses a RESOURCE_MANAGER GetProperty reply; it reports false
// when the property has the wrong format or is empty.
func NewFromGetPropertyReply(format uint8, value []byte) (*Database, bool) {
	if format == 8 && len(value) > 0 {
		return NewFromData(value), true
	}
	return nil, false
}

// NewFromData parses data with "." as the base directory of includes.
func NewFromData(data []byte) *Database {
	var entries []entry
	parseDataWithBaseDirectory(&entries, data, ".")
	return &Database{entries: entries}
}

// NewFromDataWithBaseDirectory parses data, resolving includes against basePath.
func NewFromDataWithBaseDirectory(data []byte, basePath string) *Database {
	var entries []entry
	parseDataWithBaseDirectory(&entries, data, basePath)
	return &Database{entries: entries}
}

// GetBytes returns the value of the best entry matching the query.
func (d *Database) GetBytes(resourceName, resourceClass string) ([]byte, bool) {
	return matchEntry(d.entries, resourceName, resourceClass)
}

// GetString returns the value as a string when it is valid UTF-8.
func (d *Database) GetString(resourceName, resourceClass string) (string, bool) {
	b, ok := d.GetBytes(resourceName, resourceClass)
	if !ok || !utf8.Valid(b) {
		return "", false
	}
	return string(b), true
}

// GetBool returns the value interpreted as a boolean.
func (d *Database) GetBool(resourceName, resourceClass string) (bool, bool) {
	s, ok := d.GetString(resourceName, resourceClass)
	if !ok {
		return false, false
	}
	return toBool(s)
}

// GetFloat32 is get_value::<f32>: ok reports a value is present, err a parse failure.
func (d *Database) GetFloat32(resourceName, resourceClass string) (value float32, ok bool, err error) {
	s, present := d.GetString(resourceName, resourceClass)
	if !present {
		return 0, false, nil
	}
	f, err := parseRustFloat(s)
	if err != nil {
		return 0, false, err
	}
	return f, true, nil
}

// GetUint32 is get_value::<u32>.
func (d *Database) GetUint32(resourceName, resourceClass string) (value uint32, ok bool, err error) {
	s, present := d.GetString(resourceName, resourceClass)
	if !present {
		return 0, false, nil
	}
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "+"), 10, 32)
	if err != nil || s == "+" || strings.HasPrefix(s, "++") {
		if err == nil {
			err = strconv.ErrSyntax
		}
		return 0, false, err
	}
	return uint32(v), true, nil
}

// GetInt32 is get_value::<i32>.
func (d *Database) GetInt32(resourceName, resourceClass string) (value int32, ok bool, err error) {
	s, present := d.GetString(resourceName, resourceClass)
	if !present {
		return 0, false, nil
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, false, err
	}
	return int32(v), true, nil
}

func parseRustFloat(s string) (float32, error) {
	lower := strings.ToLower(strings.TrimLeft(s, "+-"))
	if lower == "inf" || lower == "infinity" || lower == "nan" {
		v, err := strconv.ParseFloat(s, 32)
		return float32(v), err
	}
	if s == "" || strings.ContainsAny(s, "xXpP_") || strings.HasPrefix(lower, "0x") {
		return 0, strconv.ErrSyntax
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
			return float32(v), nil
		}
		return 0, err
	}
	return float32(v), nil
}

func parseDataWithBaseDirectory(result *[]entry, data []byte, basePath string) {
	budget := maxIncludeFiles
	parseDataWithIncludes(result, data, basePath, 0, &budget)
}

func parseDataWithIncludes(result *[]entry, data []byte, basePath string, depth uint8, budget *int) {
	if depth > maxInclusionDepth {
		return
	}
	parseDatabase(data, result, func(path []byte, entries *[]entry) {
		if !utf8.Valid(path) || *budget <= 0 {
			return
		}
		*budget--
		full := filepath.Join(basePath, string(path))
		if filepath.IsAbs(string(path)) {
			full = string(path)
		}
		newData, ok := readIncludeFile(full)
		if !ok {
			return
		}
		parseDataWithIncludes(entries, newData, parentOr(full), depth+1, budget)
	})
}

func readIncludeFile(path string) ([]byte, bool) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > maxIncludeFileBytes {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(f, maxIncludeFileBytes+1))
	if err != nil || len(data) > maxIncludeFileBytes {
		return nil, false
	}
	return data, true
}

func toBool(data string) (bool, bool) {
	if num, err := strconv.ParseInt(data, 10, 64); err == nil {
		return num != 0, true
	}
	switch strings.ToLower(data) {
	case "true", "on", "yes":
		return true, true
	case "false", "off", "no":
		return false, true
	}
	return false, false
}
