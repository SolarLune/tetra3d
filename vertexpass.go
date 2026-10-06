package tetra3d

// transformVertices is the vertex pass of an unskinned mesh part with no
// vertex function, shape keys, vertex snapping, normals render, or stored
// positions for the lights. For each position, it writes the clip position
// (mvp times the position), the clip position divided by w, and the clip
// code. The arithmetic is the same as the general vertex pass in
// Model.processVertices, so the results are the same to the bit.
func transformVertices(mvp *Matrix4, near, far float32, pos []Vector3, clip []Vector4, screen []Vector2, codes []uint8) {

	m00, m01, m02, m03 := mvp[0][0], mvp[0][1], mvp[0][2], mvp[0][3]
	m10, m11, m12, m13 := mvp[1][0], mvp[1][1], mvp[1][2], mvp[1][3]
	m20, m21, m22, m23 := mvp[2][0], mvp[2][1], mvp[2][2], mvp[2][3]
	m30, m31, m32, m33 := mvp[3][0], mvp[3][1], mvp[3][2], mvp[3][3]

	n := len(pos)
	clip = clip[:n]
	screen = screen[:n]
	codes = codes[:n]

	for i := range n {

		x, y, z := pos[i].X, pos[i].Y, pos[i].Z

		tx := m00*x + m10*y + m20*z + m30
		ty := m01*x + m11*y + m21*z + m31
		tz := m02*x + m12*y + m22*z + m32
		tw := m03*x + m13*y + m23*z + m33
		clip[i] = Vector4{tx, ty, tz, tw}

		w := tw
		if w < 0 {
			w = 0.000001
		}
		sx := tx / w
		sy := ty / w
		screen[i] = Vector2{sx, sy}

		code := uint8(0)
		if sx < -0.5 {
			code |= clipLeft
		}
		if sx > 0.5 {
			code |= clipRight
		}
		if sy < -0.5 {
			code |= clipBottom
		}
		if sy > 0.5 {
			code |= clipTop
		}
		if !(tz+1 >= near && tz < far) {
			code |= clipDepth
		}
		codes[i] = code

	}

}
