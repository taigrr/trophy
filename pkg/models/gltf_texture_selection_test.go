package models

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestSelectPrimaryGLTFTexturePrefersFaceMaterialBaseMap(t *testing.T) {
	wrong := image.NewRGBA(image.Rect(0, 0, 1, 1))
	wrong.Set(0, 0, color.RGBA{R: 255, A: 255})

	correct := image.NewRGBA(image.Rect(0, 0, 1, 1))
	correct.Set(0, 0, color.RGBA{G: 255, A: 255})

	mesh := &Mesh{
		Materials: []Material{
			{Name: "normal", BaseMap: wrong, HasTexture: true},
			{Name: "base", BaseMap: correct, HasTexture: true},
		},
		Faces: []Face{{Material: 1}},
	}

	textures := map[int][]byte{0: mustEncodePNG(t, wrong), 1: mustEncodePNG(t, correct)}

	selected := selectPrimaryGLTFTexture(mesh, textures)
	if selected == nil {
		t.Fatal("expected selected texture")
	}

	r, g, b, _ := selected.At(0, 0).RGBA()
	if r>>8 != 0 || g>>8 != 255 || b>>8 != 0 {
		t.Fatalf("selected wrong texture: got (%d,%d,%d)", r>>8, g>>8, b>>8)
	}
}

func TestSelectPrimaryGLTFTextureFallsBackToDecodableTexture(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{B: 255, A: 255})

	selected := selectPrimaryGLTFTexture(nil, map[int][]byte{
		0: []byte("not an image"),
		1: mustEncodePNG(t, img),
	})
	if selected == nil {
		t.Fatal("expected fallback texture")
	}

	r, g, b, _ := selected.At(0, 0).RGBA()
	if r>>8 != 0 || g>>8 != 0 || b>>8 != 255 {
		t.Fatalf("fallback selected wrong texture: got (%d,%d,%d)", r>>8, g>>8, b>>8)
	}
}

func mustEncodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "texture.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create temp png: %v", err)
	}

	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatalf("encode png: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close png: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read png: %v", err)
	}

	return data
}
