package lyon

// DebugValidator is lyon_path's private DebugValidator: it checks begin / end pairing.
type DebugValidator struct {
	inSubpath bool
}

// NewDebugValidator is DebugValidator::new.
func NewDebugValidator() DebugValidator { return DebugValidator{} }

// Begin is DebugValidator::begin.
func (v *DebugValidator) Begin() {
	if v.inSubpath {
		panic("multiple begin() calls without end()")
	}
	v.inSubpath = true
}

// End is DebugValidator::end.
func (v *DebugValidator) End() {
	if !v.inSubpath {
		panic("end() called without begin()")
	}
	v.inSubpath = false
}

// Edge is DebugValidator::edge.
func (v *DebugValidator) Edge() {
	if !v.inSubpath {
		panic("edge operation is made before begin()")
	}
}

// Build is DebugValidator::build.
func (v *DebugValidator) Build() {
	if v.inSubpath {
		panic("build() called before end()")
	}
}
