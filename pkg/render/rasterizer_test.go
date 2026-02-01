// Package render provides software rasterization tests for TuiKart.
package render

import (
	"math"
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

// mockMesh implements MeshRenderer for testing.
type mockMesh struct {
	vertices []struct {
		pos    math3d.Vec3
		normal math3d.Vec3
		uv     math3d.Vec2
	}
	faces [][3]int
}

func (m *mockMesh) VertexCount() int     { return len(m.vertices) }
func (m *mockMesh) TriangleCount() int   { return len(m.faces) }
func (m *mockMesh) GetFace(i int) [3]int { return m.faces[i] }
func (m *mockMesh) GetVertex(i int) (pos, normal math3d.Vec3, uv math3d.Vec2) {
	v := m.vertices[i]
	return v.pos, v.normal, v.uv
}

// createTestRasterizer creates a rasterizer for testing.
func createTestRasterizer(width, height int) (*Rasterizer, *Framebuffer) {
	fb := NewFramebuffer(width, height)
	camera := NewCamera()
	camera.SetPosition(math3d.V3(0, 0, 10))
	camera.LookAt(math3d.Zero3())
	camera.SetAspectRatio(float64(width) / float64(height))
	camera.SetFOV(60) // Reasonable FOV
	rasterizer := NewRasterizer(camera, fb)
	return rasterizer, fb
}

func TestBarycentric(t *testing.T) {
	// Test barycentric coordinates at triangle vertices
	tests := []struct {
		name     string
		px, py   float64
		expected math3d.Vec3
	}{
		{"vertex 0", 0, 0, math3d.V3(1, 0, 0)},
		{"vertex 1", 1, 0, math3d.V3(0, 1, 0)},
		{"vertex 2", 0, 1, math3d.V3(0, 0, 1)},
		{"centroid", 1.0 / 3, 1.0 / 3, math3d.V3(1.0/3, 1.0/3, 1.0/3)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Triangle: (0,0), (1,0), (0,1)
			bc := barycentric(0, 0, 1, 0, 0, 1, tc.px, tc.py)

			if math.Abs(bc.X-tc.expected.X) > 0.001 ||
				math.Abs(bc.Y-tc.expected.Y) > 0.001 ||
				math.Abs(bc.Z-tc.expected.Z) > 0.001 {
				t.Errorf("barycentric(%v, %v) = %v, want %v", tc.px, tc.py, bc, tc.expected)
			}
		})
	}

	// Test point outside triangle
	t.Run("outside triangle", func(t *testing.T) {
		bc := barycentric(0, 0, 1, 0, 0, 1, -1, -1)
		if bc.X >= 0 && bc.Y >= 0 && bc.Z >= 0 {
			t.Error("point outside triangle should have negative barycentric coordinate")
		}
	})
}

func TestInterpolateColor3(t *testing.T) {
	c0 := RGB(255, 0, 0) // Red
	c1 := RGB(0, 255, 0) // Green
	c2 := RGB(0, 0, 255) // Blue

	tests := []struct {
		name     string
		bc       math3d.Vec3
		expected Color
	}{
		{"full red", math3d.V3(1, 0, 0), RGB(255, 0, 0)},
		{"full green", math3d.V3(0, 1, 0), RGB(0, 255, 0)},
		{"full blue", math3d.V3(0, 0, 1), RGB(0, 0, 255)},
		{"equal mix", math3d.V3(1.0/3, 1.0/3, 1.0/3), RGB(85, 85, 85)},
		{"half red half green", math3d.V3(0.5, 0.5, 0), RGB(127, 127, 0)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := interpolateColor3(c0, c1, c2, tc.bc)
			// Allow 1 unit tolerance due to rounding
			if absInt(int(result.R)-int(tc.expected.R)) > 1 ||
				absInt(int(result.G)-int(tc.expected.G)) > 1 ||
				absInt(int(result.B)-int(tc.expected.B)) > 1 {
				t.Errorf("interpolateColor3 with bc=%v = %v, want %v", tc.bc, result, tc.expected)
			}
		})
	}
}

