//go:build linux

package screencap

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"

	gpui "github.com/megascan/gpui-go"
)

const (
	atomNone        = xproto.Atom(0)
	atomAny         = xproto.Atom(0)
	atomString      = xproto.AtomString
	atomWMName      = xproto.AtomWmName
	propertyFormat8 = 8
)

var errClosedParse = errors.New("xcb: connection closed, failed to parse data")

func getAtom(conn *xgb.Conn, atomName string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(conn, true, uint16(len(atomName)), atomName).Reply()
	if err != nil {
		return 0, err
	}
	return reply.Atom, nil
}

func getProperty(conn *xgb.Conn, win xproto.Window, prop, typ xproto.Atom, length uint32) (*xproto.GetPropertyReply, error) {
	return xproto.GetProperty(conn, false, win, prop, typ, 0, length).Reply()
}

func propertyValue(reply *xproto.GetPropertyReply, format byte) []byte {
	n := int(reply.ValueLen) * int(reply.Format) / 8
	if n == 0 {
		return nil
	}
	if reply.Format != format {
		panic(fmt.Sprintf("assertion `left == right` failed: mismatched format of xproto::GetPropertyReply::value\n  left: %d\n right: %d", reply.Format, format))
	}
	if n > len(reply.Value) {
		n = len(reply.Value)
	}
	return reply.Value[:n]
}

func propertyWindows(reply *xproto.GetPropertyReply) []xproto.Window {
	b := propertyValue(reply, 32)
	out := make([]xproto.Window, 0, len(b)/4)
	for i := 0; i+4 <= len(b); i += 4 {
		out = append(out, xproto.Window(xgb.Get32(b[i:])))
	}
	return out
}

func decodeCompoundText(value []byte) (string, error) {
	if bytes.IndexByte(value, 0) >= 0 {
		return "", errors.New("nul byte found in provided data")
	}
	if len(value) == 0 {
		return "n/a", nil
	}
	var out []rune
	utf8Mode := false
	gr := true
	for i := 0; i < len(value); {
		c := value[i]
		if c == 0x1b {
			rest := value[i:]
			switch {
			case bytes.HasPrefix(rest, []byte{0x1b, '%', 'G'}):
				utf8Mode = true
				i += 3
			case bytes.HasPrefix(rest, []byte{0x1b, '%', '@'}):
				utf8Mode = false
				i += 3
			case bytes.HasPrefix(rest, []byte{0x1b, '(', 'B'}):
				i += 3
			case bytes.HasPrefix(rest, []byte{0x1b, '-', 'A'}):
				gr = true
				i += 3
			default:
				return "n/a", nil
			}
			continue
		}
		if utf8Mode {
			r, size := utf8.DecodeRune(value[i:])
			if r == utf8.RuneError && size <= 1 {
				return "n/a", nil
			}
			out = append(out, r)
			i += size
			continue
		}
		if c >= 0x80 && !gr {
			return "n/a", nil
		}
		out = append(out, rune(c))
		i++
	}
	return string(out), nil
}

func utf8OrNA(b []byte) string {
	if !utf8.Valid(b) {
		return "n/a"
	}
	return string(b)
}

