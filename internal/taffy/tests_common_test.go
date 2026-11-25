package taffy

import (
	"math"
	"strings"
	"unsafe"
)

func newTestTree() *TaffyTree[TestNodeContext] {
	return NewTaffyTree[TestNodeContext]()
}

// TestNodeContext is the node context shared by the integration tests.
type TestNodeContext struct {
	Count       int
	MeasureData TestMeasureData
}

func newTestNodeContext(measureData TestMeasureData) TestNodeContext {
	return TestNodeContext{Count: 0, MeasureData: measureData}
}

func testNodeContextZero() TestNodeContext {
	return newTestNodeContext(TestMeasureData{kind: TestMeasureDataKindZero})
}

func testNodeContextFixed(width, height float32) TestNodeContext {
	return newTestNodeContext(TestMeasureData{kind: TestMeasureDataKindFixed, fixed: Size[float32]{Width: width, Height: height}})
}

func testNodeContextAspectRatio(width, heightRatio float32) TestNodeContext {
	data := AspectRatioMeasureData{width: width, heightRatio: heightRatio}
	return newTestNodeContext(TestMeasureData{kind: TestMeasureDataKindAspectRatio, aspectRatio: data})
}

func testNodeContextAhemText(textContent string, writingMode WritingMode) TestNodeContext {
	data := AhemTextMeasureData{TextContent: textContent, WritingMode: writingMode}
	return newTestNodeContext(TestMeasureData{kind: TestMeasureDataKindAhemText, ahemText: data})
}

type TestMeasureDataKind uint8

const (
	TestMeasureDataKindZero TestMeasureDataKind = iota
	TestMeasureDataKindFixed
	TestMeasureDataKindAspectRatio
	TestMeasureDataKindAhemText
)

// TestMeasureData is the measurement data for a test node.
type TestMeasureData struct {
	kind        TestMeasureDataKind
	fixed       Size[float32]
	aspectRatio AspectRatioMeasureData
	ahemText    AhemTextMeasureData
}

func testMeasureFunction(inputs LayoutInput, _ NodeID, context *TestNodeContext, style *Style[defaultCheapStr]) LayoutOutput {
	return ComputeLeafLayout(
		inputs,
		style.Ref(),
		func(unsafe.Pointer, float32) float32 { return 0.0 },
		func(knownDimensions Size[Option[float32]], availableSpace Size[AvailableSpace]) Size[float32] {
			if w, ok := knownDimensions.Width.Get(); ok {
				if h, ok := knownDimensions.Height.Get(); ok {
					return Size[float32]{Width: w, Height: h}
				}
			}
			if context == nil {
				return SizeOptionUnwrapOr(knownDimensions, SizeF32Zero)
			}
			context.Count++
			var computeSize Size[float32]
			switch context.MeasureData.kind {
			case TestMeasureDataKindZero:
				computeSize = SizeF32Zero
			case TestMeasureDataKindFixed:
				computeSize = context.MeasureData.fixed
			case TestMeasureDataKindAspectRatio:
				computeSize = context.MeasureData.aspectRatio.measure(knownDimensions)
			case TestMeasureDataKindAhemText:
				computeSize = context.MeasureData.ahemText.measure(knownDimensions, availableSpace)
			}
			return Size[float32]{
				Width:  knownDimensions.Width.UnwrapOr(computeSize.Width),
				Height: knownDimensions.Height.UnwrapOr(computeSize.Height),
			}
		},
	)
}

// AspectRatioMeasureData measures a node from an intrinsic aspect ratio.
type AspectRatioMeasureData struct {
	width       float32
	heightRatio float32
}

func (d AspectRatioMeasureData) measure(knownDimensions Size[Option[float32]]) Size[float32] {
	width := knownDimensions.Width.UnwrapOr(d.width)
	height, ok := knownDimensions.Height.Get()
	if !ok {
		height = float32(width * d.heightRatio)
	}
	return Size[float32]{Width: width, Height: height}
}

// WritingMode is whether text is horizontal or vertical.
type WritingMode uint8

const (
	WritingModeHorizontal WritingMode = iota
	WritingModeVertical
)

func WritingModeFromStr(s string) (WritingMode, bool) {
	if strings.Contains(s, "vertical") {
		return WritingModeVertical, true
	} else if strings.Contains(s, "horizontal") {
		return WritingModeHorizontal, true
	}
	return 0, false
}

// AhemTextMeasureData measures text set in the Ahem test font.
type AhemTextMeasureData struct {
	TextContent string
	WritingMode WritingMode
}

func (d AhemTextMeasureData) measure(knownDimensions Size[Option[float32]], availableSpace Size[AvailableSpace]) Size[float32] {
	const zws = "​"
	const hWidth float32 = 10.0
	const hHeight float32 = 10.0

	inlineAxis := AbsoluteAxisHorizontal
	if d.WritingMode == WritingModeVertical {
		inlineAxis = AbsoluteAxisVertical
	}
	blockAxis := inlineAxis.OtherAxis()
	lines := strings.Split(d.TextContent, zws)

	if len(lines) == 0 {
		return SizeF32Zero
	}

	minLineLength := 0
	maxLineLength := 0
	for _, line := range lines {
		minLineLength = max(minLineLength, len(line))
		maxLineLength += len(line)
	}
	inlineSize, ok := knownDimensions.GetAbs(inlineAxis).Get()
	if !ok {
		space := availableSpace.GetAbs(inlineAxis)
		switch space.Kind() {
		case AvailableSpaceKindMinContent:
			inlineSize = float32(float32(minLineLength) * hWidth)
		case AvailableSpaceKindMaxContent:
			inlineSize = float32(float32(maxLineLength) * hWidth)
		default:
			inlineSize = f32Min(space.Unwrap(), float32(float32(maxLineLength)*hWidth))
		}
	}
	inlineSize = f32Max(inlineSize, float32(float32(minLineLength)*hWidth))
	blockSize, ok := knownDimensions.GetAbs(blockAxis).Get()
	if !ok {
		inlineLineLength := f32ToUsize(floor(inlineSize / hWidth))
		lineCount := 1
		currentLineLength := 0
		for _, line := range lines {
			if currentLineLength+len(line) > inlineLineLength {
				if currentLineLength > 0 {
					lineCount++
				}
				currentLineLength = len(line)
			} else {
				currentLineLength += len(line)
			}
		}
		blockSize = float32(float32(lineCount) * hHeight)
	}

	if d.WritingMode == WritingModeVertical {
		return Size[float32]{Width: blockSize, Height: inlineSize}
	}
	return Size[float32]{Width: inlineSize, Height: blockSize}
}

func f32ToUsize(v float32) int {
	if !(v > 0) {
		return 0
	}
	if v >= float32(math.MaxInt) {
		return math.MaxInt
	}
	return int(v)
}
