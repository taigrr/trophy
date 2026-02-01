package math3d

import "math"

// Mat4 is a 4x4 matrix stored in column-major order.
// This matches OpenGL conventions for easier reasoning about transforms.
//
// Memory layout (indices):
// | 0  4  8  12 |
// | 1  5  9  13 |
// | 2  6  10 14 |
// | 3  7  11 15 |
//
// For a transform matrix:
// | Xx Yx Zx Tx |   X,Y,Z = basis vectors (rotation/scale)
// | Xy Yy Zy Ty |   T = translation
// | Xz Yz Zz Tz |
// | 0  0  0  1  |
type Mat4 [16]float64

// Identity returns the identity matrix.
func Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Translate creates a translation matrix.
func Translate(v Vec3) Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		v.X, v.Y, v.Z, 1,
	}
}

// Scale creates a scaling matrix.
func Scale(v Vec3) Mat4 {
	return Mat4{
		v.X, 0, 0, 0,
		0, v.Y, 0, 0,
		0, 0, v.Z, 0,
		0, 0, 0, 1,
	}
}

// ScaleUniform creates a uniform scaling matrix.
func ScaleUniform(s float64) Mat4 {
	return Scale(V3(s, s, s))
}

// RotateX creates a rotation matrix around the X axis.
func RotateX(angle float64) Mat4 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Mat4{
		1, 0, 0, 0,
		0, c, s, 0,
		0, -s, c, 0,
		0, 0, 0, 1,
	}
}

// RotateY creates a rotation matrix around the Y axis.
func RotateY(angle float64) Mat4 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Mat4{
		c, 0, -s, 0,
		0, 1, 0, 0,
		s, 0, c, 0,
		0, 0, 0, 1,
	}
}

// RotateZ creates a rotation matrix around the Z axis.
func RotateZ(angle float64) Mat4 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Mat4{
		c, s, 0, 0,
		-s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Rotate creates a rotation matrix around an arbitrary axis.
func Rotate(axis Vec3, angle float64) Mat4 {
	axis = axis.Normalize()
	c, s := math.Cos(angle), math.Sin(angle)
	t := 1 - c
	x, y, z := axis.X, axis.Y, axis.Z

	return Mat4{
		t*x*x + c, t*x*y + s*z, t*x*z - s*y, 0,
		t*x*y - s*z, t*y*y + c, t*y*z + s*x, 0,
		t*x*z + s*y, t*y*z - s*x, t*z*z + c, 0,
		0, 0, 0, 1,
	}
}

// LookAt creates a view matrix looking from eye towards center.
func LookAt(eye, center, up Vec3) Mat4 {
	f := center.Sub(eye).Normalize() // Forward
	s := f.Cross(up).Normalize()     // Right
	u := s.Cross(f)                  // Up (recomputed)

	return Mat4{
		s.X, u.X, -f.X, 0,
		s.Y, u.Y, -f.Y, 0,
		s.Z, u.Z, -f.Z, 0,
		-s.Dot(eye), -u.Dot(eye), f.Dot(eye), 1,
	}
}

// Perspective creates a perspective projection matrix.
// fovy is vertical field of view in radians.
// aspect is width/height.
// near and far are clipping planes.
func Perspective(fovy, aspect, near, far float64) Mat4 {
	f := 1.0 / math.Tan(fovy/2)
	nf := 1.0 / (near - far)

	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) * nf, -1,
		0, 0, 2 * far * near * nf, 0,
	}
}

// Orthographic creates an orthographic projection matrix.
func Orthographic(left, right, bottom, top, near, far float64) Mat4 {
	rl := 1.0 / (right - left)
	tb := 1.0 / (top - bottom)
	fn := 1.0 / (far - near)

	return Mat4{
		2 * rl, 0, 0, 0,
		0, 2 * tb, 0, 0,
		0, 0, -2 * fn, 0,
		-(right + left) * rl, -(top + bottom) * tb, -(far + near) * fn, 1,
	}
}

// Mul multiplies two matrices: a * b.
//
//nolint:st1016 // a*b naming convention is clearer for matrix multiplication
func (a Mat4) Mul(b Mat4) Mat4 {
	var m Mat4
	for col := range 4 {
		for row := range 4 {
			var sum float64
			for k := range 4 {
				sum += a[row+k*4] * b[k+col*4]
			}
			m[row+col*4] = sum
		}
	}
	return m
}

// MulVec3 transforms a Vec3 as a point (w=1).
func (m Mat4) MulVec3(v Vec3) Vec3 {
	w := m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]
	if w == 0 {
		w = 1
	}
	return Vec3{
		(m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]) / w,
		(m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]) / w,
		(m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]) / w,
	}
}

// MulVec3Dir transforms a Vec3 as a direction (w=0, no translation).
func (m Mat4) MulVec3Dir(v Vec3) Vec3 {
	return Vec3{
		m[0]*v.X + m[4]*v.Y + m[8]*v.Z,
		m[1]*v.X + m[5]*v.Y + m[9]*v.Z,
		m[2]*v.X + m[6]*v.Y + m[10]*v.Z,
	}
}

// MulVec4 transforms a Vec4.
func (m Mat4) MulVec4(v Vec4) Vec4 {
	return Vec4{
		m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]*v.W,
		m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]*v.W,
		m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]*v.W,
		m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]*v.W,
	}
}

// Transpose returns the transposed matrix.
func (m Mat4) Transpose() Mat4 {
	return Mat4{
		m[0], m[4], m[8], m[12],
		m[1], m[5], m[9], m[13],
		m[2], m[6], m[10], m[14],
		m[3], m[7], m[11], m[15],
	}
}

