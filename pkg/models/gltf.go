package models

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/qmuntal/gltf"
	"github.com/taigrr/trophy/pkg/math3d"
)

// GLTFLoader loads GLTF/GLB files into Mesh format.
type GLTFLoader struct {
	// Options
	CalculateNormals bool
	SmoothNormals    bool
}

// NewGLTFLoader creates a new GLTF loader with default options.
func NewGLTFLoader() *GLTFLoader {
	return &GLTFLoader{
		CalculateNormals: true,
		SmoothNormals:    true,
	}
}

// LoadGLB loads a binary GLTF (.glb) file.
func LoadGLB(path string) (*Mesh, error) {
	loader := NewGLTFLoader()
	return loader.Load(path)
}

// Load loads a GLTF or GLB file and returns a Mesh.
func (l *GLTFLoader) Load(path string) (*Mesh, error) {
	doc, err := gltf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open gltf: %w", err)
	}

	mesh := NewMesh(filepath.Base(path))

	// Process all meshes in the document
	for _, m := range doc.Meshes {
		if err := l.processMesh(doc, m, mesh); err != nil {
			return nil, fmt.Errorf("process mesh %q: %w", m.Name, err)
		}
	}

	// Calculate normals if needed
	hasNormals := false
	for _, v := range mesh.Vertices {
		if v.Normal.Len() > 0.001 {
			hasNormals = true
			break
		}
	}

	if l.CalculateNormals && !hasNormals {
		if l.SmoothNormals {
			mesh.CalculateSmoothNormals()
		} else {
			mesh.CalculateNormals()
		}
	}

	mesh.CalculateBounds()

	return mesh, nil
}

// processMesh extracts geometry from a GLTF mesh.
func (l *GLTFLoader) processMesh(doc *gltf.Document, m *gltf.Mesh, mesh *Mesh) error {
	for _, prim := range m.Primitives {
		if prim.Mode != gltf.PrimitiveTriangles && prim.Mode != 0 {
			// Skip non-triangle primitives (lines, points, etc)
			continue
		}

		// Get position accessor
		posIdx, ok := prim.Attributes[gltf.POSITION]
		if !ok {
			continue
		}

		positions, err := readVec3Accessor(doc, posIdx)
		if err != nil {
			return fmt.Errorf("read positions: %w", err)
		}

		// Get normals if available
		var normals []math3d.Vec3
		if normIdx, ok := prim.Attributes[gltf.NORMAL]; ok {
			normals, err = readVec3Accessor(doc, normIdx)
			if err != nil {
				return fmt.Errorf("read normals: %w", err)
			}
		}

		// Get UVs if available
		var uvs []math3d.Vec2
		if uvIdx, ok := prim.Attributes[gltf.TEXCOORD_0]; ok {
			uvs, err = readVec2Accessor(doc, uvIdx)
			if err != nil {
				return fmt.Errorf("read uvs: %w", err)
			}
		}

		// Base vertex index for this primitive
		baseVertex := len(mesh.Vertices)

		// Add vertices
		for i := range positions {
			v := MeshVertex{
				Position: positions[i],
			}
			if i < len(normals) {
				v.Normal = normals[i]
			}
			if i < len(uvs) {
				// GLTF uses top-left origin (V=0 at top), flip V for bottom-left origin
				v.UV = math3d.V2(uvs[i].X, 1.0-uvs[i].Y)
			}
			mesh.Vertices = append(mesh.Vertices, v)
		}

		// Process indices
		if prim.Indices != nil {
			indices, err := readIndices(doc, *prim.Indices)
			if err != nil {
				return fmt.Errorf("read indices: %w", err)
			}

			// Create faces from indices
			// Note: GLTF uses CCW winding for front-facing, but our engine uses CW
			// (due to Y-flip in screen space), so we reverse the winding here
			for i := 0; i+2 < len(indices); i += 3 {
				mesh.Faces = append(mesh.Faces, Face{
					V: [3]int{
						baseVertex + indices[i],
						baseVertex + indices[i+2], // swapped
						baseVertex + indices[i+1], // swapped
					},
				})
			}
		} else {
			// No indices, assume sequential triangles
			// Also reverse winding: CCW -> CW
			for i := 0; i+2 < len(positions); i += 3 {
				mesh.Faces = append(mesh.Faces, Face{
					V: [3]int{
						baseVertex + i,
						baseVertex + i + 2, // swapped
						baseVertex + i + 1, // swapped
					},
				})
			}
		}
	}

	return nil
}

// readVec3Accessor reads Vec3 data from a GLTF accessor.
func readVec3Accessor(doc *gltf.Document, accessorIdx int) ([]math3d.Vec3, error) {
	accessor := doc.Accessors[accessorIdx]
	if accessor.Type != gltf.AccessorVec3 {
		return nil, fmt.Errorf("expected VEC3, got %v", accessor.Type)
	}

	data, err := readAccessorData(doc, accessor)
	if err != nil {
		return nil, err
	}

	floats, ok := data.([][3]float32)
	if !ok {
		return nil, fmt.Errorf("unexpected data type for VEC3")
	}

	result := make([]math3d.Vec3, len(floats))
	for i, f := range floats {
		result[i] = math3d.V3(float64(f[0]), float64(f[1]), float64(f[2]))
	}

	return result, nil
}

// readVec2Accessor reads Vec2 data from a GLTF accessor.
func readVec2Accessor(doc *gltf.Document, accessorIdx int) ([]math3d.Vec2, error) {
	accessor := doc.Accessors[accessorIdx]
	if accessor.Type != gltf.AccessorVec2 {
		return nil, fmt.Errorf("expected VEC2, got %v", accessor.Type)
	}

	data, err := readAccessorData(doc, accessor)
	if err != nil {
		return nil, err
	}

	floats, ok := data.([][2]float32)
	if !ok {
		return nil, fmt.Errorf("unexpected data type for VEC2")
	}

	result := make([]math3d.Vec2, len(floats))
	for i, f := range floats {
		result[i] = math3d.V2(float64(f[0]), float64(f[1]))
	}

	return result, nil
}

