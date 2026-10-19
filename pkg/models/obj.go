package models

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/taigrr/trophy/pkg/math3d"
)

const maxOBJLineSize = 16 * 1024 * 1024

// OBJLoader loads Wavefront OBJ files.
type OBJLoader struct {
	// Options
	CalculateNormals bool // If true, calculate normals if not provided
	SmoothNormals    bool // If true, use smooth shading (averaged normals)
}

// NewOBJLoader creates a new OBJ loader with default settings.
func NewOBJLoader() *OBJLoader {
	return &OBJLoader{
		CalculateNormals: true,
		SmoothNormals:    false,
	}
}

// LoadFile loads an OBJ file from disk.
func (l *OBJLoader) LoadFile(path string) (*Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open OBJ file: %w", err)
	}
	defer f.Close()

	return l.Load(f, path)
}

// Load parses an OBJ from a reader.
func (l *OBJLoader) Load(r io.Reader, name string) (*Mesh, error) {
	mesh := NewMesh(name)

	// Temporary storage for OBJ data (1-indexed in OBJ format)
	var positions []math3d.Vec3
	var normals []math3d.Vec3
	var uvs []math3d.Vec2

	// Map to deduplicate vertices (OBJ can have different indices for pos/uv/normal)
	type vertexKey struct {
		pos, uv, normal int
	}
	vertexMap := make(map[vertexKey]int)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxOBJLineSize)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "v": // Vertex position
			if len(fields) < 4 {
				return nil, fmt.Errorf("line %d: invalid vertex (need x y z)", lineNum)
			}
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid x coordinate: %w", lineNum, err)
			}
			y, err := strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid y coordinate: %w", lineNum, err)
			}
			z, err := strconv.ParseFloat(fields[3], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid z coordinate: %w", lineNum, err)
			}
			positions = append(positions, math3d.V3(x, y, z))

		case "vt": // Texture coordinate
			if len(fields) < 3 {
				return nil, fmt.Errorf("line %d: invalid texture coord (need u v)", lineNum)
			}
			u, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid u coordinate: %w", lineNum, err)
			}
			v, err := strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid v coordinate: %w", lineNum, err)
			}
			uvs = append(uvs, math3d.V2(u, v))

		case "vn": // Vertex normal
			if len(fields) < 4 {
				return nil, fmt.Errorf("line %d: invalid normal (need x y z)", lineNum)
			}
			x, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid normal x: %w", lineNum, err)
			}
			y, err := strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid normal y: %w", lineNum, err)
			}
			z, err := strconv.ParseFloat(fields[3], 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid normal z: %w", lineNum, err)
			}
			normals = append(normals, math3d.V3(x, y, z).Normalize())

		case "f": // Face
			if len(fields) < 4 {
				return nil, fmt.Errorf("line %d: face needs at least 3 vertices", lineNum)
			}

			// Parse face vertices
			var faceVerts []int
			for i := 1; i < len(fields); i++ {
				posIdx, uvIdx, normalIdx, err := parseFaceVertex(fields[i])
				if err != nil {
					return nil, fmt.Errorf("line %d: %w", lineNum, err)
				}

				// Convert to 0-indexed, handle negative indices
				posIdx = resolveIndex(posIdx, len(positions))
				uvIdx = resolveIndex(uvIdx, len(uvs))
				normalIdx = resolveIndex(normalIdx, len(normals))

				// Check bounds
				if posIdx < 0 || posIdx >= len(positions) {
					return nil, fmt.Errorf("line %d: position index %d out of range", lineNum, posIdx+1)
				}

				// Create or reuse vertex
				key := vertexKey{posIdx, uvIdx, normalIdx}
				vertIdx, exists := vertexMap[key]
				if !exists {
					vert := MeshVertex{
						Position: positions[posIdx],
					}
					if uvIdx >= 0 && uvIdx < len(uvs) {
						vert.UV = uvs[uvIdx]
					}
					if normalIdx >= 0 && normalIdx < len(normals) {
						vert.Normal = normals[normalIdx]
					}
					vertIdx = len(mesh.Vertices)
					mesh.Vertices = append(mesh.Vertices, vert)
					vertexMap[key] = vertIdx
				}
				faceVerts = append(faceVerts, vertIdx)
			}

			// Triangulate (fan triangulation for convex polygons)
			// Note: OBJ uses CCW winding for front-facing, but our engine uses CW
			// (due to Y-flip in screen space), so we reverse the winding here
			for i := 1; i < len(faceVerts)-1; i++ {
				mesh.Faces = append(mesh.Faces, Face{
					V: [3]int{faceVerts[0], faceVerts[i+1], faceVerts[i]}, // swapped i and i+1
				})
			}

		case "o", "g": // Object/group name (use as mesh name)
			if len(fields) > 1 {
				mesh.Name = fields[1]
			}

		case "mtllib", "usemtl", "s": // Material library, material use, smoothing - ignore for now

		default:
			// Ignore unknown directives
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading OBJ: %w", err)
	}

	// Calculate bounds
	mesh.CalculateBounds()

	// Calculate normals if needed
	if l.CalculateNormals && len(normals) == 0 {
		if l.SmoothNormals {
			mesh.CalculateSmoothNormals()
		} else {
			mesh.CalculateNormals()
		}
	}

	return mesh, nil
}

// parseFaceVertex parses a face vertex in format: v, v/vt, v/vt/vn, or v//vn
// Returns 1-indexed values (0 means not specified)
func parseFaceVertex(s string) (pos, uv, normal int, err error) {
	parts := strings.Split(s, "/")

	// Position (required)
	pos, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid vertex index: %s", parts[0])
	}

	// UV (optional)
	if len(parts) > 1 && parts[1] != "" {
		uv, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid texture index: %s", parts[1])
		}
	}

	// Normal (optional)
	if len(parts) > 2 && parts[2] != "" {
		normal, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid normal index: %s", parts[2])
		}
	}

	return pos, uv, normal, nil
}

// resolveIndex converts OBJ 1-indexed (or negative) index to 0-indexed.
// Returns -1 if index was 0 (not specified).
func resolveIndex(idx, count int) int {
	if idx == 0 {
		return -1
	}
	if idx < 0 {
		return count + idx // Negative indices count from end
	}
	return idx - 1 // Convert 1-indexed to 0-indexed
}

// LoadOBJ is a convenience function to load an OBJ file with default settings.
func LoadOBJ(path string) (*Mesh, error) {
	return NewOBJLoader().LoadFile(path)
}

// LoadOBJSmooth loads an OBJ file with smooth normals.
func LoadOBJSmooth(path string) (*Mesh, error) {
	loader := NewOBJLoader()
	loader.SmoothNormals = true
	return loader.LoadFile(path)
}
