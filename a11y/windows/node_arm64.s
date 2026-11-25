//go:build windows

#include "textflag.h"

TEXT ·elementProviderFromPointTramp(SB), NOSPLIT|NOFRAME, $0-0
	MOVD  R1, R3
	FMOVD F0, R1
	FMOVD F1, R2
	MOVD  ·elementProviderFromPointTarget(SB), R16
	B     (R16)

TEXT ·rangeValueSetValueTramp(SB), NOSPLIT|NOFRAME, $0-0
	FMOVD F0, R1
	MOVD  ·rangeValueSetValueTarget(SB), R16
	B     (R16)

TEXT ·textRangeFromPointTramp(SB), NOSPLIT|NOFRAME, $0-0
	MOVD  R1, R3
	FMOVD F0, R1
	FMOVD F1, R2
	MOVD  ·textRangeFromPointTarget(SB), R16
	B     (R16)

TEXT ·elementProviderFromPointTrampAddr(SB), NOSPLIT, $0-8
	MOVD $·elementProviderFromPointTramp(SB), R0
	MOVD R0, ret+0(FP)
	RET

TEXT ·rangeValueSetValueTrampAddr(SB), NOSPLIT, $0-8
	MOVD $·rangeValueSetValueTramp(SB), R0
	MOVD R0, ret+0(FP)
	RET

TEXT ·textRangeFromPointTrampAddr(SB), NOSPLIT, $0-8
	MOVD $·textRangeFromPointTramp(SB), R0
	MOVD R0, ret+0(FP)
	RET
