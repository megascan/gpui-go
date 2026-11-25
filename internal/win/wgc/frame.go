//go:build windows

package wgc

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/megascan/gpui-go/internal/win/d3d"
)

// Frame errors (frame::Error).
var (
	ErrFrameInvalidSize         = errors.New("Invalid crop size")
	ErrFrameInvalidTitleBarSize = errors.New("Invalid title bar height")
)

// FrameError is frame::Error::WindowsError.
type FrameError struct{ Err error }

func (e *FrameError) Error() string { return fmt.Sprintf("Windows API error: %v", e.Err) }

func (e *FrameError) Unwrap() error { return e.Err }

// Frame is a captured frame, valid during the FrameArrived callback.
type Frame struct {
	d3dDevice      *d3d.ID3D11Device
	frameSurface   *IDirect3DSurface
	frameTexture   *d3d.ID3D11Texture2D
	timestamp      TimeSpan
	context        *d3d.ID3D11DeviceContext
	buffer         *[]byte
	width          uint32
	height         uint32
	colorFormat    ColorFormat
	titleBarHeight *uint32
	staging        []*d3d.ID3D11Texture2D
}

// NewFrame creates a Frame; it borrows every argument.
func NewFrame(d3dDevice *d3d.ID3D11Device, frameSurface *IDirect3DSurface, frameTexture *d3d.ID3D11Texture2D, timestamp TimeSpan, context *d3d.ID3D11DeviceContext, buffer *[]byte, width, height uint32, colorFormat ColorFormat, titleBarHeight *uint32) *Frame {
	return &Frame{
		d3dDevice:      d3dDevice,
		frameSurface:   frameSurface,
		frameTexture:   frameTexture,
		timestamp:      timestamp,
		context:        context,
		buffer:         buffer,
		width:          width,
		height:         height,
		colorFormat:    colorFormat,
		titleBarHeight: titleBarHeight,
	}
}

// Width returns the frame width.
func (f *Frame) Width() uint32 { return f.width }

// Height returns the frame height.
func (f *Frame) Height() uint32 { return f.height }

// Timestamp returns the frame's system-relative time.
func (f *Frame) Timestamp() TimeSpan { return f.timestamp }

// ColorFormat returns the frame's color format.
func (f *Frame) ColorFormat() ColorFormat { return f.colorFormat }

// AsRawSurface returns the frame's surface.
func (f *Frame) AsRawSurface() *IDirect3DSurface { return f.frameSurface }

// AsRawTexture returns the frame's texture.
func (f *Frame) AsRawTexture() *d3d.ID3D11Texture2D { return f.frameTexture }

func (f *Frame) stagingTexture(width, height uint32) (*d3d.ID3D11Texture2D, error) {
	desc := d3d.D3D11_TEXTURE2D_DESC{
		Width:          width,
		Height:         height,
		MipLevels:      1,
		ArraySize:      1,
		Format:         uint32(f.colorFormat),
		SampleDesc:     d3d.DXGI_SAMPLE_DESC{Count: 1, Quality: 0},
		Usage:          d3d.D3D11_USAGE_STAGING,
		BindFlags:      0,
		CPUAccessFlags: d3d.D3D11_CPU_ACCESS_READ | d3d.D3D11_CPU_ACCESS_WRITE,
		MiscFlags:      0,
	}
	texture, err := f.d3dDevice.CreateTexture2D(&desc, nil)
	if err != nil {
		return nil, &FrameError{Err: err}
	}
	return texture, nil
}

func (f *Frame) mapped(texture *d3d.ID3D11Texture2D, width, height uint32) (*FrameBuffer, error) {
	mapped, err := f.context.Map(texture.Resource(), 0, d3d.D3D11_MAP_READ_WRITE, 0)
	if err != nil {
		texture.Release()
		return nil, &FrameError{Err: err}
	}
	f.staging = append(f.staging, texture)
	raw := unsafe.Slice((*byte)(mapped.PData), int(height*mapped.RowPitch))
	return NewFrameBuffer(raw, f.buffer, width, height, mapped.RowPitch, mapped.DepthPitch, f.colorFormat), nil
}

// Buffer copies the frame to a CPU-readable staging texture and maps it.
func (f *Frame) Buffer() (*FrameBuffer, error) {
	texture, err := f.stagingTexture(f.width, f.height)
	if err != nil {
		return nil, err
	}
	f.context.CopyResource(texture.Resource(), f.frameTexture.Resource())
	return f.mapped(texture, f.width, f.height)
}