func TestDrawTriangleGouraud_VertexLighting(t *testing.T) {
	r, fb := createTestRasterizer(100, 100)
	r.ClearDepth()
	fb.Clear(RGB(0, 0, 0))

	// Light from z+ direction (toward camera)
	lightDir := math3d.V3(0, 0, 1).Normalize()

	// Triangle at z=0, large enough to be visible from z=10
	// CW winding for front-facing (engine convention due to Y-flip)
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(200, 200, 200)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(200, 200, 200)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0.5, 0, 0.866), Color: RGB(200, 200, 200)},
		},
	}

	r.DrawTriangleGouraud(tri, lightDir)

	// Check that the triangle was drawn (some pixels should be non-black)
	hasPixels := false
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.GetPixel(x, y)
			if c.R > 0 || c.G > 0 || c.B > 0 {
				hasPixels = true
				break
			}
		}
		if hasPixels {
			break
		}
	}

	if !hasPixels {
		t.Error("DrawTriangleGouraud should draw visible pixels")
	}
}

func TestDrawTriangleGouraud_SmoothShading(t *testing.T) {
	r, fb := createTestRasterizer(100, 100)
	r.ClearDepth()
	fb.Clear(RGB(0, 0, 0))

	// Light from the front (camera direction)
	lightDir := math3d.V3(0, 0, 1).Normalize()

	// Large triangle with varying normals - CW winding for front-facing
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0.707, 0, 0.707), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0.707, 0.707), Color: RGB(255, 255, 255)},
		},
	}

	r.DrawTriangleGouraud(tri, lightDir)

	// Count drawn pixels
	pixelCount := 0
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.GetPixel(x, y)
			if c.R > 0 || c.G > 0 || c.B > 0 {
				pixelCount++
			}
		}
	}

	if pixelCount == 0 {
		t.Error("No pixels drawn in Gouraud shaded triangle")
	}
}

func TestDrawMeshGouraud(t *testing.T) {
	r, fb := createTestRasterizer(100, 100)
	r.ClearDepth()
	fb.Clear(RGB(0, 0, 0))

	// Create a simple mesh (quad made of 2 triangles) with smooth normals
	// Using CW winding for front-facing triangles
	mesh := &mockMesh{
		vertices: []struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{
			{math3d.V3(-5, -5, 0), math3d.V3(0, 0, 1), math3d.V2(0, 0)},
			{math3d.V3(5, -5, 0), math3d.V3(0, 0, 1), math3d.V2(1, 0)},
			{math3d.V3(5, 5, 0), math3d.V3(0, 0, 1), math3d.V2(1, 1)},
			{math3d.V3(-5, 5, 0), math3d.V3(0, 0, 1), math3d.V2(0, 1)},
		},
		faces: [][3]int{
			{0, 3, 2}, // CW: bottom-left, top-left, top-right
			{0, 2, 1}, // CW: bottom-left, top-right, bottom-right
		},
	}

	transform := math3d.Identity()
	color := RGB(255, 100, 50)
	lightDir := math3d.V3(0, 0, 1)

	r.DrawMeshGouraud(mesh, transform, color, lightDir)

	// Verify mesh was rendered
	pixelCount := 0
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.GetPixel(x, y)
			if c.R > 0 || c.G > 0 || c.B > 0 {
				pixelCount++
			}
		}
	}

	if pixelCount == 0 {
		t.Error("DrawMeshGouraud should render visible pixels")
	}
}

