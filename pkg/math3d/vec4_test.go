package math3d

import (
	"testing"
)

func TestV4(t *testing.T) {
	v := V4(1, 2, 3, 4)
	if v.X != 1 || v.Y != 2 || v.Z != 3 || v.W != 4 {
		t.Fatalf("got %v", v)
	}
}

func TestV4FromV3(t *testing.T) {
	v := V4FromV3(V3(1, 2, 3), 1)
	if v != (Vec4{1, 2, 3, 1}) {
		t.Fatalf("got %v", v)
	}
}

func TestVec4ToVec3(t *testing.T) {
	v := V4(1, 2, 3, 4).Vec3()
	if v != (Vec3{1, 2, 3}) {
		t.Fatalf("got %v", v)
	}
}

func TestVec4PerspectiveDivide(t *testing.T) {
	r := V4(2, 4, 6, 2).PerspectiveDivide()
	if r != (Vec3{1, 2, 3}) {
		t.Fatalf("got %v", r)
	}
	// W=0 case
	r = V4(1, 2, 3, 0).PerspectiveDivide()
	if r != (Vec3{1, 2, 3}) {
		t.Fatalf("W=0: got %v", r)
	}
}

func TestVec4Ops(t *testing.T) {
	a, b := V4(1, 2, 3, 4), V4(5, 6, 7, 8)
	if r := a.Add(b); r != (Vec4{6, 8, 10, 12}) {
		t.Errorf("Add: %v", r)
	}
	if r := a.Sub(b); r != (Vec4{-4, -4, -4, -4}) {
		t.Errorf("Sub: %v", r)
	}
	if r := a.Scale(2); r != (Vec4{2, 4, 6, 8}) {
		t.Errorf("Scale: %v", r)
	}
	if d := a.Dot(b); d != 70 {
		t.Errorf("Dot: %f", d)
	}
}

func TestVec4LenAndNormalize(t *testing.T) {
	v := V4(1, 0, 0, 0)
	if !approxEqual(v.Len(), 1, epsilon) {
		t.Fatalf("Len = %f", v.Len())
	}
	n := V4(3, 4, 0, 0).Normalize()
	if !approxEqual(n.Len(), 1, epsilon) {
		t.Fatalf("normalized len = %f", n.Len())
	}
	z := Vec4{}.Normalize()
	if z != (Vec4{}) {
		t.Fatalf("zero normalize = %v", z)
	}
}

func TestVec4Lerp(t *testing.T) {
	a, b := V4(0, 0, 0, 0), V4(10, 20, 30, 40)
	r := a.Lerp(b, 0.5)
	if r != (Vec4{5, 10, 15, 20}) {
		t.Fatalf("got %v", r)
	}
}
