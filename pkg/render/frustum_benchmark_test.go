package render

import (
	"math"
	"math/rand"
	"testing"

	"github.com/taigrr/trophy/pkg/math3d"
)

// BenchmarkFrustumExtract benchmarks frustum plane extraction from view-projection matrix.
func BenchmarkFrustumExtract(b *testing.B) {
	fov := math.Pi / 3
	aspect := 16.0 / 9.0
	near := 0.1
	far := 100.0

	proj := math3d.Perspective(fov, aspect, near, far)
	view := math3d.Identity()
	viewProj := proj.Mul(view)

	for b.Loop() {
		_ = ExtractFrustum(viewProj)
	}
}

// BenchmarkAABBIntersection benchmarks AABB vs frustum intersection test.
func BenchmarkAABBIntersection(b *testing.B) {
	fov := math.Pi / 3
	aspect := 16.0 / 9.0
	near := 0.1
	far := 100.0

	proj := math3d.Perspective(fov, aspect, near, far)
	view := math3d.Identity()
	viewProj := proj.Mul(view)
	frustum := ExtractFrustum(viewProj)

	// AABB in front of camera (visible)
	visibleBounds := AABB{
		Min: math3d.V3(-1, -1, -15),
		Max: math3d.V3(1, 1, -5),
	}

	b.Run("visible", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = frustum.IntersectsFrustum(visibleBounds)
		}
	})

	// AABB behind camera (culled quickly)
	culledBounds := AABB{
		Min: math3d.V3(-1, -1, 5),
		Max: math3d.V3(1, 1, 15),
	}

	b.Run("culled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = frustum.IntersectsFrustum(culledBounds)
		}
	})
}

// BenchmarkTransformAABB benchmarks AABB transformation.
func BenchmarkTransformAABB(b *testing.B) {
	local := AABB{
		Min: math3d.V3(-1, -1, -1),
		Max: math3d.V3(1, 1, 1),
	}
	transform := math3d.Translate(math3d.V3(10, 5, -20)).Mul(math3d.RotateY(0.5)).Mul(math3d.ScaleUniform(2))

	for b.Loop() {
		_ = TransformAABB(local, transform)
	}
}

// BenchmarkCullingScenario simulates culling N objects, some visible, some not.
func BenchmarkCullingScenario(b *testing.B) {
	// Setup camera and frustum
	cam := NewCamera()
	cam.SetPosition(math3d.V3(0, 10, 20))
	cam.LookAt(math3d.V3(0, 0, 0))

	viewProj := cam.ViewProjectionMatrix()
	frustum := ExtractFrustum(viewProj)

	// Generate random objects: some in view, some out
	rng := rand.New(rand.NewSource(42))
	objectCount := 100

	type object struct {
		bounds    AABB
		transform math3d.Mat4
	}
	objects := make([]object, objectCount)

	for i := range objectCount {
		// Random position: X, Z in [-50, 50], Y in [0, 10]
		x := rng.Float64()*100 - 50
		y := rng.Float64() * 10
		z := rng.Float64()*100 - 50

		objects[i] = object{
			bounds: AABB{
				Min: math3d.V3(-1, -1, -1),
				Max: math3d.V3(1, 1, 1),
			},
			transform: math3d.Translate(math3d.V3(x, y, z)),
		}
	}

	b.Run("with_culling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			visible := 0
			for _, obj := range objects {
				worldBounds := TransformAABB(obj.bounds, obj.transform)
				if frustum.IntersectsFrustum(worldBounds) {
					visible++
				}
			}
			_ = visible
		}
	})

	b.Run("no_culling", func(b *testing.B) {
		// Simulate just doing work without culling
		for i := 0; i < b.N; i++ {
			visible := 0
			for range objects {
				// Pretend we "render" everything
				visible++
			}
			_ = visible
		}
	})
}

