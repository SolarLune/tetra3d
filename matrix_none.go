//go:build !amd64

package tetra3d

// Mult multiplies a Matrix4 by another provided Matrix4 - this effectively combines them.
func (matrix Matrix4) Mult(other Matrix4) Matrix4 {

	newMat := NewMatrix4()

	newMat[0][0] = matrix[0][0]*other[0][0] + matrix[0][1]*other[1][0] + matrix[0][2]*other[2][0] + matrix[0][3]*other[3][0]
	newMat[1][0] = matrix[1][0]*other[0][0] + matrix[1][1]*other[1][0] + matrix[1][2]*other[2][0] + matrix[1][3]*other[3][0]
	newMat[2][0] = matrix[2][0]*other[0][0] + matrix[2][1]*other[1][0] + matrix[2][2]*other[2][0] + matrix[2][3]*other[3][0]
	newMat[3][0] = matrix[3][0]*other[0][0] + matrix[3][1]*other[1][0] + matrix[3][2]*other[2][0] + matrix[3][3]*other[3][0]

	newMat[0][1] = matrix[0][0]*other[0][1] + matrix[0][1]*other[1][1] + matrix[0][2]*other[2][1] + matrix[0][3]*other[3][1]
	newMat[1][1] = matrix[1][0]*other[0][1] + matrix[1][1]*other[1][1] + matrix[1][2]*other[2][1] + matrix[1][3]*other[3][1]
	newMat[2][1] = matrix[2][0]*other[0][1] + matrix[2][1]*other[1][1] + matrix[2][2]*other[2][1] + matrix[2][3]*other[3][1]
	newMat[3][1] = matrix[3][0]*other[0][1] + matrix[3][1]*other[1][1] + matrix[3][2]*other[2][1] + matrix[3][3]*other[3][1]

	newMat[0][2] = matrix[0][0]*other[0][2] + matrix[0][1]*other[1][2] + matrix[0][2]*other[2][2] + matrix[0][3]*other[3][2]
	newMat[1][2] = matrix[1][0]*other[0][2] + matrix[1][1]*other[1][2] + matrix[1][2]*other[2][2] + matrix[1][3]*other[3][2]
	newMat[2][2] = matrix[2][0]*other[0][2] + matrix[2][1]*other[1][2] + matrix[2][2]*other[2][2] + matrix[2][3]*other[3][2]
	newMat[3][2] = matrix[3][0]*other[0][2] + matrix[3][1]*other[1][2] + matrix[3][2]*other[2][2] + matrix[3][3]*other[3][2]

	newMat[0][3] = matrix[0][0]*other[0][3] + matrix[0][1]*other[1][3] + matrix[0][2]*other[2][3] + matrix[0][3]*other[3][3]
	newMat[1][3] = matrix[1][0]*other[0][3] + matrix[1][1]*other[1][3] + matrix[1][2]*other[2][3] + matrix[1][3]*other[3][3]
	newMat[2][3] = matrix[2][0]*other[0][3] + matrix[2][1]*other[1][3] + matrix[2][2]*other[2][3] + matrix[2][3]*other[3][3]
	newMat[3][3] = matrix[3][0]*other[0][3] + matrix[3][1]*other[1][3] + matrix[3][2]*other[2][3] + matrix[3][3]*other[3][3]

	return newMat

}
