//go:build securityharness && linux

package x11

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jezek/xgb/xproto"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
)

func v1IncrStart(minLen string) (string, error) {
	n, err := strconv.ParseUint(minLen, 10, 32)
	if err != nil {
		return "", err
	}
	display := os.Getenv("DISPLAY")
	in, err := newClipboardInner(display)
	if err != nil {
		return "", err
	}
	reader, err := newXContext(display)
	if err != nil {
		return "", err
	}
	prop, err := xproto.InternAtom(reader.conn, false, uint16(len("V1_INCR")), "V1_INCR").Reply()
	if err != nil {
		return "", err
	}
	val := make([]byte, 16)
	binary.LittleEndian.PutUint32(val, uint32(n))
	if err := xproto.ChangePropertyChecked(reader.conn, xproto.PropModeReplace, reader.winID, prop.Atom, in.atoms.INCR, 32, 4, val).Check(); err != nil {
		return "", err
	}
	ev := xproto.SelectionNotifyEvent{Requestor: reader.winID, Selection: in.atoms.CLIPBOARD, Target: in.atoms.UTF8_STRING, Property: prop.Atom}
	owner, err := newXContext(display)
	if err != nil {
		return "", err
	}
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			xproto.ChangeProperty(owner.conn, xproto.PropModeReplace, reader.winID, prop.Atom, in.atoms.INCR, 32, 4, val)
			xproto.GetInputFocus(owner.conn).Reply()
		}
	}()
	maxCap, incr := 0, 0
	for i := 0; i < 500; i++ {
		usingIncr := false
		var data []byte
		res, _, err := in.handleReadSelectionNotify(reader, in.atoms.UTF8_STRING, &usingIncr, &data, ev)
		maxCap = max(maxCap, cap(data))
		if err == nil && res == readSelNotifyResultIncrStarted {
			incr++
			if incr >= 3 {
				break
			}
		}
	}
	return fmt.Sprintf("incrStarted=%d maxCap=%d", incr, maxCap), nil
}

func TestSecV1_S06b_IncrMinDataLenGrow(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("needs sandboxed Xvfb")
	}
	if !bombchild.IsChild(t) {
		in, err := newClipboardInner(os.Getenv("DISPLAY"))
		if err != nil {
			t.Fatalf("HARNESS-ERROR: no X server: %v", err)
		}
		in.server.conn.Close()
	}
	for _, tc := range []struct {
		name  string
		limit int64
		arg   string
	}{
		{"control_small_len", bombchild.DefaultLimit, "16"},
		{"attack_4GiB_len", bombchild.DefaultLimit, "4294967295"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, parent := bombchild.Run(t, tc.limit, 60*time.Second, tc.arg, v1IncrStart)
			if !parent {
				return
			}
			if o.Verdict == "error" {
				t.Fatalf("HARNESS-ERROR child returned error: %s", o)
			}
			bombchild.ExpectSafe(t, o, "INCR start with minDataLen="+tc.arg)
		})
	}
}
