// Package render provides software rasterization for Trophy.
package render

import (
	"math"

	"github.com/taigrr/trophy/pkg/math3d"
)

// Vertex represents a vertex with all attributes needed for rasterization.
type Vertex struct {
	Position math3d.Vec3 // World position
	Normal   math3d.Vec3 // Normal vector (for lighting)
	UV       math3d.Vec2 // Texture coordinates
	Color    Color       // Vertex color
}

// Triangle represents a triangle to be rasterized.
type Triangle struct {
	V [3]Vertex
}

// Rasterizer handles software triangle rasterization.
type Rasterizer struct {
	camera                 *Camera
	fb                     *Framebuffer
	zbuffer                []float64    // Depth buffer (1D array, row-major)
	frustum                Frustum      // Cached frustum planes
	frustumDirty           bool         // Whether frustum needs recalculation
	CullingStats           CullingStats // Statistics for debugging/benchmarking
	DisableBackfaceCulling bool         // If true, render both sides of triangles
}

// CullingStats tracks frustum culling performance.
type CullingStats struct {
	MeshesTested int // Total meshes tested for culling
	MeshesCulled int // Meshes culled (not rendered)
	MeshesDrawn  int // Meshes that passed culling
}

// NewRasterizer creates a new rasterizer.
func NewRasterizer(camera *Camera, fb *Framebuffer) *Rasterizer {
	r := &Rasterizer{
		camera:       camera,
		fb:           fb,
		frustumDirty: true,
	}
	r.Resize()
	return r
}

// Resize resizes the rasterizer's buffer to match the framebuffer.
func (r *Rasterizer) Resize() {
	if r.fb == nil {
		r.zbuffer = nil
		return
	}
	r.zbuffer = make([]float64, r.fb.Width*r.fb.Height)
}

// Width returns the framebuffer width.
func (r *Rasterizer) Width() int {
	if r.fb == nil {
		return 0
	}
	return r.fb.Width
}

// Height returns the framebuffer height.
func (r *Rasterizer) Height() int {
	if r.fb == nil {
		return 0
	}
	return r.fb.Height
}

// ClearDepth clears the Z-buffer (call before each frame).
func (r *Rasterizer) ClearDepth() {
	// Use copy-doubling for faster clearing
	n := len(r.zbuffer)
	if n == 0 {
		return
	}
	r.zbuffer[0] = math.MaxFloat64
	for i := 1; i < n; i *= 2 {
		copy(r.zbuffer[i:], r.zbuffer[:i])
	}
}

// InvalidateFrustum marks the frustum as needing recalculation.
// Call this when the camera moves or rotates.
func (r *Rasterizer) InvalidateFrustum() {
	r.frustumDirty = true
}

// UpdateFrustum recalculates the frustum planes from the camera.
func (r *Rasterizer) UpdateFrustum() {
	if r.frustumDirty {
		r.frustum = ExtractFrustum(r.camera.ViewProjectionMatrix())
		r.frustumDirty = false
	}
}

// GetFrustum returns the current frustum (updating if needed).
func (r *Rasterizer) GetFrustum() Frustum {
	r.UpdateFrustum()
	return r.frustum
}

// ResetCullingStats resets the culling statistics (call once per frame).
func (r *Rasterizer) ResetCullingStats() {
	r.CullingStats = CullingStats{}
}

// IsVisible tests if a world-space AABB is visible in the frustum.
func (r *Rasterizer) IsVisible(worldBounds AABB) bool {
	r.UpdateFrustum()
	return r.frustum.IntersectsFrustum(worldBounds)
}

// IsVisibleTransformed tests if a local-space AABB is visible after transformation.
func (r *Rasterizer) IsVisibleTransformed(localBounds AABB, transform math3d.Mat4) bool {
	worldBounds := TransformAABB(localBounds, transform)
	return r.IsVisible(worldBounds)
}

// getDepth returns the depth at (x, y).
func (r *Rasterizer) getDepth(x, y int) float64 {
	if x < 0 || x >= r.Width() || y < 0 || y >= r.Height() {
		return math.MaxFloat64
	}
	return r.zbuffer[y*r.Width()+x]
}

// setDepth sets the depth at (x, y).
func (r *Rasterizer) setDepth(x, y int, z float64) {
	if x < 0 || x >= r.Width() || y < 0 || y >= r.Height() {
		return
	}
	r.zbuffer[y*r.Width()+x] = z
}

// screenVertex holds a vertex transformed to screen space.
type screenVertex struct {
	X, Y   float64 // Screen coordinates
	Z      float64 // Depth (for Z-buffer)
	W      float64 // W coordinate (for perspective-correct interpolation)
	Color  Color
	Normal math3d.Vec3
	UV     math3d.Vec2
}

