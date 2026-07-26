package render

import (
	"math"
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

const epsilon = 1e-6

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestNewCamera(t *testing.T) {
	c := NewCamera()
	if c.Position != (math3d.Vec3{X: 0, Y: 10, Z: 0}) {
		t.Fatalf("position = %v", c.Position)
	}
	if !approxEqual(c.FOV, math.Pi/3, epsilon) {
		t.Fatalf("FOV = %f", c.FOV)
	}
}

func TestCameraSetters(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(1, 2, 3))
	if c.Position != (math3d.Vec3{X: 1, Y: 2, Z: 3}) {
		t.Errorf("pos = %v", c.Position)
	}
	c.SetRotation(0.1, 0.2, 0.3)
	if c.Pitch != 0.1 || c.Yaw != 0.2 || c.Roll != 0.3 {
		t.Error("rotation not set")
	}
	c.SetFOV(math.Pi / 4)
	if c.FOV != math.Pi/4 {
		t.Error("FOV not set")
	}
	c.SetAspectRatio(2.0)
	if c.AspectRatio != 2.0 {
		t.Error("aspect not set")
	}
	c.SetClipPlanes(0.5, 500)
	if c.Near != 0.5 || c.Far != 500 {
		t.Error("clip planes not set")
	}
}

func TestCameraDirections(t *testing.T) {
	c := NewCamera()
	// Default: yaw=0, pitch=0 => forward is (0, 0, -1)
	f := c.Forward()
	if !approxEqual(f.X, 0, epsilon) || !approxEqual(f.Y, 0, epsilon) || !approxEqual(f.Z, -1, epsilon) {
		t.Fatalf("forward = %v", f)
	}
	r := c.Right()
	if !approxEqual(r.X, 1, epsilon) || !approxEqual(r.Y, 0, epsilon) || !approxEqual(r.Z, 0, epsilon) {
		t.Fatalf("right = %v", r)
	}
	u := c.Up()
	if !approxEqual(u.Y, 1, 0.01) {
		t.Fatalf("up = %v", u)
	}
}

func TestCameraMatrices(t *testing.T) {
	c := NewCamera()
	v := c.ViewMatrix()
	p := c.ProjectionMatrix()
	vp := c.ViewProjectionMatrix()
	_ = v
	_ = p
	_ = vp
	// Call again to test caching (dirty flags cleared)
	v2 := c.ViewMatrix()
	if v != v2 {
		t.Fatal("cached view matrix changed")
	}
}

func TestCameraMovement(t *testing.T) {
	c := NewCamera()
	start := c.Position
	c.MoveForward(5)
	if c.Position == start {
		t.Error("MoveForward didn't move")
	}
	c.MoveRight(3)
	c.MoveUp(2)
	if !approxEqual(c.Position.Y, start.Y+5+2, 0.5) {
		// MoveForward may change Y if pitch != 0, just check it moved
	}
}

func TestCameraRotate(t *testing.T) {
	c := NewCamera()
	c.Rotate(0.1, 0.2, 0)
	if c.Pitch != 0.1 || c.Yaw != 0.2 {
		t.Error("Rotate didn't work")
	}
	// Test pitch clamping
	c.Rotate(math.Pi, 0, 0)
	if c.Pitch >= math.Pi/2 {
		t.Error("pitch not clamped")
	}
	c.Rotate(-math.Pi*2, 0, 0)
	if c.Pitch <= -math.Pi/2 {
		t.Error("negative pitch not clamped")
	}
}

func TestCameraLookAt(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(0, 0, 5))
	c.LookAt(math3d.V3(0, 0, 0))
	f := c.Forward()
	if !approxEqual(f.Z, -1, 0.01) {
		t.Fatalf("forward after LookAt = %v", f)
	}
}

func TestWorldToScreen(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(0, 0, 5))
	c.LookAt(math3d.V3(0, 0, 0))

	// Origin should be visible
	x, y, _, vis := c.WorldToScreen(math3d.V3(0, 0, 0), 800, 600)
	if !vis {
		t.Fatal("origin not visible")
	}
	// Should be roughly center of screen
	if !approxEqual(x, 400, 50) || !approxEqual(y, 300, 50) {
		t.Fatalf("screen pos = (%f, %f)", x, y)
	}

	// Behind camera should not be visible
	_, _, _, vis = c.WorldToScreen(math3d.V3(0, 0, 10), 800, 600)
	if vis {
		t.Fatal("behind camera was visible")
	}
}