// BenchmarkMeshRenderingComparison compares rendering with and without culling.
// This is a synthetic benchmark that estimates the savings.
func BenchmarkMeshRenderingComparison(b *testing.B) {
	// Create a simple framebuffer and rasterizer
	fb := NewFramebuffer(160, 120)
	cam := NewCamera()
	cam.SetPosition(math3d.V3(0, 10, 20))
	cam.LookAt(math3d.V3(0, 0, 0))

	rast := NewRasterizer(cam, fb)

	// Create a simple mesh (cube)
	mesh := &simpleMesh{
		vertices: []meshVertex{
			// Front face
			{pos: math3d.V3(-1, -1, 1), normal: math3d.V3(0, 0, 1)},
			{pos: math3d.V3(1, -1, 1), normal: math3d.V3(0, 0, 1)},
			{pos: math3d.V3(1, 1, 1), normal: math3d.V3(0, 0, 1)},
			{pos: math3d.V3(-1, 1, 1), normal: math3d.V3(0, 0, 1)},
			// Back face
			{pos: math3d.V3(-1, -1, -1), normal: math3d.V3(0, 0, -1)},
			{pos: math3d.V3(1, -1, -1), normal: math3d.V3(0, 0, -1)},
			{pos: math3d.V3(1, 1, -1), normal: math3d.V3(0, 0, -1)},
			{pos: math3d.V3(-1, 1, -1), normal: math3d.V3(0, 0, -1)},
		},
		faces: [][3]int{
			{0, 1, 2}, {0, 2, 3}, // Front
			{4, 6, 5}, {4, 7, 6}, // Back
			{0, 3, 7}, {0, 7, 4}, // Left
			{1, 5, 6}, {1, 6, 2}, // Right
			{3, 2, 6}, {3, 6, 7}, // Top
			{0, 4, 5}, {0, 5, 1}, // Bottom
		},
		bounds: AABB{
			Min: math3d.V3(-1, -1, -1),
			Max: math3d.V3(1, 1, 1),
		},
	}

	lightDir := math3d.V3(0.5, 1, 0.3).Normalize()
	color := RGB(100, 150, 200)

	// Generate objects: 50% visible, 50% behind camera
	rng := rand.New(rand.NewSource(42))
	objectCount := 100
	transforms := make([]math3d.Mat4, objectCount)

	for i := range objectCount {
		var z float64
		if i%2 == 0 {
			// Visible: in front of camera
			z = rng.Float64()*30 - 40 // Z from -40 to -10
		} else {
			// Culled: behind camera
			z = rng.Float64()*20 + 25 // Z from 25 to 45
		}
		x := rng.Float64()*40 - 20
		y := rng.Float64() * 10
		transforms[i] = math3d.Translate(math3d.V3(x, y, z))
	}

	b.Run("with_culling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rast.ClearDepth()
			fb.Clear(RGB(0, 0, 0))
			rast.InvalidateFrustum()
			rast.ResetCullingStats()

			for _, transform := range transforms {
				rast.DrawMeshGouraudCulled(mesh, transform, mesh.bounds, color, lightDir)
			}
		}
	})

	b.Run("without_culling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rast.ClearDepth()
			fb.Clear(RGB(0, 0, 0))

			for _, transform := range transforms {
				rast.DrawMeshGouraud(mesh, transform, color, lightDir)
			}
		}
	})
}

// simpleMesh is a test implementation of MeshRenderer.
type simpleMesh struct {
	vertices []meshVertex
	faces    [][3]int
	bounds   AABB
}

type meshVertex struct {
	pos    math3d.Vec3
	normal math3d.Vec3
	uv     math3d.Vec2
}

func (m *simpleMesh) VertexCount() int   { return len(m.vertices) }
func (m *simpleMesh) TriangleCount() int { return len(m.faces) }

func (m *simpleMesh) GetVertex(i int) (pos, normal math3d.Vec3, uv math3d.Vec2) {
	v := m.vertices[i]
	return v.pos, v.normal, v.uv
}

func (m *simpleMesh) GetFace(i int) [3]int {
	return m.faces[i]
}
