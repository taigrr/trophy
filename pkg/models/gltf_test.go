package models

import (
	"testing"
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
