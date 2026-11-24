package gotext

import "math"

func swashBlit(mask []byte, maskWidth, maskHeight uint32, x, y int32, color [4]byte, target []byte, targetWidth, targetHeight uint32) {
	if maskWidth == 0 || maskHeight == 0 || targetWidth == 0 || targetHeight == 0 {
		return
	}
	sourceWidth := int(maskWidth)
	sourceHeight := int(maskHeight)
	destWidth := int(targetWidth)
	destHeight := int(targetHeight)
	sourceX, sourceY := 0, 0
	if x < 0 {
		sourceX = int(-x)
	}
	if y < 0 {
		sourceY = int(-y)
	}
	if sourceX >= sourceWidth || sourceY >= sourceHeight {
		return
	}
	destX, destY := 0, 0
	if x >= 0 {
		destX = int(x)
	}
	if y >= 0 {
		destY = int(y)
	}
	if destX >= destWidth || destY >= destHeight {
		return
	}
	sourceEndX := min(sourceWidth, destWidth-destX+sourceX)
	sourceEndY := min(sourceHeight, destHeight-destY+sourceY)
	destPitch := int(targetWidth) * 4
	colorA := uint32(color[3])
	dy := destY
	for sy := sourceY; sy < sourceEndY; sy++ {
		srcRow := mask[sy*int(maskWidth):]
		dstRow := target[dy*destPitch:]
		dy++
		dx := destX * 4
		for sx := sourceX; sx < sourceEndX; sx++ {
			a := (uint32(srcRow[sx]) * colorA) >> 8
			if a >= 255 {
				copy(dstRow[dx:dx+3], color[:3])
				dstRow[dx+3] = 255
			} else if a != 0 {
				inverseA := 255 - a
				for i := 0; i < 3; i++ {
					d := uint32(dstRow[dx+i])
					dstRow[dx+i] = uint8((inverseA*d + a*uint32(color[i])) >> 8)
				}
				d := uint32(dstRow[dx+3])
				dstRow[dx+3] = uint8((inverseA*d + a*255) >> 8)
			}
			dx += 4
		}
	}
}

type swashFilter uint8

const (
	swashFilterNearest swashFilter = iota
	swashFilterBilinear
	swashFilterBicubic
	swashFilterMitchell
	swashFilterLanczos3
	swashFilterGaussian
)

func swashResize(img []byte, width, height, channels uint32, target []byte, targetWidth, targetHeight uint32, filter swashFilter, scratch *[]byte) bool {
	if targetWidth == 0 || targetHeight == 0 {
		return true
	}
	var tmp []byte
	if scratch == nil {
		scratch = &tmp
	}
	if len(img) < int(width*height*channels) {
		return false
	}
	if len(target) < int(targetWidth*targetHeight*channels) {
		return false
	}
	scratchSize := int(targetWidth * height * channels)
	*scratch = resizeBytes(*scratch, scratchSize)
	switch filter {
	case swashFilterNearest:
		return swashResample(img, width, height, channels, target, targetWidth, targetHeight, *scratch, 0, swashNearest)
	case swashFilterBilinear:
		return swashResample(img, width, height, channels, target, targetWidth, targetHeight, *scratch, 1, swashBilinear)
	case swashFilterBicubic:
		return swashResample(img, width, height, channels, target, targetWidth, targetHeight, *scratch, 2, swashBicubic)
	case swashFilterMitchell:
		return swashResample(img, width, height, channels, target, targetWidth, targetHeight, *scratch, 2, swashMitchell)
	case swashFilterLanczos3:
		return swashResample(img, width, height, channels, target, targetWidth, targetHeight, *scratch, 3, swashLanczos3)
	default:
		return swashResample(img, width, height, channels, target, targetWidth, targetHeight, *scratch, 3, func(x float32) float32 { return swashGaussian(x, 0.5) })
	}
}

func resizeBytes(b []byte, n int) []byte {
	if cap(b) < n {
		b = append(b[:cap(b)], make([]byte, n-cap(b))...)
	}
	old := len(b)
	b = b[:n]
	for i := old; i < n; i++ {
		b[i] = 0
	}
	return b
}

