// Package render provides software rasterization for Trophy.
package render

import (
	"github.com/taigrr/trophy/pkg/math3d"
)

// Plane represents a plane in 3D space using the equation: Ax + By + Cz + D = 0
// where (A, B, C) is the normal and D is the distance from origin.
type Plane struct {
	Normal math3d.Vec3
	D      float64
}

// Normalize normalizes the plane equation so the normal has unit length.
func (p *Plane) Normalize() {
	len := p.Normal.Len()
	if len == 0 {
		return
	}
	p.Normal = p.Normal.Scale(1.0 / len)
	p.D /= len
}

// DistanceToPoint returns the signed distance from the plane to a point.
// Positive = in front (same side as normal), negative = behind.
func (p Plane) DistanceToPoint(point math3d.Vec3) float64 {
	return p.Normal.Dot(point) + p.D
}

// SignedDistance is an alias for DistanceToPoint.
func (p Plane) SignedDistance(point math3d.Vec3) float64 {
	return p.DistanceToPoint(point)
}

// Frustum represents the 6 planes of a view frustum.
// Planes are ordered: Left, Right, Bottom, Top, Near, Far.
// Each plane's normal points inward (toward the center of the frustum).
type Frustum struct {
	Planes [6]Plane
}

// FrustumPlane indices for clarity.
const (
	FrustumLeft = iota
	FrustumRight
	FrustumBottom
	FrustumTop
	FrustumNear
	FrustumFar
)

// NewFrustumFromMatrix extracts frustum planes from a view-projection matrix.
// Uses the Gribb/Hartmann method for extracting planes from the combined matrix.
// The resulting planes have normals pointing inward.
func NewFrustumFromMatrix(m math3d.Mat4) Frustum {
	var f Frustum

	// Extract rows of the matrix for plane extraction.
	// For column-major matrix m, row i element j is at m[i + j*4].
	// Row 0: m[0], m[4], m[8], m[12]
	// Row 1: m[1], m[5], m[9], m[13]
	// Row 2: m[2], m[6], m[10], m[14]
	// Row 3: m[3], m[7], m[11], m[15]

	// Left plane: row3 + row0
	f.Planes[FrustumLeft] = Plane{
		Normal: math3d.V3(m[3]+m[0], m[7]+m[4], m[11]+m[8]),
		D:      m[15] + m[12],
	}

	// Right plane: row3 - row0
	f.Planes[FrustumRight] = Plane{
		Normal: math3d.V3(m[3]-m[0], m[7]-m[4], m[11]-m[8]),
		D:      m[15] - m[12],
	}

	// Bottom plane: row3 + row1
	f.Planes[FrustumBottom] = Plane{
		Normal: math3d.V3(m[3]+m[1], m[7]+m[5], m[11]+m[9]),
		D:      m[15] + m[13],
	}

	// Top plane: row3 - row1
	f.Planes[FrustumTop] = Plane{
		Normal: math3d.V3(m[3]-m[1], m[7]-m[5], m[11]-m[9]),
		D:      m[15] - m[13],
	}

	// Near plane: row3 + row2
	f.Planes[FrustumNear] = Plane{
		Normal: math3d.V3(m[3]+m[2], m[7]+m[6], m[11]+m[10]),
		D:      m[15] + m[14],
	}

	// Far plane: row3 - row2
	f.Planes[FrustumFar] = Plane{
		Normal: math3d.V3(m[3]-m[2], m[7]-m[6], m[11]-m[10]),
		D:      m[15] - m[14],
	}

	// Normalize all planes
	for i := range f.Planes {
		f.Planes[i].Normalize()
	}

	return f
}

// AABB represents an axis-aligned bounding box.
type AABB struct {
	Min math3d.Vec3
	Max math3d.Vec3
}

// NewAABB creates an AABB from min and max points.
func NewAABB(min, max math3d.Vec3) AABB {
	return AABB{Min: min, Max: max}
}

// Center returns the center of the AABB.
func (b AABB) Center() math3d.Vec3 {
	return b.Min.Add(b.Max).Scale(0.5)
}

// Size returns the dimensions of the AABB.
func (b AABB) Size() math3d.Vec3 {
	return b.Max.Sub(b.Min)
}

// HalfSize returns half the dimensions (extents from center).
func (b AABB) HalfSize() math3d.Vec3 {
	return b.Size().Scale(0.5)
}

// Extents is an alias for HalfSize.
func (b AABB) Extents() math3d.Vec3 {
	return b.HalfSize()
}

