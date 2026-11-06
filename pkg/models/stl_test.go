package models

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

func TestSTLLoaderASCII(t *testing.T) {
	// Simple ASCII STL cube (partial - one face)
	asciiSTL := `solid cube
  facet normal 0 0 -1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 1 1 0
    endloop
  endfacet
  facet normal 0 0 -1
    outer loop
      vertex 0 0 0
      vertex 1 1 0
      vertex 0 1 0
    endloop
  endfacet
endsolid cube`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load ASCII STL: %v", err)
	}

	if mesh.Name != "cube" {
		t.Errorf("Name = %q, want %q", mesh.Name, "cube")
	}

	if mesh.TriangleCount() != 2 {
		t.Errorf("TriangleCount = %d, want 2", mesh.TriangleCount())
	}

	// Should have 4 unique vertices (square)
	if mesh.VertexCount() != 4 {
		t.Errorf("VertexCount = %d, want 4 (deduplicated)", mesh.VertexCount())
	}
}

func TestSTLLoaderBinary(t *testing.T) {
	// Create a simple binary STL with one triangle
	var buf bytes.Buffer

	// 80-byte header
	header := make([]byte, 80)
	copy(header, []byte("Binary STL test"))
	buf.Write(header)

	// Triangle count (1)
	binary.Write(&buf, binary.LittleEndian, uint32(1))

	// Triangle: normal + 3 vertices + attribute
	// Normal: 0, 0, 1
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(1))

	// Vertex 1: 0, 0, 0
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(0))

	// Vertex 2: 1, 0, 0
	binary.Write(&buf, binary.LittleEndian, float32(1))
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(0))

	// Vertex 3: 0, 1, 0
	binary.Write(&buf, binary.LittleEndian, float32(0))
	binary.Write(&buf, binary.LittleEndian, float32(1))
	binary.Write(&buf, binary.LittleEndian, float32(0))

	// Attribute byte count
	binary.Write(&buf, binary.LittleEndian, uint16(0))

	loader := NewSTLLoader()
	mesh, err := loader.LoadBytes(buf.Bytes(), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load binary STL: %v", err)
	}

	if mesh.TriangleCount() != 1 {
		t.Errorf("TriangleCount = %d, want 1", mesh.TriangleCount())
	}

	if mesh.VertexCount() != 3 {
		t.Errorf("VertexCount = %d, want 3", mesh.VertexCount())
	}

	// Check normal
	v := mesh.Vertices[0]
	if v.Normal.Z != 1.0 {
		t.Errorf("Normal.Z = %f, want 1.0", v.Normal.Z)
	}
}

func TestSTLDetection(t *testing.T) {
	// ASCII should not be detected as binary
	ascii := []byte("solid test\nfacet normal 0 0 1\n")
	if isBinarySTL(ascii) {
		t.Error("ASCII STL detected as binary")
	}

	// Binary with matching size should be detected
	var buf bytes.Buffer
	buf.Write(make([]byte, 80))                        // header
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // 0 triangles
	if !isBinarySTL(buf.Bytes()) {
		t.Error("Binary STL not detected")
	}
}

func TestSTLVertexDeduplication(t *testing.T) {
	// Two triangles sharing an edge
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0 1 0
    endloop
  endfacet
  facet normal 0 0 1
    outer loop
      vertex 1 0 0
      vertex 1 1 0
      vertex 0 1 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// 2 triangles = 6 vertices, but 2 are shared, so 4 unique
	if mesh.VertexCount() != 4 {
		t.Errorf("VertexCount = %d, want 4 (deduplicated)", mesh.VertexCount())
	}

	if mesh.TriangleCount() != 2 {
		t.Errorf("TriangleCount = %d, want 2", mesh.TriangleCount())
	}
}

func TestSTLSmoothNormals(t *testing.T) {
	// Two triangles at 90 degrees sharing an edge
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0 0 1
    endloop
  endfacet
  facet normal 0 -1 0
    outer loop
      vertex 0 0 0
      vertex 0 0 1
      vertex 0 -1 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	loader.SmoothNormals = true
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Find the shared vertex at origin
	for _, v := range mesh.Vertices {
		if v.Position.X == 0 && v.Position.Y == 0 && v.Position.Z == 0 {
			// Shared vertex should have averaged normal
			// Between (0,0,1) and (0,-1,0), normalized
			expectedLen := math.Sqrt(v.Normal.X*v.Normal.X + v.Normal.Y*v.Normal.Y + v.Normal.Z*v.Normal.Z)
			if math.Abs(expectedLen-1.0) > 0.001 {
				t.Errorf("Smooth normal not normalized: length = %f", expectedLen)
			}
		}
	}
}

