//go:build securityharness && linux

package xim

import (
	"encoding/binary"
	"math/rand"
	"testing"
)

func b3XimCorpus() [][]byte {
	xev := XEvent{ResponseType: 2, Detail: 38, Sequence: 9, Time: 123, Root: 1, Event: 2, Child: 3, RootX: -4, RootY: 5, EventX: 6, EventY: -7, State: 8, SameScreen: true}
	ids := map[AttributeName]uint16{AttrInputStyle: 1, AttrSpotLocation: 2}
	nested := NewAttributeBuilder(ids).Nested(AttrSpotLocation, func(b *AttributeBuilder) {
		b.Point(AttrSpotLocation, Point{X: 10, Y: -20})
	}).Build()
	msgs := []Message{
		Connect{Major: 1, Minor: 0, AuthNames: []string{"MIT-MAGIC"}},
		OpenReply{IMID: 1, IMAttrs: []Attr{{ID: 0, Type: TypeStyle, Name: AttrQueryInputStyle}}, ICAttrs: []Attr{{ID: 1, Type: TypeLong, Name: AttrInputStyle}}},
		QueryExtensionReply{IMID: 1, Extensions: []Extension{{Major: 128, Minor: 1, Name: "XIM_EXT_MOVE"}}},
		EncodingNegotiation{IMID: 1, Encodings: []string{"COMPOUND_TEXT"}, EncodingInfos: []string{"info"}},
		EncodingNegotiationReply{IMID: 1, Category: 2, Index: -1},
		GetIMValuesReply{IMID: 1, Attrs: []Attribute{{ID: 0, Value: []byte{1, 2, 3}}}},
		CreateIC{IMID: 1, Attrs: nested},
		SetICValues{IMID: 1, ICID: 4, Attrs: nested},
		ResetICReply{IMID: 1, ICID: 4, PreeditString: UTF8ToCompoundText("ab")},
		ForwardEvent{IMID: 1, ICID: 4, Flag: ForwardEventSynchronous, Serial: 9, Event: xev},
		Commit{IMID: 1, ICID: 4, Data: CommitChars{Text: UTF8ToCompoundText("x")}},
		Commit{IMID: 1, ICID: 4, Data: CommitBoth{Keysym: 0x61, Text: []byte("y"), Sync: true}},
		PreeditDraw{IMID: 1, ICID: 4, Caret: 2, ChgFirst: 0, ChgLength: 1, Status: PreeditNoFeedback, PreeditString: UTF8ToCompoundText("nihao"), Feedbacks: []Feedback{FeedbackUnderline, FeedbackReverse}},
		PreeditCaret{IMID: 1, ICID: 4, Position: 3, Direction: CaretForwardChar, Style: CaretPrimary},
		ErrorMessage{IMID: 1, ICID: 4, Flag: ErrorInputMethodIDValid, Code: ErrBadProtocol, Detail: "bad"},
	}
	var out [][]byte
	for _, m := range msgs {
		out = append(out, Encode(m))
	}
	return out
}

func TestSecB3_S06a_XimDecodeFuzz(t *testing.T) {
	seed := int64(0x5eed1e)
	t.Logf("seed %#x", seed)
	rng := rand.New(rand.NewSource(seed))
	corpus := b3XimCorpus()
	for i := 0; i < 200000; i++ {
		data := append([]byte(nil), corpus[rng.Intn(len(corpus))]...)
		for range 1 + rng.Intn(4) {
			if len(data) < 6 {
				break
			}
			pos := 4 + rng.Intn(len(data)-4)
			switch rng.Intn(4) {
			case 0:
				data[pos] = byte(rng.Intn(256))
			case 1:
				if pos+2 <= len(data) {
					binary.LittleEndian.PutUint16(data[pos:], []uint16{0, 1, 0x7fff, 0x8000, 0xffff, 0xfffe, uint16(rng.Intn(64))}[rng.Intn(7)])
				}
			case 2:
				data = data[:pos]
			case 3:
				data[0] = byte(rng.Intn(256))
			}
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("FAIL: xim.Decode panicked on % x: %v", data, r)
				}
			}()
			if msg, err := Decode(data); err == nil {
				switch m := msg.(type) {
				case Commit:
					if c, ok := m.Data.(CommitChars); ok {
						_, _ = CompoundTextToUTF8(c.Text)
					}
				case PreeditDraw:
					_, _ = CompoundTextToUTF8(m.PreeditString)
				}
			}
		}()
	}
	for i := 0; i < 50000; i++ {
		n := rng.Intn(64)
		data := make([]byte, n)
		for j := range data {
			data[j] = []byte{0x1b, '$', '(', ')', '-', 'A', 'B', 'C', 'G', 0x80, 0xc3, 0xa9, 'a', 0x00, 0xff, '%', '/'}[rng.Intn(17)]
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("FAIL: CompoundTextToUTF8 panicked on % x: %v", data, r)
				}
			}()
			_, _ = CompoundTextToUTF8(data)
		}()
	}
}
