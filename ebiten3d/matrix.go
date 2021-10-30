package ebiten3d

import (
	"math"
	"strconv"

	"github.com/kvartborg/vector"
)

type Matrix4 [4][4]float64

func NewMatrix4() Matrix4 {

	mat := Matrix4{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	return mat

}

func Translate(x, y, z float64) Matrix4 {
	mat := NewMatrix4()
	mat[0][3] = x
	mat[1][3] = y
	mat[2][3] = z
	return mat
}

func Scale(x, y, z float64) Matrix4 {
	mat := NewMatrix4()
	mat[0][0] = x
	mat[1][1] = y
	mat[2][2] = z
	return mat
}

func Rotate(vector vector.Vector, angle float64) Matrix4 {

	mat := NewMatrix4()
	vector = vector.Unit()
	s := math.Sin(angle)
	c := math.Cos(angle)
	m := 1 - c

	mat[0][0] = m*vector[0]*vector[0] + c
	mat[0][1] = m*vector[0]*vector[1] + vector[2]*s
	mat[0][2] = m*vector[2]*vector[0] - vector[1]*s

	mat[1][0] = m*vector[0]*vector[1] - vector[2]*s
	mat[1][1] = m*vector[1]*vector[1] + c
	mat[1][2] = m*vector[1]*vector[2] + vector[0]*s

	mat[2][0] = m*vector[2]*vector[0] + vector[1]*s
	mat[2][1] = m*vector[1]*vector[2] - vector[0]*s
	mat[2][2] = m*vector[2]*vector[2] + c

	return mat

}

func (matrix Matrix4) Right() vector.Vector {
	return vector.Vector{
		matrix[0][0],
		matrix[0][1],
		matrix[0][2],
	}
}

func (matrix Matrix4) Up() vector.Vector {
	return vector.Vector{
		matrix[1][0],
		matrix[1][1],
		matrix[1][2],
	}
}

func (matrix Matrix4) Forward() vector.Vector {
	return vector.Vector{
		-matrix[2][0],
		-matrix[2][1],
		-matrix[2][2],
	}
}

func (matrix Matrix4) Rotate(v vector.Vector, angle float64) Matrix4 {
	mat := matrix.Clone()
	mat.MultVec(v)
	return mat
}

func (matrix Matrix4) Clone() Matrix4 {
	newMat := NewMatrix4()
	for y := 0; y < len(matrix); y++ {
		for x := 0; x < len(matrix[y]); x++ {
			newMat[y][x] = matrix[y][x]
		}
	}
	return newMat
}

func (matrix Matrix4) MultVec(vect vector.Vector) vector.Vector {

	return vector.Vector{

		matrix[0][0]*vect[0] + matrix[0][1]*vect[1] + matrix[0][2]*vect[2] + matrix[0][3],
		matrix[1][0]*vect[0] + matrix[1][1]*vect[1] + matrix[1][2]*vect[2] + matrix[1][3],
		matrix[2][0]*vect[0] + matrix[2][1]*vect[1] + matrix[2][2]*vect[2] + matrix[2][3],
		// matrix[3][0]*vect[0] + matrix[3][1]*vect[1] + matrix[3][2]*vect[2] + matrix[3][3],
		// 1 - vect[2] /

	}

}

func (matrix Matrix4) MultVecW(vect vector.Vector) vector.Vector {

	return vector.Vector{

		matrix[0][0]*vect[0] + matrix[0][1]*vect[1] + matrix[0][2]*vect[2] + matrix[0][3],
		matrix[1][0]*vect[0] + matrix[1][1]*vect[1] + matrix[1][2]*vect[2] + matrix[1][3],
		matrix[2][0]*vect[0] + matrix[2][1]*vect[1] + matrix[2][2]*vect[2] + matrix[2][3],
		matrix[3][0]*vect[0] + matrix[3][1]*vect[1] + matrix[3][2]*vect[2] + matrix[3][3],
	}

}

func (matrix Matrix4) Mult(other Matrix4) Matrix4 {

	newMat := NewMatrix4()

	for r := 0; r < len(matrix); r++ {

		row := matrix[r]

		for c := 0; c < len(other); c++ {

			column := other[c]

			sum := 0.0

			for v := range row {

				sum += row[v] * column[v]

			}

			newMat[r][c] = sum

		}

	}

	return newMat

}

func (matrix Matrix4) Columns() [][]float64 {

	columns := [][]float64{
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}

	for r := range matrix {
		for c := range matrix[r] {
			columns[c][r] = matrix[r][c]
		}
	}

	return columns
}

// func (matrix Matrix4) RowSums() []float64 {
// 	rowSums := []float64{}
// 	for _, row := range matrix {
// 		s := 0.0
// 		for _, c := range row {
// 			s += c
// 		}
// 		rowSums = append(rowSums, s)
// 	}
// 	return rowSums
// }

// func (matrix Matrix4) ColumnSums() []float64 {

// 	columnSums := []float64{}
// 	for x := 0; x < 4; x++ {
// 		s := 0.0
// 		for rowIndex := range matrix {
// 			s += matrix[x][rowIndex]
// 		}
// 		columnSums = append(columnSums, s)
// 	}
// 	return columnSums

// }

func (matrix Matrix4) String() string {
	s := "{"
	for i, y := range matrix {
		for _, x := range y {
			s += strconv.FormatFloat(x, 'f', -1, 64) + ", "
		}
		if i < len(matrix)-1 {
			s += "\n"
		}
	}
	s += "}"
	return s
}

func LookAt(eye, center, up vector.Vector) Matrix4 {
	z := eye.Sub(center).Unit()
	x, _ := up.Cross(z)
	x = x.Unit()
	y, _ := z.Cross(x)
	return Matrix4{
		{x[0], x[1], x[2], -x.Dot(eye)},
		{y[0], y[1], y[2], -y.Dot(eye)},
		{z[0], z[1], z[2], -z.Dot(eye)},
		{0, 0, 0, 1},
	}
}

// func LookAt(target, center, up vector.Vector) Matrix4 {
// 	z := target.Sub(center).Unit()
// 	x, _ := up.Cross(z)
// 	x = x.Unit()
// 	y, _ := z.Cross(x)

// 	return Matrix4{
// 		{x[0], x[1], x[2], -x.Dot(center)},
// 		{y[0], y[1], y[2], -y.Dot(center)},
// 		{z[0], z[1], z[2], -z.Dot(center)},
// 		{0, 0, 0, 1},
// 	}

// }
