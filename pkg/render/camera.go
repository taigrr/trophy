package render

import (
	"math"

	"github.com/taigrr/trophy/pkg/math3d"
)

// Camera represents a 3D camera with position and orientation.
type Camera struct {
	// Position in world space
	Position math3d.Vec3

	// Orientation (Euler angles in radians)
	Pitch float64 // Rotation around X axis (look up/down)
	Yaw   float64 // Rotation around Y axis (look left/right)
	Roll  float64 // Rotation around Z axis (tilt)

	// Projection parameters
	FOV         float64 // Vertical field of view in radians
	AspectRatio float64 // Width / Height
	Near        float64 // Near clipping plane
	Far         float64 // Far clipping plane

	// Cached matrices (computed on demand)
	viewMatrix     math3d.Mat4
	projMatrix     math3d.Mat4
	viewProjMatrix math3d.Mat4
	viewDirty      bool
	projDirty      bool
}

// NewCamera creates a new camera with default settings.
func NewCamera() *Camera {
	return &Camera{
		Position:    math3d.V3(0, 10, 0),
		Pitch:       0,
		Yaw:         0,
		Roll:        0,
		FOV:         math.Pi / 3, // 60 degrees
		AspectRatio: 16.0 / 9.0,
		Near:        0.1,
		Far:         1000,
		viewDirty:   true,
		projDirty:   true,
	}
}

// SetPosition sets the camera position.
func (c *Camera) SetPosition(pos math3d.Vec3) {
	c.Position = pos
	c.viewDirty = true
}

// SetRotation sets the camera rotation (pitch, yaw, roll in radians).
func (c *Camera) SetRotation(pitch, yaw, roll float64) {
	c.Pitch = pitch
	c.Yaw = yaw
	c.Roll = roll
	c.viewDirty = true
}

// SetFOV sets the field of view (in radians).
func (c *Camera) SetFOV(fov float64) {
	c.FOV = fov
	c.projDirty = true
}

// SetAspectRatio sets the aspect ratio.
func (c *Camera) SetAspectRatio(aspect float64) {
	c.AspectRatio = aspect
	c.projDirty = true
}

// SetClipPlanes sets the near and far clipping planes.
func (c *Camera) SetClipPlanes(near, far float64) {
	c.Near = near
	c.Far = far
	c.projDirty = true
}

// Forward returns the forward direction vector.
func (c *Camera) Forward() math3d.Vec3 {
	// Forward is -Z in camera space, rotated by yaw and pitch
	return math3d.V3(
		-math.Sin(c.Yaw)*math.Cos(c.Pitch),
		math.Sin(c.Pitch),
		-math.Cos(c.Yaw)*math.Cos(c.Pitch),
	)
}

// Right returns the right direction vector.
func (c *Camera) Right() math3d.Vec3 {
	return math3d.V3(
		math.Cos(c.Yaw),
		0,
		-math.Sin(c.Yaw),
	)
}

// Up returns the up direction vector.
func (c *Camera) Up() math3d.Vec3 {
	return c.Right().Cross(c.Forward())
}

// ViewMatrix returns the view matrix.
func (c *Camera) ViewMatrix() math3d.Mat4 {
	if c.viewDirty {
		c.computeViewMatrix()
		c.viewDirty = false
	}
	return c.viewMatrix
}

// ProjectionMatrix returns the projection matrix.
func (c *Camera) ProjectionMatrix() math3d.Mat4 {
	if c.projDirty {
		c.computeProjectionMatrix()
		c.projDirty = false
	}
	return c.projMatrix
}

// ViewProjectionMatrix returns the combined view-projection matrix.
func (c *Camera) ViewProjectionMatrix() math3d.Mat4 {
	if c.viewDirty || c.projDirty {
		_ = c.ViewMatrix()
		_ = c.ProjectionMatrix()
		c.viewProjMatrix = c.projMatrix.Mul(c.viewMatrix)
	}
	return c.viewProjMatrix
}

func (c *Camera) computeViewMatrix() {
	// Build view matrix from position and rotation
	// View = Rotation * Translation(-position)

	// Rotation matrix (inverse of camera orientation)
	rot := math3d.RotateZ(-c.Roll).Mul(
		math3d.RotateX(-c.Pitch)).Mul(
		math3d.RotateY(-c.Yaw))

	// Translation matrix (move world opposite to camera position)
	trans := math3d.Translate(c.Position.Negate())

	c.viewMatrix = rot.Mul(trans)
}

func (c *Camera) computeProjectionMatrix() {
	c.projMatrix = math3d.Perspective(c.FOV, c.AspectRatio, c.Near, c.Far)
}

// MoveForward moves the camera forward (or backward if negative).
func (c *Camera) MoveForward(distance float64) {
	c.Position = c.Position.Add(c.Forward().Scale(distance))
	c.viewDirty = true
}

// MoveRight moves the camera right (or left if negative).
func (c *Camera) MoveRight(distance float64) {
	c.Position = c.Position.Add(c.Right().Scale(distance))
	c.viewDirty = true
}

// MoveUp moves the camera up (or down if negative).
func (c *Camera) MoveUp(distance float64) {
	c.Position = c.Position.Add(math3d.Up().Scale(distance))
	c.viewDirty = true
}

// Rotate rotates the camera by the given angles (in radians).
func (c *Camera) Rotate(deltaPitch, deltaYaw, deltaRoll float64) {
	c.Pitch += deltaPitch
	c.Yaw += deltaYaw
	c.Roll += deltaRoll

	// Clamp pitch to avoid gimbal lock issues
	const maxPitch = math.Pi/2 - 0.01
	if c.Pitch > maxPitch {
		c.Pitch = maxPitch
	}
	if c.Pitch < -maxPitch {
		c.Pitch = -maxPitch
	}

	c.viewDirty = true
}

// LookAt makes the camera look at a target point.
func (c *Camera) LookAt(target math3d.Vec3) {
	dir := target.Sub(c.Position).Normalize()

	c.Pitch = math.Asin(dir.Y)
	c.Yaw = math.Atan2(-dir.X, -dir.Z)
	c.Roll = 0

	c.viewDirty = true
}

// WorldToScreen transforms a world point to screen coordinates.
// Returns (screenX, screenY, depth, visible).
func (c *Camera) WorldToScreen(worldPos math3d.Vec3, screenWidth, screenHeight int) (x, y, depth float64, visible bool) {
	// Transform to clip space
	clipPos := c.ViewProjectionMatrix().MulVec4(math3d.V4FromV3(worldPos, 1))

	// Check if behind camera
	if clipPos.W <= 0 {
		return 0, 0, 0, false
	}

	// Perspective divide to NDC (-1 to 1)
	ndc := clipPos.PerspectiveDivide()

	// Check if in view frustum
	if ndc.X < -1 || ndc.X > 1 || ndc.Y < -1 || ndc.Y > 1 || ndc.Z < -1 || ndc.Z > 1 {
		return 0, 0, 0, false
	}

	// Convert to screen coordinates
	x = (ndc.X + 1) * 0.5 * float64(screenWidth)
	y = (1 - ndc.Y) * 0.5 * float64(screenHeight) // Y is flipped
	depth = ndc.Z

	return x, y, depth, true
}
