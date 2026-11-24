//go:build windows

#include "textflag.h"

TEXT ·elementProviderFromPointTramp(SB), NOSPLIT|NOFRAME, $0-0
	MOVQ X1, DX
	MOVQ X2, R8
	MOVQ ·elementProviderFromPointTarget(SB), AX
	JMP  AX

TEXT ·rangeValueSetValueTramp(SB), NOSPLIT|NOFRAME, $0-0
	MOVQ X1, DX
	MOVQ ·rangeValueSetValueTarget(SB), AX
	JMP  AX

TEXT ·elementProviderFromPointTrampAddr(SB), NOSPLIT, $0-8
	MOVQ $·elementProviderFromPointTramp(SB), AX
	MOVQ AX, ret+0(FP)
	RET

TEXT ·rangeValueSetValueTrampAddr(SB), NOSPLIT, $0-8
	MOVQ $·rangeValueSetValueTramp(SB), AX
	MOVQ AX, ret+0(FP)
	RET
