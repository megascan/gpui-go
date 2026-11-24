//go:build securityharness

package s15

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
)

func capLimit() int64 {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 6 << 30
	}
	for _, line := range strings.Split(string(b), "\n") {
		if rest, ok := strings.CutPrefix(line, "VmSize:"); ok {
			kb, _ := strconv.ParseInt(strings.TrimSuffix(strings.TrimSpace(rest), " kB"), 10, 64)
			return kb*1024 + 3<<30
		}
	}
	return 6 << 30
}

var childLimit = capLimit()

func TestControl512MiB(t *testing.T) {
	o, parent := bombchild.Run(t, childLimit, 60*time.Second, "", func(string) (string, error) {
		b := make([]byte, 512<<20)
		for i := 0; i < len(b); i += 4096 {
			b[i] = 1
		}
		return fmt.Sprintf("touched %d bytes", len(b)), nil
	})
	if parent {
		t.Logf("cap=%d bytes", childLimit)
		if o.Verdict != "ok" {
			t.Fatalf("harness-error: control child failed under cap: %s", o)
		}
	}
}
