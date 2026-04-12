package render

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestNewGradientTexture(t *testing.T) {
	left := RGB(255, 0, 0)
	right := RGB(0, 0, 255)
	tex := NewGradientTexture(100, 10, left, right)

	if tex.Width != 100 || tex.Height != 10 {
		t.Errorf("Expected 100x10, got %dx%d", tex.Width, tex.Height)
	}

	// Left edge should be red
	c := tex.GetPixel(0, 0)
	if c.R != 255 || c.B != 0 {
		t.Errorf("Left edge should be red, got %v", c)
	}

	// Right edge should be blue
	c = tex.GetPixel(99, 0)
	if c.R != 0 || c.B != 255 {
		t.Errorf("Right edge should be blue, got %v", c)
	}

	// Middle should be roughly purple
	c = tex.GetPixel(50, 0)
	if c.R < 100 || c.R > 155 || c.B < 100 || c.B > 155 {
		t.Errorf("Middle should be ~purple, got %v", c)
	}

	// All rows should be the same (horizontal gradient)
	topRow := tex.GetPixel(50, 0)
	botRow := tex.GetPixel(50, 5)
	if topRow != botRow {
		t.Errorf("Gradient should be uniform vertically: row0=%v row5=%v", topRow, botRow)
	}
}

func TestTextureSampleBilinear(t *testing.T) {
	tex := NewTexture(2, 2)
	tex.SetPixel(0, 0, RGB(0, 0, 0))       // Black top-left
	tex.SetPixel(1, 0, RGB(255, 255, 255)) // White top-right
	tex.SetPixel(0, 1, RGB(0, 0, 0))       // Black bottom-left
	tex.SetPixel(1, 1, RGB(255, 255, 255)) // White bottom-right
	tex.FilterMode = FilterBilinear

	// Sample at center - should be interpolated
	c := tex.Sample(0.5, 0.5)
	// With bilinear, the center should be a mix
	// We just check it's not pure black or white
	if (c.R == 0 && c.G == 0 && c.B == 0) || (c.R == 255 && c.G == 255 && c.B == 255) {
		t.Errorf("Bilinear sample at center should be interpolated, got %v", c)
	}
}

func TestTextureSampleBilinearEdges(t *testing.T) {
	// Create a larger texture for more meaningful bilinear tests
	tex := NewTexture(4, 4)
	for y := range 4 {
		for x := range 4 {
			val := uint8((x + y) * 40)
			tex.SetPixel(x, y, RGB(val, val, val))
		}
	}
	tex.FilterMode = FilterBilinear

	// Just verify no panics on edge sampling
	coords := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
	for _, u := range coords {
		for _, v := range coords {
			c := tex.Sample(u, v)
			if c.A != 255 && c.A != 0 {
				// colors from RGB() should have A=255, bilinear interp should preserve
			}
			_ = c
		}
	}
}

func TestTextureFromImage(t *testing.T) {
	// Create a simple Go image
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 255})
	img.Set(0, 1, color.RGBA{0, 0, 255, 255})
	img.Set(3, 3, color.RGBA{128, 128, 128, 255})

	tex := TextureFromImage(img)

	if tex.Width != 4 || tex.Height != 4 {
		t.Errorf("Expected 4x4, got %dx%d", tex.Width, tex.Height)
	}

	c := tex.GetPixel(0, 0)
	if c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("Pixel (0,0) should be red, got %v", c)
	}

	c = tex.GetPixel(1, 0)
	if c.R != 0 || c.G != 255 || c.B != 0 {
		t.Errorf("Pixel (1,0) should be green, got %v", c)
	}

	c = tex.GetPixel(0, 1)
	if c.R != 0 || c.G != 0 || c.B != 255 {
		t.Errorf("Pixel (0,1) should be blue, got %v", c)
	}

	c = tex.GetPixel(3, 3)
	if c.R != 128 || c.G != 128 || c.B != 128 {
		t.Errorf("Pixel (3,3) should be gray, got %v", c)
	}
}

