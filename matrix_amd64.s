#include "textflag.h"

// func Matrix4_Mult(matrix Matrix4, other Matrix4) Matrix4
TEXT ·Matrix4_Mult(SB), NOSPLIT, $0-192
	MOVUPS other+64(FP), X1   // other row0
	MOVUPS other+80(FP), X2   // other row1
	MOVUPS other+96(FP), X3   // other row2
	MOVUPS other+112(FP), X4  // other row3
	// Compute ret row0 = sum (matrix row0[k] * other row k for k=0..3)
	MOVUPS matrix+0(FP), X0    // matrix row0: a00 a01 a02 a03
	MOVAPS X0, X5
	SHUFPS $0x00, X5, X5  // a00 a00 a00 a00
	MULPS  X1, X5         // a00 * other row0
	MOVAPS X0, X6
	SHUFPS $0x55, X6, X6  // a01 a01 a01 a01
	MULPS  X2, X6
	ADDPS  X6, X5
	MOVAPS X0, X6
	SHUFPS $0xAA, X6, X6  // a02 a02 a02 a02
	MULPS  X3, X6
	ADDPS  X6, X5
	MOVAPS X0, X6
	SHUFPS $0xFF, X6, X6  // a03 a03 a03 a03
	MULPS  X4, X6
	ADDPS  X6, X5
	MOVUPS X5, ret+128(FP)
	// Compute ret row1
	MOVUPS matrix+16(FP), X0   // matrix row1: a10 a11 a12 a13
	MOVAPS X0, X5
	SHUFPS $0x00, X5, X5
	MULPS  X1, X5
	MOVAPS X0, X6
	SHUFPS $0x55, X6, X6
	MULPS  X2, X6
	ADDPS  X6, X5
	MOVAPS X0, X6
	SHUFPS $0xAA, X6, X6
	MULPS  X3, X6
	ADDPS  X6, X5
	MOVAPS X0, X6
	SHUFPS $0xFF, X6, X6
	MULPS  X4, X6
	ADDPS  X6, X5
	MOVUPS X5, ret+144(FP)
	// Compute ret row2
	MOVUPS matrix+32(FP), X0   // matrix row2: a20 a21 a22 a23
	MOVAPS X0, X5
	SHUFPS $0x00, X5, X5
	MULPS  X1, X5
	MOVAPS X0, X6
	SHUFPS $0x55, X6, X6
	MULPS  X2, X6
	ADDPS  X6, X5
	MOVAPS X0, X6
	SHUFPS $0xAA, X6, X6
	MULPS  X3, X6
	ADDPS  X6, X5
	MOVAPS X0, X6
	SHUFPS $0xFF, X6, X6
	MULPS  X4, X6
	ADDPS  X6, X5
	MOVUPS X5, ret+160(FP)
	// Compute ret row3
	MOVUPS matrix+48(FP), X0   // matrix row3: a30 a31 a32 a33
	MOVAPS X0, X5
	SHUFPS $0x00, X5, X5
	MULPS  X1, X5
	MOVAPS X0, X6
	SHUFPS $0x55, X6, X6
	MULPS  X2, X6
	ADDPS  X6, X5
	MOVAPS X0, X6
	SHUFPS $0xAA, X6, X6
	MULPS  X3, X6
	ADDPS  X6, X5
	MOVAPS X0, X6
	SHUFPS $0xFF, X6, X6
	MULPS  X4, X6
	ADDPS  X6, X5
	MOVUPS X5, ret+176(FP)
	RET
