// trophy - Terminal 3D Model Viewer
// View OBJ and GLB files in your terminal with full 3D rendering.
//
// Controls:
//
//	Mouse drag  - Rotate model (yaw/pitch)
//	Scroll      - Zoom in/out
//	W/S         - Pitch up/down
//	A/D         - Yaw left/right
//	Q/E         - Roll left/right (Q rolls left, E rolls right)
//	Space       - Apply random impulse
//	R           - Reset rotation
//	T           - Toggle texture on/off
//	X           - Toggle wireframe mode (x-ray)
//	L           - Light positioning mode (move mouse, click to set, Esc to cancel)
//	?           - Toggle HUD overlay (FPS, filename, poly count, mode status)
//	+/-         - Adjust zoom
//	Esc         - Quit (or cancel light mode)
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/harmonica"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/taigrr/trophy/pkg/math3d"
	"github.com/taigrr/trophy/pkg/models"
	"github.com/taigrr/trophy/pkg/render"
)

var (
	texturePath = flag.String("texture", "", "Path to texture image (PNG/JPG)")
	targetFPS   = flag.Int("fps", 60, "Target FPS")
	bgColor     = flag.String("bg", "30,30,40", "Background color (R,G,B)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "trophy - Terminal 3D Model Viewer\n\n")
		fmt.Fprintf(os.Stderr, "Usage: trophy [options] <model.obj|model.glb>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nControls:\n")
		fmt.Fprintf(os.Stderr, "  Mouse drag  - Rotate model\n")
		fmt.Fprintf(os.Stderr, "  Scroll      - Zoom in/out\n")
		fmt.Fprintf(os.Stderr, "  W/S/A/D     - Pitch and yaw\n")
		fmt.Fprintf(os.Stderr, "  Q/E         - Roll left/right\n")
		fmt.Fprintf(os.Stderr, "  Space       - Random spin\n")
		fmt.Fprintf(os.Stderr, "  R           - Reset view\n")
		fmt.Fprintf(os.Stderr, "  T           - Toggle texture\n")
		fmt.Fprintf(os.Stderr, "  X           - Toggle wireframe\n")
		fmt.Fprintf(os.Stderr, "  L           - Position light (mouse to aim, click to set)\n")
		fmt.Fprintf(os.Stderr, "  ?           - Toggle HUD overlay\n")
		fmt.Fprintf(os.Stderr, "  Esc         - Quit\n")
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	modelPath := flag.Arg(0)

	if err := run(modelPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// RotationAxis tracks position and velocity for one rotation axis with spring decay
type RotationAxis struct {
	Position  float64
	Velocity  float64
	velSpring harmonica.Spring
	velAccel  float64 // internal spring velocity (for animating Velocity toward 0)
}

// NewRotationAxis creates an axis with harmonica spring for smooth velocity decay
func NewRotationAxis(fps int) RotationAxis {
	return RotationAxis{
		// Frequency 4.0 = moderate speed, damping 1.0 = critically damped (no overshoot)
		velSpring: harmonica.NewSpring(harmonica.FPS(fps), 4.0, 1.0),
	}
}

// Update applies velocity to position and decays velocity toward 0 using spring
func (a *RotationAxis) Update() {
	// Apply velocity to position
	a.Position += a.Velocity

	// Use spring to animate velocity toward 0 (smooth deceleration)
	a.Velocity, a.velAccel = a.velSpring.Update(a.Velocity, a.velAccel, 0)
}

// RotationState holds rotation with harmonica spring physics
type RotationState struct {
	Pitch, Yaw, Roll RotationAxis
	fps              int
}

func NewRotationState(fps int) *RotationState {
	return &RotationState{
		Pitch: NewRotationAxis(fps),
		Yaw:   NewRotationAxis(fps),
		Roll:  NewRotationAxis(fps),
		fps:   fps,
	}
}

func (r *RotationState) Update() {
	r.Pitch.Update()
	r.Yaw.Update()
	r.Roll.Update()
}

func (r *RotationState) ApplyImpulse(pitch, yaw, roll float64) {
	r.Pitch.Velocity += pitch
	r.Yaw.Velocity += yaw
	r.Roll.Velocity += roll
}

func (r *RotationState) Reset() {
	r.Pitch = NewRotationAxis(r.fps)
	r.Yaw = NewRotationAxis(r.fps)
	r.Roll = NewRotationAxis(r.fps)
}

// RenderMode controls how the mesh is drawn
type RenderMode int

const (
	RenderModeTextured  RenderMode = iota // Textured with Gouraud shading
	RenderModeFlat                        // Flat shading (no texture)
	RenderModeWireframe                   // Wireframe only
)

// ViewState holds all view-related settings (UI state, not library code)
type ViewState struct {
	TextureEnabled bool        // Whether to show textures
	RenderMode     RenderMode  // Current render mode
	LightMode      bool        // Whether in light positioning mode
	LightDir       math3d.Vec3 // Current light direction
	PendingLight   math3d.Vec3 // Light direction while positioning
	ShowHUD        bool        // Whether to show the HUD overlay
}

// NewViewState creates default view state
func NewViewState() *ViewState {
	return &ViewState{
		TextureEnabled: true,
		RenderMode:     RenderModeTextured,
		LightMode:      false,
		LightDir:       math3d.V3(0.5, 1, 0.3).Normalize(),
	}
}

// HUD renders an overlay with model info and controls
type HUD struct {
	filename  string
	polyCount int
	fps       float64
	fpsFrames int
	fpsTime   time.Time
}

// NewHUD creates a new HUD
func NewHUD(filename string, polyCount int) *HUD {
	return &HUD{
		filename:  filename,
		polyCount: polyCount,
		fpsTime:   time.Now(),
	}
}

// UpdateFPS updates the FPS counter (call once per frame)
func (h *HUD) UpdateFPS() {
	h.fpsFrames++
	elapsed := time.Since(h.fpsTime)
	if elapsed >= time.Second {
		h.fps = float64(h.fpsFrames) / elapsed.Seconds()
		h.fpsFrames = 0
		h.fpsTime = time.Now()
	}
}

// Render draws the HUD overlay directly to the terminal
func (h *HUD) Render(width, height int, viewState *ViewState) {
	// ANSI escape codes for positioning and styling
	const (
		reset     = "\x1b[0m"
		bold      = "\x1b[1m"
		dim       = "\x1b[2m"
		bgBlack   = "\x1b[40m"
		fgWhite   = "\x1b[97m"
		fgGreen   = "\x1b[92m"
		fgYellow  = "\x1b[93m"
		fgCyan    = "\x1b[96m"
		clearLine = "\x1b[2K"
	)

	// Helper to position cursor
	moveTo := func(row, col int) string {
		return fmt.Sprintf("\x1b[%d;%dH", row, col)
	}

	// Always clear the HUD rows (so toggling off works)
	fmt.Print(moveTo(1, 1) + clearLine)
	fmt.Print(moveTo(height, 1) + clearLine)

	// Light mode always shows its indicator
	if viewState.LightMode {
		lightMsg := fmt.Sprintf("%s%s%s ◉ LIGHT MODE - Move mouse to position, click to set, Esc to cancel %s",
			bgBlack, bold, fgYellow, reset)
		lightCol := max((width-60)/2, 1)
		fmt.Print(moveTo(height, lightCol) + lightMsg)
		return
	}

	// If HUD is disabled, we're done (lines already cleared)
	if !viewState.ShowHUD {
		return
	}

	// Top left: FPS
	fpsStr := fmt.Sprintf("%s%s%s %.0f FPS %s", moveTo(1, 1), bgBlack, fgGreen, h.fps, reset)
	fmt.Print(fpsStr)

	// Top middle: filename
	titleStr := fmt.Sprintf("%s%s%s %s %s", bold, bgBlack, fgWhite, h.filename, reset)
	titleCol := max((width-len(h.filename)-2)/2, 1)
	fmt.Print(moveTo(1, titleCol) + titleStr)

	// Top right: polygon count
	polyStr := fmt.Sprintf("%s%s%s %d polys %s", bgBlack, fgCyan, bold, h.polyCount, reset)
	polyCol := max(width-12, 1)
	fmt.Print(moveTo(1, polyCol) + polyStr)

	// Bottom: mode checkboxes and hint
	checkTex := "[ ]"
	if viewState.TextureEnabled && viewState.RenderMode != RenderModeWireframe {
		checkTex = "[✓]"
	}
	checkWire := "[ ]"
	if viewState.RenderMode == RenderModeWireframe {
		checkWire = "[✓]"
	}

	// Bottom: Mode checkboxes and hint
	modeStr := fmt.Sprintf("%s%s %s Texture  %s X-Ray (wireframe) %s",
		bgBlack, fgWhite, checkTex, checkWire, reset)
	fmt.Print(moveTo(height, 1) + modeStr)

	// Light hint (right side of bottom)
	hint := fmt.Sprintf("%s%s%s L: position light %s", bgBlack, dim, fgYellow, reset)
	hintCol := max(width-18, 1)
	fmt.Print(moveTo(height, hintCol) + hint)
}

// ScreenToLightDir converts a screen position to a light direction.
// Maps screen coords to a hemisphere above the object.
func (v *ViewState) ScreenToLightDir(screenX, screenY, width, height int) math3d.Vec3 {
	// Normalize to [-1, 1]
	nx := (float64(screenX)/float64(width))*2 - 1
	ny := (float64(screenY)/float64(height))*2 - 1

	// Clamp to unit circle
	lenSq := nx*nx + ny*ny
	if lenSq > 1 {
		len := math.Sqrt(lenSq)
		nx /= len
		ny /= len
		lenSq = 1
	}

	// Z component (hemisphere projection)
	nz := math.Sqrt(1 - lenSq)

	// Return as light direction (pointing toward the object)
	return math3d.V3(nx, -ny, nz).Normalize()
}

func run(modelPath string) error {
	// Parse background color
	var bgR, bgG, bgB uint8 = 30, 30, 40
	fmt.Sscanf(*bgColor, "%d,%d,%d", &bgR, &bgG, &bgB)

	// Create terminal
	term := uv.DefaultTerminal()

	width, height, err := term.GetSize()
	if err != nil {
		return fmt.Errorf("get terminal size: %w", err)
	}

	if err := term.Start(); err != nil {
		return fmt.Errorf("start terminal: %w", err)
	}

	term.EnterAltScreen()
	term.HideCursor()
	term.Resize(width, height)

	// Enable mouse mode
	fmt.Fprint(os.Stdout, "\x1b[?1003h") // Enable any-event mouse tracking
	fmt.Fprint(os.Stdout, "\x1b[?1006h") // Enable SGR extended mouse mode

	// Create renderer
	termRenderer := render.NewTerminalRenderer(term, width, height)
	fbWidth, fbHeight := termRenderer.FramebufferSize()
	fb := render.NewFramebuffer(fbWidth, fbHeight)

	// Create camera
	camera := render.NewCamera()
	camera.SetAspectRatio(float64(fbWidth) / float64(fbHeight))
	camera.SetFOV(math.Pi / 3)
	camera.SetClipPlanes(0.1, 100)
	camera.SetPosition(math3d.V3(0, 0, 5))
	camera.LookAt(math3d.V3(0, 0, 0))

	rasterizer := render.NewRasterizer(camera, fb)

	// Load texture if specified
	var texture *render.Texture
	if *texturePath != "" {
		texture, err = render.LoadTexture(*texturePath)
		if err != nil {
			fmt.Printf("Warning: could not load texture: %v\n", err)
		}
	}

	// Load model
	ext := strings.ToLower(filepath.Ext(modelPath))
	var mesh *models.Mesh

	switch ext {
	case ".glb", ".gltf":
		var embeddedImg image.Image
		mesh, embeddedImg, err = models.LoadGLBWithTexture(modelPath)
		if err != nil {
			return fmt.Errorf("load model: %w", err)
		}
		// Use embedded texture if no explicit texture and one exists
		if texture == nil && embeddedImg != nil {
			texture = render.TextureFromImage(embeddedImg)
			fmt.Printf("Using embedded texture: %dx%d\n", embeddedImg.Bounds().Dx(), embeddedImg.Bounds().Dy())
		}
	case ".obj":
		mesh, err = models.LoadOBJ(modelPath)
		if err != nil {
			return fmt.Errorf("load model: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s (use .obj or .glb)", ext)
	}

	// Generate fallback texture if none
	if texture == nil {
		texture = render.NewCheckerTexture(64, 64, 8, render.RGB(200, 200, 200), render.RGB(100, 100, 100))
	}

	fmt.Printf("Loaded: %s (%d vertices, %d triangles)\n", filepath.Base(modelPath), mesh.VertexCount(), mesh.TriangleCount())

	// Create HUD
	hud := NewHUD(filepath.Base(modelPath), mesh.TriangleCount())

	// Center and scale model
	mesh.CalculateBounds()
	center := mesh.Center()
	size := mesh.Size()
	maxDim := math.Max(size.X, math.Max(size.Y, size.Z))
	if maxDim > 0 {
		scale := 2.0 / maxDim
		transform := math3d.Scale(math3d.V3(scale, scale, scale)).Mul(math3d.Translate(center.Scale(-1)))
		mesh.Transform(transform)
	}

	// Initialize rotation and view state
	rotation := NewRotationState(*targetFPS)
	viewState := NewViewState()

	// Context for clean shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Input state
	inputTorque := struct{ pitch, yaw, roll float64 }{}
	const torqueStrength = 3.0

	// Mouse state
	var mouseDown bool
	var lastMouseX, lastMouseY int
	cameraZ := 5.0

	// Event handler
	go func() {
		for ev := range term.Events() {
			switch ev := ev.(type) {
			case uv.WindowSizeEvent:
				width, height = ev.Width, ev.Height
				term.Erase()
				term.Resize(width, height)
				termRenderer = render.NewTerminalRenderer(term, width, height)
				fbWidth, fbHeight = termRenderer.FramebufferSize()
				fb = render.NewFramebuffer(fbWidth, fbHeight)
				rasterizer = render.NewRasterizer(camera, fb)
				camera.SetAspectRatio(float64(fbWidth) / float64(fbHeight))

			case uv.KeyPressEvent:
				switch {
				case ev.MatchString("escape"):
					if viewState.LightMode {
						// Cancel light positioning mode
						viewState.LightMode = false
					} else {
						cancel()
						return
					}
				case ev.MatchString("ctrl+c"):
					cancel()
					return
				case ev.MatchString("q"):
					inputTorque.roll = -torqueStrength
				case ev.MatchString("r"):
					rotation.Reset()
					cameraZ = 5.0
					camera.SetPosition(math3d.V3(0, 0, cameraZ))
				case ev.MatchString("w", "up"):
					inputTorque.pitch = -torqueStrength
				case ev.MatchString("s", "down"):
					inputTorque.pitch = torqueStrength
				case ev.MatchString("a", "left"):
					inputTorque.yaw = -torqueStrength
				case ev.MatchString("d", "right"):
					inputTorque.yaw = torqueStrength
				case ev.MatchString("e"):
					inputTorque.roll = torqueStrength
				case ev.MatchString("space"):
					rotation.ApplyImpulse(
						(rand.Float64()-0.5)*1.5,
						(rand.Float64()-0.5)*1.5,
						(rand.Float64()-0.5)*1.5,
					)
				case ev.MatchString("+", "="):
					cameraZ = math.Max(1, cameraZ-0.5)
					camera.SetPosition(math3d.V3(0, 0, cameraZ))
				case ev.MatchString("-", "_"):
					cameraZ = math.Min(20, cameraZ+0.5)
					camera.SetPosition(math3d.V3(0, 0, cameraZ))
				case ev.MatchString("t"):
					// Toggle texture
					viewState.TextureEnabled = !viewState.TextureEnabled
				case ev.MatchString("x"):
					// Toggle wireframe mode
					if viewState.RenderMode == RenderModeWireframe {
						viewState.RenderMode = RenderModeTextured
					} else {
						viewState.RenderMode = RenderModeWireframe
					}
				case ev.MatchString("l"):
					// Enter light positioning mode
					viewState.LightMode = true
					viewState.PendingLight = viewState.LightDir
				case ev.MatchString("?"), ev.MatchString("shift+/"):
					// Toggle HUD
					viewState.ShowHUD = !viewState.ShowHUD
				}

			case uv.KeyReleaseEvent:
				switch {
				case ev.MatchString("w"), ev.MatchString("up"), ev.MatchString("s"), ev.MatchString("down"):
					inputTorque.pitch = 0
				case ev.MatchString("a"), ev.MatchString("left"), ev.MatchString("d"), ev.MatchString("right"):
					inputTorque.yaw = 0
				case ev.MatchString("q"), ev.MatchString("e"):
					inputTorque.roll = 0
				}

			case uv.MouseClickEvent:
				if viewState.LightMode {
					// Set light position and exit light mode
					viewState.LightDir = viewState.PendingLight
					viewState.LightMode = false
				} else {
					mouseDown = true
					lastMouseX, lastMouseY = ev.X, ev.Y
				}

			case uv.MouseReleaseEvent:
				if !viewState.LightMode {
					mouseDown = false
				}

			case uv.MouseMotionEvent:
				if viewState.LightMode {
					// Update pending light direction based on mouse position
					viewState.PendingLight = viewState.ScreenToLightDir(ev.X, ev.Y, width, height)
				} else if mouseDown {
					dx := ev.X - lastMouseX
					dy := ev.Y - lastMouseY
					rotation.ApplyImpulse(float64(dy)*0.03, float64(dx)*0.03, 0)
					lastMouseX, lastMouseY = ev.X, ev.Y
				}

			case uv.MouseWheelEvent:
				switch ev.Button {
				case uv.MouseWheelUp:
					cameraZ -= 0.5
					if cameraZ < 1 {
						cameraZ = 1
					}
				case uv.MouseWheelDown:
					cameraZ += 0.5
					if cameraZ > 20 {
						cameraZ = 20
					}
				}
				camera.SetPosition(math3d.V3(0, 0, cameraZ))
			}
		}
	}()

	// Main loop
	targetDuration := time.Second / time.Duration(*targetFPS)
	lastFrame := time.Now()

	cleanup := func() {
		fmt.Fprint(os.Stdout, "\x1b[?1003l")
		fmt.Fprint(os.Stdout, "\x1b[?1006l")
		term.ExitAltScreen()
		term.ShowCursor()
		term.Shutdown(context.Background())
	}

	for {
		select {
		case <-ctx.Done():
			cleanup()
			return nil
		default:
		}

		now := time.Now()
		dt := now.Sub(lastFrame).Seconds()
		lastFrame = now

		if dt > 0.1 {
			dt = 0.1
		}

		// Apply input torque and decay it (key release events unreliable)
		rotation.ApplyImpulse(
			inputTorque.pitch*dt,
			inputTorque.yaw*dt,
			inputTorque.roll*dt,
		)
		inputTorque.pitch *= 0.9
		inputTorque.yaw *= 0.9
		inputTorque.roll *= 0.9

		// Update springs (harmonica handles timing internally)
		rotation.Update()

		// Build transform
		transform := math3d.RotateX(rotation.Pitch.Position).
			Mul(math3d.RotateY(rotation.Yaw.Position)).
			Mul(math3d.RotateZ(rotation.Roll.Position))

		// Render
		fb.Clear(render.RGB(bgR, bgG, bgB))
		rasterizer.ClearDepth()

		// Choose light direction (pending if in light mode, otherwise current)
		lightDir := viewState.LightDir
		if viewState.LightMode {
			lightDir = viewState.PendingLight
		}

		// Draw mesh based on render mode
		switch viewState.RenderMode {
		case RenderModeWireframe:
			// X-ray wireframe mode
			rasterizer.DrawMeshWireframe(mesh, transform, render.RGB(0, 255, 128))
		case RenderModeFlat:
			// Flat shading (no texture)
			rasterizer.DrawMeshGouraud(mesh, transform, render.RGB(200, 200, 200), lightDir)
		default:
			// Textured mode
			if viewState.TextureEnabled {
				rasterizer.DrawMeshTexturedGouraud(mesh, transform, texture, lightDir)
			} else {
				rasterizer.DrawMeshGouraud(mesh, transform, render.RGB(200, 200, 200), lightDir)
			}
		}

		// Display
		termRenderer.Render(fb)
		if err := termRenderer.Flush(); err != nil {
			cleanup()
			return fmt.Errorf("flush: %w", err)
		}

		// HUD overlay (always update FPS, render clears lines when HUD off)
		hud.UpdateFPS()
		hud.Render(width, height, viewState)

		// Frame timing
		elapsed := time.Since(now)
		if elapsed < targetDuration {
			time.Sleep(targetDuration - elapsed)
		}
	}
}
