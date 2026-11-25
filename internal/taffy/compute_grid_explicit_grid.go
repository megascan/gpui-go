package taffy

import (
	"math"
	"slices"
	"unsafe"
)

type autoRepeatStrategy uint8

const (
	autoRepeatStrategyMaxRepetitionsThatDoNotOverflow autoRepeatStrategy = iota
	autoRepeatStrategyMinRepetitionsThatDoOverflow
)

func saturatingAddU32(a, b uint32) uint32 {
	if s := a + b; s >= a {
		return s
	}
	return math.MaxUint32
}

func f32AsU32(v float64) uint32 {
	switch {
	case math.IsNaN(v) || v <= 0:
		return 0
	case v >= math.MaxUint32:
		return math.MaxUint32
	}
	return uint32(v)
}

func trackDefiniteValue(sizingFunction TrackSizingFunction, parentSize Option[float32], calcResolver func(unsafe.Pointer, float32) float32) float32 {
	maxSize := sizingFunction.Max.DefiniteValue(parentSize, calcResolver)
	minSize := sizingFunction.Min.DefiniteValue(parentSize, calcResolver)
	if m, ok := maxSize.Get(); ok {
		return F32MaybeMax(m, minSize)
	}
	return minSize.Unwrap()
}

func computeExplicitGridSizeInAxis[S CheapCloneStr](style GridContainerStyle[S], autoFitContainerSize Option[float32], autoFitStrategy autoRepeatStrategy, resolveCalcValue func(unsafe.Pointer, float32) float32, axis AbsoluteAxis) (uint16, uint16) {
	var templateOpt Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]]
	switch axis {
	case AbsoluteAxisHorizontal:
		templateOpt = style.GridTemplateColumns()
	default:
		templateOpt = style.GridTemplateRows()
	}
	template, ok := templateOpt.Get()
	if !ok {
		return 0, 0
	}

	trackCount := len(template)
	if trackCount == 0 {
		return 0, 0
	}

	for _, trackDef := range template {
		if trackDef.Kind() == GenericGridTemplateComponentKindRepeat && trackDef.Repeat().TrackCount() == 0 {
			return 0, 0
		}
	}

	var nonAutoRepeatingTrackCount32 uint32
	for _, trackDef := range template {
		var additionalTracks uint32
		switch trackDef.Kind() {
		case GenericGridTemplateComponentKindSingle:
			additionalTracks = 1
		default:
			repeat := trackDef.Repeat()
			if c := repeat.Count(); c.Kind() == RepetitionCountKindCount {
				additionalTracks = uint32(c.Count()) * uint32(repeat.TrackCount())
			}
		}
		nonAutoRepeatingTrackCount32 = saturatingAddU32(nonAutoRepeatingTrackCount32, additionalTracks)
	}
	nonAutoRepeatingTrackCount := uint16(min(nonAutoRepeatingTrackCount32, uint32(maxGridTracks)))

	var autoRepetitionCount uint16
	allTrackDefsHaveFixedComponent := true
	for _, trackDef := range template {
		if trackDef.IsAutoRepetition() {
			autoRepetitionCount++
		}
	}
	for _, trackDef := range template {
		switch trackDef.Kind() {
		case GenericGridTemplateComponentKindSingle:
			if !TrackSizingFunctionHasFixedComponent(trackDef.Single()) {
				allTrackDefsHaveFixedComponent = false
			}
		default:
			for _, sizingFunction := range trackDef.Repeat().Tracks() {
				if !TrackSizingFunctionHasFixedComponent(sizingFunction) {
					allTrackDefsHaveFixedComponent = false
					break
				}
			}
		}
		if !allTrackDefsHaveFixedComponent {
			break
		}
	}

	templateIsValid := autoRepetitionCount == 0 || (autoRepetitionCount == 1 && allTrackDefsHaveFixedComponent)
	if !templateIsValid {
		return 0, 0
	}

	if autoRepetitionCount == 0 {
		return 0, nonAutoRepeatingTrackCount
	}

	var autoRepeatInsertionPoint uint32
	var repetitionDefinition GenericRepetition[S]
	found := false
	for _, def := range template {
		if def.Kind() == GenericGridTemplateComponentKindSingle {
			autoRepeatInsertionPoint = saturatingAddU32(autoRepeatInsertionPoint, 1)
			continue
		}
		repeat := def.Repeat()
		if c := repeat.Count(); c.Kind() == RepetitionCountKindCount {
			autoRepeatInsertionPoint = saturatingAddU32(autoRepeatInsertionPoint, uint32(c.Count())*uint32(repeat.TrackCount()))
			continue
		}
		repetitionDefinition = repeat
		found = true
		break
	}
	if !found {
		panic("called `Option::unwrap()` on a `None` value")
	}
	repetitionDefinitionTracks := repetitionDefinition.Tracks()
	repetitionTrackCount := uint16(min(len(repetitionDefinitionTracks), math.MaxUint16))

	var numRepetitions uint32
	if innerContainerSize, ok := autoFitContainerSize.Get(); !ok {
		numRepetitions = 1
	} else {
		parentSize := some(innerContainerSize)

		var nonRepeatingTrackUsedSpace float32
		for _, trackDef := range template {
			switch trackDef.Kind() {
			case GenericGridTemplateComponentKindSingle:
				nonRepeatingTrackUsedSpace += trackDefiniteValue(trackDef.Single(), parentSize, resolveCalcValue)
			default:
				repeat := trackDef.Repeat()
				if c := repeat.Count(); c.Kind() == RepetitionCountKindCount {
					var sum float32
					for _, sizingFunction := range repeat.Tracks() {
						sum += trackDefiniteValue(sizingFunction, parentSize, resolveCalcValue)
					}
					nonRepeatingTrackUsedSpace += sum * float32(c.Count())
				}
			}
		}
		gapSize := style.Gap().GetAbs(axis).ResolveOrZero(some(innerContainerSize), resolveCalcValue)

		var perRepetitionTrackUsedSpace float32
		for _, sizingFunction := range repetitionDefinitionTracks {
			perRepetitionTrackUsedSpace += trackDefiniteValue(sizingFunction, parentSize, resolveCalcValue)
		}

		gaps := uint32(nonAutoRepeatingTrackCount) + uint32(repetitionTrackCount)
		if gaps > 0 {
			gaps--
		}
		firstRepetitionAndNonRepeatingTracksUsedSpace := nonRepeatingTrackUsedSpace + perRepetitionTrackUsedSpace + float32(gaps)*gapSize

		if firstRepetitionAndNonRepeatingTracksUsedSpace > innerContainerSize {
			numRepetitions = 1
		} else {
			perRepetitionGapUsedSpace := float32(repetitionTrackCount) * gapSize
			perRepetitionUsedSpace := perRepetitionTrackUsedSpace + perRepetitionGapUsedSpace
			numRepetitionThatFit := (innerContainerSize - firstRepetitionAndNonRepeatingTracksUsedSpace) / perRepetitionUsedSpace
			switch autoFitStrategy {
			case autoRepeatStrategyMaxRepetitionsThatDoNotOverflow:
				numRepetitions = saturatingAddU32(f32AsU32(math.Floor(float64(numRepetitionThatFit))), 1)
			default:
				numRepetitions = saturatingAddU32(f32AsU32(math.Ceil(float64(numRepetitionThatFit))), 1)
			}
		}
	}

	var remainingTracks uint32
	if uint32(maxGridTracks) > autoRepeatInsertionPoint {
		remainingTracks = uint32(maxGridTracks) - autoRepeatInsertionPoint
	}
	var clampedRepetitions uint16
	if remainingTracks != 0 {
		maxRepetitions := (remainingTracks + uint32(repetitionTrackCount) - 1) / uint32(repetitionTrackCount)
		clampedRepetitions = uint16(min(max(numRepetitions, 1), maxRepetitions))
	}

	gridTemplateTrackCount := uint16(min(
		uint32(nonAutoRepeatingTrackCount)+(uint32(repetitionTrackCount)*uint32(clampedRepetitions)),
		uint32(maxGridTracks),
	))
	return clampedRepetitions, gridTemplateTrackCount
}

