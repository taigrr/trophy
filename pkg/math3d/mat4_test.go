package math3d

import (
	"math"
	"testing"
)

func matApproxEqual(a, b Mat4, eps float64) bool {
	for i := range 16 {
		if !approxEqual(a[i], b[i], eps) {
			return false
		}
	}
	return true
}

func TestIdentity(t *testing.T) {
	id := Identity()
	for i := range 4 {
		for j := range 4 {
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			if id.Get(i, j) != expected {
				t.Fatalf("Identity[%d,%d] = %f", i, j, id.Get(i, j))
			}
		}
	}
}

func TestMat4MulIdentity(t *testing.T) {
	m := Translate(V3(1, 2, 3))
	r := m.Mul(Identity())
	if !matApproxEqual(r, m, epsilon) {
		t.Fatalf("M*I != M")
	}
	r = Identity().Mul(m)
	if !matApproxEqual(r, m, epsilon) {
		t.Fatalf("I*M != M")
	}
}

func TestTranslate(t *testing.T) {
	m := Translate(V3(5, 10, 15))
	p := m.MulVec3(Zero3())
	if !approxEqual(p.X, 5, epsilon) || !approxEqual(p.Y, 10, epsilon) || !approxEqual(p.Z, 15, epsilon) {
		t.Fatalf("got %v", p)
	}
}

func TestScale(t *testing.T) {
	m := Scale(V3(2, 3, 4))
	p := m.MulVec3(V3(1, 1, 1))
	if !approxEqual(p.X, 2, epsilon) || !approxEqual(p.Y, 3, epsilon) || !approxEqual(p.Z, 4, epsilon) {
		t.Fatalf("got %v", p)
	}
}

func TestScaleUniform(t *testing.T) {
	m := ScaleUniform(3)
	p := m.MulVec3(V3(1, 2, 3))
	if !approxEqual(p.X, 3, epsilon) || !approxEqual(p.Y, 6, epsilon) || !approxEqual(p.Z, 9, epsilon) {
		t.Fatalf("got %v", p)
	}
}

func TestRotateX(t *testing.T) {
	m := RotateX(math.Pi / 2)
	p := m.MulVec3(V3(0, 1, 0))
	if !approxEqual(p.X, 0, epsilon) || !approxEqual(p.Y, 0, epsilon) || !approxEqual(p.Z, 1, epsilon) {
		t.Fatalf("got %v", p)
	}
}

func TestRotateY(t *testing.T) {
	m := RotateY(math.Pi / 2)
	p := m.MulVec3(V3(1, 0, 0))
	if !approxEqual(p.X, 0, epsilon) || !approxEqual(p.Y, 0, epsilon) || !approxEqual(p.Z, -1, epsilon) {
		t.Fatalf("got %v", p)
	}
}

func TestRotateZ(t *testing.T) {
	m := RotateZ(math.Pi / 2)
	p := m.MulVec3(V3(1, 0, 0))
	if !approxEqual(p.X, 0, epsilon) || !approxEqual(p.Y, 1, epsilon) || !approxEqual(p.Z, 0, epsilon) {
		t.Fatalf("got %v", p)
	}
}

func TestRotateArbitrary(t *testing.T) {
	// Rotating around Y axis should match RotateY
	m1 := RotateY(math.Pi / 4)
	m2 := Rotate(V3(0, 1, 0), math.Pi/4)
	if !matApproxEqual(m1, m2, epsilon) {
		t.Fatalf("Rotate(Y) != RotateY")
	}
}

func TestMulVec3Dir(t *testing.T) {
	// Direction should ignore translation
	m := Translate(V3(100, 200, 300))
	d := m.MulVec3Dir(V3(1, 0, 0))
	if !approxEqual(d.X, 1, epsilon) || !approxEqual(d.Y, 0, epsilon) || !approxEqual(d.Z, 0, epsilon) {
		t.Fatalf("got %v", d)
	}
}

func TestMulVec4(t *testing.T) {
	m := Translate(V3(1, 2, 3))
	v := m.MulVec4(V4(0, 0, 0, 1))
	if !approxEqual(v.X, 1, epsilon) || !approxEqual(v.Y, 2, epsilon) || !approxEqual(v.Z, 3, epsilon) || !approxEqual(v.W, 1, epsilon) {
		t.Fatalf("got %v", v)
	}
}

func TestTranspose(t *testing.T) {
	m := Translate(V3(1, 2, 3))
	tt := m.Transpose().Transpose()
	if !matApproxEqual(m, tt, epsilon) {
		t.Fatal("double transpose != original")
	}
}

