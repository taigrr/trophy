package models

import (
	"math"
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

// TestMaterialDefaults verifies default material values.
func TestMaterialDefaults(t *testing.T) {
	m := Material{
		Name:      "test",
		BaseColor: [4]float64{1, 1, 1, 1},
		Metallic:  0,
		Roughness: 1,
	}

	if m.BaseColor[3] != 1 {
		t.Errorf("Expected alpha=1, got %f", m.BaseColor[3])
	}
	if m.HasTexture {
		t.Errorf("HasTexture should be false by default")
	}
}

// TestFaceMaterialIndex verifies per-face material assignment.
func TestFaceMaterialIndex(t *testing.T) {
	mesh := NewMesh("test")

	mesh.Materials = []Material{
		{Name: "red", BaseColor: [4]float64{1, 0, 0, 1}},
		{Name: "green", BaseColor: [4]float64{0, 1, 0, 1}},
		{Name: "blue", BaseColor: [4]float64{0, 0, 1, 1}},
	}

	mesh.Faces = []Face{
		{V: [3]int{0, 1, 2}, Material: 0},
		{V: [3]int{3, 4, 5}, Material: 1},
		{V: [3]int{6, 7, 8}, Material: 2},
		{V: [3]int{9, 10, 11}, Material: -1},
	}

	if mesh.GetFaceMaterial(0) != 0 {
		t.Errorf("Face 0 should have material 0, got %d", mesh.GetFaceMaterial(0))
	}
	if mesh.GetFaceMaterial(1) != 1 {
		t.Errorf("Face 1 should have material 1, got %d", mesh.GetFaceMaterial(1))
	}
	if mesh.GetFaceMaterial(3) != -1 {
		t.Errorf("Face 3 should have material -1, got %d", mesh.GetFaceMaterial(3))
	}

	mat := mesh.GetMaterial(0)
	if mat == nil || mat.Name != "red" {
		t.Errorf("GetMaterial(0) should return 'red' material")
	}

	mat = mesh.GetMaterial(-1)
	if mat != nil {
		t.Errorf("GetMaterial(-1) should return nil")
	}

	mat = mesh.GetMaterial(99)
	if mat != nil {
		t.Errorf("GetMaterial(99) should return nil for out-of-bounds")
	}
}

// TestMeshClonePreservesMaterials verifies Clone copies materials.
func TestMeshClonePreservesMaterials(t *testing.T) {
	mesh := NewMesh("original")
	mesh.Materials = []Material{
		{Name: "mat1", BaseColor: [4]float64{1, 0, 0, 1}},
		{Name: "mat2", BaseColor: [4]float64{0, 1, 0, 1}},
	}
	mesh.Faces = []Face{
		{V: [3]int{0, 1, 2}, Material: 0},
		{V: [3]int{3, 4, 5}, Material: 1},
	}

	clone := mesh.Clone()

	if clone.MaterialCount() != mesh.MaterialCount() {
		t.Errorf("Clone should have %d materials, got %d", mesh.MaterialCount(), clone.MaterialCount())
	}

	clone.Materials[0].Name = "modified"
	if mesh.Materials[0].Name == "modified" {
		t.Errorf("Clone should have independent material copy")
	}

	if clone.GetFaceMaterial(0) != 0 || clone.GetFaceMaterial(1) != 1 {
		t.Errorf("Clone should preserve face material indices")
	}
}

// TestMaterialCount verifies MaterialCount helper.
func TestMaterialCount(t *testing.T) {
	mesh := NewMesh("test")

	if mesh.MaterialCount() != 0 {
		t.Errorf("Empty mesh should have 0 materials")
	}

	mesh.Materials = make([]Material, 5)
	if mesh.MaterialCount() != 5 {
		t.Errorf("Mesh should have 5 materials, got %d", mesh.MaterialCount())
	}
}

// TestQuatToMat4Identity verifies identity quaternion produces identity rotation.
func TestQuatToMat4Identity(t *testing.T) {
	m := math3d.QuatToMat4(0, 0, 0, 1)
	identity := math3d.Identity()

	for i := 0; i < 16; i++ {
		if math.Abs(m[i]-identity[i]) > 1e-10 {
			t.Errorf("QuatToMat4 identity mismatch at index %d: got %f, want %f", i, m[i], identity[i])
		}
	}
}

// TestMat4FromSlice verifies slice to matrix conversion.
func TestMat4FromSlice(t *testing.T) {
	slice := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	m := math3d.Mat4FromSlice(slice)

	for i := 0; i < 16; i++ {
		if m[i] != slice[i] {
			t.Errorf("Mat4FromSlice mismatch at index %d: got %f, want %f", i, m[i], slice[i])
		}
	}
}

// TestQuatToMat4Rotation verifies quaternion rotation produces correct matrix.
func TestQuatToMat4Rotation(t *testing.T) {
	// 90 degree rotation around Y axis
	// Quaternion: (0, sin(45°), 0, cos(45°)) = (0, 0.707, 0, 0.707)
	angle := math.Pi / 2
	qy := math.Sin(angle / 2)
	qw := math.Cos(angle / 2)

	m := math3d.QuatToMat4(0, qy, 0, qw)

	// For 90° Y rotation, X axis should map to -Z, Z should map to X
	// Check that (1,0,0) rotates to approximately (0,0,-1)
	x := m[0]*1 + m[4]*0 + m[8]*0
	z := m[2]*1 + m[6]*0 + m[10]*0

	if math.Abs(x) > 0.001 || math.Abs(z+1) > 0.001 {
		t.Errorf("90° Y rotation should map X to -Z, got (%.3f, %.3f)", x, z)
	}
}
