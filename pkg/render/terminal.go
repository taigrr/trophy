package render

import (
	"image/color"

	uv "github.com/charmbracelet/ultraviolet"
)

// TerminalRenderer converts a Framebuffer to Ultraviolet cells.
// It uses half-block characters (▀) to achieve 2x vertical resolution.
type TerminalRenderer struct {
	term   *uv.Terminal
	width  int // Terminal columns
	height int // Terminal rows
}

// NewTerminalRenderer creates a renderer for the given terminal.
func NewTerminalRenderer(term *uv.Terminal, width, height int) *TerminalRenderer {
	return &TerminalRenderer{
		term:   term,
		width:  width,
		height: height,
	}
}

// Render converts the framebuffer to terminal cells and displays them.
// The framebuffer height should be 2x the terminal height.
func (r *TerminalRenderer) Render(fb *Framebuffer) {
	// Each terminal row represents 2 framebuffer rows
	// We use ▀ (upper half block) with fg=top color and bg=bottom color

	for row := 0; row < r.height; row++ {
		topY := row * 2
		botY := topY + 1

		for col := 0; col < r.width && col < fb.Width; col++ {
			topColor := fb.GetPixel(col, topY)
			botColor := fb.GetPixel(col, botY)

			cell := &uv.Cell{
				Content: "▀",
				Width:   1,
				Style: uv.Style{
					Fg: rgbaToColor(topColor),
					Bg: rgbaToColor(botColor),
				},
			}
			r.term.SetCell(col, row, cell)
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

// Flush sends the rendered content to the terminal.
func (r *TerminalRenderer) Flush() error {
	return r.term.Display()
}

// FramebufferSize returns the recommended framebuffer size for the terminal.
// Height is 2x terminal rows for half-block rendering.
func (r *TerminalRenderer) FramebufferSize() (width, height int) {
	return r.width, r.height * 2
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