// Determinant returns the determinant of the matrix.
func (m Mat4) Determinant() float64 {
	return m[0]*(m[5]*(m[10]*m[15]-m[14]*m[11])-m[9]*(m[6]*m[15]-m[14]*m[7])+m[13]*(m[6]*m[11]-m[10]*m[7])) -
		m[4]*(m[1]*(m[10]*m[15]-m[14]*m[11])-m[9]*(m[2]*m[15]-m[14]*m[3])+m[13]*(m[2]*m[11]-m[10]*m[3])) +
		m[8]*(m[1]*(m[6]*m[15]-m[14]*m[7])-m[5]*(m[2]*m[15]-m[14]*m[3])+m[13]*(m[2]*m[7]-m[6]*m[3])) -
		m[12]*(m[1]*(m[6]*m[11]-m[10]*m[7])-m[5]*(m[2]*m[11]-m[10]*m[3])+m[9]*(m[2]*m[7]-m[6]*m[3]))
}

// Inverse returns the inverse of the matrix.
// Returns identity if the matrix is singular (det=0).
func (m Mat4) Inverse() Mat4 {
	det := m.Determinant()
	if det == 0 {
		return Identity()
	}

	invDet := 1.0 / det
	var inv Mat4

	inv[0] = (m[5]*(m[10]*m[15]-m[14]*m[11]) - m[9]*(m[6]*m[15]-m[14]*m[7]) + m[13]*(m[6]*m[11]-m[10]*m[7])) * invDet
	inv[1] = -(m[1]*(m[10]*m[15]-m[14]*m[11]) - m[9]*(m[2]*m[15]-m[14]*m[3]) + m[13]*(m[2]*m[11]-m[10]*m[3])) * invDet
	inv[2] = (m[1]*(m[6]*m[15]-m[14]*m[7]) - m[5]*(m[2]*m[15]-m[14]*m[3]) + m[13]*(m[2]*m[7]-m[6]*m[3])) * invDet
	inv[3] = -(m[1]*(m[6]*m[11]-m[10]*m[7]) - m[5]*(m[2]*m[11]-m[10]*m[3]) + m[9]*(m[2]*m[7]-m[6]*m[3])) * invDet

	inv[4] = -(m[4]*(m[10]*m[15]-m[14]*m[11]) - m[8]*(m[6]*m[15]-m[14]*m[7]) + m[12]*(m[6]*m[11]-m[10]*m[7])) * invDet
	inv[5] = (m[0]*(m[10]*m[15]-m[14]*m[11]) - m[8]*(m[2]*m[15]-m[14]*m[3]) + m[12]*(m[2]*m[11]-m[10]*m[3])) * invDet
	inv[6] = -(m[0]*(m[6]*m[15]-m[14]*m[7]) - m[4]*(m[2]*m[15]-m[14]*m[3]) + m[12]*(m[2]*m[7]-m[6]*m[3])) * invDet
	inv[7] = (m[0]*(m[6]*m[11]-m[10]*m[7]) - m[4]*(m[2]*m[11]-m[10]*m[3]) + m[8]*(m[2]*m[7]-m[6]*m[3])) * invDet

	inv[8] = (m[4]*(m[9]*m[15]-m[13]*m[11]) - m[8]*(m[5]*m[15]-m[13]*m[7]) + m[12]*(m[5]*m[11]-m[9]*m[7])) * invDet
	inv[9] = -(m[0]*(m[9]*m[15]-m[13]*m[11]) - m[8]*(m[1]*m[15]-m[13]*m[3]) + m[12]*(m[1]*m[11]-m[9]*m[3])) * invDet
	inv[10] = (m[0]*(m[5]*m[15]-m[13]*m[7]) - m[4]*(m[1]*m[15]-m[13]*m[3]) + m[12]*(m[1]*m[7]-m[5]*m[3])) * invDet
	inv[11] = -(m[0]*(m[5]*m[11]-m[9]*m[7]) - m[4]*(m[1]*m[11]-m[9]*m[3]) + m[8]*(m[1]*m[7]-m[5]*m[3])) * invDet

	inv[12] = -(m[4]*(m[9]*m[14]-m[13]*m[10]) - m[8]*(m[5]*m[14]-m[13]*m[6]) + m[12]*(m[5]*m[10]-m[9]*m[6])) * invDet
	inv[13] = (m[0]*(m[9]*m[14]-m[13]*m[10]) - m[8]*(m[1]*m[14]-m[13]*m[2]) + m[12]*(m[1]*m[10]-m[9]*m[2])) * invDet
	inv[14] = -(m[0]*(m[5]*m[14]-m[13]*m[6]) - m[4]*(m[1]*m[14]-m[13]*m[2]) + m[12]*(m[1]*m[6]-m[5]*m[2])) * invDet
	inv[15] = (m[0]*(m[5]*m[10]-m[9]*m[6]) - m[4]*(m[1]*m[10]-m[9]*m[2]) + m[8]*(m[1]*m[6]-m[5]*m[2])) * invDet

	return inv
}

// Get returns the element at (row, col).
func (m Mat4) Get(row, col int) float64 {
	return m[row+col*4]
}

// Set sets the element at (row, col).
func (m *Mat4) Set(row, col int, val float64) {
	m[row+col*4] = val
}

// Translation extracts the translation component.
func (m Mat4) Translation() Vec3 {
	return Vec3{m[12], m[13], m[14]}
}

// SetTranslation sets the translation component.
func (m *Mat4) SetTranslation(v Vec3) {
	m[12] = v.X
	m[13] = v.Y
	m[14] = v.Z
}
