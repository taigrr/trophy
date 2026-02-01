package render

import (
	"math"
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

func TestPlaneDistanceToPoint(t *testing.T) {
	// Plane at Z=0, normal pointing +Z
	plane := Plane{Normal: math3d.V3(0, 0, 1), D: 0}

	tests := []struct {
		name     string
		point    math3d.Vec3
		expected float64
	}{
		{"origin", math3d.V3(0, 0, 0), 0},
		{"in front", math3d.V3(0, 0, 5), 5},
		{"behind", math3d.V3(0, 0, -3), -3},
		{"offset XY", math3d.V3(10, -5, 2), 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dist := plane.DistanceToPoint(tc.point)
			if math.Abs(dist-tc.expected) > 1e-9 {
				t.Errorf("got %v, want %v", dist, tc.expected)
			}
		})
	}
}

func TestPlaneNormalize(t *testing.T) {
	plane := Plane{Normal: math3d.V3(0, 3, 4), D: 10}
	plane.Normalize()

	// Normal should have length 1
	length := plane.Normal.Len()
	if math.Abs(length-1.0) > 1e-9 {
		t.Errorf("normalized normal length = %v, want 1.0", length)
	}

	// Check components (3/5, 4/5)
	if math.Abs(plane.Normal.Y-0.6) > 1e-9 {
		t.Errorf("normal.Y = %v, want 0.6", plane.Normal.Y)
	}
	if math.Abs(plane.Normal.Z-0.8) > 1e-9 {
		t.Errorf("normal.Z = %v, want 0.8", plane.Normal.Z)
	}

	// D should be scaled too (10/5 = 2)
	if math.Abs(plane.D-2.0) > 1e-9 {
		t.Errorf("D = %v, want 2.0", plane.D)
	}
}

func TestAABBBasics(t *testing.T) {
	box := NewAABB(math3d.V3(-1, -2, -3), math3d.V3(1, 2, 3))

	center := box.Center()
	if center.X != 0 || center.Y != 0 || center.Z != 0 {
		t.Errorf("center = %v, want (0, 0, 0)", center)
	}

	size := box.Size()
	if size.X != 2 || size.Y != 4 || size.Z != 6 {
		t.Errorf("size = %v, want (2, 4, 6)", size)
	}

	halfSize := box.HalfSize()
	if halfSize.X != 1 || halfSize.Y != 2 || halfSize.Z != 3 {
		t.Errorf("halfSize = %v, want (1, 2, 3)", halfSize)
	}
}

func TestAABBContainsPoint(t *testing.T) {
	box := NewAABB(math3d.V3(0, 0, 0), math3d.V3(10, 10, 10))

	tests := []struct {
		name     string
		point    math3d.Vec3
		expected bool
	}{
		{"center", math3d.V3(5, 5, 5), true},
		{"corner min", math3d.V3(0, 0, 0), true},
		{"corner max", math3d.V3(10, 10, 10), true},
		{"edge", math3d.V3(5, 0, 5), true},
		{"outside X", math3d.V3(11, 5, 5), false},
		{"outside Y", math3d.V3(5, -1, 5), false},
		{"outside Z", math3d.V3(5, 5, 15), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := box.ContainsPoint(tc.point)
			if result != tc.expected {
				t.Errorf("ContainsPoint(%v) = %v, want %v", tc.point, result, tc.expected)
			}
		})
	}
}

func TestAABBTransform(t *testing.T) {
	box := NewAABB(math3d.V3(-1, -1, -1), math3d.V3(1, 1, 1))

	// Test translation
	t.Run("translation", func(t *testing.T) {
		trans := math3d.Translate(math3d.V3(10, 20, 30))
		transformed := box.Transform(trans)

		if transformed.Min.X != 9 || transformed.Min.Y != 19 || transformed.Min.Z != 29 {
			t.Errorf("translated min = %v, want (9, 19, 29)", transformed.Min)
		}
		if transformed.Max.X != 11 || transformed.Max.Y != 21 || transformed.Max.Z != 31 {
			t.Errorf("translated max = %v, want (11, 21, 31)", transformed.Max)
		}
	})

	// Test uniform scale
	t.Run("scale", func(t *testing.T) {
		scale := math3d.ScaleUniform(2.0)
		transformed := box.Transform(scale)

		if transformed.Min.X != -2 || transformed.Min.Y != -2 || transformed.Min.Z != -2 {
			t.Errorf("scaled min = %v, want (-2, -2, -2)", transformed.Min)
		}
		if transformed.Max.X != 2 || transformed.Max.Y != 2 || transformed.Max.Z != 2 {
			t.Errorf("scaled max = %v, want (2, 2, 2)", transformed.Max)
		}
	})
}

