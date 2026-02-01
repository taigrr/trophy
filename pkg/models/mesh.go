// Package models provides 3D model loading and representation for Trophy.
package models

import (
	"image"

	"github.com/taigrr/trophy/pkg/math3d"
)

// Mesh represents a 3D mesh with vertices, faces, and materials.
type Mesh struct {
	Name      string
	Vertices  []MeshVertex
	Faces     []Face
	Materials []Material

	// Bounding box (calculated on load)
	BoundsMin math3d.Vec3
	BoundsMax math3d.Vec3
}

// MeshVertex holds all vertex attributes.
type MeshVertex struct {
	Position math3d.Vec3
	Normal   math3d.Vec3
	UV       math3d.Vec2
}

// Face represents a triangle face with vertex indices and material reference.
type Face struct {
	V        [3]int // Indices into Mesh.Vertices
	Material int    // Index into Mesh.Materials (-1 for no material)
}

// Material represents a PBR material from GLTF.
type Material struct {
	Name       string
	BaseColor  [4]float64  // RGBA in 0-1 range
	Metallic   float64     // 0 = dielectric, 1 = metal
	Roughness  float64     // 0 = smooth, 1 = rough
	BaseMap    image.Image // Optional base color texture
	HasTexture bool
}

// NewMesh creates an empty mesh.
func NewMesh(name string) *Mesh {
	return &Mesh{
		Name:      name,
		Vertices:  make([]MeshVertex, 0),
		Faces:     make([]Face, 0),
		BoundsMin: math3d.V3(0, 0, 0),
		BoundsMax: math3d.V3(0, 0, 0),
	}
}

// CalculateBounds computes the axis-aligned bounding box.
func (m *Mesh) CalculateBounds() {
	if len(m.Vertices) == 0 {
		return
	}

	m.BoundsMin = m.Vertices[0].Position
	m.BoundsMax = m.Vertices[0].Position

	for _, v := range m.Vertices[1:] {
		m.BoundsMin = m.BoundsMin.Min(v.Position)
		m.BoundsMax = m.BoundsMax.Max(v.Position)
	}
}

// Center returns the center of the bounding box.
func (m *Mesh) Center() math3d.Vec3 {
	return m.BoundsMin.Add(m.BoundsMax).Scale(0.5)
}

// Size returns the dimensions of the bounding box.
func (m *Mesh) Size() math3d.Vec3 {
	return m.BoundsMax.Sub(m.BoundsMin)
}

// TriangleCount returns the number of triangles.
func (m *Mesh) TriangleCount() int {
	return len(m.Faces)
}

// VertexCount returns the number of vertices.
func (m *Mesh) VertexCount() int {
	return len(m.Vertices)
}

// CalculateNormals computes face normals and assigns them to vertices.
// This is a simple flat-shading approach; for smooth shading, normals
// should be averaged per-vertex.
func (m *Mesh) CalculateNormals() {
	for i := range m.Faces {
		f := &m.Faces[i]
		v0 := m.Vertices[f.V[0]].Position
		v1 := m.Vertices[f.V[1]].Position
		v2 := m.Vertices[f.V[2]].Position

		edge1 := v1.Sub(v0)
		edge2 := v2.Sub(v0)
		normal := edge1.Cross(edge2).Normalize()

		// Assign to vertices (flat shading - each face has its own normal)
		m.Vertices[f.V[0]].Normal = normal
		m.Vertices[f.V[1]].Normal = normal
		m.Vertices[f.V[2]].Normal = normal
	}
}

// CalculateSmoothNormals computes averaged normals for smooth shading.
func (m *Mesh) CalculateSmoothNormals() {
	// Reset all normals
	for i := range m.Vertices {
		m.Vertices[i].Normal = math3d.Zero3()
	}

	// Accumulate face normals per vertex
	for _, f := range m.Faces {
		v0 := m.Vertices[f.V[0]].Position
		v1 := m.Vertices[f.V[1]].Position
		v2 := m.Vertices[f.V[2]].Position

		edge1 := v1.Sub(v0)
		edge2 := v2.Sub(v0)
		normal := edge1.Cross(edge2) // Don't normalize yet

		m.Vertices[f.V[0]].Normal = m.Vertices[f.V[0]].Normal.Add(normal)
		m.Vertices[f.V[1]].Normal = m.Vertices[f.V[1]].Normal.Add(normal)
		m.Vertices[f.V[2]].Normal = m.Vertices[f.V[2]].Normal.Add(normal)
	}

	// Normalize all accumulated normals
	for i := range m.Vertices {
		m.Vertices[i].Normal = m.Vertices[i].Normal.Normalize()
	}
}

// Transform applies a transformation matrix to all vertices.
func (m *Mesh) Transform(mat math3d.Mat4) {
	for i := range m.Vertices {
		m.Vertices[i].Position = mat.MulVec3(m.Vertices[i].Position)
		// Transform normals with inverse transpose (for non-uniform scaling)
		// For now, just use the rotation part
		m.Vertices[i].Normal = mat.MulVec3Dir(m.Vertices[i].Normal).Normalize()
	}
	m.CalculateBounds()
}

// Clone creates a deep copy of the mesh.
func (m *Mesh) Clone() *Mesh {
	clone := &Mesh{
		Name:      m.Name,
		Vertices:  make([]MeshVertex, len(m.Vertices)),
		Faces:     make([]Face, len(m.Faces)),
		Materials: make([]Material, len(m.Materials)),
		BoundsMin: m.BoundsMin,
		BoundsMax: m.BoundsMax,
	}
	copy(clone.Vertices, m.Vertices)
	copy(clone.Faces, m.Faces)
	copy(clone.Materials, m.Materials)
	return clone
}

// GetVertex returns the position, normal, and UV for vertex i.
// Implements render.MeshRenderer interface.
func (m *Mesh) GetVertex(i int) (pos, normal math3d.Vec3, uv math3d.Vec2) {
	v := m.Vertices[i]
	return v.Position, v.Normal, v.UV
}

// GetFace returns the vertex indices for face i.
// Implements render.MeshRenderer interface.
func (m *Mesh) GetFace(i int) [3]int {
	return m.Faces[i].V
}

// GetFaceMaterial returns the material index for face i.
// Returns -1 if no material assigned.
func (m *Mesh) GetFaceMaterial(i int) int {
	return m.Faces[i].Material
}

// GetMaterial returns the material at index i.
// Returns nil if index is out of bounds or -1.
func (m *Mesh) GetMaterial(i int) *Material {
	if i < 0 || i >= len(m.Materials) {
		return nil
	}
	return &m.Materials[i]
}

// MaterialCount returns the number of materials.
func (m *Mesh) MaterialCount() int {
	return len(m.Materials)
}

// GetBounds returns the axis-aligned bounding box.
// Implements render.BoundedMeshRenderer interface.
func (m *Mesh) GetBounds() (min, max math3d.Vec3) {
	return m.BoundsMin, m.BoundsMax
}
