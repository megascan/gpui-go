//go:build windows

package windows

import (
	"math"
	"sync"
	"weak"

	"github.com/megascan/gpui-go/a11y"
	"github.com/megascan/gpui-go/a11y/consumer"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
)

func upgradeRange(weak *consumer.WeakRange, treeState *consumer.TreeState) (consumer.Range, error) {
	if r, ok := weak.Upgrade(treeState); ok {
		return r, nil
	}
	return consumer.Range{}, elementNotAvailable()
}

func upgradeRangeNode(weak *consumer.WeakRange, treeState *consumer.TreeState) (consumer.Node, error) {
	if node, ok := weak.UpgradeNode(treeState); ok {
		return node, nil
	}
	return consumer.Node{}, elementNotAvailable()
}

type comparablePosition struct {
	path  []int
	index int
}

func weakComparablePositionFromEndpoint(r *consumer.WeakRange, endpoint int32) (comparablePosition, error) {
	switch endpoint {
	case textPatternRangeEndpointStart:
		path, index := r.StartComparable()
		return comparablePosition{path: path, index: index}, nil
	case textPatternRangeEndpointEnd:
		path, index := r.EndComparable()
		return comparablePosition{path: path, index: index}, nil
	default:
		return comparablePosition{}, invalidArg()
	}
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func (p comparablePosition) cmp(other comparablePosition) int {
	for i := 0; i < len(p.path) && i < len(other.path); i++ {
		if c := cmpInt(p.path[i], other.path[i]); c != 0 {
			return c
		}
	}
	if c := cmpInt(len(p.path), len(other.path)); c != 0 {
		return c
	}
	return cmpInt(p.index, other.index)
}

func positionFromEndpoint(r *consumer.Range, endpoint int32) (consumer.Position, error) {
	switch endpoint {
	case textPatternRangeEndpointStart:
		return r.Start(), nil
	case textPatternRangeEndpointEnd:
		return r.End(), nil
	default:
		return consumer.Position{}, invalidArg()
	}
}

func setEndpointPosition(r *consumer.Range, endpoint int32, pos consumer.Position) error {
	switch endpoint {
	case textPatternRangeEndpointStart:
		r.SetStart(pos)
	case textPatternRangeEndpointEnd:
		r.SetEnd(pos)
	default:
		return invalidArg()
	}
	return nil
}

func backToUnitStart(start consumer.Position, unit int32) (consumer.Position, error) {
	switch unit {
	case textUnitCharacter:
		return start, nil
	case textUnitFormat:
		if start.IsFormatStart() {
			return start, nil
		}
		return start.BackwardToFormatStart(), nil
	case textUnitWord:
		if start.IsWordStart() {
			return start, nil
		}
		return start.BackwardToWordStart(), nil
	case textUnitLine:
		if start.IsLineStart() {
			return start, nil
		}
		return start.BackwardToLineStart(), nil
	case textUnitParagraph:
		if start.IsParagraphStart() {
			return start, nil
		}
		return start.BackwardToParagraphStart(), nil
	case textUnitPage:
		if start.IsPageStart() {
			return start, nil
		}
		return start.BackwardToPageStart(), nil
	case textUnitDocument:
		if start.IsDocumentStart() {
			return start, nil
		}
		return start.DocumentStart(), nil
	default:
		return consumer.Position{}, invalidArg()
	}
}

func moveForwardToStart(pos consumer.Position, unit int32) (consumer.Position, error) {
	switch unit {
	case textUnitCharacter:
		return pos.ForwardToCharacterStart(), nil
	case textUnitFormat:
		return pos.ForwardToFormatStart(), nil
	case textUnitWord:
		return pos.ForwardToWordStart(), nil
	case textUnitLine:
		return pos.ForwardToLineStart(), nil
	case textUnitParagraph:
		return pos.ForwardToParagraphStart(), nil
	case textUnitPage:
		return pos.ForwardToPageStart(), nil
	case textUnitDocument:
		return pos.DocumentEnd(), nil
	default:
		return consumer.Position{}, invalidArg()
	}
}

func moveForwardToEnd(pos consumer.Position, unit int32) (consumer.Position, error) {
	switch unit {
	case textUnitCharacter:
		return pos.ForwardToCharacterEnd(), nil
	case textUnitFormat:
		return pos.ForwardToFormatEnd(), nil
	case textUnitWord:
		return pos.ForwardToWordEnd(), nil
	case textUnitLine:
		return pos.ForwardToLineEnd(), nil
	case textUnitParagraph:
		return pos.ForwardToParagraphEnd(), nil
	case textUnitPage:
		return pos.ForwardToPageEnd(), nil
	case textUnitDocument:
		return pos.DocumentEnd(), nil
	default:
		return consumer.Position{}, invalidArg()
	}
}

func moveBackward(pos consumer.Position, unit int32) (consumer.Position, error) {
	switch unit {
	case textUnitCharacter:
		return pos.BackwardToCharacterStart(), nil
	case textUnitFormat:
		return pos.BackwardToFormatStart(), nil
	case textUnitWord:
		return pos.BackwardToWordStart(), nil
	case textUnitLine:
		return pos.BackwardToLineStart(), nil
	case textUnitParagraph:
		return pos.BackwardToParagraphStart(), nil
	case textUnitPage:
		return pos.BackwardToPageStart(), nil
	case textUnitDocument:
		return pos.DocumentStart(), nil
	default:
		return consumer.Position{}, invalidArg()
	}
}

func movePosition(pos consumer.Position, unit int32, toEnd bool, count int32) (consumer.Position, int32, error) {
	forward := count > 0
	if count < 0 {
		count = -count
	}
	var moved int32
	for range count {
		var atEnd bool
		if forward {
			atEnd = pos.IsDocumentEnd()
		} else {
			atEnd = pos.IsDocumentStart()
		}
		if atEnd {
			break
		}
		var err error
		if forward {
			if toEnd {
				pos, err = moveForwardToEnd(pos, unit)
			} else {
				pos, err = moveForwardToStart(pos, unit)
			}
		} else {
			pos, err = moveBackward(pos, unit)
		}
		if err != nil {
			return consumer.Position{}, 0, err
		}
		moved++
	}
	if !forward {
		moved = -moved
	}
	return pos, moved, nil
}

type platformRange struct {
	context weak.Pointer[Context]
	mu      sync.RWMutex
	state   consumer.WeakRange
}

func newPlatformRange(context weak.Pointer[Context], r consumer.Range) *com.Object {
	return newPlatformRangeObject(&platformRange{context: context, state: r.Downgrade()})
}

func newPlatformRangeObject(impl *platformRange) *com.Object {
	return com.NewObject(impl, platformRangeVtable())
}

func (p *platformRange) upgradeContext() (*Context, error) {
	return upgrade(p.context)
}

func platformRangeWithTreeStateAndContext[T any](p *platformRange, f func(*consumer.TreeState, *Context) (T, error)) (T, error) {
	context, err := p.upgradeContext()
	if err != nil {
		var zero T
		return zero, err
	}
	tree, unlock := context.readTree()
	defer unlock()
	return f(tree.State(), context)
}

func platformRangeWithTreeState[T any](p *platformRange, f func(*consumer.TreeState) (T, error)) (T, error) {
	return platformRangeWithTreeStateAndContext(p, func(state *consumer.TreeState, _ *Context) (T, error) {
		return f(state)
	})
}

func (p *platformRange) upgradeNode(treeState *consumer.TreeState) (consumer.Node, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return upgradeRangeNode(&p.state, treeState)
}

func (p *platformRange) upgradeForRead(treeState *consumer.TreeState) (consumer.Range, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return upgradeRange(&p.state, treeState)
}

func platformRangeReadWithContext[T any](p *platformRange, f func(consumer.Range, *Context) (T, error)) (T, error) {
	return platformRangeWithTreeStateAndContext(p, func(treeState *consumer.TreeState, context *Context) (T, error) {
		r, err := p.upgradeForRead(treeState)
		if err != nil {
			var zero T
			return zero, err
		}
		return f(r, context)
	})
}

func platformRangeRead[T any](p *platformRange, f func(consumer.Range) (T, error)) (T, error) {
	return platformRangeReadWithContext(p, func(r consumer.Range, _ *Context) (T, error) {
		return f(r)
	})
}

func platformRangeWrite[T any](p *platformRange, f func(*consumer.Range) (T, error)) (T, error) {
	return platformRangeWithTreeState(p, func(treeState *consumer.TreeState) (T, error) {
		p.mu.Lock()
		defer p.mu.Unlock()
		r, err := upgradeRange(&p.state, treeState)
		if err != nil {
			var zero T
			return zero, err
		}
		result, err := f(&r)
		p.state = r.Downgrade()
		return result, err
	})
}

func (p *platformRange) doAction(f func(consumer.Range, *consumer.Tree) a11y.ActionRequest) error {
	context, err := p.upgradeContext()
	if err != nil {
		return err
	}
	tree, unlock := context.readTree()
	r, err := p.upgradeForRead(tree.State())
	if err != nil {
		unlock()
		return err
	}
	request := f(r, tree)
	unlock()
	context.doAction(request)
	return nil
}

func (p *platformRange) requireSameContext(other *platformRange) error {
	if p.context == other.context {
		return nil
	}
	return invalidArg()
}

func (p *platformRange) clone() *platformRange {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &platformRange{context: p.context, state: p.state.Clone()}
}

func platformRangeOf(this *com.IUnknown) (*platformRange, error) {
	if this == nil {
		return nil, invalidArg()
	}
	p, ok := com.ImplOf[*platformRange](this)
	if !ok {
		return nil, invalidArg()
	}
	return p, nil
}

func (p *platformRange) cloneInterface() (*com.IUnknown, error) {
	return newPlatformRangeObject(p.clone()).Interface(0), nil
}

func (p *platformRange) compare(otherParam *com.IUnknown) (bool, error) {
	if _, err := requiredParam(otherParam); err != nil {
		return false, err
	}
	other, err := platformRangeOf(otherParam)
	if err != nil {
		return false, err
	}
	if p.context != other.context {
		return false, nil
	}
	if other == p {
		return true, nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	return p.state.Equal(other.state), nil
}

func (p *platformRange) compareEndpoints(endpoint int32, otherParam *com.IUnknown, otherEndpoint int32) (int32, error) {
	if _, err := requiredParam(otherParam); err != nil {
		return 0, err
	}
	other, err := platformRangeOf(otherParam)
	if err != nil {
		return 0, err
	}
	if other == p {
		p.mu.RLock()
		defer p.mu.RUnlock()
		pos, err := weakComparablePositionFromEndpoint(&p.state, endpoint)
		if err != nil {
			return 0, err
		}
		otherPos, err := weakComparablePositionFromEndpoint(&p.state, otherEndpoint)
		if err != nil {
			return 0, err
		}
		return int32(pos.cmp(otherPos)), nil
	}
	if err := p.requireSameContext(other); err != nil {
		return 0, err
	}
	return platformRangeWithTreeState(p, func(treeState *consumer.TreeState) (int32, error) {
		r, err := p.upgradeForRead(treeState)
		if err != nil {
			return 0, err
		}
		otherRange, err := other.upgradeForRead(treeState)
		if err != nil {
			return 0, err
		}
		if r.Node().ID() != otherRange.Node().ID() {
			return 0, invalidArg()
		}
		pos, err := positionFromEndpoint(&r, endpoint)
		if err != nil {
			return 0, err
		}
		otherPos, err := positionFromEndpoint(&otherRange, otherEndpoint)
		if err != nil {
			return 0, err
		}
		result, ok := pos.PartialCmp(otherPos)
		if !ok {
			panic("called `Option::unwrap()` on a `None` value")
		}
		return int32(result), nil
	})
}

func (p *platformRange) expandToEnclosingUnit(unit int32) error {
	if unit == textUnitDocument {
		_, err := platformRangeWithTreeState(p, func(treeState *consumer.TreeState) (struct{}, error) {
			p.mu.Lock()
			defer p.mu.Unlock()
			node, err := upgradeRangeNode(&p.state, treeState)
			if err != nil {
				return struct{}{}, err
			}
			p.state = node.DocumentRange().Downgrade()
			return struct{}{}, nil
		})
		return err
	}
	_, err := platformRangeWrite(p, func(r *consumer.Range) (struct{}, error) {
		start := r.Start()
		if unit == textUnitCharacter && start.IsDocumentEnd() {
			return struct{}{}, nil
		}
		start, err := backToUnitStart(start, unit)
		if err != nil {
			return struct{}{}, err
		}
		r.SetStart(start)
		if !start.IsDocumentEnd() {
			end, err := moveForwardToEnd(start, unit)
			if err != nil {
				return struct{}{}, err
			}
			r.SetEnd(end)
		}
		return struct{}{}, nil
	})
	return err
}

func (p *platformRange) findAttribute(int32, *com.VARIANT, bool) (*com.IUnknown, error) {
	return nil, nil
}

func (p *platformRange) findText(com.BSTR, bool, bool) (*com.IUnknown, error) {
	return nil, nil
}

func optionVariant[T any](convert func(T) Variant) func(gpuiutil.Option[T]) Variant {
	return func(value gpuiutil.Option[T]) Variant {
		v, ok := value.Get()
		return variantFromOption(v, ok, convert)
	}
}

func (p *platformRange) getAttributeValue(id int32) (com.VARIANT, error) {
	return platformRangeRead(p, func(r consumer.Range) (com.VARIANT, error) {
		switch id {
		case uiaIsReadOnlyAttributeID:
			return variantFromBool(r.Node().IsReadOnly()).into(), nil
		case uiaCaretPositionAttributeID:
			value := int32(caretPositionUnknown)
			if r.IsDegenerate() {
				pos := r.Start()
				if pos.IsLineStart() {
					value = caretPositionBeginningOfLine
				} else if pos.IsLineEnd() {
					value = caretPositionEndOfLine
				}
			}
			return variantFromI32(value).into(), nil
		case uiaCultureAttributeID:
			return variantFromTextRangePropertyValue(r.Language(), optionVariant(variantFromLocaleName)).into(), nil
		case uiaFontNameAttributeID:
			buffer := acquireStringBuffer()
			defer buffer.release()
			return variantFromTextRangePropertyValue(r.FontFamily(), optionVariant(func(s string) Variant {
				return variantFromString(s, &buffer.buf)
			})).into(), nil
		case uiaFontSizeAttributeID:
			return variantFromTextRangePropertyValue(r.FontSize(), optionVariant(func(value float32) Variant {
				return variantFromF64(float64(value))
			})).into(), nil
		case uiaFontWeightAttributeID:
			return variantFromTextRangePropertyValue(r.FontWeight(), optionVariant(func(value float32) Variant {
				return variantFromI32(saturatingI32(value))
			})).into(), nil
		case uiaIsItalicAttributeID:
			return variantFromTextRangePropertyValue(r.IsItalic(), variantFromBool).into(), nil
		case uiaBackgroundColorAttributeID:
			return variantFromTextRangePropertyValue(r.BackgroundColor(), optionVariant(variantFromColor)).into(), nil
		case uiaForegroundColorAttributeID:
			return variantFromTextRangePropertyValue(r.ForegroundColor(), optionVariant(variantFromColor)).into(), nil
		case uiaOverlineStyleAttributeID:
			return decorationStyle(r.Overline()), nil
		case uiaOverlineColorAttributeID:
			return decorationColor(r.Overline()), nil
		case uiaStrikethroughStyleAttributeID:
			return decorationStyle(r.Strikethrough()), nil
		case uiaStrikethroughColorAttributeID:
			return decorationColor(r.Strikethrough()), nil
		case uiaUnderlineStyleAttributeID:
			return decorationStyle(r.Underline()), nil
		case uiaUnderlineColorAttributeID:
			return decorationColor(r.Underline()), nil
		case uiaHorizontalTextAlignmentAttributeID:
			return variantFromTextRangePropertyValue(r.TextAlign(), optionVariant(variantFromTextAlign)).into(), nil
		case uiaIsSubscriptAttributeID:
			return verticalOffsetIs(r, a11y.VerticalOffsetSubscript), nil
		case uiaIsSuperscriptAttributeID:
			return verticalOffsetIs(r, a11y.VerticalOffsetSuperscript), nil
		default:
			value, err := uiaGetReservedNotSupportedValue()
			if err != nil {
				panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
			}
			return variantFromUnknown(value).into(), nil
		}
	})
}

func saturatingI32(value float32) int32 {
	switch {
	case value != value:
		return 0
	case value >= math.MaxInt32:
		return math.MaxInt32
	case value <= math.MinInt32:
		return math.MinInt32
	default:
		return int32(value)
	}
}

func decorationStyle(value consumer.TextRangePropertyValue[gpuiutil.Option[a11y.TextDecoration]]) com.VARIANT {
	mapped := consumer.RangePropertyValueMap(value, func(d a11y.TextDecoration) a11y.TextDecorationStyle { return d.Style })
	return variantFromTextRangePropertyValue(mapped, optionVariant(variantFromTextDecorationStyle)).into()
}

func decorationColor(value consumer.TextRangePropertyValue[gpuiutil.Option[a11y.TextDecoration]]) com.VARIANT {
	mapped := consumer.RangePropertyValueMap(value, func(d a11y.TextDecoration) a11y.Color { return d.Color })
	return variantFromTextRangePropertyValue(mapped, optionVariant(variantFromColor)).into()
}

func verticalOffsetIs(r consumer.Range, offset a11y.VerticalOffset) com.VARIANT {
	mapped := consumer.RangePropertyValueMap(r.VerticalOffset(), func(o a11y.VerticalOffset) bool { return o == offset })
	return variantFromTextRangePropertyValue(mapped, optionVariant(variantFromBool)).into()
}

func (p *platformRange) getBoundingRectangles() (*com.SAFEARRAY, error) {
	return platformRangeReadWithContext(p, func(r consumer.Range, context *Context) (*com.SAFEARRAY, error) {
		rects := r.BoundingBoxes()
		if len(rects) == 0 {
			return nil, nil
		}
		clientTopLeft := context.clientTopLeft()
		result := make([]float64, 0, len(rects)*4)
		for _, rect := range rects {
			result = append(result, rect.X0+clientTopLeft.X, rect.Y0+clientTopLeft.Y, rect.Width(), rect.Height())
		}
		return safeArrayFromF64Slice(result), nil
	})
}

func (p *platformRange) getEnclosingElement() (*com.IUnknown, error) {
	context, err := p.upgradeContext()
	if err != nil {
		return nil, err
	}
	tree, unlock := context.readTree()
	defer unlock()
	node, err := p.upgradeNode(tree.State())
	if err != nil {
		return nil, err
	}
	return context.getOrCreatePlatformNode(node.ID()).Interface(platformNodeSimple), nil
}

func (p *platformRange) getText(int32) (com.BSTR, error) {
	return platformRangeRead(p, func(r consumer.Range) (com.BSTR, error) {
		buffer := acquireStringBuffer()
		defer buffer.release()
		result := newWideString(&buffer.buf)
		if err := r.WriteText(result); err != nil {
			panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
		}
		return result.toBSTR(), nil
	})
}

func (p *platformRange) move(unit int32, count int32) (int32, error) {
	return platformRangeWrite(p, func(r *consumer.Range) (int32, error) {
		degenerate := r.IsDegenerate()
		start := r.Start()
		if !degenerate {
			var err error
			start, err = backToUnitStart(start, unit)
			if err != nil {
				return 0, err
			}
		}
		start, moved, err := movePosition(start, unit, false, count)
		if err != nil {
			return 0, err
		}
		if moved != 0 {
			r.SetStart(start)
			end := start
			if !degenerate && !start.IsDocumentEnd() {
				end, err = moveForwardToEnd(start, unit)
				if err != nil {
					return 0, err
				}
			}
			r.SetEnd(end)
		}
		return moved, nil
	})
}

func (p *platformRange) moveEndpointByUnit(endpoint int32, unit int32, count int32) (int32, error) {
	return platformRangeWrite(p, func(r *consumer.Range) (int32, error) {
		pos, err := positionFromEndpoint(r, endpoint)
		if err != nil {
			return 0, err
		}
		pos, moved, err := movePosition(pos, unit, endpoint == textPatternRangeEndpointEnd, count)
		if err != nil {
			return 0, err
		}
		if err := setEndpointPosition(r, endpoint, pos); err != nil {
			return 0, err
		}
		return moved, nil
	})
}

func (p *platformRange) moveEndpointByRange(endpoint int32, otherParam *com.IUnknown, otherEndpoint int32) error {
	if _, err := requiredParam(otherParam); err != nil {
		return err
	}
	other, err := platformRangeOf(otherParam)
	if err != nil {
		return err
	}
	if err := p.requireSameContext(other); err != nil {
		return err
	}
	_, err = platformRangeWithTreeState(p, func(treeState *consumer.TreeState) (struct{}, error) {
		otherRange, err := other.upgradeForRead(treeState)
		if err != nil {
			return struct{}{}, err
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		r, err := upgradeRange(&p.state, treeState)
		if err != nil {
			return struct{}{}, err
		}
		if r.Node().ID() != otherRange.Node().ID() {
			return struct{}{}, invalidArg()
		}
		pos, err := positionFromEndpoint(&otherRange, otherEndpoint)
		if err != nil {
			return struct{}{}, err
		}
		if err := setEndpointPosition(&r, endpoint, pos); err != nil {
			return struct{}{}, err
		}
		p.state = r.Downgrade()
		return struct{}{}, nil
	})
	return err
}

func locateNodeUnwrap(tree *consumer.Tree, id consumer.NodeID) (a11y.NodeID, a11y.TreeID) {
	targetNode, targetTree, ok := tree.State().LocateNode(id)
	if !ok {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return targetNode, targetTree
}

func (p *platformRange) selectRange() error {
	return p.doAction(func(r consumer.Range, tree *consumer.Tree) a11y.ActionRequest {
		targetNode, targetTree := locateNodeUnwrap(tree, r.Node().ID())
		return a11y.ActionRequest{
			Action:     a11y.ActionSetTextSelection,
			TargetTree: targetTree,
			TargetNode: targetNode,
			Data:       a11y.ActionDataSetTextSelection(r.ToTextSelection()),
		}
	})
}

func (p *platformRange) addToSelection() error {
	return invalidOperation()
}

func (p *platformRange) removeFromSelection() error {
	return invalidOperation()
}

func (p *platformRange) scrollIntoView(alignToTop bool) error {
	return p.doAction(func(r consumer.Range, tree *consumer.Tree) a11y.ActionRequest {
		position := r.End()
		hint := a11y.ScrollHintBottomEdge
		if alignToTop {
			position = r.Start()
			hint = a11y.ScrollHintTopEdge
		}
		targetNode, targetTree := locateNodeUnwrap(tree, position.InnerNode().ID())
		return a11y.ActionRequest{
			Action:     a11y.ActionScrollIntoView,
			TargetTree: targetTree,
			TargetNode: targetNode,
			Data:       a11y.ActionDataScrollHint(hint),
		}
	})
}

func (p *platformRange) getChildren() (*com.SAFEARRAY, error) {
	return safeArrayFromComSlice(nil), nil
}

func rangeOut[T any](out *T, f func(*platformRange) (T, error), this *com.IUnknown) uintptr {
	if out == nil {
		return com.E_POINTER.Uintptr()
	}
	var zero T
	*out = zero
	p, ok := com.ImplOf[*platformRange](this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	result, err := f(p)
	if err != nil {
		return hresult(err)
	}
	*out = result
	return com.S_OK.Uintptr()
}

func rangeNoOut(f func(*platformRange) error, this *com.IUnknown) uintptr {
	p, ok := com.ImplOf[*platformRange](this)
	if !ok {
		return com.E_UNEXPECTED.Uintptr()
	}
	return hresult(f(p))
}

var (
	platformRangeVtableOnce  sync.Once
	platformRangeVtableCache *com.Vtable
)

func platformRangeVtable() *com.Vtable {
	platformRangeVtableOnce.Do(func() {
		platformRangeVtableCache = newPlatformNodeVtable(iidITextRangeProvider,
			func(this *com.IUnknown, out **com.IUnknown) uintptr {
				return rangeOut(out, (*platformRange).cloneInterface, this)
			},
			func(this *com.IUnknown, other *com.IUnknown, out *int32) uintptr {
				return rangeOut(out, func(p *platformRange) (int32, error) {
					v, err := p.compare(other)
					return boolOut(v), err
				}, this)
			},
			func(this *com.IUnknown, endpoint int32, other *com.IUnknown, otherEndpoint int32, out *int32) uintptr {
				return rangeOut(out, func(p *platformRange) (int32, error) {
					return p.compareEndpoints(endpoint, other, otherEndpoint)
				}, this)
			},
			func(this *com.IUnknown, unit int32) uintptr {
				return rangeNoOut(func(p *platformRange) error { return p.expandToEnclosingUnit(unit) }, this)
			},
			func(this *com.IUnknown, id int32, value *com.VARIANT, backward int32, out **com.IUnknown) uintptr {
				return rangeOut(out, func(p *platformRange) (*com.IUnknown, error) {
					return p.findAttribute(id, value, backward != 0)
				}, this)
			},
			func(this *com.IUnknown, text com.BSTR, backward int32, ignoreCase int32, out **com.IUnknown) uintptr {
				return rangeOut(out, func(p *platformRange) (*com.IUnknown, error) {
					return p.findText(text, backward != 0, ignoreCase != 0)
				}, this)
			},
			func(this *com.IUnknown, id int32, out *com.VARIANT) uintptr {
				return rangeOut(out, func(p *platformRange) (com.VARIANT, error) { return p.getAttributeValue(id) }, this)
			},
			func(this *com.IUnknown, out **com.SAFEARRAY) uintptr {
				return rangeOut(out, (*platformRange).getBoundingRectangles, this)
			},
			func(this *com.IUnknown, out **com.IUnknown) uintptr {
				return rangeOut(out, (*platformRange).getEnclosingElement, this)
			},
			func(this *com.IUnknown, maxLength int32, out *com.BSTR) uintptr {
				return rangeOut(out, func(p *platformRange) (com.BSTR, error) { return p.getText(maxLength) }, this)
			},
			func(this *com.IUnknown, unit int32, count int32, out *int32) uintptr {
				return rangeOut(out, func(p *platformRange) (int32, error) { return p.move(unit, count) }, this)
			},
			func(this *com.IUnknown, endpoint int32, unit int32, count int32, out *int32) uintptr {
				return rangeOut(out, func(p *platformRange) (int32, error) { return p.moveEndpointByUnit(endpoint, unit, count) }, this)
			},
			func(this *com.IUnknown, endpoint int32, other *com.IUnknown, otherEndpoint int32) uintptr {
				return rangeNoOut(func(p *platformRange) error { return p.moveEndpointByRange(endpoint, other, otherEndpoint) }, this)
			},
			func(this *com.IUnknown) uintptr {
				return rangeNoOut((*platformRange).selectRange, this)
			},
			func(this *com.IUnknown) uintptr {
				return rangeNoOut((*platformRange).addToSelection, this)
			},
			func(this *com.IUnknown) uintptr {
				return rangeNoOut((*platformRange).removeFromSelection, this)
			},
			func(this *com.IUnknown, alignToTop int32) uintptr {
				return rangeNoOut(func(p *platformRange) error { return p.scrollIntoView(alignToTop != 0) }, this)
			},
			func(this *com.IUnknown, out **com.SAFEARRAY) uintptr {
				return rangeOut(out, (*platformRange).getChildren, this)
			},
		)
	})
	return platformRangeVtableCache
}