// DrawTriangle rasterizes a single triangle.
func (r *Rasterizer) DrawTriangle(tri Triangle) {
	// Transform vertices to screen space
	var sv [3]screenVertex
	allBehind := true

	viewProj := r.camera.ViewProjectionMatrix()

	for i := range 3 {
		// Transform to clip space
		clipPos := viewProj.MulVec4(math3d.V4FromV3(tri.V[i].Position, 1))

		// Check if behind camera
		if clipPos.W > 0 {
			allBehind = false
		}

		// Perspective divide
		if clipPos.W != 0 {
			sv[i].X = clipPos.X / clipPos.W
			sv[i].Y = clipPos.Y / clipPos.W
			sv[i].Z = clipPos.Z / clipPos.W
		}
		sv[i].W = clipPos.W

		// NDC to screen coordinates
		sv[i].X = (sv[i].X + 1) * 0.5 * float64(r.Width())
		sv[i].Y = (1 - sv[i].Y) * 0.5 * float64(r.Height()) // Y flipped

		// Copy other attributes
		sv[i].Color = tri.V[i].Color
		sv[i].Normal = tri.V[i].Normal
		sv[i].UV = tri.V[i].UV
	}

	// Skip if entirely behind camera
	if allBehind {
		return
	}

	// Backface culling (using screen-space winding)
	edge1 := math3d.V2(sv[1].X-sv[0].X, sv[1].Y-sv[0].Y)
	edge2 := math3d.V2(sv[2].X-sv[0].X, sv[2].Y-sv[0].Y)
	cross := edge1.X*edge2.Y - edge1.Y*edge2.X
	if cross < 0 {
		return // Back-facing
	}

	// Find bounding box
	minX := int(math.Max(0, math.Floor(min3(sv[0].X, sv[1].X, sv[2].X))))
	maxX := int(math.Min(float64(r.Width()-1), math.Ceil(max3(sv[0].X, sv[1].X, sv[2].X))))
	minY := int(math.Max(0, math.Floor(min3(sv[0].Y, sv[1].Y, sv[2].Y))))
	maxY := int(math.Min(float64(r.Height()-1), math.Ceil(max3(sv[0].Y, sv[1].Y, sv[2].Y))))

	// Rasterize using barycentric coordinates
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5

			// Calculate barycentric coordinates
			bc := barycentric(
				sv[0].X, sv[0].Y,
				sv[1].X, sv[1].Y,
				sv[2].X, sv[2].Y,
				px, py,
			)

			// Check if inside triangle
			if bc.X < 0 || bc.Y < 0 || bc.Z < 0 {
				continue
			}

			// Interpolate depth (perspective-correct)
			z := bc.X*sv[0].Z + bc.Y*sv[1].Z + bc.Z*sv[2].Z

			// Z-buffer test
			if z >= r.getDepth(x, y) {
				continue
			}

			// Interpolate color (perspective-correct would use W, but flat is fine for now)
			color := interpolateColor3(sv[0].Color, sv[1].Color, sv[2].Color, bc)

			// Set pixel
			r.setDepth(x, y, z)
			r.fb.SetPixel(x, y, color)
		}
	}
}

// DrawTriangleTextured rasterizes a textured triangle with perspective-correct UV interpolation.
func (r *Rasterizer) DrawTriangleTextured(tri Triangle, tex *Texture, lightDir math3d.Vec3) {
	// Transform vertices to screen space
	var sv [3]screenVertex
	allBehind := true

	viewProj := r.camera.ViewProjectionMatrix()

	for i := range 3 {
		// Transform to clip space
		clipPos := viewProj.MulVec4(math3d.V4FromV3(tri.V[i].Position, 1))

		// Check if behind camera
		if clipPos.W > 0 {
			allBehind = false
		}

		// Perspective divide
		if clipPos.W != 0 {
			sv[i].X = clipPos.X / clipPos.W
			sv[i].Y = clipPos.Y / clipPos.W
			sv[i].Z = clipPos.Z / clipPos.W
		}
		sv[i].W = clipPos.W

		// NDC to screen coordinates
		sv[i].X = (sv[i].X + 1) * 0.5 * float64(r.Width())
		sv[i].Y = (1 - sv[i].Y) * 0.5 * float64(r.Height()) // Y flipped

		// Copy other attributes
		sv[i].Color = tri.V[i].Color
		sv[i].Normal = tri.V[i].Normal
		sv[i].UV = tri.V[i].UV
	}

	// Skip if entirely behind camera
	if allBehind {
		return
	}

	// Backface culling (using screen-space winding)
	edge1 := math3d.V2(sv[1].X-sv[0].X, sv[1].Y-sv[0].Y)
	edge2 := math3d.V2(sv[2].X-sv[0].X, sv[2].Y-sv[0].Y)
	cross := edge1.X*edge2.Y - edge1.Y*edge2.X
	if cross < 0 {
		return // Back-facing
	}

	// Calculate face normal for lighting (from original vertices)
	e1 := tri.V[1].Position.Sub(tri.V[0].Position)
	e2 := tri.V[2].Position.Sub(tri.V[0].Position)
	faceNormal := e1.Cross(e2).Normalize()
	intensity := math.Max(0.2, faceNormal.Dot(lightDir.Normalize()))
	intensity = 0.3 + 0.7*intensity // Ambient + diffuse

	// Find bounding box
	minX := int(math.Max(0, math.Floor(min3(sv[0].X, sv[1].X, sv[2].X))))
	maxX := int(math.Min(float64(r.Width()-1), math.Ceil(max3(sv[0].X, sv[1].X, sv[2].X))))
	minY := int(math.Max(0, math.Floor(min3(sv[0].Y, sv[1].Y, sv[2].Y))))
	maxY := int(math.Min(float64(r.Height()-1), math.Ceil(max3(sv[0].Y, sv[1].Y, sv[2].Y))))

	// Precompute perspective-correct interpolation factors (1/w for each vertex)
	var invW [3]float64
	for i := range 3 {
		if sv[i].W != 0 {
			invW[i] = 1.0 / sv[i].W
		} else {
			invW[i] = 0
		}
	}

	// Rasterize using barycentric coordinates with perspective correction
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5

			// Calculate barycentric coordinates
			bc := barycentric(
				sv[0].X, sv[0].Y,
				sv[1].X, sv[1].Y,
				sv[2].X, sv[2].Y,
				px, py,
			)

			// Check if inside triangle
			if bc.X < 0 || bc.Y < 0 || bc.Z < 0 {
				continue
			}

			// Interpolate depth
			z := bc.X*sv[0].Z + bc.Y*sv[1].Z + bc.Z*sv[2].Z

			// Z-buffer test
			if z >= r.getDepth(x, y) {
				continue
			}

			// Perspective-correct UV interpolation
			// Interpolate UV/W and 1/W, then divide to get correct UV
			w0, w1, w2 := bc.X*invW[0], bc.Y*invW[1], bc.Z*invW[2]
			oneOverW := w0 + w1 + w2
			if oneOverW == 0 {
				continue
			}

			u := (w0*sv[0].UV.X + w1*sv[1].UV.X + w2*sv[2].UV.X) / oneOverW
			v := (w0*sv[0].UV.Y + w1*sv[1].UV.Y + w2*sv[2].UV.Y) / oneOverW

			// Sample texture
			texColor := tex.Sample(u, v)

			// Apply lighting
			litColor := MultiplyColor(texColor, intensity)

			// Set pixel
			r.setDepth(x, y, z)
			r.fb.SetPixel(x, y, litColor)
		}
	}
}

