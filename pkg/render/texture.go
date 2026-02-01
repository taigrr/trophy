// Package render provides software rasterization for TuiKart.
package render

import (
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"math"
	"os"
)

// WrapMode determines how texture coordinates outside [0,1] are handled.
type WrapMode int

const (
	WrapRepeat WrapMode = iota // Tile the texture
	WrapClamp                  // Clamp to edge
)

// FilterMode determines how texture sampling is performed.
type FilterMode int

const (
	FilterNearest  FilterMode = iota // Nearest-neighbor (pixelated)
	FilterBilinear                   // Bilinear interpolation (smooth)
)

// Texture holds a 2D image for texture mapping.
type Texture struct {
	Width      int
	Height     int
	Pixels     []Color    // Row-major pixel data
	WrapU      WrapMode   // Horizontal wrap mode
	WrapV      WrapMode   // Vertical wrap mode
	FilterMode FilterMode // Sampling filter mode
}

// NewTexture creates an empty texture with the given dimensions.
func NewTexture(width, height int) *Texture {
	return &Texture{
		Width:      width,
		Height:     height,
		Pixels:     make([]Color, width*height),
		WrapU:      WrapRepeat,
		WrapV:      WrapRepeat,
		FilterMode: FilterNearest,
	}
}

// LoadTexture loads a texture from an image file.
func LoadTexture(path string) (*Texture, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open texture: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	tex := NewTexture(width, height)

	for y := range height {
		for x := range width {
			c := img.At(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b, a := c.RGBA()
			// RGBA returns 16-bit values, scale to 8-bit
			tex.SetPixel(x, y, Color{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	return tex, nil
}

// TextureFromImage creates a texture from an image.Image.
func TextureFromImage(img image.Image) *Texture {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	tex := NewTexture(width, height)

	for y := range height {
		for x := range width {
			c := img.At(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b, a := c.RGBA()
			// RGBA returns 16-bit values, scale to 8-bit
			tex.SetPixel(x, y, Color{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	return tex
}

// NewCheckerTexture creates a procedural checkerboard texture.
func NewCheckerTexture(width, height, checkSize int, c1, c2 Color) *Texture {
	tex := NewTexture(width, height)
	for y := range height {
		for x := range width {
			cx := x / checkSize
			cy := y / checkSize
			if (cx+cy)%2 == 0 {
				tex.SetPixel(x, y, c1)
			} else {
				tex.SetPixel(x, y, c2)
			}
		}
	}
	return tex
}

// NewGradientTexture creates a horizontal gradient texture.
func NewGradientTexture(width, height int, left, right Color) *Texture {
	tex := NewTexture(width, height)
	for y := range height {
		for x := range width {
			t := float64(x) / float64(width-1)
			tex.SetPixel(x, y, lerpColor(left, right, t))
		}
	}
	return tex
}

// SetPixel sets a pixel in the texture.
func (t *Texture) SetPixel(x, y int, c Color) {
	if x < 0 || x >= t.Width || y < 0 || y >= t.Height {
		return
	}
	t.Pixels[y*t.Width+x] = c
}

// GetPixel returns the pixel at (x, y) with bounds checking.
func (t *Texture) GetPixel(x, y int) Color {
	if x < 0 || x >= t.Width || y < 0 || y >= t.Height {
		return Color{}
	}
	return t.Pixels[y*t.Width+x]
}

// Sample samples the texture at UV coordinates (0-1 range).
func (t *Texture) Sample(u, v float64) Color {
	// Apply wrap mode
	u = t.wrapCoord(u, t.WrapU)
	v = t.wrapCoord(v, t.WrapV)

	// Flip V coordinate (image Y=0 at top, UV V=0 at bottom)
	v = 1.0 - v

	switch t.FilterMode {
	case FilterBilinear:
		return t.sampleBilinear(u, v)
	default:
		return t.sampleNearest(u, v)
	}
}

// wrapCoord applies the wrap mode to a coordinate.
func (t *Texture) wrapCoord(coord float64, mode WrapMode) float64 {
	switch mode {
	case WrapRepeat:
		coord = coord - math.Floor(coord) // fmod to [0,1)
	case WrapClamp:
		coord = math.Max(0, math.Min(1, coord))
	}
	return coord
}

// sampleNearest returns the nearest pixel.
func (t *Texture) sampleNearest(u, v float64) Color {
	x := int(u * float64(t.Width))
	y := int(v * float64(t.Height))

	// Clamp to valid range
	if x >= t.Width {
		x = t.Width - 1
	}
	if y >= t.Height {
		y = t.Height - 1
	}

	return t.GetPixel(x, y)
}

// sampleBilinear returns bilinearly interpolated color.
func (t *Texture) sampleBilinear(u, v float64) Color {
	// Convert to pixel coordinates
	fx := u*float64(t.Width) - 0.5
	fy := v*float64(t.Height) - 0.5

	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	x1 := x0 + 1
	y1 := y0 + 1

	// Fractional parts
	tx := fx - float64(x0)
	ty := fy - float64(y0)

	// Wrap coordinates for sampling
	x0 = t.wrapPixelCoord(x0, t.Width, t.WrapU)
	x1 = t.wrapPixelCoord(x1, t.Width, t.WrapU)
	y0 = t.wrapPixelCoord(y0, t.Height, t.WrapV)
	y1 = t.wrapPixelCoord(y1, t.Height, t.WrapV)

	// Sample 4 pixels
	c00 := t.GetPixel(x0, y0)
	c10 := t.GetPixel(x1, y0)
	c01 := t.GetPixel(x0, y1)
	c11 := t.GetPixel(x1, y1)

	// Bilinear interpolation
	top := lerpColor(c00, c10, tx)
	bot := lerpColor(c01, c11, tx)
	return lerpColor(top, bot, ty)
}

// wrapPixelCoord wraps a pixel coordinate.
func (t *Texture) wrapPixelCoord(x, size int, mode WrapMode) int {
	switch mode {
	case WrapRepeat:
		x = x % size
		if x < 0 {
			x += size
		}
	case WrapClamp:
		if x < 0 {
			x = 0
		} else if x >= size {
			x = size - 1
		}
	}
	return x
}

// lerpColor linearly interpolates between two colors.
func lerpColor(a, b Color, t float64) Color {
	return Color{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t),
	}
}

// MultiplyColor multiplies a color by a scalar (for lighting).
func MultiplyColor(c Color, intensity float64) Color {
	return Color{
		R: uint8(math.Min(255, float64(c.R)*intensity)),
		G: uint8(math.Min(255, float64(c.G)*intensity)),
		B: uint8(math.Min(255, float64(c.B)*intensity)),
		A: c.A,
	}
}

// ModulateColor modulates one color by another (texture * vertex color).
func ModulateColor(a, b Color) Color {
	return Color{
		R: uint8((int(a.R) * int(b.R)) / 255),
		G: uint8((int(a.G) * int(b.G)) / 255),
		B: uint8((int(a.B) * int(b.B)) / 255),
		A: uint8((int(a.A) * int(b.A)) / 255),
	}
}