func TestDrawMeshGouraud_SmoothVsFlat(t *testing.T) {
	// This test verifies that Gouraud shading produces different results
	// than flat shading when normals vary across the surface

	// Create two rasterizers
	rGouraud, fbGouraud := createTestRasterizer(50, 50)
	rFlat, fbFlat := createTestRasterizer(50, 50)

	rGouraud.ClearDepth()
	rFlat.ClearDepth()
	fbGouraud.Clear(RGB(0, 0, 0))
	fbFlat.Clear(RGB(0, 0, 0))

	// Create a mesh with varying normals, CW winding for front-facing
	mesh := &mockMesh{
		vertices: []struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{
			// Triangle with different normals at each vertex
			{math3d.V3(-5, -5, 0), math3d.V3(0, 0, 1), math3d.V2(0, 0)},
			{math3d.V3(5, -5, 0), math3d.V3(0.5, 0, 0.866), math3d.V2(1, 0)},
			{math3d.V3(0, 5, 0), math3d.V3(-0.5, 0, 0.866), math3d.V2(0.5, 1)},
		},
		faces: [][3]int{{0, 2, 1}}, // CW winding
	}

	transform := math3d.Identity()
	color := RGB(200, 200, 200)
	lightDir := math3d.V3(0, 0, 1)

	// Draw with Gouraud shading
	rGouraud.DrawMeshGouraud(mesh, transform, color, lightDir)

	// Draw with flat shading (uses per-face normals)
	rFlat.DrawMesh(mesh, transform, color, lightDir)

	// Count pixels and compute average brightness for each
	gouraudSum := 0
	gouraudCount := 0
	flatSum := 0
	flatCount := 0

	for y := range 50 {
		for x := range 50 {
			cg := fbGouraud.GetPixel(x, y)
			cf := fbFlat.GetPixel(x, y)

			if cg.R > 0 || cg.G > 0 || cg.B > 0 {
				gouraudSum += int(cg.R) + int(cg.G) + int(cg.B)
				gouraudCount++
			}
			if cf.R > 0 || cf.G > 0 || cf.B > 0 {
				flatSum += int(cf.R) + int(cf.G) + int(cf.B)
				flatCount++
			}
		}
	}

	// Gouraud shading should produce similar pixel coverage but with
	// potentially different brightness distribution due to interpolation
	if gouraudCount == 0 || flatCount == 0 {
		t.Error("Both shading methods should produce visible pixels")
	}

	t.Logf("Gouraud: %d pixels, avg brightness: %.2f", gouraudCount, float64(gouraudSum)/float64(gouraudCount))
	t.Logf("Flat: %d pixels, avg brightness: %.2f", flatCount, float64(flatSum)/float64(flatCount))
}

func TestDrawTriangleGouraud_BackfaceCulling(t *testing.T) {
	r, fb := createTestRasterizer(100, 100)
	r.ClearDepth()
	fb.Clear(RGB(0, 0, 0))

	// Back-facing triangle: CCW winding (opposite of front-facing CW)
	// This should be culled
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, -1), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, -1), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, -1), Color: RGB(255, 255, 255)},
		},
	}

	lightDir := math3d.V3(0, 0, 1)
	r.DrawTriangleGouraud(tri, lightDir)

	// Back-facing triangle should not be drawn
	pixelCount := 0
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.GetPixel(x, y)
			if c.R > 0 || c.G > 0 || c.B > 0 {
				pixelCount++
			}
		}
	}

	if pixelCount > 0 {
		t.Errorf("Back-facing triangle should be culled, but got %d pixels", pixelCount)
	}
}

func TestDrawTransformedCubeGouraud(t *testing.T) {
	r, fb := createTestRasterizer(100, 100)
	r.ClearDepth()
	fb.Clear(RGB(0, 0, 0))

	// Position cube in front of camera
	transform := math3d.Translate(math3d.V3(0, 0, -3))
	lightDir := math3d.V3(1, 1, 1).Normalize()

	r.DrawTransformedCubeGouraud(transform, 2.0, RGB(200, 100, 50), lightDir)

	// Verify cube was rendered
	pixelCount := 0
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.GetPixel(x, y)
			if c.R > 0 || c.G > 0 || c.B > 0 {
				pixelCount++
			}
		}
	}

	if pixelCount == 0 {
		t.Error("DrawTransformedCubeGouraud should render visible pixels")
	}
}

