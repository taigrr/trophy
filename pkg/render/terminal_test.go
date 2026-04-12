package render

import (
	"image/color"
	"testing"
)

func TestRGBAToColor(t *testing.T) {
	tests := []struct {
		name   string
		input  color.RGBA
		isNil  bool
		expect color.RGBA
	}{
		{"transparent returns nil", color.RGBA{0, 0, 0, 0}, true, color.RGBA{}},
		{"opaque returns color", color.RGBA{255, 0, 0, 255}, false, color.RGBA{255, 0, 0, 255}},
		{"partial alpha returns color", color.RGBA{0, 128, 0, 128}, false, color.RGBA{0, 128, 0, 128}},
		{"alpha 1 returns color", color.RGBA{0, 0, 0, 1}, false, color.RGBA{0, 0, 0, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rgbaToColor(tt.input)
			if tt.isNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Error("expected non-nil, got nil")
				} else if result != tt.expect {
					t.Errorf("expected %v, got %v", tt.expect, result)
				}
			}
		})
	}
}

func TestRGB(t *testing.T) {
	c := RGB(10, 20, 30)
	if c.R != 10 || c.G != 20 || c.B != 30 || c.A != 255 {
		t.Errorf("RGB(10,20,30) = %v, want {10,20,30,255}", c)
	}
}

func TestRGBA(t *testing.T) {
	c := RGBA(10, 20, 30, 40)
	if c.R != 10 || c.G != 20 || c.B != 30 || c.A != 40 {
		t.Errorf("RGBA(10,20,30,40) = %v, want {10,20,30,40}", c)
	}
}

func TestColorConstants(t *testing.T) {
	// Verify a few constants are correctly defined
	if ColorBlack.R != 0 || ColorBlack.G != 0 || ColorBlack.B != 0 || ColorBlack.A != 255 {
		t.Errorf("ColorBlack = %v, want {0,0,0,255}", ColorBlack)
	}
	if ColorWhite.R != 255 || ColorWhite.G != 255 || ColorWhite.B != 255 || ColorWhite.A != 255 {
		t.Errorf("ColorWhite = %v, want {255,255,255,255}", ColorWhite)
	}
	if ColorRed.R != 255 || ColorRed.G != 0 || ColorRed.B != 0 {
		t.Errorf("ColorRed = %v, want {255,0,0,255}", ColorRed)
	}
	if ColorGreen.R != 0 || ColorGreen.G != 255 || ColorGreen.B != 0 {
		t.Errorf("ColorGreen = %v, want {0,255,0,255}", ColorGreen)
	}
	if ColorBlue.R != 0 || ColorBlue.G != 0 || ColorBlue.B != 255 {
		t.Errorf("ColorBlue = %v, want {0,0,255,255}", ColorBlue)
	}
}

func TestDrawFramebufferToScreen(t *testing.T) {
	// Create a small framebuffer (4x4, which represents 2 terminal rows)
	fb := NewFramebuffer(4, 4)

	// Set some pixels - top two rows
	fb.SetPixel(0, 0, RGB(255, 0, 0))   // Row 0: red
	fb.SetPixel(0, 1, RGB(0, 255, 0))   // Row 1: green
	fb.SetPixel(1, 0, RGB(0, 0, 255))   // Row 0: blue
	fb.SetPixel(1, 1, RGB(255, 255, 0)) // Row 1: yellow

	// Row 2-3: one transparent, one colored
	fb.SetPixel(0, 2, color.RGBA{0, 0, 0, 0}) // Transparent
	fb.SetPixel(0, 3, RGB(128, 128, 128))     // Gray

	// We can't easily test Draw() without a real ultraviolet.Screen,
	// but we can verify the framebuffer setup is correct
	c := fb.GetPixel(0, 0)
	if c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("GetPixel(0,0) = %v, want red", c)
	}

	c = fb.GetPixel(0, 1)
	if c.R != 0 || c.G != 255 || c.B != 0 {
		t.Errorf("GetPixel(0,1) = %v, want green", c)
	}

	// Test out-of-bounds pixel returns zero
	c = fb.GetPixel(-1, 0)
	if c.A != 0 {
		t.Errorf("GetPixel(-1,0) should return transparent, got %v", c)
	}

	c = fb.GetPixel(100, 0)
	if c.A != 0 {
		t.Errorf("GetPixel(100,0) should return transparent, got %v", c)
	}
}
