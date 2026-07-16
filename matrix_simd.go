//go:build amd64

package tetra3d

func (matrix Matrix4) Mult(other Matrix4) Matrix4 {
	return Matrix4_Mult(matrix, other)
}

func Matrix4_Mult(matrix Matrix4, other Matrix4) Matrix4
