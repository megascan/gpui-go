package taffy

import (
	"math"
	"sync/atomic"

	"github.com/megascan/gpui-go/gpuiutil"
)

const cacheSize = 9

const (
	infinityBits    uint32 = 0b_0_11111111_00000000000000000000000
	negInfinityBits uint32 = 0b_1_11111111_00000000000000000000000
)

const (
	signBit1         uint64 = 1 << 63
	signBit2         uint64 = 1 << 31
	bothSignBitsMask        = signBit1 | signBit2
	nonSignBitsMask         = ^bothSignBitsMask
	xAxisValueMask   uint64 = uint64(math.MaxUint32) << 32
)

func optionCacheKey(input Option[float32]) uint32 {
	if value, ok := input.Get(); ok {
		return math.Float32bits(value)
	}
	return infinityBits
}

func sizeOptionCacheKey(input Size[Option[float32]]) uint64 {
	return uint64(optionCacheKey(input.Width))<<32 | uint64(optionCacheKey(input.Height))
}

func availableSpaceCacheKey(input AvailableSpace) uint32 {
	switch input.kind {
	case AvailableSpaceKindDefinite:
		return math.Float32bits(-input.value)
	case AvailableSpaceKindMinContent:
		return negInfinityBits
	default:
		return infinityBits
	}
}

func sizeAvailableSpaceCacheKey(input Size[AvailableSpace]) uint64 {
	return uint64(availableSpaceCacheKey(input.Width))<<32 | uint64(availableSpaceCacheKey(input.Height))
}

func mixedCacheKey(kd Option[float32], avs AvailableSpace) uint32 {
	if kd, ok := kd.Get(); ok {
		return math.Float32bits(kd)
	}
	return availableSpaceCacheKey(avs)
}

func sizeMixedCacheKey(kd Size[Option[float32]], avs Size[AvailableSpace]) uint64 {
	return uint64(mixedCacheKey(kd.Width, avs.Width))<<32 | uint64(mixedCacheKey(kd.Height, avs.Height))
}

type cacheKey struct {
	kdAvailableSpace           uint64
	parentSize                 uint64
	knownDimensionsAreDefinite Size[bool]
}

func (k cacheKey) parentSizeValue() uint64 {
	return k.parentSize & nonSignBitsMask
}

func (k cacheKey) xAxisParentSize() uint64 {
	return k.parentSize & (xAxisValueMask & nonSignBitsMask)
}

func (k cacheKey) requestedAxisBits() uint64 {
	return k.parentSize & bothSignBitsMask
}

func (k cacheKey) sizeIsValidFor(other *cacheKey) bool {
	entryAxis := k.requestedAxisBits()
	return entryAxis == bothSignBitsMask || entryAxis == other.requestedAxisBits()
}

func cacheKeyFromLayoutInput(input *LayoutInput) cacheKey {
	var extraBits uint64
	switch input.Axis {
	case RequestedAxisHorizontal:
		extraBits = signBit1
	case RequestedAxisVertical:
		extraBits = signBit2
	default:
		extraBits = signBit1 | signBit2
	}
	return cacheKey{
		kdAvailableSpace: sizeMixedCacheKey(input.KnownDimensions, input.AvailableSpace),
		parentSize:       (sizeOptionCacheKey(input.ParentSize) & nonSignBitsMask) | extraBits,
		knownDimensionsAreDefinite: input.KnownDimensionsAreDefinite.ZipMap(input.KnownDimensions, func(isDefinite bool, kd Option[float32]) bool {
			return isDefinite || kd.IsNone()
		}),
	}
}

type cacheMode uint8

const (
	cacheModeInherentSizeBit           uint8 = 0b001
	cacheModeCollapsibleStartMarginBit uint8 = 0b010
	cacheModeCollapsibleEndMarginBit   uint8 = 0b100
)

const cacheModeInitial cacheMode = 0

func boolToU8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func cacheModeFromLayoutInput(input *LayoutInput) cacheMode {
	inherentSize := boolToU8(input.SizingMode == SizingModeInherentSize)
	collapsible := input.VerticalMarginsAreCollapsible
	return cacheMode((inherentSize * cacheModeInherentSizeBit) |
		(boolToU8(collapsible.Start) * cacheModeCollapsibleStartMarginBit) |
		(boolToU8(collapsible.End) * cacheModeCollapsibleEndMarginBit))
}

var modeChangeEvictions atomic.Int64

// CacheModeChangeEvictions returns how many times a result was stored into a non-empty Cache whose entries had a
// different sizing mode or margin collapsibility, dropping them. Counted only when gpuiutil.DebugAssertions is set.
func CacheModeChangeEvictions() int {
	return int(modeChangeEvictions.Load())
}

type cacheEntry[T any] struct {
	key     cacheKey
	content T
}

// Cache caches the results of sizing a grid or flexbox item. The zero value is an empty cache.
type Cache struct {
	finalLayoutEntry    Option[cacheEntry[LayoutOutput]]
	measureEntries      [cacheSize]Option[cacheEntry[Size[float32]]]
	recentlyUsedEntries uint16
	nextMeasureEntry    uint8
	nonEmpty            bool
	mode                cacheMode
}

