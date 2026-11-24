package gpui

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"log/slog"
	"math/bits"
	"reflect"

	"github.com/megascan/gpui-go/collections"
	"github.com/megascan/gpui-go/scheduler"
)

type cachedLoadState[T any] struct {
	loaded  bool
	evicted bool
	views   collections.HashSet[EntityID]
	result  T
}

type cachedLoad[T any] struct {
	state *cachedLoadState[T]
	task  *Task[struct{}]
}

type cachedLoadReleaser interface {
	release()
	evict(cx *App)
}

func newCachedLoad[T any](future func(s *scheduler.Scope) T, cx *App) *cachedLoad[T] {
	return newObservedCachedLoad(future, cx, nil)
}

func newObservedCachedLoad[T any](future func(s *scheduler.Scope) T, cx *App, onLoaded func(result T, cx *App)) *cachedLoad[T] {
	state := &cachedLoadState[T]{views: collections.HashSet[EntityID]{}}
	task := cx.BackgroundExecutor().Spawn(func(s *scheduler.Scope) (T, error) {
		return future(s), nil
	})
	fg := cx.Spawn(func(cx *AsyncApp) (struct{}, error) {
		result, _ := task.Await(cx)
		if state.evicted {
			return struct{}{}, nil
		}
		if state.loaded {
			panic("a cached load completes only once")
		}
		views := state.views
		state.loaded = true
		state.views = nil
		state.result = result
		cx.Update(func(cx *App) {
			if onLoaded != nil {
				onLoaded(result, cx)
			}
			for view := range views {
				cx.Notify(view)
			}
		})
		return struct{}{}, nil
	})
	return &cachedLoad[T]{state: state, task: fg}
}

func (l *cachedLoad[T]) get() (T, bool) {
	if !l.state.loaded {
		var zero T
		return zero, false
	}
	return l.state.result, true
}

func (l *cachedLoad[T]) useBy(view EntityID) (T, bool) {
	if !l.state.loaded {
		l.state.views[view] = struct{}{}
		var zero T
		return zero, false
	}
	return l.state.result, true
}

func (l *cachedLoad[T]) release() {
	l.state.evicted = true
	l.task.Cancel()
}

func (l *cachedLoad[T]) evict(cx *App) {
	views := l.state.views
	l.release()
	if len(views) == 0 {
		return
	}
	cx.Defer(func(cx *App) {
		for view := range views {
			cx.Notify(view)
		}
	})
}

const (
	maxAssetCacheEntries       = 4096
	maxAssetCacheBytes   int64 = 1 << 30
)

type assetCache struct {
	entries map[loadingAssetKey]*list.Element
	order   list.List
	bytes   int64
}

type assetSlot struct {
	key   loadingAssetKey
	load  cachedLoadReleaser
	bytes int64
}

func newAssetCache() *assetCache {
	return &assetCache{entries: map[loadingAssetKey]*list.Element{}}
}

func (c *assetCache) has(key loadingAssetKey) bool {
	_, ok := c.entries[key]
	return ok
}

func (c *assetCache) get(key loadingAssetKey) (*assetSlot, bool) {
	element, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(element)
	return element.Value.(*assetSlot), true
}

func (c *assetCache) insert(slot *assetSlot, cx *App) {
	c.entries[slot.key] = c.order.PushFront(slot)
	c.trim(slot, cx)
}

func (c *assetCache) remove(key loadingAssetKey) (*assetSlot, bool) {
	element, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	slot := element.Value.(*assetSlot)
	delete(c.entries, key)
	c.order.Remove(element)
	c.bytes -= slot.bytes
	return slot, true
}

func (c *assetCache) loaded(key loadingAssetKey, bytes int64, cx *App) {
	element, ok := c.entries[key]
	if !ok {
		return
	}
	slot := element.Value.(*assetSlot)
	c.bytes += bytes - slot.bytes
	slot.bytes = bytes
	c.trim(slot, cx)
}

func (c *assetCache) trim(keep *assetSlot, cx *App) {
	for len(c.entries) > maxAssetCacheEntries || c.bytes > maxAssetCacheBytes {
		element := c.order.Back()
		if element == nil || element.Value.(*assetSlot) == keep {
			return
		}
		slot, _ := c.remove(element.Value.(*assetSlot).key)
		slot.load.evict(cx)
	}
}

func assetResultBytes(result any) int64 {
	switch result := result.(type) {
	case scheduler.Result[*RenderImage]:
		return renderImageBytes(result.Value)
	case *RenderImage:
		return renderImageBytes(result)
	case scheduler.Result[*Image]:
		if result.Value != nil {
			return int64(len(result.Value.Bytes))
		}
	case []byte:
		return int64(len(result))
	case string:
		return int64(len(result))
	}
	return 0
}

func renderImageBytes(image *RenderImage) int64 {
	if image == nil {
		return 0
	}
	var total int64
	for _, frame := range image.data {
		if frame.Buffer != nil {
			total += int64(len(frame.Buffer.Pix))
		}
	}
	return total
}

