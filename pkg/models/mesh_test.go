package models

import (
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

func TestFaceKey(t *testing.T) {
	tests := []struct {
		name       string
		v0, v1, v2 int
		want       [3]int
	}{
		{"already sorted", 0, 1, 2, [3]int{0, 1, 2}},
		{"reverse order", 2, 1, 0, [3]int{0, 1, 2}},
		{"middle first", 1, 0, 2, [3]int{0, 1, 2}},
		{"rotated", 1, 2, 0, [3]int{0, 1, 2}},
		{"with gaps", 5, 10, 3, [3]int{3, 5, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := faceKey(tt.v0, tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("faceKey(%d, %d, %d) = %v, want %v", tt.v0, tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestDeduplicateFaces(t *testing.T) {
	// Create a mesh with duplicate faces
	mesh := NewMesh("test")
	mesh.Vertices = []MeshVertex{
		{Position: math3d.V3(0, 0, 0)},
		{Position: math3d.V3(1, 0, 0)},
		{Position: math3d.V3(0, 1, 0)},
		{Position: math3d.V3(1, 1, 0)},
	}
	mesh.Faces = []Face{
		{V: [3]int{0, 1, 2}}, // unique
		{V: [3]int{0, 1, 2}}, // exact duplicate
		{V: [3]int{2, 0, 1}}, // same vertices, different order
		{V: [3]int{1, 2, 3}}, // unique
	}

	removed := mesh.DeduplicateFaces()

	if removed != 2 {
		t.Errorf("DeduplicateFaces() removed %d faces, want 2", removed)
	}

	if mesh.TriangleCount() != 2 {
		t.Errorf("After dedup: TriangleCount = %d, want 2", mesh.TriangleCount())
	}
}

func TestRemoveInternalFaces(t *testing.T) {
	// Create a mesh with opposing faces (internal geometry)
	mesh := NewMesh("test")
	mesh.Vertices = []MeshVertex{
		{Position: math3d.V3(0, 0, 0)},
		{Position: math3d.V3(1, 0, 0)},
		{Position: math3d.V3(0, 1, 0)},
		{Position: math3d.V3(1, 1, 0)},
	}
	// Two faces with same vertices but opposite winding (opposite normals)
	mesh.Faces = []Face{
		{V: [3]int{0, 1, 2}}, // normal points +Z
		{V: [3]int{0, 2, 1}}, // normal points -Z (reversed winding)
		{V: [3]int{1, 2, 3}}, // unique face, should be kept
	}

	removed := mesh.RemoveInternalFaces()

	if removed != 2 {
		t.Errorf("RemoveInternalFaces() removed %d faces, want 2", removed)
	}

	if mesh.TriangleCount() != 1 {
		t.Errorf("After removal: TriangleCount = %d, want 1", mesh.TriangleCount())
	}

	// The remaining face should be the unique one
	if mesh.Faces[0].V != [3]int{1, 2, 3} {
		t.Errorf("Remaining face = %v, want [1,2,3]", mesh.Faces[0].V)
	}
}

func TestRemoveDegenerateFaces(t *testing.T) {
	mesh := NewMesh("test")
	mesh.Vertices = []MeshVertex{
		{Position: math3d.V3(0, 0, 0)},
		{Position: math3d.V3(1, 0, 0)},
		{Position: math3d.V3(0, 1, 0)},
		{Position: math3d.V3(0, 0, 0)}, // duplicate of vertex 0
	}
	mesh.Faces = []Face{
		{V: [3]int{0, 1, 2}}, // valid face
		{V: [3]int{0, 0, 1}}, // degenerate: duplicate vertex index
		{V: [3]int{0, 1, 0}}, // degenerate: duplicate vertex index
		{V: [3]int{0, 3, 1}}, // nearly degenerate: vertices 0 and 3 are at same position (collinear)
	}

	removed := mesh.RemoveDegenerateFaces()

	if removed != 3 {
		t.Errorf("RemoveDegenerateFaces() removed %d faces, want 3", removed)
	}

	if mesh.TriangleCount() != 1 {
		t.Errorf("After removal: TriangleCount = %d, want 1", mesh.TriangleCount())
	}
}

func TestRemoveUnreferencedVertices(t *testing.T) {
	mesh := NewMesh("test")
	mesh.Vertices = []MeshVertex{
		{Position: math3d.V3(0, 0, 0)}, // index 0 - used
		{Position: math3d.V3(1, 0, 0)}, // index 1 - used
		{Position: math3d.V3(2, 0, 0)}, // index 2 - NOT used
		{Position: math3d.V3(0, 1, 0)}, // index 3 - used
	}
	mesh.Faces = []Face{
		{V: [3]int{0, 1, 3}}, // uses vertices 0, 1, 3 (not 2)
	}

	mesh.RemoveUnreferencedVertices()

	if mesh.VertexCount() != 3 {
		t.Errorf("After removal: VertexCount = %d, want 3", mesh.VertexCount())
	}

	// Face indices should be remapped
	// Old: 0,1,3 -> New: 0,1,2
	if mesh.Faces[0].V != [3]int{0, 1, 2} {
		t.Errorf("Face after remap = %v, want [0,1,2]", mesh.Faces[0].V)
	}
}

func TestCleanMesh(t *testing.T) {
	mesh := NewMesh("test")
	mesh.Vertices = []MeshVertex{
		{Position: math3d.V3(0, 0, 0)},
		{Position: math3d.V3(1, 0, 0)},
		{Position: math3d.V3(0, 1, 0)},
		{Position: math3d.V3(1, 1, 0)},
		{Position: math3d.V3(2, 2, 0)}, // unreferenced
	}
	mesh.Faces = []Face{
		{V: [3]int{0, 1, 2}}, // valid, unique
		{V: [3]int{0, 1, 2}}, // duplicate
		{V: [3]int{1, 2, 3}}, // valid, unique
		{V: [3]int{0, 0, 1}}, // degenerate
	}

	removed := mesh.CleanMesh()

	// Should remove: 1 degenerate + 1 duplicate = 2
	if removed != 2 {
		t.Errorf("CleanMesh() removed %d faces, want 2", removed)
	}

	if mesh.TriangleCount() != 2 {
		t.Errorf("After clean: TriangleCount = %d, want 2", mesh.TriangleCount())
	}

	// Unreferenced vertex should be removed
	if mesh.VertexCount() != 4 {
		t.Errorf("After clean: VertexCount = %d, want 4", mesh.VertexCount())
	}
}

func TestCleanMeshWithInternalFaces(t *testing.T) {
	// Simulate a merged mesh scenario where two cubes overlap
	// and share an internal face that should be removed
	mesh := NewMesh("test")
	mesh.Vertices = []MeshVertex{
		{Position: math3d.V3(0, 0, 0)}, // 0
		{Position: math3d.V3(1, 0, 0)}, // 1
		{Position: math3d.V3(0, 1, 0)}, // 2
		{Position: math3d.V3(1, 1, 0)}, // 3
	}

	// Internal face pair - same vertices, opposite normals
	mesh.Faces = []Face{
		{V: [3]int{0, 1, 2}}, // face A, normal +Z
		{V: [3]int{0, 2, 1}}, // face B, normal -Z (reversed)
		{V: [3]int{1, 2, 3}}, // external face, should remain
	}

	removed := mesh.CleanMesh()

	// Should remove the internal face pair (2 faces)
	if removed != 2 {
		t.Errorf("CleanMesh() removed %d faces, want 2", removed)
	}

	if mesh.TriangleCount() != 1 {
		t.Errorf("After clean: TriangleCount = %d, want 1", mesh.TriangleCount())
	}
}
