package render

import (
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

func TestEdgeCoeffs(t *testing.T) {
	// Edge from (0,0) to (1,0) - horizontal edge
	a, b, c := edgeCoeffs(0, 0, 1, 0)
	// A = y0 - y1 = 0
	// B = x1 - x0 = 1
	// C = x0*y1 - x1*y0 = 0
	if a != 0 || b != 1 || c != 0 {
		t.Errorf("edgeCoeffs(0,0,1,0) = (%f,%f,%f), want (0,1,0)", a, b, c)
	}

	// Edge from (0,0) to (0,1) - vertical edge
	a, b, c = edgeCoeffs(0, 0, 0, 1)
	// A = 0 - 1 = -1
	// B = 0 - 0 = 0
	// C = 0*1 - 0*0 = 0
	if a != -1 || b != 0 || c != 0 {
		t.Errorf("edgeCoeffs(0,0,0,1) = (%f,%f,%f), want (-1,0,0)", a, b, c)
	}
}

func TestEdgeFunc(t *testing.T) {
	// A*x + B*y + C
	result := edgeFunc(1, 2, 3, 4, 5)
	expected := 1.0*4 + 2.0*5 + 3.0
	if result != expected {
		t.Errorf("edgeFunc(1,2,3,4,5) = %f, want %f", result, expected)
	}
}

func TestDrawTriangleGouraudOpt_Visible(t *testing.T) {
	rast, fb := createTestRasterizer(100, 100)
	fb.BG = RGB(0, 0, 0)
	rast.ClearDepth()
	fb.Clear()

	// Front-facing triangle (CW winding in screen space after projection)
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(255, 0, 0)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(0, 255, 0)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(0, 0, 255)},
		},
	}

	lightDir := math3d.V3(0, 0, 1)
	rast.DrawTriangleGouraudOpt(tri, lightDir)

	pixelCount := countNonBlackPixels(fb)
	if pixelCount == 0 {
		t.Error("DrawTriangleGouraudOpt should render visible pixels")
	}
}

func TestDrawTriangleGouraudOpt_BackfaceCulled(t *testing.T) {
	rast, fb := createTestRasterizer(100, 100)
	fb.BG = RGB(0, 0, 0)
	rast.ClearDepth()
	fb.Clear()

	// CCW winding should be culled
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, -1), Color: RGB(255, 0, 0)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, -1), Color: RGB(0, 255, 0)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, -1), Color: RGB(0, 0, 255)},
		},
	}

	lightDir := math3d.V3(0, 0, 1)
	rast.DrawTriangleGouraudOpt(tri, lightDir)

	pixelCount := countNonBlackPixels(fb)
	if pixelCount > 0 {
		t.Errorf("Back-facing triangle should be culled, got %d pixels", pixelCount)
	}
}

func TestDrawTriangleGouraudOpt_BehindCamera(t *testing.T) {
	rast, fb := createTestRasterizer(100, 100)
	fb.BG = RGB(0, 0, 0)
	rast.ClearDepth()
	fb.Clear()

	// Triangle behind camera (z > camera z=10)
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 20), Normal: math3d.V3(0, 0, 1), Color: RGB(255, 0, 0)},
			{Position: math3d.V3(0, 5, 20), Normal: math3d.V3(0, 0, 1), Color: RGB(0, 255, 0)},
			{Position: math3d.V3(5, -5, 20), Normal: math3d.V3(0, 0, 1), Color: RGB(0, 0, 255)},
		},
	}

	lightDir := math3d.V3(0, 0, 1)
	rast.DrawTriangleGouraudOpt(tri, lightDir)

	pixelCount := countNonBlackPixels(fb)
	if pixelCount > 0 {
		t.Errorf("Triangle behind camera should not render, got %d pixels", pixelCount)
	}
}

func TestDrawMeshGouraudOpt(t *testing.T) {
	rast, fb := createTestRasterizer(100, 100)
	fb.BG = RGB(0, 0, 0)
	rast.ClearDepth()
	fb.Clear()

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
			{0, 3, 2},
			{0, 2, 1},
		},
	}

	transform := math3d.Identity()
	meshColor := RGB(200, 100, 50)
	lightDir := math3d.V3(0, 0, 1)

	rast.DrawMeshGouraudOpt(mesh, transform, meshColor, lightDir)

	pixelCount := countNonBlackPixels(fb)
	if pixelCount == 0 {
		t.Error("DrawMeshGouraudOpt should render visible pixels")
	}
}

