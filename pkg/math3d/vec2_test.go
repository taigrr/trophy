package math3d

import (
	"math"
	"testing"
)

const epsilon = 1e-10

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestV2(t *testing.T) {
	v := V2(3, 4)
	if v.X != 3 || v.Y != 4 {
		t.Fatalf("V2(3,4) = %v", v)
	}
}

func TestZero2(t *testing.T) {
	v := Zero2()
	if v.X != 0 || v.Y != 0 {
		t.Fatalf("Zero2() = %v", v)
	}
}

func TestVec2Ops(t *testing.T) {
	tests := []struct {
		name string
		got  Vec2
		want Vec2
	}{
		{"Add", V2(1, 2).Add(V2(3, 4)), V2(4, 6)},
		{"Sub", V2(5, 7).Sub(V2(2, 3)), V2(3, 4)},
		{"Scale", V2(2, 3).Scale(3), V2(6, 9)},
		{"Mul", V2(2, 3).Mul(V2(4, 5)), V2(8, 15)},
		{"Negate", V2(1, -2).Negate(), V2(-1, 2)},
		{"Perpendicular", V2(1, 0).Perpendicular(), V2(0, 1)},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestVec2Dot(t *testing.T) {
	if d := V2(1, 2).Dot(V2(3, 4)); d != 11 {
		t.Fatalf("got %f", d)
	}
}

func TestVec2Len(t *testing.T) {
	if l := V2(3, 4).Len(); !approxEqual(l, 5, epsilon) {
		t.Fatalf("got %f", l)
	}
	if l := V2(3, 4).LenSq(); l != 25 {
		t.Fatalf("got %f", l)
	}
}

func TestVec2Normalize(t *testing.T) {
	n := V2(3, 4).Normalize()
	if !approxEqual(n.Len(), 1, epsilon) {
		t.Fatalf("len = %f", n.Len())
	}
	z := Zero2().Normalize()
	if z != (Vec2{}) {
		t.Fatalf("zero normalize = %v", z)
	}
}

func TestVec2Lerp(t *testing.T) {
	r := V2(0, 0).Lerp(V2(10, 20), 0.5)
	if r.X != 5 || r.Y != 10 {
		t.Fatalf("got %v", r)
	}
}

func TestVec2Rotate(t *testing.T) {
	r := V2(1, 0).Rotate(math.Pi / 2)
	if !approxEqual(r.X, 0, epsilon) || !approxEqual(r.Y, 1, epsilon) {
		t.Fatalf("got %v", r)
	}
}

func TestVec2Angle(t *testing.T) {
	if a := V2(1, 0).Angle(); !approxEqual(a, 0, epsilon) {
		t.Fatalf("got %f", a)
	}
	if a := V2(0, 1).Angle(); !approxEqual(a, math.Pi/2, epsilon) {
		t.Fatalf("got %f", a)
	}
}

func TestVec2AngleTo(t *testing.T) {
	a := V2(0, 0).AngleTo(V2(1, 1))
	if !approxEqual(a, math.Pi/4, epsilon) {
		t.Fatalf("got %f", a)
	}
}

func TestVec2Distance(t *testing.T) {
	if d := V2(0, 0).Distance(V2(3, 4)); !approxEqual(d, 5, epsilon) {
		t.Fatalf("got %f", d)
	}
}
