package main

import (
	"math"
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

func TestNewRotationAxis(t *testing.T) {
	axis := NewRotationAxis(60)
	if axis.Position != 0 {
		t.Errorf("expected Position=0, got %f", axis.Position)
	}
	if axis.Velocity != 0 {
		t.Errorf("expected Velocity=0, got %f", axis.Velocity)
	}
}

func TestRotationAxisUpdate(t *testing.T) {
	axis := NewRotationAxis(60)
	axis.Velocity = 1.0
	axis.Update(false) // no damping
	if axis.Position != 1.0 {
		t.Errorf("expected Position=1.0 after update, got %f", axis.Position)
	}
	// Velocity unchanged without damping
	if axis.Velocity != 1.0 {
		t.Errorf("expected Velocity=1.0 without damping, got %f", axis.Velocity)
	}
}

func TestRotationAxisUpdateWithDamping(t *testing.T) {
	axis := NewRotationAxis(60)
	axis.Velocity = 1.0
	for range 120 {
		axis.Update(true)
	}
	// After many damped updates, velocity should decay toward 0
	if math.Abs(axis.Velocity) > 0.01 {
		t.Errorf("expected velocity near 0 after damping, got %f", axis.Velocity)
	}
}

func TestNewRotationState(t *testing.T) {
	rs := NewRotationState(60)
	if rs.fps != 60 {
		t.Errorf("expected fps=60, got %d", rs.fps)
	}
	if rs.Pitch.Position != 0 || rs.Yaw.Position != 0 || rs.Roll.Position != 0 {
		t.Error("expected all positions to start at 0")
	}
}

func TestRotationStateApplyImpulse(t *testing.T) {
	rs := NewRotationState(60)
	rs.ApplyImpulse(1.0, 2.0, 3.0)
	if rs.Pitch.Velocity != 1.0 {
		t.Errorf("expected Pitch.Velocity=1.0, got %f", rs.Pitch.Velocity)
	}
	if rs.Yaw.Velocity != 2.0 {
		t.Errorf("expected Yaw.Velocity=2.0, got %f", rs.Yaw.Velocity)
	}
	if rs.Roll.Velocity != 3.0 {
		t.Errorf("expected Roll.Velocity=3.0, got %f", rs.Roll.Velocity)
	}
}

func TestRotationStateReset(t *testing.T) {
	rs := NewRotationState(60)
	rs.ApplyImpulse(1.0, 2.0, 3.0)
	rs.Update(false)
	rs.Reset()
	if rs.Pitch.Position != 0 || rs.Yaw.Position != 0 || rs.Roll.Position != 0 {
		t.Error("expected all positions to be 0 after reset")
	}
	if rs.Pitch.Velocity != 0 || rs.Yaw.Velocity != 0 || rs.Roll.Velocity != 0 {
		t.Error("expected all velocities to be 0 after reset")
	}
}

func TestRotationStateUpdate(t *testing.T) {
	rs := NewRotationState(60)
	rs.ApplyImpulse(0.1, 0.2, 0.3)
	rs.Update(false)
	if rs.Pitch.Position != 0.1 {
		t.Errorf("expected Pitch.Position=0.1, got %f", rs.Pitch.Position)
	}
	if rs.Yaw.Position != 0.2 {
		t.Errorf("expected Yaw.Position=0.2, got %f", rs.Yaw.Position)
	}
	if rs.Roll.Position != 0.3 {
		t.Errorf("expected Roll.Position=0.3, got %f", rs.Roll.Position)
	}
}

func TestRenderModeConstants(t *testing.T) {
	if RenderModeTextured != 0 {
		t.Error("expected RenderModeTextured=0")
	}
	if RenderModeFlat != 1 {
		t.Error("expected RenderModeFlat=1")
	}
	if RenderModeWireframe != 2 {
		t.Error("expected RenderModeWireframe=2")
	}
}

func TestNewViewState(t *testing.T) {
	vs := NewViewState()
	if !vs.TextureEnabled {
		t.Error("expected TextureEnabled=true by default")
	}
	if vs.RenderMode != RenderModeTextured {
		t.Errorf("expected RenderMode=RenderModeTextured, got %d", vs.RenderMode)
	}
	if vs.LightMode {
		t.Error("expected LightMode=false by default")
	}
	if vs.ShowHUD {
		t.Error("expected ShowHUD=false by default")
	}
	if vs.SpinMode {
		t.Error("expected SpinMode=false by default")
	}
	if vs.BackfaceCull {
		t.Error("expected BackfaceCull=false by default")
	}
}

func TestScreenToLightDir(t *testing.T) {
	vs := NewViewState()

	// Center of screen should give (0, 0, 1) — light pointing straight at object
	dir := vs.ScreenToLightDir(50, 50, 100, 100)
	if math.Abs(dir.Z-1.0) > 0.01 {
		t.Errorf("center should have Z~1.0, got %f", dir.Z)
	}
	if math.Abs(dir.X) > 0.01 || math.Abs(dir.Y) > 0.01 {
		t.Errorf("center should have X,Y~0, got (%f, %f)", dir.X, dir.Y)
	}

	// Top-left corner — should have negative X, positive Y (inverted), and Z > 0
	dir = vs.ScreenToLightDir(0, 0, 100, 100)
	if dir.X >= 0 {
		t.Errorf("top-left should have X<0, got %f", dir.X)
	}
	if dir.Y <= 0 {
		t.Errorf("top-left should have Y>0 (inverted), got %f", dir.Y)
	}

	// Result should be normalized
	length := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y + dir.Z*dir.Z)
	if math.Abs(length-1.0) > 0.001 {
		t.Errorf("expected normalized vector (length=1.0), got %f", length)
	}
}

