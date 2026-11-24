//go:build linux

package wl

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// Interface describes a Wayland protocol interface. Generated code declares one per
// protocol interface and fills Methods, Events and New in init.
type Interface struct {
	Name    string
	Version uint32
	Methods []Message
	Events  []Message
	New     func() Object

	c *cInterface
}

// Message describes one request or event. Signature holds the argument type codes
// (i, u, f, s, o, n, a, h, with ? before a nullable argument) without the since prefix;
// Types has one entry per argument code, the interface of object and new_id arguments
// and nil for the rest.
type Message struct {
	Name       string
	Signature  string
	Since      uint32
	Types      []*Interface
	Destructor bool
}

type cMessage struct {
	name      *byte
	signature *byte
	types     **cInterface
}

type cInterface struct {
	name        *byte
	version     int32
	methodCount int32
	methods     *cMessage
	eventCount  int32
	events      *cMessage
}

var (
	tablesMu sync.Mutex
	pinner   runtime.Pinner
)

// CSignature returns the signature string libwayland reads: the since version when it
// is above 1, followed by Signature.
func (m *Message) CSignature() string {
	if m.Since > 1 {
		return strconv.FormatUint(uint64(m.Since), 10) + m.Signature
	}
	return m.Signature
}

// C returns the address of the interface's wl_interface table, building it in pinned
// Go memory on first use. The table lives for the rest of the process.
func (i *Interface) C() uintptr {
	tablesMu.Lock()
	defer tablesMu.Unlock()
	return uintptr(unsafe.Pointer(i.build()))
}

func (i *Interface) build() *cInterface {
	if i.c != nil {
		return i.c
	}
	c := &cInterface{}
	pinner.Pin(c)
	i.c = c
	c.name = pinnedString(i.Name)
	c.version = int32(i.Version)
	c.methodCount = int32(len(i.Methods))
	c.methods = buildMessages(i.Methods)
	c.eventCount = int32(len(i.Events))
	c.events = buildMessages(i.Events)
	return c
}

func buildMessages(ms []Message) *cMessage {
	if len(ms) == 0 {
		return nil
	}
	out := make([]cMessage, len(ms))
	pinner.Pin(&out[0])
	for k := range ms {
		m := &ms[k]
		out[k].name = pinnedString(m.Name)
		out[k].signature = pinnedString(m.CSignature())
		if len(m.Types) > 0 {
			types := make([]*cInterface, len(m.Types))
			pinner.Pin(&types[0])
			for j, t := range m.Types {
				if t != nil {
					types[j] = t.build()
				}
			}
			out[k].types = &types[0]
		}
	}
	return &out[0]
}

func pinnedString(s string) *byte {
	b := cString(s)
	pinner.Pin(&b[0])
	return &b[0]
}

type argCode struct {
	kind     byte
	nullable bool
}

func parseSignature(sig string) []argCode {
	out := make([]argCode, 0, len(sig))
	nullable := false
	for k := 0; k < len(sig); k++ {
		ch := sig[k]
		switch {
		case ch == '?':
			nullable = true
		case ch >= '0' && ch <= '9':
		default:
			out = append(out, argCode{kind: ch, nullable: nullable})
			nullable = false
		}
	}
	return out
}

func (m *Message) codes() []argCode {
	return parseSignature(m.Signature)
}

// DescribeC renders the wl_interface table at addr as text: name, version and every
// message with its signature and the interface names of its object arguments. Tests use
// it to compare generated tables with libwayland's.
func DescribeC(addr uintptr) string {
	var b strings.Builder
	describeC(&b, addr, -1, -1)
	return b.String()
}

// DescribeCPrefix is DescribeC limited to the first methods requests and events events.
func DescribeCPrefix(addr uintptr, methods, events int) string {
	var b strings.Builder
	describeC(&b, addr, methods, events)
	return b.String()
}

// MessageCounts returns the request and event counts of the wl_interface table at addr.
func MessageCounts(addr uintptr) (int, int) {
	return int(int32(load32(addr + 12))), int(int32(load32(addr + 24)))
}

func describeC(b *strings.Builder, addr uintptr, maxMethods, maxEvents int) {
	if addr == 0 {
		b.WriteString("<nil>")
		return
	}
	fmt.Fprintf(b, "%s v%d", goString(loadPtr(addr)), int32(load32(addr+8)))
	methods, events := MessageCounts(addr)
	if maxMethods >= 0 {
		methods = min(methods, maxMethods)
	}
	if maxEvents >= 0 {
		events = min(events, maxEvents)
	}
	describeMessages(b, "request", loadPtr(addr+16), methods)
	describeMessages(b, "event", loadPtr(addr+32), events)
}

func describeMessages(b *strings.Builder, kind string, msgs uintptr, n int) {
	for k := 0; k < n; k++ {
		m := msgs + uintptr(k)*24
		sig := goString(loadPtr(m + 8))
		fmt.Fprintf(b, "\n  %s %s %q", kind, goString(loadPtr(m)), sig)
		types := loadPtr(m + 16)
		for j, c := range parseSignature(sig) {
			if c.kind != 'o' && c.kind != 'n' {
				continue
			}
			name := "<nil>"
			if types != 0 {
				if t := loadPtr(types + uintptr(j)*8); t != 0 {
					name = goString(loadPtr(t))
				}
			}
			fmt.Fprintf(b, " %d:%s", j, name)
		}
	}
}
