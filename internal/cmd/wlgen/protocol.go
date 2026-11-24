package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
)

type protocol struct {
	Name       string  `xml:"name,attr"`
	Interfaces []iface `xml:"interface"`
	file       string
}

type iface struct {
	Name     string    `xml:"name,attr"`
	Version  string    `xml:"version,attr"`
	Requests []message `xml:"request"`
	Events   []message `xml:"event"`
	Enums    []enum    `xml:"enum"`
}

type message struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Since string `xml:"since,attr"`
	Args  []arg  `xml:"arg"`
}

type arg struct {
	Name      string `xml:"name,attr"`
	Type      string `xml:"type,attr"`
	Interface string `xml:"interface,attr"`
	AllowNull string `xml:"allow-null,attr"`
	Enum      string `xml:"enum,attr"`
}

type enum struct {
	Name     string  `xml:"name,attr"`
	Bitfield string  `xml:"bitfield,attr"`
	Entries  []entry `xml:"entry"`
}

type entry struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

func parseProtocol(path string) (*protocol, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("wlgen: read %s: %w", path, err)
	}
	var p protocol
	if err := xml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("wlgen: parse %s: %w", path, err)
	}
	p.file = path
	return &p, nil
}

func (m *message) since() (uint32, error) {
	if m.Since == "" {
		return 1, nil
	}
	v, err := strconv.ParseUint(m.Since, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("wlgen: %s: bad since %q: %w", m.Name, m.Since, err)
	}
	return uint32(v), nil
}

func (a *arg) nullable() bool { return a.AllowNull == "true" }

func (m *message) destructor() bool { return m.Type == "destructor" }

func (i *iface) version() (uint32, error) {
	v, err := strconv.ParseUint(i.Version, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("wlgen: %s: bad version %q: %w", i.Name, i.Version, err)
	}
	return uint32(v), nil
}