// DrawTriangleFlat draws a triangle with flat shading (single color).
func (r *Rasterizer) DrawTriangleFlat(v0, v1, v2 math3d.Vec3, color Color) {
	tri := Triangle{
		V: [3]Vertex{
			{Position: v0, Color: color},
			{Position: v1, Color: color},
			{Position: v2, Color: color},
		},
	}
	r.DrawTriangle(tri)
}

// DrawTriangleLit draws a triangle with simple directional lighting.
func (r *Rasterizer) DrawTriangleLit(v0, v1, v2 math3d.Vec3, baseColor Color, lightDir math3d.Vec3) {
	// Calculate face normal
	edge1 := v1.Sub(v0)
	edge2 := v2.Sub(v0)
	normal := edge1.Cross(edge2).Normalize()

	// Calculate lighting intensity
	intensity := math.Max(0, normal.Dot(lightDir.Normalize()))
	intensity = 0.3 + 0.7*intensity // Ambient + diffuse

	// Apply lighting to color
	litColor := RGB(
		uint8(float64(baseColor.R)*intensity),
		uint8(float64(baseColor.G)*intensity),
		uint8(float64(baseColor.B)*intensity),
	)

	r.DrawTriangleFlat(v0, v1, v2, litColor)
}

// DrawQuad draws a quad as two triangles.
func (r *Rasterizer) DrawQuad(v0, v1, v2, v3 math3d.Vec3, color Color) {
	r.DrawTriangleFlat(v0, v1, v2, color)
	r.DrawTriangleFlat(v0, v2, v3, color)
}

// DrawCube draws a solid cube with lighting.
func (r *Rasterizer) DrawCube(center math3d.Vec3, size float64, color Color, lightDir math3d.Vec3) {
	h := size / 2

	// 8 vertices
	v := [8]math3d.Vec3{
		{X: center.X - h, Y: center.Y - h, Z: center.Z - h}, // 0: left-bottom-back
		{X: center.X + h, Y: center.Y - h, Z: center.Z - h}, // 1: right-bottom-back
		{X: center.X + h, Y: center.Y + h, Z: center.Z - h}, // 2: right-top-back
		{X: center.X - h, Y: center.Y + h, Z: center.Z - h}, // 3: left-top-back
		{X: center.X - h, Y: center.Y - h, Z: center.Z + h}, // 4: left-bottom-front
		{X: center.X + h, Y: center.Y - h, Z: center.Z + h}, // 5: right-bottom-front
		{X: center.X + h, Y: center.Y + h, Z: center.Z + h}, // 6: right-top-front
		{X: center.X - h, Y: center.Y + h, Z: center.Z + h}, // 7: left-top-front
	}

	// 6 faces (2 triangles each)
	faces := [][4]int{
		{0, 1, 2, 3}, // Back
		{5, 4, 7, 6}, // Front
		{4, 0, 3, 7}, // Left
		{1, 5, 6, 2}, // Right
		{3, 2, 6, 7}, // Top
		{4, 5, 1, 0}, // Bottom
	}

	for _, f := range faces {
		r.DrawTriangleLit(v[f[0]], v[f[1]], v[f[2]], color, lightDir)
		r.DrawTriangleLit(v[f[0]], v[f[2]], v[f[3]], color, lightDir)
	}
}

// DrawTransformedCube draws a cube with a transformation matrix.
func (r *Rasterizer) DrawTransformedCube(transform math3d.Mat4, size float64, color Color, lightDir math3d.Vec3) {
	h := size / 2

	// Local vertices
	local := [8]math3d.Vec3{
		{X: -h, Y: -h, Z: -h},
		{X: h, Y: -h, Z: -h},
		{X: h, Y: h, Z: -h},
		{X: -h, Y: h, Z: -h},
		{X: -h, Y: -h, Z: h},
		{X: h, Y: -h, Z: h},
		{X: h, Y: h, Z: h},
		{X: -h, Y: h, Z: h},
	}

	// Transform vertices
	var v [8]math3d.Vec3
	for i := range local {
		v[i] = transform.MulVec3(local[i])
	}

	// Transform light direction to local space for consistent lighting
	invTransform := transform.Inverse()
	localLight := invTransform.MulVec3Dir(lightDir).Normalize()

	faces := [][4]int{
		{0, 1, 2, 3}, // Back
		{5, 4, 7, 6}, // Front
		{4, 0, 3, 7}, // Left
		{1, 5, 6, 2}, // Right
		{3, 2, 6, 7}, // Top
		{4, 5, 1, 0}, // Bottom
	}

	for _, f := range faces {
		r.DrawTriangleLit(v[f[0]], v[f[1]], v[f[2]], color, localLight)
		r.DrawTriangleLit(v[f[0]], v[f[2]], v[f[3]], color, localLight)
	}
}

