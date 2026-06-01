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

	// Extract materials first
	mesh.Materials = extractMaterials(doc, path)

	// Process scene nodes with transforms (handles node hierarchy)
	processedMeshes := make(map[int]bool)

	if len(doc.Scenes) > 0 {
		sceneIdx := 0
		if doc.Scene != nil {
			sceneIdx = int(*doc.Scene)
		}
		scene := doc.Scenes[sceneIdx]
		for _, nodeIdx := range scene.Nodes {
			l.processNode(doc, int(nodeIdx), math3d.Identity(), mesh, processedMeshes)
		}
	} else {
		// No scenes defined, process all root nodes
		for i := range doc.Nodes {
			isRoot := true
			for _, n := range doc.Nodes {
				for _, child := range n.Children {
					if int(child) == i {
						isRoot = false
						break
					}
				}
				if !isRoot {
					break
				}
			}
			if isRoot {
				l.processNode(doc, i, math3d.Identity(), mesh, processedMeshes)
			}
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

// processNode recursively processes a node and its children, accumulating transforms.
func (l *GLTFLoader) processNode(doc *gltf.Document, nodeIdx int, parentTransform math3d.Mat4, mesh *Mesh, processedMeshes map[int]bool) {
	node := doc.Nodes[nodeIdx]

	// Build this node's local transform
	localTransform := math3d.Identity()

	if node.Translation != [3]float64{0, 0, 0} {
		localTransform = localTransform.Mul(math3d.Translate(math3d.V3(
			node.Translation[0],
			node.Translation[1],
			node.Translation[2],
		)))
	}

	if node.Rotation != [4]float64{0, 0, 0, 1} {
		localTransform = localTransform.Mul(math3d.QuatToMat4(
			node.Rotation[0],
			node.Rotation[1],
			node.Rotation[2],
			node.Rotation[3],
		))
	}

	if node.Scale != [3]float64{1, 1, 1} && node.Scale != [3]float64{0, 0, 0} {
		localTransform = localTransform.Mul(math3d.Scale(math3d.V3(
			node.Scale[0],
			node.Scale[1],
			node.Scale[2],
		)))
	}

	if node.Matrix != [16]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} {
		localTransform = math3d.Mat4FromSlice(node.Matrix[:])
	}

	worldTransform := parentTransform.Mul(localTransform)

	if node.Mesh != nil {
		meshIdx := int(*node.Mesh)
		gltfMesh := doc.Meshes[meshIdx]
		l.processMeshWithTransform(doc, gltfMesh, mesh, worldTransform)
		processedMeshes[meshIdx] = true
	}

	for _, childIdx := range node.Children {
		l.processNode(doc, int(childIdx), worldTransform, mesh, processedMeshes)
	}
}

// processMeshWithTransform extracts geometry from a GLTF mesh, applying the given transform.
func (l *GLTFLoader) processMeshWithTransform(doc *gltf.Document, m *gltf.Mesh, mesh *Mesh, transform math3d.Mat4) error {
	for _, prim := range m.Primitives {
		if prim.Mode != gltf.PrimitiveTriangles && prim.Mode != 0 {
			continue
		}

		posIdx, ok := prim.Attributes[gltf.POSITION]
		if !ok {
			continue
		}

		positions, err := readVec3Accessor(doc, posIdx)
		if err != nil {
			return fmt.Errorf("read positions: %w", err)
		}

		var normals []math3d.Vec3
		if normIdx, ok := prim.Attributes[gltf.NORMAL]; ok {
			normals, err = readVec3Accessor(doc, normIdx)
			if err != nil {
				return fmt.Errorf("read normals: %w", err)
			}
		}

		var uvs []math3d.Vec2
		if uvIdx, ok := prim.Attributes[gltf.TEXCOORD_0]; ok {
			uvs, err = readVec2Accessor(doc, uvIdx)
			if err != nil {
				return fmt.Errorf("read uvs: %w", err)
			}
		}

		materialIdx := -1
		if prim.Material != nil {
			materialIdx = int(*prim.Material)
		}

		baseVertex := len(mesh.Vertices)

		for i := range positions {
			worldPos := transform.MulVec3(positions[i])

			v := MeshVertex{
				Position: worldPos,
			}

			if i < len(normals) {
				v.Normal = transform.MulVec3Dir(normals[i]).Normalize()
			}
			if i < len(uvs) {
				v.UV = math3d.V2(uvs[i].X, 1.0-uvs[i].Y)
			}
			mesh.Vertices = append(mesh.Vertices, v)
		}

		if prim.Indices != nil {
			indices, err := readIndices(doc, *prim.Indices)
			if err != nil {
				return fmt.Errorf("read indices: %w", err)
			}

			for i := 0; i+2 < len(indices); i += 3 {
				mesh.Faces = append(mesh.Faces, Face{
					V: [3]int{
						baseVertex + indices[i],
						baseVertex + indices[i+2],
						baseVertex + indices[i+1],
					},
					Material: materialIdx,
				})
			}
		} else {
			for i := 0; i+2 < len(positions); i += 3 {
				mesh.Faces = append(mesh.Faces, Face{
					V: [3]int{
						baseVertex + i,
						baseVertex + i + 2,
						baseVertex + i + 1,
					},
					Material: materialIdx,
				})
			}
		}
	}

	return nil
}

// extractMaterials extracts all materials from a GLTF document.
func extractMaterials(doc *gltf.Document, basePath string) []Material {
	materials := make([]Material, len(doc.Materials))

	for i, mat := range doc.Materials {
		m := Material{
			Name:      mat.Name,
			BaseColor: [4]float64{1, 1, 1, 1}, // Default white
			Metallic:  0,
			Roughness: 1,
		}

		if mat.PBRMetallicRoughness != nil {
			pbr := mat.PBRMetallicRoughness

			// Extract base color
			if pbr.BaseColorFactor != nil {
				m.BaseColor = [4]float64{
					float64(pbr.BaseColorFactor[0]),
					float64(pbr.BaseColorFactor[1]),
					float64(pbr.BaseColorFactor[2]),
					float64(pbr.BaseColorFactor[3]),
				}
			}

			// Extract metallic/roughness
			if pbr.MetallicFactor != nil {
				m.Metallic = float64(*pbr.MetallicFactor)
			}
			if pbr.RoughnessFactor != nil {
				m.Roughness = float64(*pbr.RoughnessFactor)
			}

			// Extract base color texture if present
			if pbr.BaseColorTexture != nil {
				texIdx := pbr.BaseColorTexture.Index
				if int(texIdx) < len(doc.Textures) {
					tex := doc.Textures[texIdx]
					if tex.Source != nil && int(*tex.Source) < len(doc.Images) {
						img := doc.Images[*tex.Source]
						texImg := loadGLTFImage(doc, img, basePath)
						if texImg != nil {
							m.BaseMap = texImg
							m.HasTexture = true
						}
					}
				}
			}
		}

		materials[i] = m
	}

	return materials
}

// loadGLTFImage loads an image from GLTF (embedded or external).
func loadGLTFImage(doc *gltf.Document, img *gltf.Image, basePath string) image.Image {
	if img.BufferView != nil {
		// Embedded image
		bv := doc.BufferViews[*img.BufferView]
		buf := doc.Buffers[bv.Buffer]
		if buf.Data != nil {
			start := bv.ByteOffset
			end := start + bv.ByteLength
			decoded, _, err := image.Decode(bytes.NewReader(buf.Data[start:end]))
			if err == nil {
				return decoded
			}
		}
	} else if img.URI != "" {
		// External image file
		dir := filepath.Dir(basePath)
		imgPath := filepath.Join(dir, img.URI)
		data, err := os.ReadFile(imgPath)
		if err == nil {
			decoded, _, err := image.Decode(bytes.NewReader(data))
			if err == nil {
				return decoded
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

// LoadGLBWithTexture loads a GLB file and returns the mesh plus its primary base-color texture.
// Returns (mesh, texture image, error). Texture may be nil if none embedded.
func LoadGLBWithTexture(path string) (*Mesh, image.Image, error) {
	mesh, textures, err := LoadGLTFWithTextures(path)
	if err != nil {
		return nil, nil, err
	}

	if textureImg := selectPrimaryGLTFTexture(mesh, textures); textureImg != nil {
		return mesh, textureImg, nil
	}

	return mesh, nil, nil
}

func selectPrimaryGLTFTexture(mesh *Mesh, textures map[int][]byte) image.Image {
	if mesh != nil {
		for _, face := range mesh.Faces {
			material := mesh.GetMaterial(face.Material)
			if material != nil && material.HasTexture && material.BaseMap != nil {
				return material.BaseMap
			}
		}

		for _, material := range mesh.Materials {
			if material.HasTexture && material.BaseMap != nil {
				return material.BaseMap
			}
		}
	}

	for _, data := range textures {
		if len(data) == 0 {
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err == nil {
			return img
		}
	}

	return nil
}