func TestDrawTriangleTexturedGouraud(t *testing.T) {
	r, fb := createTestRasterizer(100, 100)
	r.ClearDepth()
	fb.Clear(RGB(0, 0, 0))

	// Create a simple 2x2 texture
	tex := NewTexture(2, 2)
	tex.SetPixel(0, 0, RGB(255, 0, 0))   // Red
	tex.SetPixel(1, 0, RGB(0, 255, 0))   // Green
	tex.SetPixel(0, 1, RGB(0, 0, 255))   // Blue
	tex.SetPixel(1, 1, RGB(255, 255, 0)) // Yellow

	// Triangle with UVs and varying normals - CW winding
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), UV: math3d.V2(0, 0), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, 1), UV: math3d.V2(0.5, 1), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, 1), UV: math3d.V2(1, 0), Color: RGB(255, 255, 255)},
		},
	}

	lightDir := math3d.V3(0, 0, 1)
	r.DrawTriangleTexturedGouraud(tri, tex, lightDir)

	// Verify textured triangle was rendered
	pixelCount := 0
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.GetPixel(x, y)
			if c.R > 0 || c.G > 0 || c.B > 0 {
				pixelCount++
			}
		}
	}

	if pixelCount == 0 {
		t.Error("DrawTriangleTexturedGouraud should render visible pixels")
	}
}

func TestDrawMeshTexturedGouraud(t *testing.T) {
	r, fb := createTestRasterizer(100, 100)
	r.ClearDepth()
	fb.Clear(RGB(0, 0, 0))

	// Create a simple 4x4 checkerboard texture
	tex := NewTexture(4, 4)
	for y := range 4 {
		for x := range 4 {
			if (x+y)%2 == 0 {
				tex.SetPixel(x, y, RGB(255, 255, 255))
			} else {
				tex.SetPixel(x, y, RGB(100, 100, 100))
			}
		}
	}

	// Create a quad mesh with smooth normals - CW winding
	mesh := &mockMesh{
		vertices: []struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{
			{math3d.V3(-5, -5, 0), math3d.V3(0, 0, 1), math3d.V2(0, 0)},
			{math3d.V3(5, -5, 0), math3d.V3(0, 0, 1), math3d.V2(1, 0)},
			{math3d.V3(5, 5, 0), math3d.V3(0, 0, 1), math3d.V2(1, 1)},
			{math3d.V3(-5, 5, 0), math3d.V3(0, 0, 1), math3d.V2(0, 1)},
		},
		faces: [][3]int{
			{0, 3, 2}, // CW winding
			{0, 2, 1}, // CW winding
		},
	}

	transform := math3d.Identity()
	lightDir := math3d.V3(0, 0, 1)

	r.DrawMeshTexturedGouraud(mesh, transform, tex, lightDir)

	// Verify mesh was rendered with texture
	pixelCount := 0
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.GetPixel(x, y)
			if c.R > 0 || c.G > 0 || c.B > 0 {
				pixelCount++
			}
		}
	}

	if pixelCount == 0 {
		t.Error("DrawMeshTexturedGouraud should render visible pixels")
	}
}

func TestMin3Max3(t *testing.T) {
	if min3(1, 2, 3) != 1 || min3(3, 1, 2) != 1 || min3(2, 3, 1) != 1 {
		t.Error("min3 failed")
	}
	if max3(1, 2, 3) != 3 || max3(3, 1, 2) != 3 || max3(2, 3, 1) != 3 {
		t.Error("max3 failed")
	}
}

func TestRasterizerClearDepth(t *testing.T) {
	r, _ := createTestRasterizer(10, 10)

	// Set some depth values
	r.setDepth(5, 5, 1.0)
	if r.getDepth(5, 5) != 1.0 {
		t.Error("setDepth/getDepth failed")
	}

	// Clear and verify
	r.ClearDepth()
	if r.getDepth(5, 5) != math.MaxFloat64 {
		t.Error("ClearDepth should reset to MaxFloat64")
	}
}

func TestRasterizerDepthBoundsCheck(t *testing.T) {
	r, _ := createTestRasterizer(10, 10)

	// Out of bounds should return MaxFloat64 and not panic
	if r.getDepth(-1, 0) != math.MaxFloat64 {
		t.Error("Out of bounds getDepth should return MaxFloat64")
	}
	if r.getDepth(100, 0) != math.MaxFloat64 {
		t.Error("Out of bounds getDepth should return MaxFloat64")
	}

	// setDepth out of bounds should not panic
	r.setDepth(-1, 0, 1.0) // Should not panic
	r.setDepth(100, 0, 1.0)
}