// NewCache creates a new empty cache (also upstream's Default).
func NewCache() Cache {
	return Cache{
		recentlyUsedEntries: 0,
		nextMeasureEntry:    0,
		nonEmpty:            false,
		mode:                cacheModeInitial,
	}
}

// Get tries to retrieve a cached result.
func (c *Cache) Get(input *LayoutInput) Option[LayoutOutput] {
	if c.mode != cacheModeFromLayoutInput(input) {
		return none[LayoutOutput]()
	}
	key := cacheKeyFromLayoutInput(input)
	switch input.RunMode {
	case RunModePerformLayout:
		if entry, ok := c.finalLayoutEntry.Get(); ok && entry.key == key {
			return some(entry.content.Clone())
		}
		return none[LayoutOutput]()
	case RunModeComputeSize:
		for index := range c.measureEntries {
			slot := &c.measureEntries[index]
			entry, ok := slot.Get()
			if !ok {
				continue
			}
			if entry.key.kdAvailableSpace == key.kdAvailableSpace &&
				entry.key.knownDimensionsAreDefinite == key.knownDimensionsAreDefinite &&
				entry.key.xAxisParentSize() == key.xAxisParentSize() &&
				entry.key.sizeIsValidFor(&key) {
				c.recentlyUsedEntries |= 1 << index
				return some(LayoutOutputFromOuterSize(entry.content))
			}
		}
		return none[LayoutOutput]()
	default:
		return none[LayoutOutput]()
	}
}

// Store stores a computed size in the cache.
func (c *Cache) Store(input *LayoutInput, layoutOutput LayoutOutput) {
	if input.RunMode == RunModePerformHiddenLayout {
		return
	}
	mode := cacheModeFromLayoutInput(input)
	if c.mode != mode {
		if gpuiutil.DebugAssertions && c.nonEmpty {
			modeChangeEvictions.Add(1)
		}
		c.Clear()
		c.mode = mode
	}
	key := cacheKeyFromLayoutInput(input)
	switch input.RunMode {
	case RunModePerformLayout:
		c.nonEmpty = true
		c.finalLayoutEntry = some(cacheEntry[LayoutOutput]{key: key, content: layoutOutput})
	case RunModeComputeSize:
		if layoutOutput.MarginsCanCollapseThrough ||
			layoutOutput.TopMargin != CollapsibleMarginSetZero ||
			layoutOutput.BottomMargin != CollapsibleMarginSetZero {
			return
		}
		c.nonEmpty = true
		for index := range c.measureEntries {
			if entry, ok := c.measureEntries[index].Get(); ok && entry.key == key {
				c.measureEntries[index].AsMut().content = layoutOutput.Size
				c.recentlyUsedEntries |= 1 << index
				return
			}
		}
		for c.recentlyUsedEntries&(1<<c.nextMeasureEntry) != 0 {
			c.recentlyUsedEntries &^= 1 << c.nextMeasureEntry
			c.nextMeasureEntry++
			if c.nextMeasureEntry == cacheSize {
				c.nextMeasureEntry = 0
			}
		}
		entryIndex := int(c.nextMeasureEntry)
		c.measureEntries[entryIndex] = some(cacheEntry[Size[float32]]{key: key, content: layoutOutput.Size})
		c.recentlyUsedEntries |= 1 << entryIndex
		c.nextMeasureEntry++
		if c.nextMeasureEntry == cacheSize {
			c.nextMeasureEntry = 0
		}
	}
}

// Clear clears all cache entries and reports the outcome.
func (c *Cache) Clear() ClearState {
	if !c.nonEmpty {
		return ClearStateAlreadyEmpty
	}
	c.nonEmpty = false
	c.finalLayoutEntry = none[cacheEntry[LayoutOutput]]()
	c.measureEntries = [cacheSize]Option[cacheEntry[Size[float32]]]{}
	c.recentlyUsedEntries = 0
	c.nextMeasureEntry = 0
	return ClearStateCleared
}

// IsEmpty reports whether all cache entries are None.
func (c *Cache) IsEmpty() bool {
	if c.finalLayoutEntry.IsSome() {
		return false
	}
	for index := range c.measureEntries {
		entry := &c.measureEntries[index]
		if entry.IsSome() {
			return false
		}
	}
	return true
}

// Clone returns a deep copy of the cache (upstream's Clone).
func (c *Cache) Clone() Cache {
	cloned := *c
	if entry, ok := c.finalLayoutEntry.Get(); ok {
		cloned.finalLayoutEntry = some(cacheEntry[LayoutOutput]{key: entry.key, content: entry.content.Clone()})
	}
	return cloned
}

// ClearState is the outcome of Cache.Clear.
type ClearState uint8

const (
	ClearStateCleared ClearState = iota
	ClearStateAlreadyEmpty
)