func getX11Targets() ([]gpui.ScapTarget, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := randr.Init(conn); err != nil {
		return nil, err
	}
	screens := xproto.Setup(conn).Roots

	wmClientList, err := getAtom(conn, "_NET_CLIENT_LIST")
	if err != nil {
		return nil, err
	}
	if wmClientList == atomNone {
		panic("EWMH not supported")
	}
	atomNetWMName, err := getAtom(conn, "_NET_WM_NAME")
	if err != nil {
		return nil, err
	}
	atomText, err := getAtom(conn, "TEXT")
	if err != nil {
		return nil, err
	}
	atomUTF8String, err := getAtom(conn, "UTF8_STRING")
	if err != nil {
		return nil, err
	}
	atomCompoundText, err := getAtom(conn, "COMPOUND_TEXT")
	if err != nil {
		return nil, err
	}

	var targets []gpui.ScapTarget
	for _, screen := range screens {
		windowList, err := getProperty(conn, screen.Root, wmClientList, atomNone, 100)
		if err != nil {
			return nil, err
		}
		for _, client := range propertyWindows(windowList) {
			cr, err := getProperty(conn, client, atomNetWMName, atomString, 4096)
			if err != nil {
				return nil, err
			}
			if len(propertyValue(cr, 32)) != 0 {
				title := propertyValue(cr, propertyFormat8)
				if !utf8.Valid(title) {
					return nil, errClosedParse
				}
				targets = append(targets, &gpui.ScapWindow{ID: 0, Title: string(title), RawHandle: uintptr(client)})
				continue
			}

			reply, err := getProperty(conn, client, atomWMName, atomAny, 4096)
			if err != nil {
				return nil, err
			}
			value := propertyValue(reply, propertyFormat8)
			if len(value) != 0 {
				ttype := reply.Type
				var title string
				switch {
				case ttype == atomString || ttype == atomUTF8String || ttype == atomText:
					title = utf8OrNA(value)
				case ttype == atomCompoundText:
					t, err := decodeCompoundText(value)
					if err != nil {
						return nil, errClosedParse
					}
					title = t
				default:
					title = utf8OrNA(value)
				}
				targets = append(targets, &gpui.ScapWindow{ID: 0, Title: title, RawHandle: uintptr(client)})
				continue
			}
			targets = append(targets, &gpui.ScapWindow{ID: 0, Title: "n/a", RawHandle: uintptr(client)})
		}

		resources, err := randr.GetScreenResources(conn, screen.Root).Reply()
		if err != nil {
			return nil, err
		}
		for _, output := range resources.Outputs {
			info, err := randr.GetOutputInfo(conn, output, 0).Reply()
			if err != nil {
				return nil, err
			}
			if info.Connection == randr.ConnectionConnected {
				crtc := info.Crtc
				crtcInfo, err := randr.GetCrtcInfo(conn, crtc, 0).Reply()
				if err != nil {
					return nil, err
				}
				targets = append(targets, &gpui.ScapDisplay{
					ID:        uint32(crtc),
					Title:     utf8OrNA(info.Name),
					Width:     crtcInfo.Width,
					Height:    crtcInfo.Height,
					XOffset:   crtcInfo.X,
					YOffset:   crtcInfo.Y,
					RawHandle: uintptr(screen.Root),
				})
			}
		}
	}
	return targets, nil
}

func linuxGetAllTargets() ([]gpui.ScapTarget, error) {
	if _, ok := os.LookupEnv("WAYLAND_DISPLAY"); ok {
		return []gpui.ScapTarget{}, nil
	}
	if _, ok := os.LookupEnv("DISPLAY"); ok {
		return getX11Targets()
	}
	return nil, errors.New(unsupportedPlatformMsg)
}

func getDefaultXDisplay(conn *xgb.Conn, screen *xproto.ScreenInfo) (*gpui.ScapDisplay, error) {
	primary, err := randr.GetOutputPrimary(conn, screen.Root).Reply()
	if err != nil {
		return nil, err
	}
	info, err := randr.GetOutputInfo(conn, primary.Output, 0).Reply()
	if err != nil {
		return nil, err
	}
	crtc := info.Crtc
	crtcInfo, err := randr.GetCrtcInfo(conn, crtc, 0).Reply()
	if err != nil {
		return nil, err
	}
	title := "default"
	if utf8.Valid(info.Name) {
		title = string(info.Name)
	}
	return &gpui.ScapDisplay{
		ID:        uint32(crtc),
		Title:     title,
		Width:     crtcInfo.Width,
		Height:    crtcInfo.Height,
		XOffset:   crtcInfo.X,
		YOffset:   crtcInfo.Y,
		RawHandle: uintptr(screen.Root),
	}, nil
}

func linuxGetMainDisplay() (*gpui.ScapDisplay, error) {
	if _, ok := os.LookupEnv("WAYLAND_DISPLAY"); ok {
		return nil, errors.New("Getting main display not currently supported on Wayland.")
	}
	if _, ok := os.LookupEnv("DISPLAY"); ok {
		conn, err := xgb.NewConn()
		if err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		defer conn.Close()
		if err := randr.Init(conn); err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
		roots := xproto.Setup(conn).Roots
		if conn.DefaultScreen >= len(roots) {
			panic("called `Option::unwrap()` on a `None` value")
		}
		d, err := getDefaultXDisplay(conn, &roots[conn.DefaultScreen])
		if err != nil {
			return nil, fmt.Errorf("Failed to get main X11 display.: %w", err)
		}
		return d, nil
	}
	return nil, errors.New(unsupportedPlatformMsg)
}

func linuxGetTargetDimensions(target gpui.ScapTarget) (uint64, uint64) {
	switch t := target.(type) {
	case *gpui.ScapDisplay:
		return uint64(t.Width), uint64(t.Height)
	}
	return 0, 0
}
