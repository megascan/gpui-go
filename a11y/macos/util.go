//go:build darwin && !ios

package macos

import (
	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func fromNSRange(node consumer.Node, nsRange objc.NSRange) (consumer.TextRange, bool) {
	pos, ok := node.TextPositionFromGlobalUTF16Index(int(nsRange.Location))
	if !ok {
		return consumer.TextRange{}, false
	}
	r := pos.ToDegenerateRange()
	if nsRange.Length > 0 {
		end, ok := node.TextPositionFromGlobalUTF16Index(int(nsRange.Location + nsRange.Length))
		if !ok {
			return consumer.TextRange{}, false
		}
		r.SetEnd(end)
	}
	return r, true
}

func toNSRange(r consumer.TextRange) objc.NSRange {
	start := r.Start().ToGlobalUTF16Index()
	end := r.End().ToGlobalUTF16Index()
	return objc.MakeNSRange(objc.NSUInteger(start), objc.NSUInteger(end-start))
}

func toNSRangeForCharacter(pos consumer.TextPosition) objc.NSRange {
	r := pos.ToDegenerateRange()
	if !pos.IsDocumentEnd() {
		r.SetEnd(pos.ForwardToCharacterEnd())
	}
	return toNSRange(r)
}

func viewWindow(view objc.ID) objc.ID {
	window := objc.Send[objc.ID](view, objc.Sel("window"))
	if window == objc.Nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return window
}

func fromNSPoint(view objc.ID, node consumer.Node, point objc.CGPoint) a11y.Point {
	window := viewWindow(view)
	point = objc.Send[objc.CGPoint](window, objc.Sel("convertPointFromScreen:"), point)
	point = objc.Send[objc.CGPoint](view, objc.Sel("convertPoint:fromView:"), point, objc.Nil)
	factor := objc.Send[objc.CGFloat](window, objc.Sel("backingScaleFactor"))
	var y float64
	if objc.Send[bool](view, objc.Sel("isFlipped")) {
		y = point.Y * factor
	} else {
		viewBounds := objc.Send[objc.CGRect](view, objc.Sel("bounds"))
		y = (viewBounds.Size.Height - point.Y) * factor
	}
	p := a11y.NewPoint(point.X*factor, y)
	return node.Transform().Inverse().MulPoint(p)
}

func toNSRect(view objc.ID, rect a11y.Rect) objc.CGRect {
	window := viewWindow(view)
	factor := objc.Send[objc.CGFloat](window, objc.Sel("backingScaleFactor"))
	var y float64
	if objc.Send[bool](view, objc.Sel("isFlipped")) {
		y = rect.Y0 / factor
	} else {
		viewBounds := objc.Send[objc.CGRect](view, objc.Sel("bounds"))
		y = viewBounds.Size.Height - rect.Y1/factor
	}
	r := objc.MakeCGRect(rect.X0/factor, y, rect.Width()/factor, rect.Height()/factor)
	r = objc.Send[objc.CGRect](view, objc.Sel("convertRect:toView:"), r, objc.Nil)
	window = viewWindow(view)
	return objc.Send[objc.CGRect](window, objc.Sel("convertRectToScreen:"), r)
}

func colorChannelToF64(channel uint8) float64 {
	return float64(channel) / 255.0
}

func toColorAttribute(color a11y.Color) objc.ID {
	nsColor := objc.SendClass[objc.ID](objc.GetClass("NSColor"), objc.Sel("colorWithSRGBRed:green:blue:alpha:"),
		colorChannelToF64(color.Red),
		colorChannelToF64(color.Green),
		colorChannelToF64(color.Blue),
		colorChannelToF64(color.Alpha),
	)
	cgColor := objc.Send[objc.ID](nsColor, objc.Sel("CGColor"))
	return objc.Retain(cgColor)
}