// DrawTransformedCubeGouraud draws a cube with Gouraud shading (smooth corners).
// Uses per-vertex normals averaged at corners for smoother lighting transitions.
func (r *Rasterizer) DrawTransformedCubeGouraud(transform math3d.Mat4, size float64, color Color, lightDir math3d.Vec3) {
	h := size / 2

	// Local vertices
	local := [8]math3d.Vec3{
		{X: -h, Y: -h, Z: -h}, {X: h, Y: -h, Z: -h}, {X: h, Y: h, Z: -h}, {X: -h, Y: h, Z: -h}, // back
		{X: -h, Y: -h, Z: h}, {X: h, Y: -h, Z: h}, {X: h, Y: h, Z: h}, {X: -h, Y: h, Z: h}, // front
	}

	// Vertex normals (pointing outward from cube corners, averaged from 3 adjacent faces)
	// For a cube, each corner touches 3 faces, so normal is diagonal
	sqrt3 := 1.0 / math.Sqrt(3)
	vertexNormals := [8]math3d.Vec3{
		{X: -sqrt3, Y: -sqrt3, Z: -sqrt3},
		{X: sqrt3, Y: -sqrt3, Z: -sqrt3},
		{X: sqrt3, Y: sqrt3, Z: -sqrt3},
		{X: -sqrt3, Y: sqrt3, Z: -sqrt3},
		{X: -sqrt3, Y: -sqrt3, Z: sqrt3},
		{X: sqrt3, Y: -sqrt3, Z: sqrt3},
		{X: sqrt3, Y: sqrt3, Z: sqrt3},
		{X: -sqrt3, Y: sqrt3, Z: sqrt3},
	}

	// Transform vertices and normals
	var v [8]math3d.Vec3
	var n [8]math3d.Vec3
	for i := range local {
		v[i] = transform.MulVec3(local[i])
		n[i] = transform.MulVec3Dir(vertexNormals[i]).Normalize()
	}

	// Face definitions (same as before)
	faces := [][4]int{
		{0, 1, 2, 3}, // Back
		{5, 4, 7, 6}, // Front
		{4, 0, 3, 7}, // Left
		{1, 5, 6, 2}, // Right
		{3, 2, 6, 7}, // Top
		{4, 5, 1, 0}, // Bottom
	}

	for _, f := range faces {
		// Triangle 1: f[0], f[1], f[2]
		tri1 := Triangle{
			V: [3]Vertex{
				{Position: v[f[0]], Normal: n[f[0]], Color: color},
				{Position: v[f[1]], Normal: n[f[1]], Color: color},
				{Position: v[f[2]], Normal: n[f[2]], Color: color},
			},
		}
		r.DrawTriangleGouraud(tri1, lightDir)

		// Triangle 2: f[0], f[2], f[3]
		tri2 := Triangle{
			V: [3]Vertex{
				{Position: v[f[0]], Normal: n[f[0]], Color: color},
				{Position: v[f[2]], Normal: n[f[2]], Color: color},
				{Position: v[f[3]], Normal: n[f[3]], Color: color},
			},
		}
		r.DrawTriangleGouraud(tri2, lightDir)
	}
}

// DrawTexturedCube draws a cube with texture mapping.
func (r *Rasterizer) DrawTexturedCube(transform math3d.Mat4, size float64, tex *Texture, lightDir math3d.Vec3) {
	h := size / 2

	// Local vertices
	local := [8]math3d.Vec3{
		{X: -h, Y: -h, Z: -h}, {X: h, Y: -h, Z: -h}, {X: h, Y: h, Z: -h}, {X: -h, Y: h, Z: -h}, // back
		{X: -h, Y: -h, Z: h}, {X: h, Y: -h, Z: h}, {X: h, Y: h, Z: h}, {X: -h, Y: h, Z: h}, // front
	}

	// UV coordinates for each corner of a face
	uvBL := math3d.V2(0, 0)
	uvBR := math3d.V2(1, 0)
	uvTR := math3d.V2(1, 1)
	uvTL := math3d.V2(0, 1)

	// Transform vertices
	var v [8]math3d.Vec3
	for i := range local {
		v[i] = transform.MulVec3(local[i])
	}

	// Face definitions: 4 vertices per face (v0, v1, v2, v3 forming two triangles)
	// Each face: triangle 1 (v0, v1, v2), triangle 2 (v0, v2, v3)
	faces := [][4]int{
		{0, 1, 2, 3}, // Back  (-Z)
		{5, 4, 7, 6}, // Front (+Z)
		{4, 0, 3, 7}, // Left  (-X)
		{1, 5, 6, 2}, // Right (+X)
		{3, 2, 6, 7}, // Top   (+Y)
		{4, 5, 1, 0}, // Bottom(-Y)
	}

	// Face normals (local space)
	normals := []math3d.Vec3{
		{X: 0, Y: 0, Z: -1}, // Back
		{X: 0, Y: 0, Z: 1},  // Front
		{X: -1, Y: 0, Z: 0}, // Left
		{X: 1, Y: 0, Z: 0},  // Right
		{X: 0, Y: 1, Z: 0},  // Top
		{X: 0, Y: -1, Z: 0}, // Bottom
	}

	for fi, f := range faces {
		normal := transform.MulVec3Dir(normals[fi]).Normalize()

		// Triangle 1: v0, v1, v2 (BL, BR, TR)
		tri1 := Triangle{
			V: [3]Vertex{
				{Position: v[f[0]], Normal: normal, UV: uvBL, Color: RGB(255, 255, 255)},
				{Position: v[f[1]], Normal: normal, UV: uvBR, Color: RGB(255, 255, 255)},
				{Position: v[f[2]], Normal: normal, UV: uvTR, Color: RGB(255, 255, 255)},
			},
		}
		r.DrawTriangleTextured(tri1, tex, lightDir)

		// Triangle 2: v0, v2, v3 (BL, TR, TL)
		tri2 := Triangle{
			V: [3]Vertex{
				{Position: v[f[0]], Normal: normal, UV: uvBL, Color: RGB(255, 255, 255)},
				{Position: v[f[2]], Normal: normal, UV: uvTR, Color: RGB(255, 255, 255)},
				{Position: v[f[3]], Normal: normal, UV: uvTL, Color: RGB(255, 255, 255)},
			},
		}
		r.DrawTriangleTextured(tri2, tex, lightDir)
	}
}