// readIndices reads index data from a GLTF accessor.
func readIndices(doc *gltf.Document, accessorIdx int) ([]int, error) {
	accessor := doc.Accessors[accessorIdx]

	data, err := readAccessorData(doc, accessor)
	if err != nil {
		return nil, err
	}

	switch v := data.(type) {
	case []uint8:
		result := make([]int, len(v))
		for i, x := range v {
			result[i] = int(x)
		}
		return result, nil
	case []uint16:
		result := make([]int, len(v))
		for i, x := range v {
			result[i] = int(x)
		}
		return result, nil
	case []uint32:
		result := make([]int, len(v))
		for i, x := range v {
			result[i] = int(x)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unexpected index type: %T", data)
	}
}

// readAccessorData reads raw data from a GLTF accessor.
func readAccessorData(doc *gltf.Document, accessor *gltf.Accessor) (any, error) {
	if accessor.BufferView == nil {
		return nil, fmt.Errorf("accessor has no buffer view")
	}

	bufferView := doc.BufferViews[*accessor.BufferView]
	buffer := doc.Buffers[bufferView.Buffer]

	// Get buffer data
	var bufData []byte
	if buffer.URI == "" {
		// Embedded data (GLB)
		bufData = buffer.Data
	} else {
		// External file - need to load relative to document
		return nil, fmt.Errorf("external buffers not supported yet")
	}

	if bufData == nil {
		return nil, fmt.Errorf("buffer has no data")
	}

	// Calculate data bounds
	start := bufferView.ByteOffset + accessor.ByteOffset
	stride := bufferView.ByteStride
	count := accessor.Count

	// Read based on component type and accessor type
	switch accessor.Type {
	case gltf.AccessorVec3:
		if stride == 0 {
			stride = 12 // 3 floats * 4 bytes
		}
		result := make([][3]float32, count)
		for i := range count {
			offset := start + i*stride
			for j := range 3 {
				result[i][j] = readFloat32(bufData[offset+j*4:])
			}
		}
		return result, nil

	case gltf.AccessorVec2:
		if stride == 0 {
			stride = 8 // 2 floats * 4 bytes
		}
		result := make([][2]float32, count)
		for i := range count {
			offset := start + i*stride
			for j := range 2 {
				result[i][j] = readFloat32(bufData[offset+j*4:])
			}
		}
		return result, nil

	case gltf.AccessorScalar:
		if stride == 0 {
			switch accessor.ComponentType {
			case gltf.ComponentUbyte:
				stride = 1
			case gltf.ComponentUshort:
				stride = 2
			case gltf.ComponentUint:
				stride = 4
			}
		}

		switch accessor.ComponentType {
		case gltf.ComponentUbyte:
			result := make([]uint8, count)
			for i := range count {
				result[i] = bufData[start+i*stride]
			}
			return result, nil
		case gltf.ComponentUshort:
			result := make([]uint16, count)
			for i := range count {
				offset := start + i*stride
				result[i] = uint16(bufData[offset]) | uint16(bufData[offset+1])<<8
			}
			return result, nil
		case gltf.ComponentUint:
			result := make([]uint32, count)
			for i := range count {
				offset := start + i*stride
				result[i] = uint32(bufData[offset]) |
					uint32(bufData[offset+1])<<8 |
					uint32(bufData[offset+2])<<16 |
					uint32(bufData[offset+3])<<24
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("unsupported accessor type: %v / %v", accessor.Type, accessor.ComponentType)
}

// readFloat32 reads a little-endian float32.
func readFloat32(b []byte) float32 {
	bits := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return float32frombits(bits)
}

// float32frombits converts bits to float32.
func float32frombits(b uint32) float32 {
	return *(*float32)(unsafe.Pointer(&b))
}

// LoadGLTFWithTextures loads a GLTF file and extracts embedded textures.
// Returns the mesh and a map of image index to texture data.
func LoadGLTFWithTextures(path string) (*Mesh, map[int][]byte, error) {
	doc, err := gltf.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open gltf: %w", err)
	}

	loader := NewGLTFLoader()
	mesh, err := loader.Load(path)
	if err != nil {
		return nil, nil, err
	}

	// Extract textures
	textures := make(map[int][]byte)
	for i, img := range doc.Images {
		if img.BufferView != nil {
			bv := doc.BufferViews[*img.BufferView]
			buf := doc.Buffers[bv.Buffer]
			if buf.Data != nil {
				start := bv.ByteOffset
				end := start + bv.ByteLength
				textures[i] = buf.Data[start:end]
			}
		} else if img.URI != "" {
			// External texture file
			dir := filepath.Dir(path)
			texPath := filepath.Join(dir, img.URI)
			data, err := os.ReadFile(texPath)
			if err == nil {
				textures[i] = data
			}
		}
	}

	return mesh, textures, nil
}

// LoadGLBWithTexture loads a GLB file and returns the mesh plus the first embedded texture.
// Returns (mesh, texture image, error). Texture may be nil if none embedded.
func LoadGLBWithTexture(path string) (*Mesh, image.Image, error) {
	mesh, textures, err := LoadGLTFWithTextures(path)
	if err != nil {
		return nil, nil, err
	}

	// Find the first texture
	var textureImg image.Image
	for _, data := range textures {
		if len(data) > 0 {
			img, _, err := image.Decode(bytes.NewReader(data))
			if err == nil {
				textureImg = img
				break
			}
		}
	}

	return mesh, textureImg, nil
}
