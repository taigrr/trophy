package models

import (
	"strings"
	"testing"

	"github.com/qmuntal/gltf"
	"github.com/taigrr/trophy/pkg/math3d"
)

func TestLoadGLBInvalidPath(t *testing.T) {
	_, err := LoadGLB("/nonexistent/path.glb")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestGLTFLoaderCreation(t *testing.T) {
	loader := NewGLTFLoader()
	if loader == nil {
		t.Error("NewGLTFLoader returned nil")
		return
	}
	if !loader.CalculateNormals {
		t.Error("CalculateNormals should default to true")
	}
	if !loader.SmoothNormals {
		t.Error("SmoothNormals should default to true")
	}
}

func TestGLTFProcessNodePropagatesMeshErrors(t *testing.T) {
	loader := NewGLTFLoader()
	doc := &gltf.Document{
		Accessors: []*gltf.Accessor{
			{
				BufferView:    gltf.Index(0),
				ComponentType: gltf.ComponentFloat,
				Count:         1,
				Type:          gltf.AccessorScalar,
			},
		},
		Meshes: []*gltf.Mesh{
			{
				Primitives: []*gltf.Primitive{
					{
						Attributes: gltf.PrimitiveAttributes{
							gltf.POSITION: 0,
						},
					},
				},
			},
		},
		Nodes: []*gltf.Node{
			{
				Mesh: gltf.Index(0),
			},
		},
	}

	err := loader.processNode(doc, 0, math3d.Identity(), NewMesh("bad"), map[int]bool{})
	if err == nil {
		t.Fatal("expected invalid POSITION accessor error")
	}
	if !strings.Contains(err.Error(), "expected VEC3") {
		t.Fatalf("expected VEC3 error, got %v", err)
	}
}