// Transform returns an AABB that bounds the original AABB after transformation.
// This computes a new AABB that contains all 8 transformed corners.
func (b AABB) Transform(m math3d.Mat4) AABB {
	// Get all 8 corners
	corners := [8]math3d.Vec3{
		{X: b.Min.X, Y: b.Min.Y, Z: b.Min.Z},
		{X: b.Max.X, Y: b.Min.Y, Z: b.Min.Z},
		{X: b.Min.X, Y: b.Max.Y, Z: b.Min.Z},
		{X: b.Max.X, Y: b.Max.Y, Z: b.Min.Z},
		{X: b.Min.X, Y: b.Min.Y, Z: b.Max.Z},
		{X: b.Max.X, Y: b.Min.Y, Z: b.Max.Z},
		{X: b.Min.X, Y: b.Max.Y, Z: b.Max.Z},
		{X: b.Max.X, Y: b.Max.Y, Z: b.Max.Z},
	}

	// Transform all corners and find new bounds
	transformed := m.MulVec3(corners[0])
	newMin := transformed
	newMax := transformed

	for i := 1; i < 8; i++ {
		transformed = m.MulVec3(corners[i])
		newMin = newMin.Min(transformed)
		newMax = newMax.Max(transformed)
	}

	return AABB{Min: newMin, Max: newMax}
}

// ContainsPoint returns true if the point is inside the AABB.
func (b AABB) ContainsPoint(p math3d.Vec3) bool {
	return p.X >= b.Min.X && p.X <= b.Max.X &&
		p.Y >= b.Min.Y && p.Y <= b.Max.Y &&
		p.Z >= b.Min.Z && p.Z <= b.Max.Z
}

// IntersectAABB tests if the AABB intersects or is inside the frustum.
// Returns true if any part of the AABB is visible.
// Uses the "positive vertex" optimization for faster rejection.
func (f Frustum) IntersectAABB(box AABB) bool {
	for i := range f.Planes {
		plane := f.Planes[i]

		// Find the "positive vertex" - the corner of the AABB furthest in the direction of the plane normal.
		// This is the corner that would be outside if the entire box is outside.
		pVertex := math3d.V3(
			selectComponent(plane.Normal.X >= 0, box.Max.X, box.Min.X),
			selectComponent(plane.Normal.Y >= 0, box.Max.Y, box.Min.Y),
			selectComponent(plane.Normal.Z >= 0, box.Max.Z, box.Min.Z),
		)

		// If the positive vertex is outside this plane, the entire box is outside the frustum
		if plane.DistanceToPoint(pVertex) < 0 {
			return false
		}
	}

	// The box is at least partially inside all planes
	return true
}

// ContainsAABB tests if the AABB is completely inside the frustum.
// Returns true only if all 8 corners are inside all 6 planes.
func (f Frustum) ContainsAABB(box AABB) bool {
	for i := range f.Planes {
		plane := f.Planes[i]

		// Find the "negative vertex" - the corner closest to the plane in the normal direction.
		nVertex := math3d.V3(
			selectComponent(plane.Normal.X >= 0, box.Min.X, box.Max.X),
			selectComponent(plane.Normal.Y >= 0, box.Min.Y, box.Max.Y),
			selectComponent(plane.Normal.Z >= 0, box.Min.Z, box.Max.Z),
		)

		// If the negative vertex is outside, the box is not fully contained
		if plane.DistanceToPoint(nVertex) < 0 {
			return false
		}
	}

	return true
}

// ContainsPoint tests if a point is inside the frustum.
func (f Frustum) ContainsPoint(p math3d.Vec3) bool {
	for i := range f.Planes {
		if f.Planes[i].DistanceToPoint(p) < 0 {
			return false
		}
	}
	return true
}

// IntersectsSphere tests if a sphere intersects the frustum.
// center is the sphere center, radius is the sphere radius.
func (f Frustum) IntersectsSphere(center math3d.Vec3, radius float64) bool {
	for i := range f.Planes {
		if f.Planes[i].DistanceToPoint(center) < -radius {
			return false
		}
	}
	return true
}

// selectComponent is a branchless conditional selection helper.
func selectComponent(cond bool, a, b float64) float64 {
	if cond {
		return a
	}
	return b
}

// GetFrustum returns the current view frustum from the camera.
func (c *Camera) GetFrustum() Frustum {
	return NewFrustumFromMatrix(c.ViewProjectionMatrix())
}

// ExtractFrustum is an alias for NewFrustumFromMatrix for API consistency.
func ExtractFrustum(m math3d.Mat4) Frustum {
	return NewFrustumFromMatrix(m)
}

// IntersectsFrustum is an alias for IntersectAABB for API consistency.
func (f Frustum) IntersectsFrustum(box AABB) bool {
	return f.IntersectAABB(box)
}

// TransformAABB transforms an AABB by a matrix and returns the new bounds.
// This is a convenience function wrapping AABB.Transform.
func TransformAABB(box AABB, m math3d.Mat4) AABB {
	return box.Transform(m)
}
