//go:build securityharness

package gotext

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"
)

func TestSecV1_S03_StrikeCountTrusted(t *testing.T) {
	for _, tc := range []struct {
		name  string
		count uint32
		sbix  bool
	}{
		{"control_cblc_count_2", 2, false},
		{"attack_cblc_count_max", 0xFFFFFFFF, false},
		{"attack_sbix_count_max", 0xFFFFFFFF, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := make([]byte, 8+2*48)
			binary.BigEndian.PutUint32(table[4:], tc.count)
			strikes := newSwashBitmapStrikes(table, nil, 1000, tc.sbix, false)
			done := make(chan struct{})
			go func() {
				strikes.findByNearestPpem(16, 1)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("findByNearestPpem with strike count %#x from a %d-byte table still looping after 5s", tc.count, len(table))
			}
		})
	}
}

func TestSecV1_S03_BitmapDecodeLengthPanic(t *testing.T) {
	for _, tc := range []struct {
		name string
		size uint32
	}{
		{"control_size_matches", 4},
		{"attack_size_short", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 64)
			loc := swashLocation{format: 5, offset: 0, size: tc.size, ppem: 16, bitDepth: 8, width: 2, height: 2}
			bitmap, ok := loc.get(data, 1000, false)
			if !ok {
				t.Fatalf("location rejected")
			}
			var panicked any
			func() {
				defer func() { panicked = recover() }()
				target := resizeBytes(nil, bitmap.format.bufferSize(bitmap.width, bitmap.height))
				bitmap.decode(target)
			}()
			if panicked != nil {
				t.Fatalf("decode panicked on font-controlled size %d for 2x2 8-bit glyph: %v", tc.size, fmt.Sprint(panicked))
			}
		})
	}
}