func assetKey[A Asset[S, O], S, O any](source S) loadingAssetKey {
	return loadingAssetKey{assetType: reflect.TypeFor[A](), hash: keyedHash(source)}
}

// ResourceKind tells which variant a Resource is.
type ResourceKind uint8

const (
	// ResourceKindURI is a resource at a given URI.
	ResourceKindURI ResourceKind = iota
	// ResourceKindPath is a resource at a given path in the file system.
	ResourceKindPath
	// ResourceKindEmbedded is a resource embedded in the application binary.
	ResourceKindEmbedded
)

// Resource is a resource an image or SVG is loaded from: a URI, a file system path or an asset
// embedded in the application binary. It is a comparable tagged struct.
type Resource struct {
	// Kind is the variant.
	Kind ResourceKind
	// URI is set for ResourceKindURI.
	URI SharedURI
	// Path is set for ResourceKindPath.
	Path string
	// Embedded is set for ResourceKindEmbedded.
	Embedded SharedString
}

// ResourceURI returns a resource at the given URI.
func ResourceURI(uri SharedURI) Resource {
	return Resource{Kind: ResourceKindURI, URI: uri}
}

// ResourcePath returns a resource at the given file system path.
func ResourcePath(path string) Resource {
	return Resource{Kind: ResourceKindPath, Path: path}
}

// ResourceEmbedded returns a resource embedded in the application binary.
func ResourceEmbedded(path SharedString) Resource {
	return Resource{Kind: ResourceKindEmbedded, Embedded: path}
}

// Hash feeds the resource to h as its derived Hash impl does: the variant index, then the
// variant's field.
func (r Resource) Hash(h *FxHasher) {
	h.WriteU64(uint64(r.Kind))
	switch r.Kind {
	case ResourceKindURI:
		h.WriteString(string(r.URI))
	case ResourceKindPath:
		h.WriteString(r.Path)
	case ResourceKindEmbedded:
		h.WriteString(string(r.Embedded))
	}
}

// Asset is implemented by a marker type A that loads an asset of type O from a source of type
// S. Load runs on the main thread and returns the body that loads the asset on a background
// thread.
type Asset[S, O any] interface {
	Load(source S, cx *App) func(s *scheduler.Scope) O
}

// AssetLogger is an asset loader that logs the error of a failed load of A.
type AssetLogger[A Asset[S, scheduler.Result[R]], S, R any] struct{}

// Load loads the asset with A and logs its error.
func (AssetLogger[A, S, R]) Load(source S, cx *App) func(s *scheduler.Scope) scheduler.Result[R] {
	var loader A
	load := loader.Load(source, cx)
	return func(s *scheduler.Scope) scheduler.Result[R] {
		result := load(s)
		if result.Err != nil {
			slog.Error(fmt.Sprintf("Failed to load asset: %v", result.Err))
		}
		return result
	}
}

// RemoveAsset removes an asset from GPUI's cache, canceling its load if it is still pending.
func (cx *App) RemoveAsset[A Asset[S, O], S, O any](source S) {
	if slot, ok := cx.loadingAssets.remove(assetKey[A](source)); ok {
		slot.load.release()
	}
}

// HasAsset reports whether an asset is present in GPUI's cache, loading or loaded, without
// fetching it.
func (cx *App) HasAsset[A Asset[S, O], S, O any](source S) bool {
	return cx.loadingAssets.has(assetKey[A](source))
}

// FetchAsset starts loading an uncached asset and returns its result once available. Pending
// loads and completed results are cached until RemoveAsset or until the least recently used
// entries are evicted to keep the cache bounded. It does not subscribe a view to completion
// notifications.
func (cx *App) FetchAsset[A Asset[S, O], S, O any](source S) (O, bool) {
	return assetEntry[A](cx, source).get()
}

func assetEntry[A Asset[S, O], S, O any](cx *App, source S) *cachedLoad[O] {
	key := assetKey[A](source)
	slot, ok := cx.loadingAssets.get(key)
	if !ok {
		var loader A
		future := loader.Load(source, cx)
		slot = &assetSlot{key: key}
		slot.load = newObservedCachedLoad(future, cx, func(result O, cx *App) {
			cx.loadingAssets.loaded(key, assetResultBytes(result), cx)
		})
		cx.loadingAssets.insert(slot, cx)
	}
	entry, ok := slot.load.(*cachedLoad[O])
	if !ok {
		panic("asset cache entries are keyed by their asset type")
	}
	return entry
}

// Hashable is implemented by values that feed themselves to an FxHasher, as a Rust Hash impl
// does. Hash uses it before its built-in cases.
type Hashable interface {
	Hash(h *FxHasher)
}

var hashSeed = maphash.MakeSeed()

// Hash returns a quick, non-cryptographically secure hash of data, as upstream's
// FxBuildHasher.hash_one. Integers, bools, strings, byte slices, SharedString and Hashable
// values hash as their Rust counterparts; other comparable values hash with hash/maphash.
func Hash[T any](data T) uint64 {
	var h FxHasher
	hashInto(&h, data)
	return h.Finish()
}