// BufferCrop copies the region [start, end) of the frame to a staging texture and maps it; a
// region that is empty or extends past the frame fails with ErrFrameInvalidSize.
func (f *Frame) BufferCrop(startWidth, startHeight, endWidth, endHeight uint32) (*FrameBuffer, error) {
	if startWidth >= endWidth || startHeight >= endHeight || endWidth > f.width || endHeight > f.height {
		return nil, ErrFrameInvalidSize
	}
	textureWidth := endWidth - startWidth
	textureHeight := endHeight - startHeight
	texture, err := f.stagingTexture(textureWidth, textureHeight)
	if err != nil {
		return nil, err
	}
	resourceBox := d3d.D3D11_BOX{
		Left:   startWidth,
		Top:    startHeight,
		Front:  0,
		Right:  endWidth,
		Bottom: endHeight,
		Back:   1,
	}
	f.context.CopySubresourceRegion(texture.Resource(), 0, 0, 0, 0, f.frameTexture.Resource(), 0, &resourceBox)
	return f.mapped(texture, textureWidth, textureHeight)
}

// BufferWithoutTitleBar is Buffer cropped below the window's title bar.
func (f *Frame) BufferWithoutTitleBar() (*FrameBuffer, error) {
	if f.titleBarHeight != nil {
		titleBarHeight := *f.titleBarHeight
		if titleBarHeight >= f.height {
			return nil, ErrFrameInvalidTitleBarSize
		}
		return f.BufferCrop(0, titleBarHeight, f.width, f.height)
	}
	return f.Buffer()
}

// Release unmaps and drops the staging textures the frame's buffers were mapped from.
func (f *Frame) Release() {
	for _, t := range f.staging {
		f.context.Unmap(t.Resource(), 0)
		t.Release()
	}
	f.staging = nil
}

// FrameBuffer is a mapped frame.
type FrameBuffer struct {
	rawBuffer   []byte
	buffer      *[]byte
	width       uint32
	height      uint32
	rowPitch    uint32
	depthPitch  uint32
	colorFormat ColorFormat
}

// NewFrameBuffer creates a FrameBuffer.
func NewFrameBuffer(rawBuffer []byte, buffer *[]byte, width, height, rowPitch, depthPitch uint32, colorFormat ColorFormat) *FrameBuffer {
	return &FrameBuffer{rawBuffer: rawBuffer, buffer: buffer, width: width, height: height, rowPitch: rowPitch, depthPitch: depthPitch, colorFormat: colorFormat}
}

// Width returns the buffer width.
func (b *FrameBuffer) Width() uint32 { return b.width }

// Height returns the buffer height.
func (b *FrameBuffer) Height() uint32 { return b.height }

// RowPitch returns the row pitch in bytes.
func (b *FrameBuffer) RowPitch() uint32 { return b.rowPitch }

// DepthPitch returns the depth pitch in bytes.
func (b *FrameBuffer) DepthPitch() uint32 { return b.depthPitch }

// HasPadding reports whether rows are padded.
func (b *FrameBuffer) HasPadding() bool { return b.width*4 != b.rowPitch }

// AsRawBuffer returns the mapped bytes, rows padded to RowPitch.
func (b *FrameBuffer) AsRawBuffer() []byte { return b.rawBuffer }

// AsNopaddingBuffer returns the pixels with rows packed to width times the pixel size.
func (b *FrameBuffer) AsNopaddingBuffer() ([]byte, error) {
	if !b.HasPadding() {
		return b.rawBuffer, nil
	}
	var multiplier uint32
	switch b.colorFormat {
	case ColorFormatRgba16F:
		multiplier = 8
	case ColorFormatRgba8, ColorFormatBgra8:
		multiplier = 4
	}
	frameSize := int(b.width * b.height * multiplier)
	if cap(*b.buffer) < frameSize {
		*b.buffer = append(*b.buffer, make([]byte, frameSize-len(*b.buffer))...)
	}
	widthSize := int(b.width * multiplier)
	out := (*b.buffer)[:frameSize]
	for y := 0; y < int(b.height); y++ {
		index := y * int(b.rowPitch)
		copy(out[y*widthSize:(y+1)*widthSize], b.rawBuffer[index:index+widthSize])
	}
	return out, nil
}
