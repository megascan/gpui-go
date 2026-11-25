package taffy

import (
	"fmt"
	"slices"
	"testing"
)

var nonPercentageTags = []int{
	CompactLengthLengthTag,
	CompactLengthAutoTag,
	CompactLengthFrTag,
	CompactLengthMinContentTag,
	CompactLengthMaxContentTag,
	CompactLengthFitContentPxTag,
	CompactLengthFitContentKeywordTag,
	CompactLengthStretchTag,
	CompactLengthContentTag,
}

var percentageTags = []int{CompactLengthPercentTag, CompactLengthFitContentPercentTag}

const calcFeature = 0

func TestStyleCompactLength_TagsAreUniqueAndFitInAByte(t *testing.T) {
	all := slices.Concat(nonPercentageTags, percentageTags)
	for _, tag := range all {
		if tag > 0xFF {
			t.Fatalf("tag %#010b does not fit in a byte", tag)
		}
	}
	slices.Sort(all)
	all = slices.Compact(all)
	if got, want := len(all), len(nonPercentageTags)+len(percentageTags)+2*calcFeature; got != want {
		t.Fatalf("unique tags = %d, want %d", got, want)
	}
}

func TestStyleCompactLength_NonPointerTagsHaveNonPointerBitSet(t *testing.T) {
	for _, tag := range slices.Concat(nonPercentageTags, percentageTags) {
		if tag&CompactLengthNonPointerBit == 0 {
			t.Errorf("tag %#010b", tag)
		}
	}
}

func TestStyleCompactLength_KeywordBitsMatchPredicates(t *testing.T) {
	for _, raw := range slices.Concat(nonPercentageTags, percentageTags) {
		value := compactLengthFromTag(raw)
		tag := value.Tag()
		isPlain := tag == CompactLengthLengthTag || tag == CompactLengthPercentTag
		if (tag&CompactLengthKeywordBit == 0) != isPlain {
			t.Errorf("tag %#010b: keyword bit", tag)
		}
		if value.IsLengthOrPercentage() != isPlain {
			t.Errorf("tag %#010b: IsLengthOrPercentage", tag)
		}
		isSizingKeyword := false
		switch tag {
		case CompactLengthMinContentTag, CompactLengthMaxContentTag, CompactLengthFitContentKeywordTag,
			CompactLengthFitContentPxTag, CompactLengthFitContentPercentTag, CompactLengthStretchTag:
			isSizingKeyword = true
		}
		if value.IsSizingKeyword() != isSizingKeyword {
			t.Errorf("tag %#010b: IsSizingKeyword", tag)
		}
	}
}

func TestStyleCompactLength_PercentageBitMatchesUsesPercentage(t *testing.T) {
	for _, tag := range nonPercentageTags {
		if tag&CompactLengthPercentageBit != 0 {
			t.Errorf("tag %#010b", tag)
		}
	}
	for _, tag := range percentageTags {
		if tag&CompactLengthPercentageBit == 0 {
			t.Errorf("tag %#010b", tag)
		}
	}
	cases := []struct {
		value CompactLength
		want  bool
	}{
		{CompactLengthLength(10.0), false},
		{CompactLengthAuto(), false},
		{CompactLengthFr(1.0), false},
		{CompactLengthMinContent(), false},
		{CompactLengthMaxContent(), false},
		{CompactLengthFitContentPx(10.0), false},
		{CompactLengthFitContentKeyword(), false},
		{CompactLengthStretch(), false},
		{CompactLengthContent(), false},
		{CompactLengthPercent(0.5), true},
		{CompactLengthFitContentPercent(0.5), true},
	}
	for i, c := range cases {
		if got := c.value.UsesPercentage(); got != c.want {
			t.Errorf("case %d: UsesPercentage() = %v, want %v", i, got, c.want)
		}
	}
}

func TestStyleCompactLength_BitwiseEquality(t *testing.T) {
	nan := float32FromBits(0x7fc00000)
	if CompactLengthLength(nan) != CompactLengthLength(nan) {
		t.Error("NaN lengths with the same bits must be equal")
	}
	if CompactLengthLength(negZero()).IsZero() {
		t.Error("-0.0 is not ZERO")
	}
	if !CompactLengthLength(0).IsZero() {
		t.Error("0.0 is ZERO")
	}
	if CompactLengthAuto() == CompactLengthLength(0) {
		t.Error("auto must differ from 0 px")
	}
}

func TestStyleCompactLength_FitContent(t *testing.T) {
	if got := (CompactLength{}).FitContent(LengthPercentageLength(10)); got != CompactLengthFitContentPx(10) {
		t.Errorf("got %v", got)
	}
	if got := (CompactLength{}).FitContent(LengthPercentagePercent(0.5)); got != CompactLengthFitContentPercent(0.5) {
		t.Errorf("got %v", got)
	}
	if got := CompactLengthPercent(0.5).ResolvedPercentageSize(10, nil); got != some[float32](5) {
		t.Errorf("got %v", got)
	}
	if got := CompactLengthLength(3).ResolvedPercentageSize(10, nil); got.IsSome() {
		t.Errorf("got %v", got)
	}
	defer func() {
		if r := fmt.Sprint(recover()); r != "internal error: entered unreachable code" {
			t.Errorf("panic = %q", r)
		}
	}()
	(CompactLength{}).FitContent(LengthPercentageFromRaw(CompactLengthAuto()))
}