var assetKeySeed = maphash.MakeSeed()

func keyedHash[T any](data T) uint64 {
	var keyed maphash.Hash
	keyed.SetSeed(assetKeySeed)
	h := FxHasher{keyed: &keyed}
	hashInto(&h, data)
	return keyed.Sum64()
}

func hashInto[T any](h *FxHasher, data T) {
	switch v := any(data).(type) {
	case Hashable:
		v.Hash(h)
	case []byte:
		h.WriteUsize(uint64(len(v)))
		h.Write(v)
	case string:
		h.WriteString(v)
	case SharedString:
		h.WriteString(string(v))
	case bool:
		if v {
			h.WriteU8(1)
		} else {
			h.WriteU8(0)
		}
	case uint8:
		h.WriteU8(v)
	case uint16:
		h.WriteU64(uint64(v))
	case uint32:
		h.WriteU64(uint64(v))
	case uint64:
		h.WriteU64(v)
	case uint:
		h.WriteUsize(uint64(v))
	case int8:
		h.WriteU8(uint8(v))
	case int16:
		h.WriteU64(uint64(uint16(v)))
	case int32:
		h.WriteU64(uint64(uint32(v)))
	case int64:
		h.WriteU64(uint64(v))
	case int:
		h.WriteUsize(uint64(v))
	default:
		h.WriteU64(maphash.Comparable(hashSeed, any(data)))
	}
}

const (
	fxK                          uint64 = 0xf1357aea2e62a9c5
	fxSeed1                      uint64 = 0x243f6a8885a308d3
	fxSeed2                      uint64 = 0x13198a2e03707344
	fxPreventTrivialZeroCollapse uint64 = 0xa4093822299f31d0
)

// FxHasher is the 64-bit FxHasher of rustc-hash 2.1, upstream's collections::FxBuildHasher.
type FxHasher struct {
	hash  uint64
	keyed *maphash.Hash
}

func (h *FxHasher) addToHash(i uint64) {
	if h.keyed != nil {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], i)
		_, _ = h.keyed.Write(buf[:])
	}
	h.hash = (h.hash + i) * fxK
}

// Write feeds bytes to the hasher.
func (h *FxHasher) Write(bytes []byte) {
	if h.keyed != nil {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(len(bytes)))
		_, _ = h.keyed.Write(buf[:])
		_, _ = h.keyed.Write(bytes)
	}
	h.WriteU64(fxHashBytes(bytes))
}

// WriteU8 feeds a u8.
func (h *FxHasher) WriteU8(i uint8) {
	h.addToHash(uint64(i))
}

// WriteU64 feeds a u64 (also u16 and u32, which FxHasher widens the same way).
func (h *FxHasher) WriteU64(i uint64) {
	h.addToHash(i)
}

// WriteUsize feeds a usize.
func (h *FxHasher) WriteUsize(i uint64) {
	h.addToHash(i)
}

// WriteString feeds a str as Rust's Hash for str does: its bytes, then 0xff.
func (h *FxHasher) WriteString(s string) {
	h.Write([]byte(s))
	h.WriteU8(0xff)
}

// Finish returns the hash.
func (h *FxHasher) Finish() uint64 {
	return bits.RotateLeft64(h.hash, 26)
}

func fxMultiplyMix(x, y uint64) uint64 {
	hi, lo := bits.Mul64(x, y)
	return lo ^ hi
}

func fxHashBytes(bytes []byte) uint64 {
	length := len(bytes)
	s0 := fxSeed1
	s1 := fxSeed2
	if length <= 16 {
		switch {
		case length >= 8:
			s0 ^= binary.LittleEndian.Uint64(bytes[0:8])
			s1 ^= binary.LittleEndian.Uint64(bytes[length-8:])
		case length >= 4:
			s0 ^= uint64(binary.LittleEndian.Uint32(bytes[0:4]))
			s1 ^= uint64(binary.LittleEndian.Uint32(bytes[length-4:]))
		case length > 0:
			lo := bytes[0]
			mid := bytes[length/2]
			hi := bytes[length-1]
			s0 ^= uint64(lo)
			s1 ^= uint64(hi)<<8 | uint64(mid)
		}
	} else {
		bulk := bytes[:length-1]
		for len(bulk) >= 16 {
			x := binary.LittleEndian.Uint64(bulk[0:8])
			y := binary.LittleEndian.Uint64(bulk[8:16])
			t := fxMultiplyMix(s0^x, fxPreventTrivialZeroCollapse^y)
			s0 = s1
			s1 = t
			bulk = bulk[16:]
		}
		suffix := bytes[length-16:]
		s0 ^= binary.LittleEndian.Uint64(suffix[0:8])
		s1 ^= binary.LittleEndian.Uint64(suffix[8:16])
	}
	return fxMultiplyMix(s0, s1) ^ uint64(length)
}
