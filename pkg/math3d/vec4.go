package math3d

import "math"

// Vec4 represents a 4D vector (or homogeneous 3D point).
type Vec4 struct {
	X, Y, Z, W float64
}

// V4 creates a new Vec4.
func V4(x, y, z, w float64) Vec4 {
	return Vec4{x, y, z, w}
}

// V4FromV3 creates a Vec4 from Vec3 with specified W.
func V4FromV3(v Vec3, w float64) Vec4 {
	return Vec4{v.X, v.Y, v.Z, w}
}

// Vec3 returns the Vec3 portion (ignoring W).
func (v Vec4) Vec3() Vec3 {
	return Vec3{v.X, v.Y, v.Z}
}

// PerspectiveDivide returns Vec3 after dividing by W.
func (v Vec4) PerspectiveDivide() Vec3 {
	if v.W == 0 {
		return Vec3{v.X, v.Y, v.Z}
	}
	return Vec3{v.X / v.W, v.Y / v.W, v.Z / v.W}
}

// Add returns the vector sum.
//
//nolint:st1016 // a+b naming convention is clearer for vector operations
func (a Vec4) Add(b Vec4) Vec4 {
	return Vec4{a.X + b.X, a.Y + b.Y, a.Z + b.Z, a.W + b.W}
}

// Sub returns the vector difference.
//
//nolint:st1016 // a-b naming convention is clearer for vector operations
func (a Vec4) Sub(b Vec4) Vec4 {
	return Vec4{a.X - b.X, a.Y - b.Y, a.Z - b.Z, a.W - b.W}
}

// Scale returns the scalar product.
func (v Vec4) Scale(s float64) Vec4 {
	return Vec4{v.X * s, v.Y * s, v.Z * s, v.W * s}
}

// Dot returns the dot product.
//
//nolint:st1016 // a·b naming convention is clearer for vector operations
func (a Vec4) Dot(b Vec4) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z + a.W*b.W
}

// Len returns the length.
func (v Vec4) Len() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z + v.W*v.W)
}

// Normalize returns the unit vector.
func (v Vec4) Normalize() Vec4 {
	l := v.Len()
	if l == 0 {
		return Vec4{}
	}
	return Vec4{v.X / l, v.Y / l, v.Z / l, v.W / l}
}

// Lerp returns linear interpolation.
//
//nolint:st1016 // a,b naming convention is clearer for interpolation
func (a Vec4) Lerp(b Vec4, t float64) Vec4 {
	return Vec4{
		a.X + (b.X-a.X)*t,
		a.Y + (b.Y-a.Y)*t,
		a.Z + (b.Z-a.Z)*t,
		a.W + (b.W-a.W)*t,
	}
}
