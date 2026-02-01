// Package render provides terminal rendering capabilities for Trophy.
package render

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// Framebuffer is a 2D array of pixels that can be rendered to the terminal.
// We use double vertical resolution by using half-block characters (▀▄).
type Framebuffer struct {
	Width  int          // Width in "pixels" (same as terminal columns)
	Height int          // Height in "pixels" (2x terminal rows due to half-blocks)
	Pixels []color.RGBA // Row-major pixel data
}

// NewFramebuffer creates a new framebuffer with the given dimensions.
// Height should be 2x the desired terminal rows for half-block rendering.
func NewFramebuffer(width, height int) *Framebuffer {
	return &Framebuffer{
		Width:  width,
		Height: height,
		Pixels: make([]color.RGBA, width*height),
	}
}

// Clear fills the framebuffer with a solid color.
func (fb *Framebuffer) Clear(c color.RGBA) {
	for i := range fb.Pixels {
		fb.Pixels[i] = c
	}
}

// SetPixel sets a pixel at (x, y) to the given color.
// Bounds checking is performed.
func (fb *Framebuffer) SetPixel(x, y int, c color.RGBA) {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return
	}
	fb.Pixels[y*fb.Width+x] = c
}

// GetPixel returns the color at (x, y).
// Returns transparent black if out of bounds.
func (fb *Framebuffer) GetPixel(x, y int) color.RGBA {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return color.RGBA{}
	}
	return fb.Pixels[y*fb.Width+x]
}

// DrawLine draws a line from (x0, y0) to (x1, y1) using Bresenham's algorithm.
func (fb *Framebuffer) DrawLine(x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		fb.SetPixel(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// DrawRect draws a filled rectangle.
func (fb *Framebuffer) DrawRect(x, y, w, h int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			fb.SetPixel(px, py, c)
		}
	}
}

// DrawRectOutline draws a rectangle outline.
func (fb *Framebuffer) DrawRectOutline(x, y, w, h int, c color.RGBA) {
	// Top and bottom
	for px := x; px < x+w; px++ {
		fb.SetPixel(px, y, c)
		fb.SetPixel(px, y+h-1, c)
	}
	// Left and right
	for py := y; py < y+h; py++ {
		fb.SetPixel(x, py, c)
		fb.SetPixel(x+w-1, py, c)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ToImage converts the framebuffer to a standard Go image.RGBA.
func (fb *Framebuffer) ToImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, fb.Width, fb.Height))
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			img.SetRGBA(x, y, fb.Pixels[y*fb.Width+x])
		}
	}
	return img
}

// SavePNG saves the framebuffer as a PNG file.
func (fb *Framebuffer) SavePNG(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, fb.ToImage())
}