func TestDrawTriangleTexturedOpt_Visible(t *testing.T) {
	rast, fb := createTestRasterizer(100, 100)
	fb.BG = RGB(0, 0, 0)
	rast.ClearDepth()
	fb.Clear()

	tex := NewCheckerTexture(8, 8, 2, RGB(255, 255, 255), RGB(128, 128, 128))

	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), UV: math3d.V2(0, 0), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, 1), UV: math3d.V2(0.5, 1), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, 1), UV: math3d.V2(1, 0), Color: RGB(255, 255, 255)},
		},
	}

	lightDir := math3d.V3(0, 0, 1)
	rast.DrawTriangleTexturedOpt(tri, tex, lightDir)

	pixelCount := countNonBlackPixels(fb)
	if pixelCount == 0 {
		t.Error("DrawTriangleTexturedOpt should render visible pixels")
	}
}

func TestDrawTriangleTexturedOpt_BackfaceCulled(t *testing.T) {
	rast, fb := createTestRasterizer(100, 100)
	fb.BG = RGB(0, 0, 0)
	rast.ClearDepth()
	fb.Clear()

	tex := NewCheckerTexture(4, 4, 1, RGB(255, 255, 255), RGB(128, 128, 128))

	// CCW winding
	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, -1), UV: math3d.V2(0, 0), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, -1), UV: math3d.V2(1, 0), Color: RGB(255, 255, 255)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, -1), UV: math3d.V2(0.5, 1), Color: RGB(255, 255, 255)},
		},
	}

	lightDir := math3d.V3(0, 0, 1)
	rast.DrawTriangleTexturedOpt(tri, tex, lightDir)

	pixelCount := countNonBlackPixels(fb)
	if pixelCount > 0 {
		t.Errorf("Back-facing textured triangle should be culled, got %d pixels", pixelCount)
	}
}

func TestDrawMeshTexturedOpt(t *testing.T) {
	rast, fb := createTestRasterizer(100, 100)
	fb.BG = RGB(0, 0, 0)
	rast.ClearDepth()
	fb.Clear()

	tex := NewCheckerTexture(8, 8, 2, RGB(200, 200, 200), RGB(100, 100, 100))

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
			{0, 3, 2},
			{0, 2, 1},
		},
	}

	transform := math3d.Identity()
	lightDir := math3d.V3(0, 0, 1)

	rast.DrawMeshTexturedOpt(mesh, transform, tex, lightDir)

	pixelCount := countNonBlackPixels(fb)
	if pixelCount == 0 {
		t.Error("DrawMeshTexturedOpt should render visible pixels")
	}
}

func TestOptVsNonOptConsistency(t *testing.T) {
	// Both optimized and non-optimized should produce similar results
	rastOpt, fbOpt := createTestRasterizer(50, 50)
	rastReg, fbReg := createTestRasterizer(50, 50)

	fbOpt.BG = RGB(0, 0, 0)
	fbReg.BG = RGB(0, 0, 0)
	rastOpt.ClearDepth()
	rastReg.ClearDepth()
	fbOpt.Clear()
	fbReg.Clear()

	tri := Triangle{
		V: [3]Vertex{
			{Position: math3d.V3(-5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(200, 100, 50)},
			{Position: math3d.V3(0, 5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(200, 100, 50)},
			{Position: math3d.V3(5, -5, 0), Normal: math3d.V3(0, 0, 1), Color: RGB(200, 100, 50)},
		},
	}

	lightDir := math3d.V3(0, 0, 1)

	rastOpt.DrawTriangleGouraudOpt(tri, lightDir)
	rastReg.DrawTriangleGouraud(tri, lightDir)

	optPixels := countNonBlackPixels(fbOpt)
	regPixels := countNonBlackPixels(fbReg)

	// They should produce similar pixel counts (within 10%)
	if optPixels == 0 || regPixels == 0 {
		t.Errorf("Both methods should render pixels: opt=%d reg=%d", optPixels, regPixels)
		return
	}

	ratio := float64(optPixels) / float64(regPixels)
	if ratio < 0.8 || ratio > 1.2 {
		t.Errorf("Opt and regular rasterizers produce very different results: opt=%d reg=%d ratio=%.2f",
			optPixels, regPixels, ratio)
	}
}

// countNonBlackPixels counts pixels that aren't fully black.
func countNonBlackPixels(fb *Framebuffer) int {
	count := 0
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			c := fb.GetPixel(x, y)
			if c.R > 0 || c.G > 0 || c.B > 0 {
				count++
			}
		}
	}
	return count
}
