package taffy

// OptionF32MaybeMin returns the minimum of self and rhs; a None self stays None and a None rhs is ignored.
func OptionF32MaybeMin(self, rhs Option[float32]) Option[float32] {
	l, lok := self.Get()
	r, rok := rhs.Get()
	switch {
	case lok && rok:
		return some(f32Min(l, r))
	case lok:
		return self
	}
	return none[float32]()
}

// OptionF32MaybeMax returns the maximum of self and rhs; a None self stays None and a None rhs is ignored.
func OptionF32MaybeMax(self, rhs Option[float32]) Option[float32] {
	l, lok := self.Get()
	r, rok := rhs.Get()
	switch {
	case lok && rok:
		return some(f32Max(l, r))
	case lok:
		return self
	}
	return none[float32]()
}

// OptionF32MaybeClamp returns self clamped between min and max, ignoring a None bound.
func OptionF32MaybeClamp(self, min, max Option[float32]) Option[float32] {
	base, bok := self.Get()
	if !bok {
		return none[float32]()
	}
	lo, lok := min.Get()
	hi, hok := max.Get()
	switch {
	case lok && hok:
		return some(f32Max(f32Min(base, hi), lo))
	case hok:
		return some(f32Min(base, hi))
	case lok:
		return some(f32Max(base, lo))
	}
	return self
}

// OptionF32MaybeAdd adds self and rhs; a None self stays None and a None rhs is ignored.
func OptionF32MaybeAdd(self, rhs Option[float32]) Option[float32] {
	l, lok := self.Get()
	r, rok := rhs.Get()
	switch {
	case lok && rok:
		return some(l + r)
	case lok:
		return self
	}
	return none[float32]()
}

// OptionF32MaybeSub subtracts rhs from self; a None self stays None and a None rhs is ignored.
func OptionF32MaybeSub(self, rhs Option[float32]) Option[float32] {
	l, lok := self.Get()
	r, rok := rhs.Get()
	switch {
	case lok && rok:
		return some(l - r)
	case lok:
		return self
	}
	return none[float32]()
}

// OptionF32MaybeMinF32 returns the minimum of self and rhs, or None.
func OptionF32MaybeMinF32(self Option[float32], rhs float32) Option[float32] {
	if v, ok := self.Get(); ok {
		return some(f32Min(v, rhs))
	}
	return none[float32]()
}

// OptionF32MaybeMaxF32 returns the maximum of self and rhs, or None.
func OptionF32MaybeMaxF32(self Option[float32], rhs float32) Option[float32] {
	if v, ok := self.Get(); ok {
		return some(f32Max(v, rhs))
	}
	return none[float32]()
}

// OptionF32MaybeClampF32 returns self clamped between min and max, or None.
func OptionF32MaybeClampF32(self Option[float32], min, max float32) Option[float32] {
	if v, ok := self.Get(); ok {
		return some(f32Max(f32Min(v, max), min))
	}
	return none[float32]()
}

// OptionF32MaybeAddF32 adds rhs to self, or returns None.
func OptionF32MaybeAddF32(self Option[float32], rhs float32) Option[float32] {
	if v, ok := self.Get(); ok {
		return some(v + rhs)
	}
	return none[float32]()
}

// OptionF32MaybeSubF32 subtracts rhs from self, or returns None.
func OptionF32MaybeSubF32(self Option[float32], rhs float32) Option[float32] {
	if v, ok := self.Get(); ok {
		return some(v - rhs)
	}
	return none[float32]()
}

// F32MaybeMin returns the minimum of self and rhs, ignoring a None rhs.
func F32MaybeMin(self float32, rhs Option[float32]) float32 {
	if v, ok := rhs.Get(); ok {
		return f32Min(self, v)
	}
	return self
}

// F32MaybeMax returns the maximum of self and rhs, ignoring a None rhs.
func F32MaybeMax(self float32, rhs Option[float32]) float32 {
	if v, ok := rhs.Get(); ok {
		return f32Max(self, v)
	}
	return self
}

// F32MaybeClamp returns self clamped between min and max, ignoring a None bound.
func F32MaybeClamp(self float32, min, max Option[float32]) float32 {
	lo, lok := min.Get()
	hi, hok := max.Get()
	switch {
	case lok && hok:
		return f32Max(f32Min(self, hi), lo)
	case hok:
		return f32Min(self, hi)
	case lok:
		return f32Max(self, lo)
	}
	return self
}