func TestFrustumFromPerspective(t *testing.T) {
	// Create a typical perspective projection
	proj := math3d.Perspective(math.Pi/3, 16.0/9.0, 0.1, 100)
	view := math3d.Identity() // Camera at origin looking down -Z
	viewProj := proj.Mul(view)

	frustum := NewFrustumFromMatrix(viewProj)

	// Verify planes are normalized
	for i, plane := range frustum.Planes {
		length := plane.Normal.Len()
		if math.Abs(length-1.0) > 1e-6 {
			t.Errorf("plane %d normal length = %v, want 1.0", i, length)
		}
	}
}

func TestFrustumContainsPoint(t *testing.T) {
	// Create frustum from typical camera setup
	fov := math.Pi / 3 // 60 degrees
	aspect := 16.0 / 9.0
	near := 0.1
	far := 100.0

	proj := math3d.Perspective(fov, aspect, near, far)
	view := math3d.Identity()
	frustum := NewFrustumFromMatrix(proj.Mul(view))

	tests := []struct {
		name     string
		point    math3d.Vec3
		expected bool
	}{
		{"center near", math3d.V3(0, 0, -1), true},
		{"center mid", math3d.V3(0, 0, -50), true},
		{"center far", math3d.V3(0, 0, -99), true},
		{"behind camera", math3d.V3(0, 0, 1), false},
		{"too far", math3d.V3(0, 0, -200), false},
		{"too close", math3d.V3(0, 0, -0.01), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := frustum.ContainsPoint(tc.point)
			if result != tc.expected {
				t.Errorf("ContainsPoint(%v) = %v, want %v", tc.point, result, tc.expected)
			}
		})
	}
}

func TestFrustumIntersectAABB(t *testing.T) {
	// Create frustum from typical camera setup
	fov := math.Pi / 3
	aspect := 16.0 / 9.0
	near := 1.0
	far := 100.0

	proj := math3d.Perspective(fov, aspect, near, far)
	view := math3d.Identity()
	frustum := NewFrustumFromMatrix(proj.Mul(view))

	tests := []struct {
		name     string
		box      AABB
		expected bool
	}{
		{
			"fully inside",
			NewAABB(math3d.V3(-1, -1, -10), math3d.V3(1, 1, -5)),
			true,
		},
		{
			"partially visible",
			NewAABB(math3d.V3(-1, -1, -2), math3d.V3(1, 1, 2)), // Crosses near plane and goes behind
			true,
		},
		{
			"behind camera",
			NewAABB(math3d.V3(-1, -1, 5), math3d.V3(1, 1, 10)),
			false,
		},
		{
			"beyond far plane",
			NewAABB(math3d.V3(-1, -1, -150), math3d.V3(1, 1, -120)),
			false,
		},
		{
			"far to the right",
			NewAABB(math3d.V3(100, -1, -10), math3d.V3(110, 1, -5)),
			false,
		},
		{
			"large box containing frustum",
			NewAABB(math3d.V3(-200, -200, -200), math3d.V3(200, 200, 200)),
			true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := frustum.IntersectAABB(tc.box)
			if result != tc.expected {
				t.Errorf("IntersectAABB(%v) = %v, want %v", tc.box, result, tc.expected)
			}
		})
	}
}