// barycentric calculates barycentric coordinates for point (px, py) in triangle.
func barycentric(x0, y0, x1, y1, x2, y2, px, py float64) math3d.Vec3 {
	v0x, v0y := x2-x0, y2-y0
	v1x, v1y := x1-x0, y1-y0
	v2x, v2y := px-x0, py-y0

	dot00 := v0x*v0x + v0y*v0y
	dot01 := v0x*v1x + v0y*v1y
	dot02 := v0x*v2x + v0y*v2y
	dot11 := v1x*v1x + v1y*v1y
	dot12 := v1x*v2x + v1y*v2y

	invDenom := 1.0 / (dot00*dot11 - dot01*dot01)
	u := (dot11*dot02 - dot01*dot12) * invDenom
	v := (dot00*dot12 - dot01*dot02) * invDenom

	return math3d.V3(1-u-v, v, u)
}

// interpolateColor3 interpolates between 3 colors using barycentric coords.
func interpolateColor3(c0, c1, c2 Color, bc math3d.Vec3) Color {
	return RGB(
		uint8(float64(c0.R)*bc.X+float64(c1.R)*bc.Y+float64(c2.R)*bc.Z),
		uint8(float64(c0.G)*bc.X+float64(c1.G)*bc.Y+float64(c2.G)*bc.Z),
		uint8(float64(c0.B)*bc.X+float64(c1.B)*bc.Y+float64(c2.B)*bc.Z),
	)
}

func min3(a, b, c float64) float64 {
	return math.Min(a, math.Min(b, c))
}

func max3(a, b, c float64) float64 {
	return math.Max(a, math.Max(b, c))
}

// DrawTriangleGouraud rasterizes a triangle with Gouraud shading (per-vertex lighting).
// Lighting is calculated at each vertex and interpolated across the triangle.
func (r *Rasterizer) DrawTriangleGouraud(tri Triangle, lightDir math3d.Vec3) {
	// Transform vertices to screen space
	var sv [3]screenVertex
	allBehind := true

	viewProj := r.camera.ViewProjectionMatrix()
	normLight := lightDir.Normalize()

	for i := range 3 {
		// Transform to clip space
		clipPos := viewProj.MulVec4(math3d.V4FromV3(tri.V[i].Position, 1))

		// Check if behind camera
		if clipPos.W > 0 {
			allBehind = false
		}

		// Perspective divide
		if clipPos.W != 0 {
			sv[i].X = clipPos.X / clipPos.W
			sv[i].Y = clipPos.Y / clipPos.W
			sv[i].Z = clipPos.Z / clipPos.W
		}
		sv[i].W = clipPos.W

		// NDC to screen coordinates
		sv[i].X = (sv[i].X + 1) * 0.5 * float64(r.Width())
		sv[i].Y = (1 - sv[i].Y) * 0.5 * float64(r.Height()) // Y flipped

		// Calculate per-vertex lighting intensity
		intensity := math.Max(0, tri.V[i].Normal.Dot(normLight))
		intensity = 0.3 + 0.7*intensity // Ambient + diffuse

		// Apply lighting to vertex color
		sv[i].Color = RGB(
			uint8(float64(tri.V[i].Color.R)*intensity),
			uint8(float64(tri.V[i].Color.G)*intensity),
			uint8(float64(tri.V[i].Color.B)*intensity),
		)
		sv[i].Normal = tri.V[i].Normal
		sv[i].UV = tri.V[i].UV
	}

	// Skip if entirely behind camera
	if allBehind {
		return
	}

	// Backface culling (using screen-space winding)
	edge1 := math3d.V2(sv[1].X-sv[0].X, sv[1].Y-sv[0].Y)
	edge2 := math3d.V2(sv[2].X-sv[0].X, sv[2].Y-sv[0].Y)
	cross := edge1.X*edge2.Y - edge1.Y*edge2.X
	if cross < 0 {
		return // Back-facing
	}

	// Find bounding box
	minX := int(math.Max(0, math.Floor(min3(sv[0].X, sv[1].X, sv[2].X))))
	maxX := int(math.Min(float64(r.Width()-1), math.Ceil(max3(sv[0].X, sv[1].X, sv[2].X))))
	minY := int(math.Max(0, math.Floor(min3(sv[0].Y, sv[1].Y, sv[2].Y))))
	maxY := int(math.Min(float64(r.Height()-1), math.Ceil(max3(sv[0].Y, sv[1].Y, sv[2].Y))))

	// Rasterize using barycentric coordinates
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5

			// Calculate barycentric coordinates
			bc := barycentric(
				sv[0].X, sv[0].Y,
				sv[1].X, sv[1].Y,
				sv[2].X, sv[2].Y,
				px, py,
			)

			// Check if inside triangle
			if bc.X < 0 || bc.Y < 0 || bc.Z < 0 {
				continue
			}

			// Interpolate depth
			z := bc.X*sv[0].Z + bc.Y*sv[1].Z + bc.Z*sv[2].Z

			// Z-buffer test
			if z >= r.getDepth(x, y) {
				continue
			}

			// Interpolate lit vertex colors (Gouraud shading)
			color := interpolateColor3(sv[0].Color, sv[1].Color, sv[2].Color, bc)

			// Set pixel
			r.setDepth(x, y, z)
			r.fb.SetPixel(x, y, color)
		}
	}
}

