package models

import (
	"testing"
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

	// Add some materials
	mesh.Materials = []Material{
		{Name: "red", BaseColor: [4]float64{1, 0, 0, 1}},
		{Name: "green", BaseColor: [4]float64{0, 1, 0, 1}},
		{Name: "blue", BaseColor: [4]float64{0, 0, 1, 1}},
	}

	// Add faces with different materials
	mesh.Faces = []Face{
		{V: [3]int{0, 1, 2}, Material: 0}, // red
		{V: [3]int{3, 4, 5}, Material: 1}, // green
		{V: [3]int{6, 7, 8}, Material: 2}, // blue
		{V: [3]int{9, 10, 11}, Material: -1}, // no material
	}

	// Verify material indices
	if mesh.GetFaceMaterial(0) != 0 {
		t.Errorf("Face 0 should have material 0, got %d", mesh.GetFaceMaterial(0))
	}
	if mesh.GetFaceMaterial(1) != 1 {
		t.Errorf("Face 1 should have material 1, got %d", mesh.GetFaceMaterial(1))
	}
	if mesh.GetFaceMaterial(3) != -1 {
		t.Errorf("Face 3 should have material -1, got %d", mesh.GetFaceMaterial(3))
	}

	// Verify GetMaterial
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

	// Verify materials are copied, not shared
	clone.Materials[0].Name = "modified"
	if mesh.Materials[0].Name == "modified" {
		t.Errorf("Clone should have independent material copy")
	}

	// Verify face material indices
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
