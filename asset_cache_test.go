package gpui

import (
	"math"
	"testing"

	"github.com/megascan/gpui-go/scheduler"
)

type hashBytes []byte

func (b hashBytes) Hash(h *FxHasher) {
	h.Write(b)
}

func TestAssetCache_HashMatchesRustcHash(t *testing.T) {
	cases := []struct {
		name string
		got  uint64
		want uint64
	}{
		{"0u8", Hash(uint8(0)), 0},
		{"1u8", Hash(uint8(1)), 12157901119326311915},
		{"100u8", Hash(uint8(100)), 16751747135202103309},
		{"u8max", Hash(uint8(math.MaxUint8)), 1211781028898739645},
		{"u16max", Hash(uint16(math.MaxUint16)), 16279819243059860173},
		{"u32max", Hash(uint32(math.MaxUint32)), 7729994835221066939},
		{"u64max", Hash(uint64(math.MaxUint64)), 6288842954450348564},
		{"i8min", Hash(int8(math.MinInt8)), 6684841074112525780},
		{"i16max", Hash(int16(math.MaxInt16)), 2060959061933882993},
		{"i32min", Hash(int32(math.MinInt32)), 9943947977240134995},
		{"i64min", Hash(int64(math.MinInt64)), 33554432},
		{"i64max", Hash(int64(math.MaxInt64)), 6288842954483902996},
		{"empty", Hash(hashBytes{}), 17606491139363777937},
		{"zero", Hash(hashBytes{0}), 5448590020104574886},
		{"six zeros", Hash(hashBytes{0, 0, 0, 0, 0, 0}), 16766921560080789783},
		{"one", Hash(hashBytes{1}), 5922447956811044110},
		{"two", Hash(hashBytes{2}), 5229781508510959783},
		{"uwu", Hash(hashBytes("uwu")), 7168164714682931527},
		{"long", Hash(hashBytes("These are some bytes for testing rustc_hash.")), 2349210501944688211},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestAssetCache_HashSliceAndStringPrefixes(t *testing.T) {
	var slice FxHasher
	slice.WriteUsize(3)
	slice.Write([]byte("uwu"))
	if got := Hash([]byte("uwu")); got != slice.Finish() {
		t.Errorf("[]byte: got %d, want %d", got, slice.Finish())
	}
	var str FxHasher
	str.Write([]byte("uwu"))
	str.WriteU8(0xff)
	if got := Hash("uwu"); got != str.Finish() {
		t.Errorf("string: got %d, want %d", got, str.Finish())
	}
	if Hash(SharedString("uwu")) != Hash("uwu") {
		t.Error("SharedString hashes differently from its str")
	}
}

type inferenceAsset struct{}

type inferenceSource struct{ id int }

func (inferenceAsset) Load(source inferenceSource, cx *App) func(s *scheduler.Scope) scheduler.Result[int] {
	return func(*scheduler.Scope) scheduler.Result[int] { return scheduler.Result[int]{Value: source.id} }
}

var _ = func(cx *App) {
	var _, _ = cx.FetchAsset[inferenceAsset](inferenceSource{})
	var _ bool = cx.HasAsset[inferenceAsset](inferenceSource{})
	cx.RemoveAsset[inferenceAsset](inferenceSource{})
	var _, _ = cx.FetchAsset[AssetLogger[inferenceAsset, inferenceSource, int]](inferenceSource{})
}