// DrawTriangleTexturedGouraud rasterizes a textured triangle with Gouraud shading.
// Per-vertex lighting is calculated and interpolated, then modulated with texture.
func (r *Rasterizer) DrawTriangleTexturedGouraud(tri Triangle, tex *Texture, lightDir math3d.Vec3) {
	// Transform vertices to screen space
	var sv [3]screenVertex
	var vertexIntensity [3]float64 // Store lighting intensity per vertex
	allBehind := true

	viewProj := r.camera.ViewProjectionMatrix()
	normLight := lightDir.Normalize()

	for i := range 3 {
		// Transform to clip space
		clipPos := viewProj.MulVec4(math3d.V4FromV3(tri.V[i].Position, 1))

		// Check if behind camera
		if clipPos.W > 0 {
			allBehind = false
		}

		// Perspective divide
		if clipPos.W != 0 {
			sv[i].X = clipPos.X / clipPos.W
			sv[i].Y = clipPos.Y / clipPos.W
			sv[i].Z = clipPos.Z / clipPos.W
		}
		sv[i].W = clipPos.W

		// NDC to screen coordinates
		sv[i].X = (sv[i].X + 1) * 0.5 * float64(r.Width())
		sv[i].Y = (1 - sv[i].Y) * 0.5 * float64(r.Height()) // Y flipped

		// Calculate per-vertex lighting intensity
		intensity := math.Max(0, tri.V[i].Normal.Dot(normLight))
		vertexIntensity[i] = 0.3 + 0.7*intensity // Ambient + diffuse

		// Copy other attributes
		sv[i].Color = tri.V[i].Color
		sv[i].Normal = tri.V[i].Normal
		sv[i].UV = tri.V[i].UV
	}

	// Skip if entirely behind camera
	if allBehind {
		return
	}

	// Backface culling (using screen-space winding)
	edge1 := math3d.V2(sv[1].X-sv[0].X, sv[1].Y-sv[0].Y)
	edge2 := math3d.V2(sv[2].X-sv[0].X, sv[2].Y-sv[0].Y)
	cross := edge1.X*edge2.Y - edge1.Y*edge2.X
	if cross < 0 {
		return // Back-facing
	}

	// Find bounding box
	minX := int(math.Max(0, math.Floor(min3(sv[0].X, sv[1].X, sv[2].X))))
	maxX := int(math.Min(float64(r.Width()-1), math.Ceil(max3(sv[0].X, sv[1].X, sv[2].X))))
	minY := int(math.Max(0, math.Floor(min3(sv[0].Y, sv[1].Y, sv[2].Y))))
	maxY := int(math.Min(float64(r.Height()-1), math.Ceil(max3(sv[0].Y, sv[1].Y, sv[2].Y))))

	// Precompute perspective-correct interpolation factors (1/w for each vertex)
	var invW [3]float64
	for i := range 3 {
		if sv[i].W != 0 {
			invW[i] = 1.0 / sv[i].W
		} else {
			invW[i] = 0
		}
	}

	// Rasterize using barycentric coordinates with perspective correction
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5

			// Calculate barycentric coordinates
			bc := barycentric(
				sv[0].X, sv[0].Y,
				sv[1].X, sv[1].Y,
				sv[2].X, sv[2].Y,
				px, py,
			)

			// Check if inside triangle
			if bc.X < 0 || bc.Y < 0 || bc.Z < 0 {
				continue
			}

			// Interpolate depth
			z := bc.X*sv[0].Z + bc.Y*sv[1].Z + bc.Z*sv[2].Z

			// Z-buffer test
			if z >= r.getDepth(x, y) {
				continue
			}

			// Perspective-correct interpolation
			w0, w1, w2 := bc.X*invW[0], bc.Y*invW[1], bc.Z*invW[2]
			oneOverW := w0 + w1 + w2
			if oneOverW == 0 {
				continue
			}

			// Perspective-correct UV interpolation
			u := (w0*sv[0].UV.X + w1*sv[1].UV.X + w2*sv[2].UV.X) / oneOverW
			v := (w0*sv[0].UV.Y + w1*sv[1].UV.Y + w2*sv[2].UV.Y) / oneOverW

			// Perspective-correct lighting intensity interpolation
			intensity := (w0*vertexIntensity[0] + w1*vertexIntensity[1] + w2*vertexIntensity[2]) / oneOverW

			// Sample texture
			texColor := tex.Sample(u, v)

			// Apply interpolated lighting (Gouraud)
			litColor := MultiplyColor(texColor, intensity)

			// Set pixel
			r.setDepth(x, y, z)
			r.fb.SetPixel(x, y, litColor)
		}
	}
}

// MeshRenderer is imported from models to avoid circular deps.
// This interface allows drawing meshes without importing the models package.
type MeshRenderer interface {
	VertexCount() int
	TriangleCount() int
	GetVertex(i int) (pos, normal math3d.Vec3, uv math3d.Vec2)
	GetFace(i int) [3]int
}

// BoundedMeshRenderer extends MeshRenderer with bounding box support for frustum culling.
type BoundedMeshRenderer interface {
	MeshRenderer
	GetBounds() (min, max math3d.Vec3)
}