func TestScreenToLightDirEdge(t *testing.T) {
	vs := NewViewState()
	// Far corner — clamped to unit circle, should still be normalized
	dir := vs.ScreenToLightDir(100, 100, 100, 100)
	length := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y + dir.Z*dir.Z)
	if math.Abs(length-1.0) > 0.001 {
		t.Errorf("expected normalized vector at edge, got length=%f", length)
	}
}

func TestNewHUD(t *testing.T) {
	vs := NewViewState()
	hud := NewHUD("test.obj", 1000, vs)
	if hud.filename != "test.obj" {
		t.Errorf("expected filename=test.obj, got %s", hud.filename)
	}
	if hud.polyCount != 1000 {
		t.Errorf("expected polyCount=1000, got %d", hud.polyCount)
	}
	if hud.fps != 0 {
		t.Errorf("expected initial fps=0, got %f", hud.fps)
	}
}

func TestHUDUpdateFPS(t *testing.T) {
	vs := NewViewState()
	hud := NewHUD("test.obj", 1000, vs)
	// Just verify it doesn't panic
	for range 10 {
		hud.UpdateFPS()
	}
}

func TestRotationStateMultipleImpulses(t *testing.T) {
	rs := NewRotationState(60)
	rs.ApplyImpulse(1.0, 0, 0)
	rs.ApplyImpulse(0.5, 0, 0)
	if rs.Pitch.Velocity != 1.5 {
		t.Errorf("expected accumulated velocity=1.5, got %f", rs.Pitch.Velocity)
	}
}

func TestViewStateLightDirDefault(t *testing.T) {
	vs := NewViewState()
	// Default light direction should be normalized
	dir := vs.LightDir
	length := math.Sqrt(dir.X*dir.X + dir.Y*dir.Y + dir.Z*dir.Z)
	if math.Abs(length-1.0) > 0.001 {
		t.Errorf("expected normalized default light dir, got length=%f", length)
	}
}

func TestScreenToLightDirSymmetry(t *testing.T) {
	vs := NewViewState()
	// Left and right of center should be symmetric in X
	left := vs.ScreenToLightDir(25, 50, 100, 100)
	right := vs.ScreenToLightDir(75, 50, 100, 100)
	if math.Abs(left.X+right.X) > 0.001 {
		t.Errorf("expected X symmetry: left.X=%f, right.X=%f", left.X, right.X)
	}
	if math.Abs(left.Z-right.Z) > 0.001 {
		t.Errorf("expected same Z: left.Z=%f, right.Z=%f", left.Z, right.Z)
	}
}

func TestRunInfoUnsupportedFormat(t *testing.T) {
	err := runInfo("test.xyz")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestRunInfoMissingFile(t *testing.T) {
	err := runInfo("nonexistent.obj")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// Verify Vec3 import works correctly with the test helpers
func TestVec3Integration(t *testing.T) {
	v := math3d.V3(1, 2, 3)
	if v.X != 1 || v.Y != 2 || v.Z != 3 {
		t.Errorf("expected (1,2,3), got (%f,%f,%f)", v.X, v.Y, v.Z)
	}
}
