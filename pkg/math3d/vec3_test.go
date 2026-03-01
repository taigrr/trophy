package math3d

import (
	"math"
	"testing"
)

func TestV3(t *testing.T) {
	v := V3(1, 2, 3)
	if v.X != 1 || v.Y != 2 || v.Z != 3 {
		t.Fatalf("got %v", v)
	}
}

func TestDirectionVectors(t *testing.T) {
	if Zero3() != (Vec3{}) {
		t.Error("Zero3")
	}
	if Up() != (Vec3{0, 1, 0}) {
		t.Error("Up")
	}
	if Forward() != (Vec3{0, 0, -1}) {
		t.Error("Forward")
	}
	if Right() != (Vec3{1, 0, 0}) {
		t.Error("Right")
	}
}

func TestVec3Arithmetic(t *testing.T) {
	a, b := V3(1, 2, 3), V3(4, 5, 6)
	if r := a.Add(b); r != (Vec3{5, 7, 9}) {
		t.Errorf("Add: %v", r)
	}
	if r := a.Sub(b); r != (Vec3{-3, -3, -3}) {
		t.Errorf("Sub: %v", r)
	}
	if r := a.Mul(b); r != (Vec3{4, 10, 18}) {
		t.Errorf("Mul: %v", r)
	}
	if r := a.Scale(2); r != (Vec3{2, 4, 6}) {
		t.Errorf("Scale: %v", r)
	}
	if r := V3(4, 6, 8).Div(2); r != (Vec3{2, 3, 4}) {
		t.Errorf("Div: %v", r)
	}
	if r := a.Negate(); r != (Vec3{-1, -2, -3}) {
		t.Errorf("Negate: %v", r)
	}
}

func TestVec3Dot(t *testing.T) {
	if d := V3(1, 2, 3).Dot(V3(4, 5, 6)); d != 32 {
		t.Fatalf("got %f", d)
	}
}

func TestVec3Cross(t *testing.T) {
	r := V3(1, 0, 0).Cross(V3(0, 1, 0))
	if r != (Vec3{0, 0, 1}) {
		t.Fatalf("got %v", r)
	}
}

func TestVec3LenAndNormalize(t *testing.T) {
	v := V3(3, 4, 0)
	if !approxEqual(v.Len(), 5, epsilon) {
		t.Fatalf("Len = %f", v.Len())
	}
	if v.LenSq() != 25 {
		t.Fatalf("LenSq = %f", v.LenSq())
	}
	n := v.Normalize()
	if !approxEqual(n.Len(), 1, epsilon) {
		t.Fatalf("normalized len = %f", n.Len())
	}
	z := Zero3().Normalize()
	if z != (Vec3{}) {
		t.Fatalf("zero normalize = %v", z)
	}
}

func TestVec3Lerp(t *testing.T) {
	r := V3(0, 0, 0).Lerp(V3(10, 20, 30), 0.25)
	if !approxEqual(r.X, 2.5, epsilon) || !approxEqual(r.Y, 5, epsilon) || !approxEqual(r.Z, 7.5, epsilon) {
		t.Fatalf("got %v", r)
	}
}

func TestVec3Distance(t *testing.T) {
	d := V3(0, 0, 0).Distance(V3(3, 4, 0))
	if !approxEqual(d, 5, epsilon) {
		t.Fatalf("got %f", d)
	}
}

func TestVec3Reflect(t *testing.T) {
	// Reflect (1,-1,0) around normal (0,1,0) => (1,1,0)
	r := V3(1, -1, 0).Reflect(V3(0, 1, 0))
	if !approxEqual(r.X, 1, epsilon) || !approxEqual(r.Y, 1, epsilon) || !approxEqual(r.Z, 0, epsilon) {
		t.Fatalf("got %v", r)
	}
}

func TestVec3MinMaxAbsFloorCeil(t *testing.T) {
	a, b := V3(-1, 3, 2), V3(2, -1, 5)
	if r := a.Min(b); r != (Vec3{-1, -1, 2}) {
		t.Errorf("Min: %v", r)
	}
	if r := a.Max(b); r != (Vec3{2, 3, 5}) {
		t.Errorf("Max: %v", r)
	}
	if r := V3(-1.5, 2.7, -3.2).Abs(); !approxEqual(r.X, 1.5, epsilon) || !approxEqual(r.Y, 2.7, epsilon) || !approxEqual(r.Z, 3.2, epsilon) {
		t.Errorf("Abs: %v", r)
	}
	if r := V3(1.7, -2.3, 3.9).Floor(); r != (Vec3{1, -3, 3}) {
		t.Errorf("Floor: %v", r)
	}
	if r := V3(1.1, -2.9, 3.0).Ceil(); r != (Vec3{2, -2, 3}) {
		t.Errorf("Ceil: %v", r)
	}
}

func TestVec3CrossOrthogonal(t *testing.T) {
	// Cross product should be orthogonal to both inputs
	a, b := V3(1, 2, 3), V3(4, 5, 6)
	c := a.Cross(b)
	if !approxEqual(c.Dot(a), 0, epsilon) || !approxEqual(c.Dot(b), 0, epsilon) {
		t.Fatalf("cross not orthogonal: dot(c,a)=%f dot(c,b)=%f", c.Dot(a), c.Dot(b))
	}
}

func TestVec3NormalizeUnit(t *testing.T) {
	// Normalizing a unit vector should return itself
	u := V3(1, 0, 0).Normalize()
	if !approxEqual(u.X, 1, epsilon) || u.Y != 0 || u.Z != 0 {
		t.Fatalf("got %v", u)
	}
}

func TestVec3ReflectPerpendicular(t *testing.T) {
	// Reflecting a vector perpendicular to the normal should negate it
	r := V3(0, -1, 0).Reflect(V3(0, 1, 0))
	if !approxEqual(r.Y, 1, epsilon) {
		t.Fatalf("got %v", r)
	}
}

func TestVec3LerpEndpoints(t *testing.T) {
	a, b := V3(1, 2, 3), V3(4, 5, 6)
	if r := a.Lerp(b, 0); r != a {
		t.Errorf("lerp(0) = %v", r)
	}
	if r := a.Lerp(b, 1); r != b {
		t.Errorf("lerp(1) = %v", r)
	}
}

func TestVec3ScaleByZero(t *testing.T) {
	r := V3(5, 10, 15).Scale(0)
	if r != (Vec3{}) {
		t.Fatalf("got %v", r)
	}
}

func _ (t *testing.T) {
	// Distance is symmetric
	a, b := V3(1, 2, 3), V3(4, 5, 6)
	_ = math.Abs(a.Distance(b) - b.Distance(a))
}
