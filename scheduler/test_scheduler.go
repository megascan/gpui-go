package scheduler

// Yield is a future that wakes itself and stays pending for a number of polls before
// completing.
type Yield int

// Poll completes once the remaining count is zero, otherwise decrements it and wakes itself.
func (y *Yield) Poll(cx *Context) (struct{}, bool) {
	if *y == 0 {
		return struct{}{}, true
	}
	*y--
	cx.Waker().WakeByRef()
	return struct{}{}, false
}