func TestFrustumIntersectsSphere(t *testing.T) {
	// Create frustum
	proj := math3d.Perspective(math.Pi/3, 16.0/9.0, 1.0, 100.0)
	view := math3d.Identity()
	frustum := NewFrustumFromMatrix(proj.Mul(view))

	tests := []struct {
		name     string
		center   math3d.Vec3
		radius   float64
		expected bool
	}{
		{"inside", math3d.V3(0, 0, -10), 1.0, true},
		{"partially visible", math3d.V3(0, 0, -0.5), 1.0, true}, // Near the near plane
		{"behind", math3d.V3(0, 0, 5), 1.0, false},
		{"far behind", math3d.V3(0, 0, 20), 1.0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := frustum.IntersectsSphere(tc.center, tc.radius)
			if result != tc.expected {
				t.Errorf("IntersectsSphere(%v, %v) = %v, want %v", tc.center, tc.radius, result, tc.expected)
			}
		})
	}
}

func TestFrustumWithRotatedCamera(t *testing.T) {
	// Camera at origin looking at a target along +X axis
	proj := math3d.Perspective(math.Pi/3, 1.0, 1.0, 100.0)
	eye := math3d.V3(0, 0, 0)
	target := math3d.V3(10, 0, 0) // Looking along +X
	up := math3d.V3(0, 1, 0)
	view := math3d.LookAt(eye, target, up)
	frustum := NewFrustumFromMatrix(proj.Mul(view))

	// Point in front of camera (along +X at distance 10)
	inFront := math3d.V3(10, 0, 0)
	if !frustum.ContainsPoint(inFront) {
		t.Error("point in front of rotated camera should be visible")
	}

	// Point behind camera (along -X)
	behind := math3d.V3(-10, 0, 0)
	if frustum.ContainsPoint(behind) {
		t.Error("point behind rotated camera should not be visible")
	}
}

func TestExtractFrustumAlias(t *testing.T) {
	// Verify the alias works
	proj := math3d.Perspective(math.Pi/3, 1.0, 1.0, 100.0)
	f1 := NewFrustumFromMatrix(proj)
	f2 := ExtractFrustum(proj)

	// Should produce identical results
	for i := range f1.Planes {
		if f1.Planes[i].Normal != f2.Planes[i].Normal || f1.Planes[i].D != f2.Planes[i].D {
			t.Errorf("plane %d mismatch between NewFrustumFromMatrix and ExtractFrustum", i)
		}
	}
}

func TestTransformAABBAlias(t *testing.T) {
	box := NewAABB(math3d.V3(-1, -1, -1), math3d.V3(1, 1, 1))
	trans := math3d.Translate(math3d.V3(5, 0, 0))

	t1 := box.Transform(trans)
	t2 := TransformAABB(box, trans)

	if t1.Min != t2.Min || t1.Max != t2.Max {
		t.Error("Transform and TransformAABB should produce identical results")
	}
}

func BenchmarkFrustumIntersectAABB(b *testing.B) {
	proj := math3d.Perspective(math.Pi/3, 16.0/9.0, 0.1, 1000.0)
	view := math3d.Identity()
	frustum := NewFrustumFromMatrix(proj.Mul(view))
	box := NewAABB(math3d.V3(-1, -1, -10), math3d.V3(1, 1, -5))

	for b.Loop() {
		_ = frustum.IntersectAABB(box)
	}
}

func BenchmarkFrustumIntersectsSphere(b *testing.B) {
	proj := math3d.Perspective(math.Pi/3, 16.0/9.0, 0.1, 1000.0)
	view := math3d.Identity()
	frustum := NewFrustumFromMatrix(proj.Mul(view))
	center := math3d.V3(0, 0, -10)
	radius := 2.0

	for b.Loop() {
		_ = frustum.IntersectsSphere(center, radius)
	}
}

func BenchmarkFrustumExtraction(b *testing.B) {
	proj := math3d.Perspective(math.Pi/3, 16.0/9.0, 0.1, 1000.0)
	view := math3d.LookAt(math3d.V3(0, 10, 20), math3d.V3(0, 0, 0), math3d.V3(0, 1, 0))
	viewProj := proj.Mul(view)

	for b.Loop() {
		_ = NewFrustumFromMatrix(viewProj)
	}
}

func BenchmarkAABBTransform(b *testing.B) {
	box := NewAABB(math3d.V3(-1, -1, -1), math3d.V3(1, 1, 1))
	trans := math3d.Translate(math3d.V3(10, 0, 0)).Mul(math3d.RotateY(0.5))

	for b.Loop() {
		_ = box.Transform(trans)
	}
}
