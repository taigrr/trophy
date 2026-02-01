package render

import (
	"github.com/taigrr/trophy/pkg/math3d"
)

// Wireframe renders 3D wireframe objects.
type Wireframe struct {
	camera *Camera
	fb     *Framebuffer
}

// NewWireframe creates a new wireframe renderer.
func NewWireframe(camera *Camera, fb *Framebuffer) *Wireframe {
	return &Wireframe{
		camera: camera,
		fb:     fb,
	}
}

// DrawLine3D draws a line in 3D space.
func (w *Wireframe) DrawLine3D(p1, p2 math3d.Vec3, color Color) {
	// Project both endpoints
	x1, y1, _, vis1 := w.camera.WorldToScreen(p1, w.fb.Width, w.fb.Height)
	x2, y2, _, vis2 := w.camera.WorldToScreen(p2, w.fb.Width, w.fb.Height)

	// Simple clipping: only draw if at least one point is visible
	// (proper line clipping would be more complex)
	if !vis1 && !vis2 {
		return
	}

	// Draw the line
	w.fb.DrawLine(int(x1), int(y1), int(x2), int(y2), color)
}

// DrawCube draws a wireframe cube.
func (w *Wireframe) DrawCube(center math3d.Vec3, size float64, color Color) {
	half := size / 2

	// 8 vertices of the cube
	vertices := [8]math3d.Vec3{
		{X: center.X - half, Y: center.Y - half, Z: center.Z - half}, // 0: bottom-left-back
		{X: center.X + half, Y: center.Y - half, Z: center.Z - half}, // 1: bottom-right-back
		{X: center.X + half, Y: center.Y + half, Z: center.Z - half}, // 2: top-right-back
		{X: center.X - half, Y: center.Y + half, Z: center.Z - half}, // 3: top-left-back
		{X: center.X - half, Y: center.Y - half, Z: center.Z + half}, // 4: bottom-left-front
		{X: center.X + half, Y: center.Y - half, Z: center.Z + half}, // 5: bottom-right-front
		{X: center.X + half, Y: center.Y + half, Z: center.Z + half}, // 6: top-right-front
		{X: center.X - half, Y: center.Y + half, Z: center.Z + half}, // 7: top-left-front
	}

	// 12 edges of the cube
	edges := [][2]int{
		// Back face
		{0, 1},
		{1, 2},
		{2, 3},
		{3, 0},
		// Front face
		{4, 5},
		{5, 6},
		{6, 7},
		{7, 4},
		// Connecting edges
		{0, 4},
		{1, 5},
		{2, 6},
		{3, 7},
	}

	for _, edge := range edges {
		w.DrawLine3D(vertices[edge[0]], vertices[edge[1]], color)
	}
}

// DrawTransformedCube draws a wireframe cube with a transformation matrix.
func (w *Wireframe) DrawTransformedCube(transform math3d.Mat4, size float64, color Color) {
	half := size / 2

	// Local vertices (centered at origin)
	localVerts := [8]math3d.Vec3{
		{X: -half, Y: -half, Z: -half},
		{X: half, Y: -half, Z: -half},
		{X: half, Y: half, Z: -half},
		{X: -half, Y: half, Z: -half},
		{X: -half, Y: -half, Z: half},
		{X: half, Y: -half, Z: half},
		{X: half, Y: half, Z: half},
		{X: -half, Y: half, Z: half},
	}

	// Transform vertices
	var worldVerts [8]math3d.Vec3
	for i, v := range localVerts {
		worldVerts[i] = transform.MulVec3(v)
	}

	// 12 edges
	edges := [][2]int{
		{0, 1},
		{1, 2},
		{2, 3},
		{3, 0},
		{4, 5},
		{5, 6},
		{6, 7},
		{7, 4},
		{0, 4},
		{1, 5},
		{2, 6},
		{3, 7},
	}

	for _, edge := range edges {
		w.DrawLine3D(worldVerts[edge[0]], worldVerts[edge[1]], color)
	}
}

// DrawAxes draws the coordinate axes at the origin.
func (w *Wireframe) DrawAxes(length float64) {
	origin := math3d.Zero3()
	w.DrawLine3D(origin, math3d.V3(length, 0, 0), ColorRed)   // X axis
	w.DrawLine3D(origin, math3d.V3(0, length, 0), ColorGreen) // Y axis
	w.DrawLine3D(origin, math3d.V3(0, 0, length), ColorBlue)  // Z axis
}

// DrawGrid draws a grid on the XZ plane at y=0.
func (w *Wireframe) DrawGrid(size, step float64, color Color) {
	half := size / 2
	for x := -half; x <= half; x += step {
		w.DrawLine3D(math3d.V3(x, 0, -half), math3d.V3(x, 0, half), color)
	}
	for z := -half; z <= half; z += step {
		w.DrawLine3D(math3d.V3(-half, 0, z), math3d.V3(half, 0, z), color)
	}
}

// DrawPoint draws a point as a small cross.
func (w *Wireframe) DrawPoint(pos math3d.Vec3, size float64, color Color) {
	halfSize := size / 2
	w.DrawLine3D(
		math3d.V3(pos.X-halfSize, pos.Y, pos.Z),
		math3d.V3(pos.X+halfSize, pos.Y, pos.Z),
		color,
	)
	w.DrawLine3D(
		math3d.V3(pos.X, pos.Y-halfSize, pos.Z),
		math3d.V3(pos.X, pos.Y+halfSize, pos.Z),
		color,
	)
	w.DrawLine3D(
		math3d.V3(pos.X, pos.Y, pos.Z-halfSize),
		math3d.V3(pos.X, pos.Y, pos.Z+halfSize),
		color,
	)
}