func TestSTLBounds(t *testing.T) {
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex -1 -2 -3
      vertex 4 5 6
      vertex 0 0 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	min, max := mesh.GetBounds()

	if min.X != -1 || min.Y != -2 || min.Z != -3 {
		t.Errorf("BoundsMin = %v, want (-1, -2, -3)", min)
	}

	if max.X != 4 || max.Y != 5 || max.Z != 6 {
		t.Errorf("BoundsMax = %v, want (4, 5, 6)", max)
	}
}

// TestSTLFaceIndicesValid ensures face indices point to valid vertices
func TestSTLFaceIndicesValid(t *testing.T) {
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0 1 0
    endloop
  endfacet
  facet normal 0 0 1
    outer loop
      vertex 1 0 0
      vertex 1 1 0
      vertex 0 1 0
    endloop
  endfacet
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 0 1 0
      vertex -1 0 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	vertexCount := mesh.VertexCount()

	for i, face := range mesh.Faces {
		for j, idx := range face.V {
			if idx < 0 || idx >= vertexCount {
				t.Errorf("Face %d vertex %d: index %d out of range [0, %d)", i, j, idx, vertexCount)
			}
		}
	}
}

// TestSTLNormalsNotZero ensures all vertices have non-zero normals
func TestSTLNormalsNotZero(t *testing.T) {
	asciiSTL := `solid test
  facet normal 0.577 0.577 0.577
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0 1 0
    endloop
  endfacet
  facet normal 0 0 1
    outer loop
      vertex 1 0 0
      vertex 1 1 0
      vertex 0 1 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	for i, v := range mesh.Vertices {
		lenSq := v.Normal.X*v.Normal.X + v.Normal.Y*v.Normal.Y + v.Normal.Z*v.Normal.Z
		if lenSq < 0.001 {
			t.Errorf("Vertex %d has zero/near-zero normal: %v", i, v.Normal)
		}
	}
}

// TestSTLWindingOrder tests that face winding matches GLTF/OBJ convention
func TestSTLWindingOrder(t *testing.T) {
	// Single triangle with known normal
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0 1 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if len(mesh.Faces) != 1 {
		t.Fatalf("Expected 1 face, got %d", len(mesh.Faces))
	}

	face := mesh.Faces[0]
	v0 := mesh.Vertices[face.V[0]].Position
	v1 := mesh.Vertices[face.V[1]].Position
	v2 := mesh.Vertices[face.V[2]].Position

	// Calculate face normal from cross product (winding-derived)
	edge1 := v1.Sub(v0)
	edge2 := v2.Sub(v0)
	windingNormal := edge1.Cross(edge2).Normalize()

	// STL uses CCW winding, we reverse to match GLTF/OBJ (CW convention)
	// So winding-derived normal should be OPPOSITE of STL normal
	if math.Abs(windingNormal.Z+1.0) > 0.001 {
		t.Errorf("Winding normal = %v, want (0, 0, -1) for reversed winding", windingNormal)
	}

	// But stored vertex normals should still come from STL file
	if math.Abs(mesh.Vertices[face.V[0]].Normal.Z-1.0) > 0.001 {
		t.Errorf("Stored normal = %v, want (0, 0, 1) from STL", mesh.Vertices[face.V[0]].Normal)
	}
}

// TestSTLSharedVertexNormals tests that shared vertices get accumulated normals
func TestSTLSharedVertexNormals(t *testing.T) {
	// Two triangles with SAME normal direction sharing vertex at (1,0,0)
	// (using same-direction normals to avoid cancellation)
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0.5 1 0
    endloop
  endfacet
  facet normal 0 1 0
    outer loop
      vertex 1 0 0
      vertex 2 0 0
      vertex 1.5 1 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Find vertex at (1, 0, 0) - shared between both triangles
	var sharedVertex *MeshVertex
	for i := range mesh.Vertices {
		v := &mesh.Vertices[i]
		if v.Position.X == 1 && v.Position.Y == 0 && v.Position.Z == 0 {
			sharedVertex = v
			break
		}
	}

	if sharedVertex == nil {
		t.Fatal("Could not find shared vertex at (1, 0, 0)")
	}

	// With normal accumulation, the shared vertex should have an averaged normal
	// (0,0,1) + (0,1,0) normalized = (0, 0.707, 0.707) approximately
	t.Logf("Shared vertex normal: %v", sharedVertex.Normal)

	// Check that the normal is not zero
	lenSq := vec3LenSq(sharedVertex.Normal)
	if lenSq < 0.001 {
		t.Error("Shared vertex has zero normal")
	}

	// Check that the normal is normalized (length ~1)
	length := math.Sqrt(lenSq)
	if math.Abs(length-1.0) > 0.01 {
		t.Errorf("Shared vertex normal not normalized: length = %f", length)
	}

	// Check that both Y and Z components are non-zero (averaged from both faces)
	if math.Abs(sharedVertex.Normal.Y) < 0.1 || math.Abs(sharedVertex.Normal.Z) < 0.1 {
		t.Errorf("Shared vertex normal doesn't appear to be averaged: %v", sharedVertex.Normal)
	}
}

// TestSTLTeapotLoad tests loading the actual teapot file
func TestSTLTeapotLoad(t *testing.T) {
	loader := NewSTLLoader()
	mesh, err := loader.LoadFile("../../docs/teapot.stl")
	if err != nil {
		t.Skipf("Skipping teapot test (file not found): %v", err)
	}

	// Verify basic properties
	if mesh.TriangleCount() == 0 {
		t.Error("Teapot has no triangles")
	}

	if mesh.VertexCount() == 0 {
		t.Error("Teapot has no vertices")
	}

	// Check all face indices are valid
	vertexCount := mesh.VertexCount()
	for i, face := range mesh.Faces {
		for j, idx := range face.V {
			if idx < 0 || idx >= vertexCount {
				t.Errorf("Face %d vertex %d: index %d out of range [0, %d)", i, j, idx, vertexCount)
			}
		}
	}

	// Check no degenerate triangles (all three vertices same)
	for i, face := range mesh.Faces {
		v0 := mesh.Vertices[face.V[0]].Position
		v1 := mesh.Vertices[face.V[1]].Position
		v2 := mesh.Vertices[face.V[2]].Position

		if v0 == v1 && v1 == v2 {
			t.Errorf("Face %d is degenerate (all vertices same)", i)
		}
	}

	// Check all normals are non-zero
	zeroNormals := 0
	for _, v := range mesh.Vertices {
		if vec3LenSq(v.Normal) < 0.0001 {
			zeroNormals++
		}
	}
	if zeroNormals > 0 {
		t.Errorf("%d vertices have zero/near-zero normals", zeroNormals)
	}

	t.Logf("Teapot: %d triangles, %d vertices", mesh.TriangleCount(), mesh.VertexCount())
}

// TestSTLNoDedupe tests loading without deduplication for comparison
func TestSTLNoDedupe(t *testing.T) {
	asciiSTL := `solid test
  facet normal 0 0 1
    outer loop
      vertex 0 0 0
      vertex 1 0 0
      vertex 0 1 0
    endloop
  endfacet
  facet normal 0 0 1
    outer loop
      vertex 1 0 0
      vertex 1 1 0
      vertex 0 1 0
    endloop
  endfacet
endsolid test`

	loader := NewSTLLoader()
	loader.NoDedupe = true
	mesh, err := loader.Load(bytes.NewReader([]byte(asciiSTL)), "test.stl")
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Without deduplication: 2 triangles * 3 vertices = 6 vertices
	if mesh.VertexCount() != 6 {
		t.Errorf("VertexCount = %d, want 6 (no deduplication)", mesh.VertexCount())
	}

	// Each face should reference sequential vertices with reversed winding [0, 2, 1]
	for i, face := range mesh.Faces {
		expectedStart := i * 3
		if face.V[0] != expectedStart || face.V[1] != expectedStart+2 || face.V[2] != expectedStart+1 {
			t.Errorf("Face %d: V = %v, want [%d, %d, %d] (reversed winding)",
				i, face.V, expectedStart, expectedStart+2, expectedStart+1)
		}
	}
}

// Helper to check if Vec3 has non-trivial length
func vec3LenSq(v math3d.Vec3) float64 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

func TestSTLLoaderRejectsOverflowingTriangleCounts(t *testing.T) {
	for _, count := range []uint32{1 << 31, math.MaxUint32} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			data := make([]byte, 84)
			binary.LittleEndian.PutUint32(data[80:], count)
			mesh, err := NewSTLLoader().LoadBytes(data, "overflow")
			if err == nil || mesh != nil {
				t.Fatalf("expected truncated STL error and no mesh, got mesh=%v, err=%v", mesh, err)
			}
		})
	}
}

func TestIsBinarySTLDoesNotWrapExpectedSize(t *testing.T) {
	data := make([]byte, 84)
	copy(data, "solid binary")
	binary.LittleEndian.PutUint32(data[80:], 1<<31)
	if isBinarySTL(data) {
		t.Fatal("overflowing triangle count must not match the file size")
	}
}
