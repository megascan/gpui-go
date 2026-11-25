//go:build securityharness && linux

package s09b

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/megascan/gpui-go/internal/linux/xkb"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
)

func TestR2_S09b_KeymapTruncateAfterFstat(t *testing.T) {
	const seed uint64 = 0x5209b
	t.Logf("seed=%#x", seed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 50*time.Second, "", func(string) (string, error) {
		if err := xkb.Load(); err != nil {
			return "", fmt.Errorf("harness-error: %v", err)
		}
		ctx := xkb.NewContext(xkb.ContextNoFlags)
		rng := rand.New(rand.NewPCG(seed, 1))
		content := append([]byte("xkb_keymap {\n"), bytes.Repeat([]byte("// padding comment line to make the keymap span many pages\n"), 1<<14)...)
		deadline := time.Now().Add(30 * time.Second)
		iters := 0
		for time.Now().Before(deadline) {
			fd, err := unix.MemfdCreate("r2-keymap", 0)
			if err != nil {
				return "", err
			}
			if _, err := unix.Write(fd, content); err != nil {
				return "", err
			}
			peer, err := unix.Dup(fd)
			if err != nil {
				return "", err
			}
			delay := time.Duration(rng.IntN(400)) * time.Microsecond
			go func() {
				time.Sleep(delay)
				_ = unix.Ftruncate(peer, 0)
				_ = unix.Close(peer)
			}()
			km, _ := xkb.NewKeymapFromFd(ctx, fd, len(content), xkb.KeymapFormatTextV1, xkb.KeymapCompileNoFlags)
			if km != nil {
				km.Close()
			}
			iters++
		}
		return fmt.Sprintf("%d iterations without SIGBUS", iters), nil
	})
	if parent {
		t.Logf("%s", o)
		if !o.Safe() {
			t.Errorf("SIGBUS: the compositor shrinking the keymap memfd between NewKeymapFromFd's fstat and libxkbcommon's read crashes the client: %s", o.Verdict)
		}
	}
}
