// Package math3d provides 3D math primitives for the TuiKart engine.
package math3d

import "math"

// Vec3 represents a 3D vector.
type Vec3 struct {
	X, Y, Z float64
}

// V3 creates a new Vec3.
func V3(x, y, z float64) Vec3 {
	return Vec3{x, y, z}
}

// Zero3 returns the zero vector.
func Zero3() Vec3 {
	return Vec3{}
}

// Up returns the world up vector (0, 1, 0).
func Up() Vec3 {
	return Vec3{0, 1, 0}
}

// Forward returns the world forward vector (0, 0, -1).
func Forward() Vec3 {
	return Vec3{0, 0, -1}
}

// Right returns the world right vector (1, 0, 0).
func Right() Vec3 {
	return Vec3{1, 0, 0}
}

// Add returns the vector sum a + b.
func (a Vec3) Add(b Vec3) Vec3 {
	return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z}
}

// Sub returns the vector difference a - b.
func (a Vec3) Sub(b Vec3) Vec3 {
	return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z}
}

// Mul returns the component-wise product a * b.
func (a Vec3) Mul(b Vec3) Vec3 {
	return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z}
}

// Scale returns the scalar product a * s.
func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

// Div returns the scalar division a / s.
func (a Vec3) Div(s float64) Vec3 {
	return Vec3{a.X / s, a.Y / s, a.Z / s}
}

// Dot returns the dot product a · b.
func (a Vec3) Dot(b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

// Cross returns the cross product a × b.
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

// Len returns the length (magnitude) of the vector.
func (a Vec3) Len() float64 {
	return math.Sqrt(a.X*a.X + a.Y*a.Y + a.Z*a.Z)
}

// LenSq returns the squared length (faster, no sqrt).
func (a Vec3) LenSq() float64 {
	return a.X*a.X + a.Y*a.Y + a.Z*a.Z
}

// Normalize returns the unit vector in the same direction.
func (a Vec3) Normalize() Vec3 {
	l := a.Len()
	if l == 0 {
		return Vec3{}
	}
	return Vec3{a.X / l, a.Y / l, a.Z / l}
}

// Negate returns the negated vector.
func (a Vec3) Negate() Vec3 {
	return Vec3{-a.X, -a.Y, -a.Z}
}

// Lerp returns the linear interpolation between a and b by t.
func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return Vec3{
		a.X + (b.X-a.X)*t,
		a.Y + (b.Y-a.Y)*t,
		a.Z + (b.Z-a.Z)*t,
	}
}

// Distance returns the distance between two points.
func (a Vec3) Distance(b Vec3) float64 {
	return a.Sub(b).Len()
}

// Reflect returns the reflection of a around normal n.
func (a Vec3) Reflect(n Vec3) Vec3 {
	return a.Sub(n.Scale(2 * a.Dot(n)))
}

// Min returns the component-wise minimum.
func (a Vec3) Min(b Vec3) Vec3 {
	return Vec3{
		math.Min(a.X, b.X),
		math.Min(a.Y, b.Y),
		math.Min(a.Z, b.Z),
	}
}

// Max returns the component-wise maximum.
func (a Vec3) Max(b Vec3) Vec3 {
	return Vec3{
		math.Max(a.X, b.X),
		math.Max(a.Y, b.Y),
		math.Max(a.Z, b.Z),
	}
}

// Abs returns the component-wise absolute value.
func (a Vec3) Abs() Vec3 {
	return Vec3{
		math.Abs(a.X),
		math.Abs(a.Y),
		math.Abs(a.Z),
	}
}

// Floor returns the component-wise floor.
func (a Vec3) Floor() Vec3 {
	return Vec3{
		math.Floor(a.X),
		math.Floor(a.Y),
		math.Floor(a.Z),
	}
}

// Ceil returns the component-wise ceiling.
func (a Vec3) Ceil() Vec3 {
	return Vec3{
		math.Ceil(a.X),
		math.Ceil(a.Y),
		math.Ceil(a.Z),
	}
}
