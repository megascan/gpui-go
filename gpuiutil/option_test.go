package gpuiutil

import (
	"encoding/json/v2"
	"testing"
)

func TestOption_Basics(t *testing.T) {
	var zero Option[int]
	if zero.IsSome() || !zero.IsNone() || zero.UnwrapOr(7) != 7 || zero.String() != "None" {
		t.Fatal("zero value is not None")
	}
	s := Some(0)
	if v, ok := s.Get(); !ok || v != 0 || s == zero || s.String() != "Some(0)" {
		t.Fatal("Some(0)")
	}
	if None[int]() != zero {
		t.Fatal("None")
	}
	type wrap struct {
		A Option[int] `json:"a,omitzero"`
		B Option[int] `json:"b,omitzero"`
	}
	b, err := json.Marshal(wrap{A: Some(2)})
	if err != nil || string(b) != `{"a":2}` {
		t.Fatalf("marshal = %s, %v", b, err)
	}
	var w wrap
	if err := json.Unmarshal([]byte(`{"a":null,"b":5}`), &w); err != nil || w.A.IsSome() || w.B != Some(5) {
		t.Fatalf("unmarshal = %+v, %v", w, err)
	}
}