func swashResample(img []byte, width, height, channels uint32, target []byte, targetWidth, targetHeight uint32, scratch []byte, support float32, filter func(float32) float32) bool {
	tmpWidth := targetWidth
	tmpHeight := height
	const s = float32(1.) / 255.
	switch channels {
	case 1:
		swashSampleDir(
			func(x, y uint32) [4]float32 { return [4]float32{0, 0, 0, float32(img[y*width+x]) * s} },
			width, height, targetWidth, filter, support,
			func(x, y uint32, p *[4]float32) { scratch[y*tmpWidth+x] = satU8(p[3] * 255) },
		)
		swashSampleDir(
			func(y, x uint32) [4]float32 { return [4]float32{0, 0, 0, float32(scratch[y*tmpWidth+x]) * s} },
			tmpHeight, tmpWidth, targetHeight, filter, support,
			func(y, x uint32, p *[4]float32) { target[y*targetWidth+x] = satU8(p[3] * 255) },
		)
		return true
	case 4:
		swashSampleDir(
			func(x, y uint32) [4]float32 {
				row := y*width*channels + x*channels
				return [4]float32{float32(img[row]) * s, float32(img[row+1]) * s, float32(img[row+2]) * s, float32(img[row+3]) * s}
			},
			width, height, targetWidth, filter, support,
			func(x, y uint32, p *[4]float32) {
				row := y*targetWidth*channels + x*channels
				scratch[row] = satU8(p[0] * 255)
				scratch[row+1] = satU8(p[1] * 255)
				scratch[row+2] = satU8(p[2] * 255)
				scratch[row+3] = satU8(p[3] * 255)
			},
		)
		swashSampleDir(
			func(y, x uint32) [4]float32 {
				row := y*tmpWidth*channels + x*channels
				return [4]float32{float32(scratch[row]) * s, float32(scratch[row+1]) * s, float32(scratch[row+2]) * s, float32(scratch[row+3]) * s}
			},
			tmpHeight, tmpWidth, targetHeight, filter, support,
			func(y, x uint32, p *[4]float32) {
				row := y*targetWidth*channels + x*channels
				target[row] = satU8(p[0] * 255)
				target[row+1] = satU8(p[1] * 255)
				target[row+2] = satU8(p[2] * 255)
				target[row+3] = satU8(p[3] * 255)
			},
		)
		return true
	}
	return false
}

func swashSampleDir(input func(uint32, uint32) [4]float32, width, height, newWidth uint32, filter func(float32) float32, support float32, output func(uint32, uint32, *[4]float32)) {
	const maxWeights = 64
	var weights [maxWeights]float32
	ratio := float32(width) / float32(newWidth)
	sratio := max(ratio, 1)
	srcSupport := support * sratio
	isratio := 1 / sratio
	for outx := uint32(0); outx < newWidth; outx++ {
		inx := (float32(outx) + 0.5) * ratio
		left := int(math.Floor(float64(inx - srcSupport)))
		left = min(max(left, 0), int(width)-1)
		right := int(math.Ceil(float64(inx + srcSupport)))
		right = min(max(right, left+1), int(width))
		inx -= 0.5
		for right-left > maxWeights {
			right--
			left++
		}
		numWeights := 0
		var sum float32
		for i := left; i < right; i++ {
			w := filter((float32(i) - inx) * isratio)
			weights[numWeights] = w
			numWeights++
			sum += w
		}
		isum := 1 / sum
		ws := weights[:numWeights]
		for y := uint32(0); y < height; y++ {
			var accum [4]float32
			for i, w := range ws {
				p := input(uint32(left+i), y)
				a := p[3]
				accum[0] += p[0] * w * a
				accum[1] += p[1] * w * a
				accum[2] += p[2] * w * a
				accum[3] += p[3] * w
			}
			if accum[3] != 0 {
				a := 1 / accum[3]
				accum[0] *= a
				accum[1] *= a
				accum[2] *= a
				accum[3] *= isum
			}
			output(outx, y, &accum)
		}
	}
}

func swashSinc(t float32) float32 {
	a := t * math.Pi
	if t == 0 {
		return 1
	}
	return float32(math.Sin(float64(a))) / a
}

func swashLanczos3(x float32) float32 {
	if abs32(x) < 3 {
		return abs32(swashSinc(x) * swashSinc(x/3))
	}
	return 0
}

func swashBilinear(x float32) float32 {
	x = abs32(x)
	if x < 1 {
		return 1 - x
	}
	return 0
}

func swashBicubic(x float32) float32 {
	a := abs32(x)
	const b = float32(0)
	const c = float32(0.5)
	var k float32
	switch {
	case a < 1:
		k = (12-9*b-6*c)*a*a*a + (-18+12*b+6*c)*a*a + (6 - 2*b)
	case a < 2:
		k = (-b-6*c)*a*a*a + (6*b+30*c)*a*a + (-12*b-48*c)*a + (8*b + 24*c)
	}
	return abs32(k / 6)
}

func swashMitchell(x float32) float32 {
	x = abs32(x)
	if x < 1 {
		return abs32((16 + x*x*(21*x-36)) / 18)
	} else if x < 2 {
		return abs32((32 + x*(-60+x*(36-7*x))) / 18)
	}
	return 0
}

func swashNearest(float32) float32 { return 1 }

func swashGaussian(x, r float32) float32 {
	return 1 / (float32(math.Sqrt(2*math.Pi)) * r) * float32(math.Exp(float64(-x*x/(2*r*r))))
}

func abs32(x float32) float32 { return float32(math.Abs(float64(x))) }

func satU8(v float32) uint8 {
	switch {
	case v != v || v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return uint8(v)
}
