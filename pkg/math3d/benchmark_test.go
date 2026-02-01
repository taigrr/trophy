package math3d

import (
	"testing"
)

func BenchmarkMat4Mul(b *testing.B) {
	m1 := Translate(V3(1, 2, 3))
	m2 := RotateY(0.5)

	for b.Loop() {
		_ = m1.Mul(m2)
	}
}

func BenchmarkMat4MulVec4(b *testing.B) {
	m := Translate(V3(1, 2, 3)).Mul(RotateY(0.5))
	v := V4(1, 2, 3, 1)

	for b.Loop() {
		_ = m.MulVec4(v)
	}
}

func BenchmarkMat4MulVec3(b *testing.B) {
	m := Translate(V3(1, 2, 3)).Mul(RotateY(0.5))
	v := V3(1, 2, 3)

	for b.Loop() {
		_ = m.MulVec3(v)
	}
}

func BenchmarkMat4Inverse(b *testing.B) {
	m := Translate(V3(1, 2, 3)).Mul(RotateY(0.5)).Mul(Scale(V3(2, 2, 2)))

	for b.Loop() {
		_ = m.Inverse()
	}
}

func BenchmarkVec3Normalize(b *testing.B) {
	v := V3(1, 2, 3)

	for b.Loop() {
		_ = v.Normalize()
	}
}

func BenchmarkVec3Cross(b *testing.B) {
	v1 := V3(1, 2, 3)
	v2 := V3(4, 5, 6)

	for b.Loop() {
		_ = v1.Cross(v2)
	}
}

func BenchmarkVec3Dot(b *testing.B) {
	v1 := V3(1, 2, 3)
	v2 := V3(4, 5, 6)

	for b.Loop() {
		_ = v1.Dot(v2)
	}
}

func BenchmarkPerspective(b *testing.B) {
	for b.Loop() {
		_ = Perspective(60.0, 1.333, 0.1, 100.0)
	}
}

func BenchmarkLookAt(b *testing.B) {
	eye := V3(0, 0, 10)
	target := V3(0, 0, 0)
	up := V3(0, 1, 0)

	for b.Loop() {
		_ = LookAt(eye, target, up)
	}
}

func BenchmarkViewProjection(b *testing.B) {
	// Simulate building view-projection matrix like the rasterizer does
	eye := V3(0, 0, 10)
	target := V3(0, 0, 0)
	up := V3(0, 1, 0)
	view := LookAt(eye, target, up)
	proj := Perspective(60.0, 1.333, 0.1, 100.0)

	for b.Loop() {
		_ = proj.Mul(view)
	}
}