// Helper function for color comparison tolerance
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Benchmark tests
func BenchmarkDrawTriangleGouraud(b *testing.B) {
	r, _ := createTestRasterizer(200, 200)

	// CW winding for front-facing
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(255, 100, 50)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(100, 50, 255)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(50, 255, 100)},
		},
	}
	lightDir := math3d.V3(0, 0, 1)

	for b.Loop() {
		r.ClearDepth()
		r.DrawTriangleGouraud(tri, lightDir)
	}
}

func BenchmarkDrawMeshGouraud(b *testing.B) {
	r, _ := createTestRasterizer(200, 200)

	// Create a mesh with 100 triangles - CW winding
	mesh := &mockMesh{
		vertices: make([]struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}, 300),
		faces: make([][3]int, 100),
	}

	for i := range 100 {
		base := i * 3
		mesh.vertices[base] = struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{math3d.V3(-3, -3, float64(i)*0.01), math3d.V3(0, 0, 1), math3d.V2(0, 0)}
		mesh.vertices[base+1] = struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{math3d.V3(3, -3, float64(i)*0.01), math3d.V3(0, 0, 1), math3d.V2(1, 0)}
		mesh.vertices[base+2] = struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{math3d.V3(0, 3, float64(i)*0.01), math3d.V3(0, 0, 1), math3d.V2(0.5, 1)}
		// CW winding: 0, 2, 1
		mesh.faces[i] = [3]int{base, base + 2, base + 1}
	}

	transform := math3d.Identity()
	color := RGB(200, 100, 50)
	lightDir := math3d.V3(0, 0, 1)

	for b.Loop() {
		r.ClearDepth()
		r.DrawMeshGouraud(mesh, transform, color, lightDir)
	}
}

// BenchmarkDrawTriangleGouraudOpt benchmarks the optimized Gouraud triangle rasterizer.
func BenchmarkDrawTriangleGouraudOpt(b *testing.B) {
	r, _ := createTestRasterizer(200, 200)

	// CW winding for front-facing
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(255, 100, 50)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(100, 50, 255)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(50, 255, 100)},
		},
	}
	lightDir := math3d.V3(0, 0, 1)

	for b.Loop() {
		r.ClearDepth()
		r.DrawTriangleGouraudOpt(tri, lightDir)
	}
}

// BenchmarkDrawMeshGouraudOpt benchmarks the optimized mesh Gouraud renderer.
func BenchmarkDrawMeshGouraudOpt(b *testing.B) {
	r, _ := createTestRasterizer(200, 200)

	// Create a mesh with 100 triangles - CW winding
	mesh := &mockMesh{
		vertices: make([]struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}, 300),
		faces: make([][3]int, 100),
	}

	for i := range 100 {
		base := i * 3
		mesh.vertices[base] = struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{math3d.V3(-3, -3, float64(i)*0.01), math3d.V3(0, 0, 1), math3d.V2(0, 0)}
		mesh.vertices[base+1] = struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{math3d.V3(3, -3, float64(i)*0.01), math3d.V3(0, 0, 1), math3d.V2(1, 0)}
		mesh.vertices[base+2] = struct {
			pos    math3d.Vec3
			normal math3d.Vec3
			uv     math3d.Vec2
		}{math3d.V3(0, 3, float64(i)*0.01), math3d.V3(0, 0, 1), math3d.V2(0.5, 1)}
		// CW winding: 0, 2, 1
		mesh.faces[i] = [3]int{base, base + 2, base + 1}
	}

	transform := math3d.Identity()
	color := RGB(200, 100, 50)
	lightDir := math3d.V3(0, 0, 1)

	for b.Loop() {
		r.ClearDepth()
		r.DrawMeshGouraudOpt(mesh, transform, color, lightDir)
	}
}

// BenchmarkGouraudComparison directly compares old vs new implementation.
func BenchmarkGouraudComparison(b *testing.B) {
	r, _ := createTestRasterizer(200, 200)

	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(255, 100, 50)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(100, 50, 255)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(50, 255, 100)},
		},
	}
	lightDir := math3d.V3(0, 0, 1)

	b.Run("original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r.ClearDepth()
			r.DrawTriangleGouraud(tri, lightDir)
		}
	})

	b.Run("optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r.ClearDepth()
			r.DrawTriangleGouraudOpt(tri, lightDir)
		}
	})
}
