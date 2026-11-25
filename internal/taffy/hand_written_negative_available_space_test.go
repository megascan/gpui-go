package taffy

import (
	"fmt"
	"testing"
)

func negativeAvailableSpaceLayoutWithOversizedMargins(display Display, flexDirection FlexDirection) (float32, []string) {
	taffy := newTestTree()
	ls := defaultStyle()
	ls.Margin = Rect[LengthPercentageAuto]{Left: hwLPA(30.0), Right: hwLPA(10.0), Top: LengthPercentageAutoZero, Bottom: LengthPercentageAutoZero}
	ls.Size = Size[Dimension]{Width: DimensionAuto(), Height: DimensionAuto()}
	leaf := hwUnwrap(taffy.NewLeafWithContext(ls, testNodeContextZero()))
	rs := defaultStyle()
	rs.Display = display
	rs.FlexDirection = flexDirection
	rs.Size = Size[Dimension]{Width: hwDim(20.0), Height: DimensionAuto()}
	rs.GridTemplateColumns = []GridTemplateComponent[defaultCheapStr]{GridTemplateComponentSingle[defaultCheapStr](Length[TrackSizingFunction](20.0))}
	root := hwUnwrap(taffy.NewWithChildren(rs, []NodeID{leaf}))

	var negativeInputs []string
	hwOK(taffy.ComputeLayoutWithMeasure(
		root,
		SizeMaxContent[AvailableSpace](),
		func(inputs LayoutInput, nodeID NodeID, context *TestNodeContext, style *Style[defaultCheapStr]) LayoutOutput {
			known := inputs.KnownDimensions
			avail := inputs.AvailableSpace
			isNegative := func(value Option[float32]) bool {
				v, ok := value.Get()
				return ok && v < 0.0
			}
			isNegativeAvail := func(value AvailableSpace) bool {
				v, ok := value.IntoOption().Get()
				return ok && v < 0.0
			}
			if isNegative(known.Width) || isNegative(known.Height) || isNegativeAvail(avail.Width) || isNegativeAvail(avail.Height) {
				negativeInputs = append(negativeInputs, fmt.Sprintf("known=%+v available=%+v", known, avail))
			}
			return testMeasureFunction(inputs, nodeID, context, style)
		},
	))

	return hwLayout(taffy, leaf).Size.Width, negativeInputs
}

func negativeAvailableSpaceAssertFlooredAtZero(t *testing.T, display Display, flexDirection FlexDirection) {
	t.Helper()
	leafWidth, negativeInputs := negativeAvailableSpaceLayoutWithOversizedMargins(display, flexDirection)
	if len(negativeInputs) != 0 {
		t.Fatalf("leaf received negative layout inputs: %v", negativeInputs)
	}
	hwAssertEq(t, leafWidth, 0.0)
}

func TestHandWrittenNegativeAvailableSpace_BlockChildMarginsExceedContainerWidth(t *testing.T) {
	negativeAvailableSpaceAssertFlooredAtZero(t, DisplayBlock, FlexDirectionRow)
}

func TestHandWrittenNegativeAvailableSpace_FlexColumnChildMarginsExceedContainerWidth(t *testing.T) {
	negativeAvailableSpaceAssertFlooredAtZero(t, DisplayFlex, FlexDirectionColumn)
}

func TestHandWrittenNegativeAvailableSpace_GridChildMarginsExceedContainerWidth(t *testing.T) {
	negativeAvailableSpaceAssertFlooredAtZero(t, DisplayGrid, FlexDirectionRow)
}
