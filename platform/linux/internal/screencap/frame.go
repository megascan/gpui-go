//go:build linux

package screencap

import gpui "github.com/megascan/gpui-go"

// FrameData borrows a frame's pixels (scap::frame::FrameData): FrameDataNV12 or FrameDataBGR0.
type FrameData interface{ isFrameData() }

// FrameDataNV12 is FrameData::NV12.
type FrameDataNV12 struct{ Frame *gpui.ScapFrameYUV }

// FrameDataBGR0 is FrameData::BGR0.
type FrameDataBGR0 []byte

func (FrameDataNV12) isFrameData() {}
func (FrameDataBGR0) isFrameData() {}

// RemoveAlphaChannel drops every fourth byte of frameData.
func RemoveAlphaChannel(frameData []byte) []byte {
	width := len(frameData)
	widthWithoutAlpha := (width / 4) * 3
	data := make([]byte, widthWithoutAlpha)
	for i := 0; i+4 <= len(frameData) && (i/4)*3+3 <= len(data); i += 4 {
		dst := data[(i/4)*3:]
		dst[0] = frameData[i]
		dst[1] = frameData[i+1]
		dst[2] = frameData[i+2]
	}
	return data
}

// ConvertBGRAToRGB converts packed BGRA pixels to packed RGB.
func ConvertBGRAToRGB(frameData []byte) []byte {
	width := len(frameData)
	widthWithoutAlpha := (width / 4) * 3
	data := make([]byte, widthWithoutAlpha)
	for i := 0; i+4 <= len(frameData) && (i/4)*3+3 <= len(data); i += 4 {
		dst := data[(i/4)*3:]
		dst[0] = frameData[i+2]
		dst[1] = frameData[i+1]
		dst[2] = frameData[i]
	}
	return data
}

// GetCroppedData keeps the first width pixels of each row of a curWidth-wide 4-byte image; data
// of any other length is returned unchanged.
func GetCroppedData(data []byte, curWidth, height, width int32) []byte {
	if int32(len(data)) != height*curWidth*4 {
		return data
	}
	n := 4 * height * width
	if n < 0 {
		panic("called `Result::unwrap()` on an `Err` value: TryFromIntError(())")
	}
	cropped := make([]byte, n)
	idx := 0
	for i, item := range data {
		x := int32(i) % (curWidth * 4)
		if x < width*4 {
			cropped[idx] = item
			idx++
		}
	}
	return cropped
}
