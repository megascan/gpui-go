package zeno

// Fill is the fill rule.
type Fill uint8

const (
	// FillNonZero is the non-zero fill rule.
	FillNonZero Fill = iota
	// FillEvenOdd is the even-odd fill rule.
	FillEvenOdd
)