// F32MaybeAdd adds rhs to self, ignoring a None rhs.
func F32MaybeAdd(self float32, rhs Option[float32]) float32 {
	if v, ok := rhs.Get(); ok {
		return self + v
	}
	return self
}

// F32MaybeSub subtracts rhs from self, ignoring a None rhs.
func F32MaybeSub(self float32, rhs Option[float32]) float32 {
	if v, ok := rhs.Get(); ok {
		return self - v
	}
	return self
}

// MaybeMinF32 returns Definite(min(value, rhs)); MinContent and MaxContent become Definite(rhs).
func (a AvailableSpace) MaybeMinF32(rhs float32) AvailableSpace {
	switch a.kind {
	case AvailableSpaceKindDefinite:
		return AvailableSpaceDefinite(f32Min(a.value, rhs))
	default:
		return AvailableSpaceDefinite(rhs)
	}
}

// MaybeMaxF32 returns Definite(max(value, rhs)); MinContent and MaxContent are unchanged.
func (a AvailableSpace) MaybeMaxF32(rhs float32) AvailableSpace {
	if a.kind == AvailableSpaceKindDefinite {
		return AvailableSpaceDefinite(f32Max(a.value, rhs))
	}
	return a
}

// MaybeClampF32 clamps a definite value between min and max; MinContent and MaxContent are unchanged.
func (a AvailableSpace) MaybeClampF32(min, max float32) AvailableSpace {
	if a.kind == AvailableSpaceKindDefinite {
		return AvailableSpaceDefinite(f32Max(f32Min(a.value, max), min))
	}
	return a
}

// MaybeAddF32 adds rhs to a definite value; MinContent and MaxContent are unchanged.
func (a AvailableSpace) MaybeAddF32(rhs float32) AvailableSpace {
	if a.kind == AvailableSpaceKindDefinite {
		return AvailableSpaceDefinite(a.value + rhs)
	}
	return a
}

// MaybeSubF32 subtracts rhs from a definite value; MinContent and MaxContent are unchanged.
func (a AvailableSpace) MaybeSubF32(rhs float32) AvailableSpace {
	if a.kind == AvailableSpaceKindDefinite {
		return AvailableSpaceDefinite(a.value - rhs)
	}
	return a
}

// MaybeMin returns the minimum with a Some rhs; MinContent and MaxContent become Definite(rhs) when rhs is Some.
func (a AvailableSpace) MaybeMin(rhs Option[float32]) AvailableSpace {
	r, rok := rhs.Get()
	switch a.kind {
	case AvailableSpaceKindDefinite:
		if rok {
			return AvailableSpaceDefinite(f32Min(a.value, r))
		}
		return AvailableSpaceDefinite(a.value)
	case AvailableSpaceKindMinContent:
		if rok {
			return AvailableSpaceDefinite(r)
		}
		return AvailableSpaceMinContent
	default:
		if rok {
			return AvailableSpaceDefinite(r)
		}
		return AvailableSpaceMaxContent
	}
}

// MaybeMax returns the maximum of a definite value and a Some rhs; MinContent and MaxContent are unchanged.
func (a AvailableSpace) MaybeMax(rhs Option[float32]) AvailableSpace {
	switch a.kind {
	case AvailableSpaceKindDefinite:
		if r, ok := rhs.Get(); ok {
			return AvailableSpaceDefinite(f32Max(a.value, r))
		}
		return AvailableSpaceDefinite(a.value)
	case AvailableSpaceKindMinContent:
		return AvailableSpaceMinContent
	default:
		return AvailableSpaceMaxContent
	}
}

// MaybeClamp clamps a definite value between the Some bounds; MinContent and MaxContent are unchanged.
func (a AvailableSpace) MaybeClamp(min, max Option[float32]) AvailableSpace {
	switch a.kind {
	case AvailableSpaceKindDefinite:
		lo, lok := min.Get()
		hi, hok := max.Get()
		switch {
		case lok && hok:
			return AvailableSpaceDefinite(f32Max(f32Min(a.value, hi), lo))
		case hok:
			return AvailableSpaceDefinite(f32Min(a.value, hi))
		case lok:
			return AvailableSpaceDefinite(f32Max(a.value, lo))
		}
		return AvailableSpaceDefinite(a.value)
	case AvailableSpaceKindMinContent:
		return AvailableSpaceMinContent
	default:
		return AvailableSpaceMaxContent
	}
}