func TestLoadTexture(t *testing.T) {
	// Create a temp PNG file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")

	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.Set(x, y, color.RGBA{uint8(x * 30), uint8(y * 30), 100, 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	tex, err := LoadTexture(path)
	if err != nil {
		t.Fatalf("LoadTexture failed: %v", err)
	}

	if tex.Width != 8 || tex.Height != 8 {
		t.Errorf("Expected 8x8, got %dx%d", tex.Width, tex.Height)
	}

	// Verify a pixel value
	c := tex.GetPixel(0, 0)
	if c.R != 0 || c.G != 0 || c.B != 100 {
		t.Errorf("Pixel (0,0) expected {0,0,100,255}, got %v", c)
	}
}

func TestLoadTextureErrors(t *testing.T) {
	// Non-existent file
	_, err := LoadTexture("/nonexistent/path.png")
	if err == nil {
		t.Error("LoadTexture should fail for nonexistent file")
	}

	// Invalid image file
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(badPath, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadTexture(badPath)
	if err == nil {
		t.Error("LoadTexture should fail for invalid image data")
	}
}

func TestSetPixelOutOfBounds(t *testing.T) {
	tex := NewTexture(4, 4)
	// These should not panic
	tex.SetPixel(-1, 0, RGB(255, 0, 0))
	tex.SetPixel(0, -1, RGB(255, 0, 0))
	tex.SetPixel(4, 0, RGB(255, 0, 0))
	tex.SetPixel(0, 4, RGB(255, 0, 0))
	tex.SetPixel(100, 100, RGB(255, 0, 0))
}

func TestGetPixelOutOfBounds(t *testing.T) {
	tex := NewTexture(4, 4)
	tex.SetPixel(0, 0, RGB(255, 0, 0))

	zero := Color{}
	if c := tex.GetPixel(-1, 0); c != zero {
		t.Errorf("GetPixel(-1,0) should return zero, got %v", c)
	}
	if c := tex.GetPixel(4, 0); c != zero {
		t.Errorf("GetPixel(4,0) should return zero, got %v", c)
	}
	if c := tex.GetPixel(0, -1); c != zero {
		t.Errorf("GetPixel(0,-1) should return zero, got %v", c)
	}
	if c := tex.GetPixel(0, 4); c != zero {
		t.Errorf("GetPixel(0,4) should return zero, got %v", c)
	}
}

func TestWrapPixelCoord(t *testing.T) {
	tex := NewTexture(4, 4)

	// Repeat mode
	if got := tex.wrapPixelCoord(-1, 4, WrapRepeat); got != 3 {
		t.Errorf("wrapPixelCoord(-1, 4, Repeat) = %d, want 3", got)
	}
	if got := tex.wrapPixelCoord(5, 4, WrapRepeat); got != 1 {
		t.Errorf("wrapPixelCoord(5, 4, Repeat) = %d, want 1", got)
	}
	if got := tex.wrapPixelCoord(0, 4, WrapRepeat); got != 0 {
		t.Errorf("wrapPixelCoord(0, 4, Repeat) = %d, want 0", got)
	}

	// Clamp mode
	if got := tex.wrapPixelCoord(-5, 4, WrapClamp); got != 0 {
		t.Errorf("wrapPixelCoord(-5, 4, Clamp) = %d, want 0", got)
	}
	if got := tex.wrapPixelCoord(10, 4, WrapClamp); got != 3 {
		t.Errorf("wrapPixelCoord(10, 4, Clamp) = %d, want 3", got)
	}
	if got := tex.wrapPixelCoord(2, 4, WrapClamp); got != 2 {
		t.Errorf("wrapPixelCoord(2, 4, Clamp) = %d, want 2", got)
	}
}

func TestWrapCoord(t *testing.T) {
	tex := NewTexture(4, 4)

	// Repeat: negative values should wrap
	got := tex.wrapCoord(-0.25, WrapRepeat)
	if math.Abs(got-0.75) > 0.001 {
		t.Errorf("wrapCoord(-0.25, Repeat) = %f, want 0.75", got)
	}

	// Repeat: values > 1 should wrap
	got = tex.wrapCoord(1.5, WrapRepeat)
	if math.Abs(got-0.5) > 0.001 {
		t.Errorf("wrapCoord(1.5, Repeat) = %f, want 0.5", got)
	}

	// Clamp: negative clamped to 0
	got = tex.wrapCoord(-0.5, WrapClamp)
	if got != 0 {
		t.Errorf("wrapCoord(-0.5, Clamp) = %f, want 0", got)
	}

	// Clamp: >1 clamped to 1
	got = tex.wrapCoord(1.5, WrapClamp)
	if got != 1 {
		t.Errorf("wrapCoord(1.5, Clamp) = %f, want 1", got)
	}
}