func initializeGridTracks[S CheapCloneStr](tracks *[]gridTrack, counts trackCounts, style GridContainerStyle[S], axis AbsoluteAxis, autoRepetitionCount uint16, trackHasItems func(int) bool) {
	var trackTemplate Option[[]GenericGridTemplateComponent[S, GenericRepetition[S]]]
	var autoTracks []TrackSizingFunction
	var gap LengthPercentage
	switch axis {
	case AbsoluteAxisHorizontal:
		trackTemplate = style.GridTemplateColumns()
		autoTracks = style.GridAutoColumns()
		gap = style.Gap().Width
	default:
		trackTemplate = style.GridTemplateRows()
		autoTracks = style.GridAutoRows()
		gap = style.Gap().Height
	}

	*tracks = slices.Grow((*tracks)[:0], counts.len()*2+1)
	*tracks = append(*tracks, gridTrackGutter(gap))

	autoTrackCount := len(autoTracks)
	trackAuto := TrackSizingFunction{Min: MinTrackSizingFunctionAuto(), Max: MaxTrackSizingFunctionAuto()}

	if counts.negativeImplicit > 0 {
		if autoTrackCount == 0 {
			createImplicitTracks(tracks, counts.negativeImplicit, func(int) TrackSizingFunction { return trackAuto }, gap)
		} else {
			offset := autoTrackCount - (int(counts.negativeImplicit) % autoTrackCount)
			createImplicitTracks(tracks, counts.negativeImplicit, func(i int) TrackSizingFunction { return autoTracks[(offset+i)%autoTrackCount] }, gap)
		}
	}

	currentTrackIndex := int(counts.negativeImplicit)
	explicitTrackLimit := int(counts.negativeImplicit + counts.explicit)

	if counts.explicit > 0 {
		if template, ok := trackTemplate.Get(); ok {
			for _, trackSizingFunction := range template {
				switch trackSizingFunction.Kind() {
				case GenericGridTemplateComponentKindSingle:
					sizingFunction := trackSizingFunction.Single()
					if currentTrackIndex < explicitTrackLimit {
						*tracks = append(*tracks, newGridTrack(TrackSizingFunctionMinSizingFunction(sizingFunction), TrackSizingFunctionMaxSizingFunction(sizingFunction)))
						*tracks = append(*tracks, gridTrackGutter(gap))
						currentTrackIndex++
					}
				default:
					repeat := trackSizingFunction.Repeat()
					repeatTracks := repeat.Tracks()
					count := repeat.Count()
					if count.Kind() == RepetitionCountKindCount {
						repeatedTrackCount := min(int(repeat.TrackCount())*int(count.Count()), explicitTrackLimit-currentTrackIndex)
						for i := range repeatedTrackCount {
							sizingFunction := repeatTracks[i%len(repeatTracks)]
							*tracks = append(*tracks, newGridTrack(TrackSizingFunctionMinSizingFunction(sizingFunction), TrackSizingFunctionMaxSizingFunction(sizingFunction)))
							*tracks = append(*tracks, gridTrackGutter(gap))
							currentTrackIndex++
						}
						continue
					}
					autoRepeatedTrackCount := min(int(repeat.TrackCount())*int(autoRepetitionCount), explicitTrackLimit-currentTrackIndex)
					for i := range autoRepeatedTrackCount {
						trackDef := repeatTracks[i%len(repeatTracks)]
						track := newGridTrack(TrackSizingFunctionMinSizingFunction(trackDef), TrackSizingFunctionMaxSizingFunction(trackDef))
						gutter := gridTrackGutter(gap)
						if count == RepetitionCountAutoFit && !trackHasItems(currentTrackIndex) {
							track.collapse()
							gutter.collapse()
						}
						*tracks = append(*tracks, track, gutter)
						currentTrackIndex++
					}

					isLast := currentTrackIndex == counts.len()
					if count == RepetitionCountAutoFit && isLast {
						for j := len(*tracks) - 1; j >= 0; j-- {
							previousTrack := &(*tracks)[j]
							if previousTrack.kind == gridTrackKindTrack && !previousTrack.isCollapsed {
								break
							}
							previousTrack.collapse()
						}
					}
				}
			}
		}
	}

	gridAreaTracks := (counts.negativeImplicit + counts.explicit) - uint16(currentTrackIndex)

	if autoTrackCount == 0 {
		createImplicitTracks(tracks, counts.positiveImplicit+gridAreaTracks, func(int) TrackSizingFunction { return trackAuto }, gap)
	} else {
		createImplicitTracks(tracks, counts.positiveImplicit+gridAreaTracks, func(i int) TrackSizingFunction { return autoTracks[i%autoTrackCount] }, gap)
	}

	(*tracks)[0].collapse()
	(*tracks)[len(*tracks)-1].collapse()
}

func createImplicitTracks(tracks *[]gridTrack, count uint16, autoTracksIter func(int) TrackSizingFunction, gap LengthPercentage) {
	for i := range int(count) {
		trackDef := autoTracksIter(i)
		*tracks = append(*tracks, newGridTrack(TrackSizingFunctionMinSizingFunction(trackDef), TrackSizingFunctionMaxSizingFunction(trackDef)))
		*tracks = append(*tracks, gridTrackGutter(gap))
	}
}