// tryFrustumCull attempts to cull a mesh using its bounds if available.
// Returns true if the mesh should be culled (not visible).
func (r *Rasterizer) tryFrustumCull(mesh MeshRenderer, transform math3d.Mat4) bool {
	// Check if mesh supports bounds for frustum culling
	bounded, ok := mesh.(BoundedMeshRenderer)
	if !ok {
		// No bounds available, can't cull
		return false
	}

	r.CullingStats.MeshesTested++

	// Get local bounds and transform to world space
	minBounds, maxBounds := bounded.GetBounds()
	localBounds := AABB{Min: minBounds, Max: maxBounds}

	// Check if visible
	if !r.IsVisibleTransformed(localBounds, transform) {
		r.CullingStats.MeshesCulled++
		return true
	}

	r.CullingStats.MeshesDrawn++
	return false
}

// DrawMesh renders a mesh with the given transform and color.
// Automatically performs frustum culling if the mesh provides bounds.
func (r *Rasterizer) DrawMesh(mesh MeshRenderer, transform math3d.Mat4, color Color, lightDir math3d.Vec3) {
	// Frustum culling check
	if r.tryFrustumCull(mesh, transform) {
		return
	}

	// Transform light to local space
	invTransform := transform.Inverse()
	localLight := invTransform.MulVec3Dir(lightDir).Normalize()

	for i := 0; i < mesh.TriangleCount(); i++ {
		face := mesh.GetFace(i)

		// Get vertices
		p0, _, _ := mesh.GetVertex(face[0])
		p1, _, _ := mesh.GetVertex(face[1])
		p2, _, _ := mesh.GetVertex(face[2])

		// Transform to world space
		v0 := transform.MulVec3(p0)
		v1 := transform.MulVec3(p1)
		v2 := transform.MulVec3(p2)

		r.DrawTriangleLit(v0, v1, v2, color, localLight)
	}
}

// DrawMeshTextured renders a mesh with texture mapping.
// Automatically performs frustum culling if the mesh provides bounds.
func (r *Rasterizer) DrawMeshTextured(mesh MeshRenderer, transform math3d.Mat4, tex *Texture, lightDir math3d.Vec3) {
	// Frustum culling check
	if r.tryFrustumCull(mesh, transform) {
		return
	}

	for i := 0; i < mesh.TriangleCount(); i++ {
		face := mesh.GetFace(i)

		// Get vertices with all attributes
		p0, n0, uv0 := mesh.GetVertex(face[0])
		p1, n1, uv1 := mesh.GetVertex(face[1])
		p2, n2, uv2 := mesh.GetVertex(face[2])

		// Transform positions to world space
		v0 := transform.MulVec3(p0)
		v1 := transform.MulVec3(p1)
		v2 := transform.MulVec3(p2)

		// Transform normals (using rotation only, ignoring translation/scale for now)
		wn0 := transform.MulVec3Dir(n0).Normalize()
		wn1 := transform.MulVec3Dir(n1).Normalize()
		wn2 := transform.MulVec3Dir(n2).Normalize()

		// Build triangle with all attributes
		tri := Triangle{
			V: [3]Vertex{
				{Position: v0, Normal: wn0, UV: uv0, Color: RGB(255, 255, 255)},
				{Position: v1, Normal: wn1, UV: uv1, Color: RGB(255, 255, 255)},
				{Position: v2, Normal: wn2, UV: uv2, Color: RGB(255, 255, 255)},
			},
		}

		r.DrawTriangleTextured(tri, tex, lightDir)
	}
}

// DrawMeshGouraud renders a mesh with Gouraud shading (per-vertex lighting).
// This produces smoother shading than flat shading by interpolating lighting across triangles.
// Automatically performs frustum culling if the mesh provides bounds.
func (r *Rasterizer) DrawMeshGouraud(mesh MeshRenderer, transform math3d.Mat4, color Color, lightDir math3d.Vec3) {
	// Frustum culling check
	if r.tryFrustumCull(mesh, transform) {
		return
	}

	for i := 0; i < mesh.TriangleCount(); i++ {
		face := mesh.GetFace(i)

		// Get vertices with all attributes
		p0, n0, _ := mesh.GetVertex(face[0])
		p1, n1, _ := mesh.GetVertex(face[1])
		p2, n2, _ := mesh.GetVertex(face[2])

		// Transform positions to world space
		v0 := transform.MulVec3(p0)
		v1 := transform.MulVec3(p1)
		v2 := transform.MulVec3(p2)

		// Transform normals
		wn0 := transform.MulVec3Dir(n0).Normalize()
		wn1 := transform.MulVec3Dir(n1).Normalize()
		wn2 := transform.MulVec3Dir(n2).Normalize()

		// Build triangle with per-vertex normals for Gouraud
		tri := Triangle{
			V: [3]Vertex{
				{Position: v0, Normal: wn0, Color: color},
				{Position: v1, Normal: wn1, Color: color},
				{Position: v2, Normal: wn2, Color: color},
			},
		}

		r.DrawTriangleGouraud(tri, lightDir)
	}
}

// DrawMeshTexturedGouraud renders a mesh with texture mapping and Gouraud shading.
// Combines perspective-correct texture mapping with smooth per-vertex lighting.
// Automatically performs frustum culling if the mesh provides bounds.
func (r *Rasterizer) DrawMeshTexturedGouraud(mesh MeshRenderer, transform math3d.Mat4, tex *Texture, lightDir math3d.Vec3) {
	// Frustum culling check
	if r.tryFrustumCull(mesh, transform) {
		return
	}

	for i := 0; i < mesh.TriangleCount(); i++ {
		face := mesh.GetFace(i)

		// Get vertices with all attributes
		p0, n0, uv0 := mesh.GetVertex(face[0])
		p1, n1, uv1 := mesh.GetVertex(face[1])
		p2, n2, uv2 := mesh.GetVertex(face[2])

		// Transform positions to world space
		v0 := transform.MulVec3(p0)
		v1 := transform.MulVec3(p1)
		v2 := transform.MulVec3(p2)

		// Transform normals
		wn0 := transform.MulVec3Dir(n0).Normalize()
		wn1 := transform.MulVec3Dir(n1).Normalize()
		wn2 := transform.MulVec3Dir(n2).Normalize()

		// Build triangle with all attributes
		tri := Triangle{
			V: [3]Vertex{
				{Position: v0, Normal: wn0, UV: uv0, Color: RGB(255, 255, 255)},
				{Position: v1, Normal: wn1, UV: uv1, Color: RGB(255, 255, 255)},
				{Position: v2, Normal: wn2, UV: uv2, Color: RGB(255, 255, 255)},
			},
		}

		r.DrawTriangleTexturedGouraud(tri, tex, lightDir)
	}
}