// MaybeAdd adds a Some rhs to a definite value; MinContent and MaxContent are unchanged.
func (a AvailableSpace) MaybeAdd(rhs Option[float32]) AvailableSpace {
	switch a.kind {
	case AvailableSpaceKindDefinite:
		if r, ok := rhs.Get(); ok {
			return AvailableSpaceDefinite(a.value + r)
		}
		return AvailableSpaceDefinite(a.value)
	case AvailableSpaceKindMinContent:
		return AvailableSpaceMinContent
	default:
		return AvailableSpaceMaxContent
	}
}

// MaybeSub subtracts a Some rhs from a definite value; MinContent and MaxContent are unchanged.
func (a AvailableSpace) MaybeSub(rhs Option[float32]) AvailableSpace {
	switch a.kind {
	case AvailableSpaceKindDefinite:
		if r, ok := rhs.Get(); ok {
			return AvailableSpaceDefinite(a.value - r)
		}
		return AvailableSpaceDefinite(a.value)
	case AvailableSpaceKindMinContent:
		return AvailableSpaceMinContent
	default:
		return AvailableSpaceMaxContent
	}
}

// SizeOptionF32MaybeMin applies OptionF32MaybeMin to each component.
func SizeOptionF32MaybeMin(self, rhs Size[Option[float32]]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeMin(self.Width, rhs.Width), Height: OptionF32MaybeMin(self.Height, rhs.Height)}
}

// SizeOptionF32MaybeMinF32 applies OptionF32MaybeMinF32 to each component.
func SizeOptionF32MaybeMinF32(self Size[Option[float32]], rhs Size[float32]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeMinF32(self.Width, rhs.Width), Height: OptionF32MaybeMinF32(self.Height, rhs.Height)}
}

// SizeF32MaybeMin applies F32MaybeMin to each component.
func SizeF32MaybeMin(self Size[float32], rhs Size[Option[float32]]) Size[float32] {
	return Size[float32]{Width: F32MaybeMin(self.Width, rhs.Width), Height: F32MaybeMin(self.Height, rhs.Height)}
}

// SizeAvailableSpaceMaybeMinF32 applies AvailableSpace.MaybeMinF32 to each component.
func SizeAvailableSpaceMaybeMinF32(self Size[AvailableSpace], rhs Size[float32]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeMinF32(rhs.Width), Height: self.Height.MaybeMinF32(rhs.Height)}
}

// SizeAvailableSpaceMaybeMin applies AvailableSpace.MaybeMin to each component.
func SizeAvailableSpaceMaybeMin(self Size[AvailableSpace], rhs Size[Option[float32]]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeMin(rhs.Width), Height: self.Height.MaybeMin(rhs.Height)}
}

// SizeOptionF32MaybeMax applies OptionF32MaybeMax to each component.
func SizeOptionF32MaybeMax(self, rhs Size[Option[float32]]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeMax(self.Width, rhs.Width), Height: OptionF32MaybeMax(self.Height, rhs.Height)}
}

// SizeOptionF32MaybeMaxF32 applies OptionF32MaybeMaxF32 to each component.
func SizeOptionF32MaybeMaxF32(self Size[Option[float32]], rhs Size[float32]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeMaxF32(self.Width, rhs.Width), Height: OptionF32MaybeMaxF32(self.Height, rhs.Height)}
}

// SizeF32MaybeMax applies F32MaybeMax to each component.
func SizeF32MaybeMax(self Size[float32], rhs Size[Option[float32]]) Size[float32] {
	return Size[float32]{Width: F32MaybeMax(self.Width, rhs.Width), Height: F32MaybeMax(self.Height, rhs.Height)}
}

// SizeAvailableSpaceMaybeMaxF32 applies AvailableSpace.MaybeMaxF32 to each component.
func SizeAvailableSpaceMaybeMaxF32(self Size[AvailableSpace], rhs Size[float32]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeMaxF32(rhs.Width), Height: self.Height.MaybeMaxF32(rhs.Height)}
}

// SizeAvailableSpaceMaybeMax applies AvailableSpace.MaybeMax to each component.
func SizeAvailableSpaceMaybeMax(self Size[AvailableSpace], rhs Size[Option[float32]]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeMax(rhs.Width), Height: self.Height.MaybeMax(rhs.Height)}
}

// SizeOptionF32MaybeAdd applies OptionF32MaybeAdd to each component.
func SizeOptionF32MaybeAdd(self, rhs Size[Option[float32]]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeAdd(self.Width, rhs.Width), Height: OptionF32MaybeAdd(self.Height, rhs.Height)}
}

// SizeOptionF32MaybeAddF32 applies OptionF32MaybeAddF32 to each component.
func SizeOptionF32MaybeAddF32(self Size[Option[float32]], rhs Size[float32]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeAddF32(self.Width, rhs.Width), Height: OptionF32MaybeAddF32(self.Height, rhs.Height)}
}