func TestDeterminantIdentity(t *testing.T) {
	if d := Identity().Determinant(); !approxEqual(d, 1, epsilon) {
		t.Fatalf("got %f", d)
	}
}

func TestDeterminantScale(t *testing.T) {
	m := Scale(V3(2, 3, 4))
	if d := m.Determinant(); !approxEqual(d, 24, epsilon) {
		t.Fatalf("got %f", d)
	}
}

func TestInverse(t *testing.T) {
	m := Translate(V3(5, 10, 15)).Mul(RotateY(0.7)).Mul(Scale(V3(2, 3, 4)))
	inv := m.Inverse()
	product := m.Mul(inv)
	if !matApproxEqual(product, Identity(), 1e-6) {
		t.Fatal("M * M^-1 != I")
	}
}

func TestInverseSingular(t *testing.T) {
	// Singular matrix (all zeros) should return identity
	var zero Mat4
	inv := zero.Inverse()
	if !matApproxEqual(inv, Identity(), epsilon) {
		t.Fatal("singular inverse != identity")
	}
}

func TestGetSet(t *testing.T) {
	var m Mat4
	m.Set(2, 3, 42)
	if m.Get(2, 3) != 42 {
		t.Fatalf("got %f", m.Get(2, 3))
	}
}

func TestTranslationExtractAndSet(t *testing.T) {
	m := Translate(V3(1, 2, 3))
	if tr := m.Translation(); tr != (Vec3{1, 2, 3}) {
		t.Fatalf("got %v", tr)
	}
	m.SetTranslation(V3(9, 8, 7))
	if tr := m.Translation(); tr != (Vec3{9, 8, 7}) {
		t.Fatalf("got %v", tr)
	}
}

func TestMat4FromSlice(t *testing.T) {
	s := make([]float64, 16)
	s[0] = 1
	s[5] = 1
	s[10] = 1
	s[15] = 1
	m := Mat4FromSlice(s)
	if !matApproxEqual(m, Identity(), epsilon) {
		t.Fatal("Mat4FromSlice != Identity")
	}
	// Short slice
	short := Mat4FromSlice([]float64{5, 6})
	if short[0] != 5 || short[1] != 6 || short[2] != 0 {
		t.Fatalf("short: %v", short)
	}
}

func TestQuatToMat4Identity(t *testing.T) {
	// Identity quaternion (0,0,0,1) should produce identity rotation
	m := QuatToMat4(0, 0, 0, 1)
	if !matApproxEqual(m, Identity(), epsilon) {
		t.Fatal("identity quat != identity matrix")
	}
}

func TestQuatToMat490Y(t *testing.T) {
	// 90 degree rotation around Y
	angle := math.Pi / 2
	q := math.Sin(angle / 2)
	m := QuatToMat4(0, q, 0, math.Cos(angle/2))
	p := m.MulVec3(V3(1, 0, 0))
	if !approxEqual(p.X, 0, 1e-6) || !approxEqual(p.Z, -1, 1e-6) {
		t.Fatalf("got %v", p)
	}
}

func TestLookAt(t *testing.T) {
	eye := V3(0, 0, 5)
	center := V3(0, 0, 0)
	up := V3(0, 1, 0)
	m := LookAt(eye, center, up)
	// Origin should map to (0, 0, -5) in view space (in front of camera)
	p := m.MulVec3(Zero3())
	if !approxEqual(p.Z, -5, 1e-6) {
		t.Fatalf("got z=%f, want -5", p.Z)
	}
}

func TestPerspective(t *testing.T) {
	m := Perspective(math.Pi/3, 16.0/9.0, 0.1, 1000)
	// Should not be identity
	if matApproxEqual(m, Identity(), epsilon) {
		t.Fatal("perspective == identity")
	}
	// m[15] should be 0 for perspective
	if m[15] != 0 {
		t.Fatalf("m[15] = %f", m[15])
	}
}

func TestOrthographic(t *testing.T) {
	m := Orthographic(-1, 1, -1, 1, 0.1, 100)
	if matApproxEqual(m, Identity(), epsilon) {
		t.Fatal("orthographic == identity")
	}
	// m[15] should be 1 for orthographic
	if !approxEqual(m[15], 1, epsilon) {
		t.Fatalf("m[15] = %f", m[15])
	}
}

func TestMulVec3W0(t *testing.T) {
	// When w computes to 0, should default to 1
	var m Mat4 // all zeros, w will be 0
	m[0] = 1
	m[5] = 1
	m[10] = 1
	// m[15] = 0, so w = 0 for any input
	p := m.MulVec3(V3(1, 2, 3))
	if p.X != 1 || p.Y != 2 || p.Z != 3 {
		t.Fatalf("got %v", p)
	}
}