// DrawMeshGouraudCulled renders a mesh with Gouraud shading, with frustum culling.
// localBounds should be the mesh's local-space bounding box (e.g., mesh.BoundsMin/Max).
// Returns true if the mesh was drawn, false if it was culled.
func (r *Rasterizer) DrawMeshGouraudCulled(mesh MeshRenderer, transform math3d.Mat4, localBounds AABB, color Color, lightDir math3d.Vec3) bool {
	r.CullingStats.MeshesTested++

	// Transform bounds to world space and test against frustum
	if !r.IsVisibleTransformed(localBounds, transform) {
		r.CullingStats.MeshesCulled++
		return false
	}

	r.CullingStats.MeshesDrawn++
	r.DrawMeshGouraud(mesh, transform, color, lightDir)
	return true
}

// DrawMeshTexturedGouraudCulled renders a textured mesh with Gouraud shading, with frustum culling.
// localBounds should be the mesh's local-space bounding box (e.g., mesh.BoundsMin/Max).
// Returns true if the mesh was drawn, false if it was culled.
func (r *Rasterizer) DrawMeshTexturedGouraudCulled(mesh MeshRenderer, transform math3d.Mat4, localBounds AABB, tex *Texture, lightDir math3d.Vec3) bool {
	r.CullingStats.MeshesTested++

	// Transform bounds to world space and test against frustum
	if !r.IsVisibleTransformed(localBounds, transform) {
		r.CullingStats.MeshesCulled++
		return false
	}

	r.CullingStats.MeshesDrawn++
	r.DrawMeshTexturedGouraud(mesh, transform, tex, lightDir)
	return true
}

// DrawMeshCulled renders a mesh with frustum culling.
// Returns true if the mesh was drawn, false if it was culled.
func (r *Rasterizer) DrawMeshCulled(mesh MeshRenderer, transform math3d.Mat4, localBounds AABB, color Color, lightDir math3d.Vec3) bool {
	r.CullingStats.MeshesTested++

	if !r.IsVisibleTransformed(localBounds, transform) {
		r.CullingStats.MeshesCulled++
		return false
	}

	r.CullingStats.MeshesDrawn++
	r.DrawMesh(mesh, transform, color, lightDir)
	return true
}

// DrawMeshTexturedCulled renders a textured mesh with frustum culling.
// Returns true if the mesh was drawn, false if it was culled.
func (r *Rasterizer) DrawMeshTexturedCulled(mesh MeshRenderer, transform math3d.Mat4, localBounds AABB, tex *Texture, lightDir math3d.Vec3) bool {
	r.CullingStats.MeshesTested++

	if !r.IsVisibleTransformed(localBounds, transform) {
		r.CullingStats.MeshesCulled++
		return false
	}

	r.CullingStats.MeshesDrawn++
	r.DrawMeshTextured(mesh, transform, tex, lightDir)
	return true
}

// DrawMeshWireframe renders a mesh as wireframe.
// Automatically performs frustum culling if the mesh provides bounds.
func (r *Rasterizer) DrawMeshWireframe(mesh MeshRenderer, transform math3d.Mat4, color Color) {
	// Frustum culling check
	if r.tryFrustumCull(mesh, transform) {
		return
	}

	for i := 0; i < mesh.TriangleCount(); i++ {
		face := mesh.GetFace(i)

		p0, _, _ := mesh.GetVertex(face[0])
		p1, _, _ := mesh.GetVertex(face[1])
		p2, _, _ := mesh.GetVertex(face[2])

		v0 := transform.MulVec3(p0)
		v1 := transform.MulVec3(p1)
		v2 := transform.MulVec3(p2)

		// Project and draw lines (using framebuffer directly for now)
		r.drawLine3D(v0, v1, color)
		r.drawLine3D(v1, v2, color)
		r.drawLine3D(v2, v0, color)
	}
}

// drawLine3D draws a 3D line (projected to screen).
func (r *Rasterizer) drawLine3D(a, b math3d.Vec3, color Color) {
	viewProj := r.camera.ViewProjectionMatrix()

	// Transform to clip space
	clipA := viewProj.MulVec4(math3d.V4FromV3(a, 1))
	clipB := viewProj.MulVec4(math3d.V4FromV3(b, 1))

	// Skip if both behind camera
	if clipA.W <= 0 && clipB.W <= 0 {
		return
	}

	// Perspective divide and NDC to screen
	if clipA.W > 0 {
		clipA.X /= clipA.W
		clipA.Y /= clipA.W
	}
	if clipB.W > 0 {
		clipB.X /= clipB.W
		clipB.Y /= clipB.W
	}

	x0 := int((clipA.X + 1) * 0.5 * float64(r.Width()))
	y0 := int((1 - clipA.Y) * 0.5 * float64(r.Height()))
	x1 := int((clipB.X + 1) * 0.5 * float64(r.Width()))
	y1 := int((1 - clipB.Y) * 0.5 * float64(r.Height()))

	r.fb.DrawLine(x0, y0, x1, y1, color)
}