// SizeF32MaybeAdd applies F32MaybeAdd to each component.
func SizeF32MaybeAdd(self Size[float32], rhs Size[Option[float32]]) Size[float32] {
	return Size[float32]{Width: F32MaybeAdd(self.Width, rhs.Width), Height: F32MaybeAdd(self.Height, rhs.Height)}
}

// SizeAvailableSpaceMaybeAddF32 applies AvailableSpace.MaybeAddF32 to each component.
func SizeAvailableSpaceMaybeAddF32(self Size[AvailableSpace], rhs Size[float32]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeAddF32(rhs.Width), Height: self.Height.MaybeAddF32(rhs.Height)}
}

// SizeAvailableSpaceMaybeAdd applies AvailableSpace.MaybeAdd to each component.
func SizeAvailableSpaceMaybeAdd(self Size[AvailableSpace], rhs Size[Option[float32]]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeAdd(rhs.Width), Height: self.Height.MaybeAdd(rhs.Height)}
}

// SizeOptionF32MaybeSub applies OptionF32MaybeSub to each component.
func SizeOptionF32MaybeSub(self, rhs Size[Option[float32]]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeSub(self.Width, rhs.Width), Height: OptionF32MaybeSub(self.Height, rhs.Height)}
}

// SizeOptionF32MaybeSubF32 applies OptionF32MaybeSubF32 to each component.
func SizeOptionF32MaybeSubF32(self Size[Option[float32]], rhs Size[float32]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeSubF32(self.Width, rhs.Width), Height: OptionF32MaybeSubF32(self.Height, rhs.Height)}
}

// SizeF32MaybeSub applies F32MaybeSub to each component.
func SizeF32MaybeSub(self Size[float32], rhs Size[Option[float32]]) Size[float32] {
	return Size[float32]{Width: F32MaybeSub(self.Width, rhs.Width), Height: F32MaybeSub(self.Height, rhs.Height)}
}

// SizeAvailableSpaceMaybeSubF32 applies AvailableSpace.MaybeSubF32 to each component.
func SizeAvailableSpaceMaybeSubF32(self Size[AvailableSpace], rhs Size[float32]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeSubF32(rhs.Width), Height: self.Height.MaybeSubF32(rhs.Height)}
}

// SizeAvailableSpaceMaybeSub applies AvailableSpace.MaybeSub to each component.
func SizeAvailableSpaceMaybeSub(self Size[AvailableSpace], rhs Size[Option[float32]]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeSub(rhs.Width), Height: self.Height.MaybeSub(rhs.Height)}
}

// SizeOptionF32MaybeClamp applies OptionF32MaybeClamp to each component.
func SizeOptionF32MaybeClamp(self, min, max Size[Option[float32]]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeClamp(self.Width, min.Width, max.Width), Height: OptionF32MaybeClamp(self.Height, min.Height, max.Height)}
}

// SizeOptionF32MaybeClampF32 applies OptionF32MaybeClampF32 to each component.
func SizeOptionF32MaybeClampF32(self Size[Option[float32]], min, max Size[float32]) Size[Option[float32]] {
	return Size[Option[float32]]{Width: OptionF32MaybeClampF32(self.Width, min.Width, max.Width), Height: OptionF32MaybeClampF32(self.Height, min.Height, max.Height)}
}

// SizeF32MaybeClamp applies F32MaybeClamp to each component.
func SizeF32MaybeClamp(self Size[float32], min, max Size[Option[float32]]) Size[float32] {
	return Size[float32]{Width: F32MaybeClamp(self.Width, min.Width, max.Width), Height: F32MaybeClamp(self.Height, min.Height, max.Height)}
}

// SizeAvailableSpaceMaybeClampF32 applies AvailableSpace.MaybeClampF32 to each component.
func SizeAvailableSpaceMaybeClampF32(self Size[AvailableSpace], min, max Size[float32]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeClampF32(min.Width, max.Width), Height: self.Height.MaybeClampF32(min.Height, max.Height)}
}

// SizeAvailableSpaceMaybeClamp applies AvailableSpace.MaybeClamp to each component.
func SizeAvailableSpaceMaybeClamp(self Size[AvailableSpace], min, max Size[Option[float32]]) Size[AvailableSpace] {
	return Size[AvailableSpace]{Width: self.Width.MaybeClamp(min.Width, max.Width), Height: self.Height.MaybeClamp(min.Height, max.Height)}
}
