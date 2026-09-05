package models

import (
	"strconv"
	"strings"
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

func TestLoadSimpleOBJ(t *testing.T) {
	// Simple triangle
	objData := `
# Simple triangle
v 0 0 0
v 1 0 0
v 0.5 1 0
f 1 2 3
`
	loader := NewOBJLoader()
	mesh, err := loader.Load(strings.NewReader(objData), "triangle")
	if err != nil {
		t.Fatalf("failed to load OBJ: %v", err)
	}

	if mesh.VertexCount() != 3 {
		t.Errorf("expected 3 vertices, got %d", mesh.VertexCount())
	}

	if mesh.TriangleCount() != 1 {
		t.Errorf("expected 1 triangle, got %d", mesh.TriangleCount())
	}
}

func TestLoadCubeOBJ(t *testing.T) {
	objData := `
# Cube
v -0.5 -0.5 -0.5
v  0.5 -0.5 -0.5
v  0.5  0.5 -0.5
v -0.5  0.5 -0.5
v -0.5 -0.5  0.5
v  0.5 -0.5  0.5
v  0.5  0.5  0.5
v -0.5  0.5  0.5

# Back face
f 1 2 3 4
# Front face
f 5 6 7 8
# Left face
f 1 4 8 5
# Right face
f 2 6 7 3
# Top face
f 4 3 7 8
# Bottom face
f 1 5 6 2
`
	loader := NewOBJLoader()
	mesh, err := loader.Load(strings.NewReader(objData), "cube")
	if err != nil {
		t.Fatalf("failed to load cube: %v", err)
	}

	// 6 faces * 2 triangles per quad = 12 triangles
	if mesh.TriangleCount() != 12 {
		t.Errorf("expected 12 triangles (6 quads), got %d", mesh.TriangleCount())
	}

	// Check bounds
	expectedMin := math3d.V3(-0.5, -0.5, -0.5)
	expectedMax := math3d.V3(0.5, 0.5, 0.5)

	if mesh.BoundsMin != expectedMin {
		t.Errorf("expected min bounds %v, got %v", expectedMin, mesh.BoundsMin)
	}
	if mesh.BoundsMax != expectedMax {
		t.Errorf("expected max bounds %v, got %v", expectedMax, mesh.BoundsMax)
	}
}

func TestLoadOBJWithUVsAndNormals(t *testing.T) {
	objData := `
v 0 0 0
v 1 0 0
v 0.5 1 0
vt 0 0
vt 1 0
vt 0.5 1
vn 0 0 1
f 1/1/1 2/2/1 3/3/1
`
	loader := NewOBJLoader()
	loader.CalculateNormals = false // Use provided normals
	mesh, err := loader.Load(strings.NewReader(objData), "tri_with_attrs")
	if err != nil {
		t.Fatalf("failed to load OBJ: %v", err)
	}

	// Check UV coordinates
	pos0, norm0, uv0 := mesh.GetVertex(0)
	if uv0.X != 0 || uv0.Y != 0 {
		t.Errorf("expected UV (0,0), got %v", uv0)
	}

	// Check normal
	expectedNormal := math3d.V3(0, 0, 1)
	if norm0 != expectedNormal {
		t.Errorf("expected normal %v, got %v", expectedNormal, norm0)
	}

	_ = pos0
}

func TestNegativeIndices(t *testing.T) {
	// OBJ allows negative indices (counting from end)
	objData := `
v 0 0 0
v 1 0 0
v 0.5 1 0
f -3 -2 -1
`
	loader := NewOBJLoader()
	mesh, err := loader.Load(strings.NewReader(objData), "neg_indices")
	if err != nil {
		t.Fatalf("failed to load OBJ: %v", err)
	}

	if mesh.TriangleCount() != 1 {
		t.Errorf("expected 1 triangle, got %d", mesh.TriangleCount())
	}
}

func TestLoadOBJWithLongFaceLine(t *testing.T) {
	const vertexCount = 15000

	var objData strings.Builder
	for i := 0; i < vertexCount; i++ {
		objData.WriteString("v ")
		objData.WriteString("0 ")
		objData.WriteString("0 ")
		objData.WriteString("0\n")
	}
	objData.WriteString("f")
	for i := 1; i <= vertexCount; i++ {
		objData.WriteString(" ")
		objData.WriteString(strconv.Itoa(i))
	}
	objData.WriteString("\n")

	loader := NewOBJLoader()
	mesh, err := loader.Load(strings.NewReader(objData.String()), "long_face")
	if err != nil {
		t.Fatalf("failed to load OBJ with long face line: %v", err)
	}

	wantTriangles := vertexCount - 2
	if mesh.TriangleCount() != wantTriangles {
		t.Errorf("expected %d triangles, got %d", wantTriangles, mesh.TriangleCount())
	}
}

func TestMeshClone(t *testing.T) {
	mesh := NewMesh("test")
	mesh.Vertices = append(mesh.Vertices, MeshVertex{
		Position: math3d.V3(1, 2, 3),
	})
	mesh.Faces = append(mesh.Faces, Face{V: [3]int{0, 0, 0}})
	mesh.CalculateBounds()

	clone := mesh.Clone()

	// Modify original
	mesh.Vertices[0].Position = math3d.V3(9, 9, 9)

	// Clone should be unaffected
	if clone.Vertices[0].Position.X != 1 {
		t.Error("clone was affected by original modification")
	}
}
