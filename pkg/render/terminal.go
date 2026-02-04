package render

import (
	"image/color"

	uv "github.com/charmbracelet/ultraviolet"
)

// Draw converts the internal framebuffer to terminal cells and draws them on
// the screen.
// The framebuffer height should be 2x the terminal height.
func (r *Framebuffer) Draw(scr uv.Screen, area uv.Rectangle) {
	// Each terminal row represents 2 framebuffer rows
	// We use ▀ (upper half block) with fg=top color and bg=bottom color

	for row := area.Min.Y; row < area.Max.Y; row++ {
		topY := row * 2
		botY := topY + 1

		for col := area.Min.X; col < area.Max.X && col < r.Width; col++ {
			topColor := r.GetPixel(col, topY)
			botColor := r.GetPixel(col, botY)

			cell := &uv.Cell{
				Content: "▀",
				Width:   1,
				Style: uv.Style{
					Fg: rgbaToColor(topColor),
					Bg: rgbaToColor(botColor),
				},
			}
			scr.SetCell(col, row, cell)
		}
	}
}

// rgbaToColor converts color.RGBA to Go's color.Color interface.
func rgbaToColor(c color.RGBA) color.Color {
	if c.A == 0 {
		return nil // Transparent = no color
	}
	return c
}

// Color is an alias for color.RGBA for convenience.
type Color = color.RGBA

// Colors for convenience
var (
	ColorBlack   = color.RGBA{0, 0, 0, 255}
	ColorWhite   = color.RGBA{255, 255, 255, 255}
	ColorRed     = color.RGBA{255, 0, 0, 255}
	ColorGreen   = color.RGBA{0, 255, 0, 255}
	ColorBlue    = color.RGBA{0, 0, 255, 255}
	ColorYellow  = color.RGBA{255, 255, 0, 255}
	ColorCyan    = color.RGBA{0, 255, 255, 255}
	ColorMagenta = color.RGBA{255, 0, 255, 255}
	ColorGray    = color.RGBA{128, 128, 128, 255}
	ColorSky     = color.RGBA{135, 206, 235, 255}
	ColorGrass   = color.RGBA{34, 139, 34, 255}
	ColorRoad    = color.RGBA{64, 64, 64, 255}
)

// RGB creates a color from RGB values.
func RGB(r, g, b uint8) color.RGBA {
	return color.RGBA{r, g, b, 255}
}

// RGBA creates a color from RGBA values.
func RGBA(r, g, b, a uint8) color.RGBA {
	return color.RGBA{r, g, b, a}
}
