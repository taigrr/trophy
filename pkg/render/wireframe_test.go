package render

import (
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

func TestNewWireframe(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(0, 0, 10))
	c.LookAt(math3d.V3(0, 0, 0))
	fb := NewFramebuffer(80, 40)
	w := NewWireframe(c, fb)
	if w == nil {
		t.Fatal("nil wireframe")
	}
}

func TestWireframeDrawLine3D(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(0, 0, 10))
	c.LookAt(math3d.V3(0, 0, 0))
	fb := NewFramebuffer(80, 40)
	w := NewWireframe(c, fb)

	// Visible line
	w.DrawLine3D(math3d.V3(-1, 0, 0), math3d.V3(1, 0, 0), ColorWhite)

	// Both behind camera - should not crash
	w.DrawLine3D(math3d.V3(0, 0, 20), math3d.V3(0, 0, 30), ColorWhite)
}

func TestWireframeDrawCube(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(0, 5, 15))
	c.LookAt(math3d.V3(0, 0, 0))
	fb := NewFramebuffer(80, 40)
	w := NewWireframe(c, fb)
	w.DrawCube(math3d.V3(0, 0, 0), 2, ColorWhite)
}

func TestWireframeDrawTransformedCube(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(0, 5, 15))
	c.LookAt(math3d.V3(0, 0, 0))
	fb := NewFramebuffer(80, 40)
	w := NewWireframe(c, fb)
	transform := math3d.Translate(math3d.V3(2, 0, 0)).Mul(math3d.RotateY(0.5))
	w.DrawTransformedCube(transform, 2, ColorRed)
}

func TestWireframeDrawAxes(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(5, 5, 5))
	c.LookAt(math3d.V3(0, 0, 0))
	fb := NewFramebuffer(80, 40)
	w := NewWireframe(c, fb)
	w.DrawAxes(3)
}

func TestWireframeDrawGrid(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(0, 10, 10))
	c.LookAt(math3d.V3(0, 0, 0))
	fb := NewFramebuffer(80, 40)
	w := NewWireframe(c, fb)
	w.DrawGrid(10, 2, ColorGreen)
}

func TestWireframeDrawPoint(t *testing.T) {
	c := NewCamera()
	c.SetPosition(math3d.V3(0, 0, 5))
	c.LookAt(math3d.V3(0, 0, 0))
	fb := NewFramebuffer(80, 40)
	w := NewWireframe(c, fb)
	w.DrawPoint(math3d.V3(0, 0, 0), 0.5, ColorBlue)
}
