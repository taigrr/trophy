// Package render provides optimized rasterization routines using edge function rasterization.
// Uses incremental updates to avoid recomputing barycentric coordinates per pixel.
package render

import (
	"math"

	"github.com/taigrr/trophy/pkg/math3d"
)

// edgeFunction computes the signed area of the parallelogram formed by vertices.
// Positive = left of edge, negative = right of edge, zero = on edge.
// Returns A, B, C coefficients for: edge(x,y) = A*x + B*y + C
func edgeCoeffs(x0, y0, x1, y1 float64) (A, B, C float64) {
	A = y0 - y1 // dy
	B = x1 - x0 // -dx
	C = x0*y1 - x1*y0
	return
}

// edgeFunc evaluates edge function at point (x, y)
func edgeFunc(A, B, C, x, y float64) float64 {
	return A*x + B*y + C
}

// DrawTriangleGouraudOpt is an optimized version using edge functions with incremental updates.
func (r *Rasterizer) DrawTriangleGouraudOpt(tri Triangle, lightDir math3d.Vec3) {
	// Transform vertices to screen space
	var sv [3]screenVertex
	allBehind := true

	viewProj := r.camera.ViewProjectionMatrix()
	normLight := lightDir.Normalize()

	for i := range 3 {
		clipPos := viewProj.MulVec4(math3d.V4FromV3(tri.V[i].Position, 1))

		if clipPos.W > 0 {
			allBehind = false
		}

		if clipPos.W != 0 {
			invW := 1.0 / clipPos.W
			sv[i].X = clipPos.X * invW
			sv[i].Y = clipPos.Y * invW
			sv[i].Z = clipPos.Z * invW
		}
		sv[i].W = clipPos.W

		// NDC to screen coordinates
		sv[i].X = (sv[i].X + 1) * 0.5 * float64(r.Width())
		sv[i].Y = (1 - sv[i].Y) * 0.5 * float64(r.Height())

		// Per-vertex lighting
		intensity := math.Max(0, tri.V[i].Normal.Dot(normLight))
		intensity = 0.3 + 0.7*intensity

		sv[i].Color = RGB(
			uint8(float64(tri.V[i].Color.R)*intensity),
			uint8(float64(tri.V[i].Color.G)*intensity),
			uint8(float64(tri.V[i].Color.B)*intensity),
		)
	}

	if allBehind {
		return
	}

	// Backface culling
	edge1X := sv[1].X - sv[0].X
	edge1Y := sv[1].Y - sv[0].Y
	edge2X := sv[2].X - sv[0].X
	edge2Y := sv[2].Y - sv[0].Y
	cross := edge1X*edge2Y - edge1Y*edge2X
	if cross < 0 && !r.DisableBackfaceCulling {
		return
	}

	// Bounding box (clamped to screen)
	minX := int(math.Max(0, math.Floor(min3(sv[0].X, sv[1].X, sv[2].X))))
	maxX := int(math.Min(float64(r.Width()-1), math.Ceil(max3(sv[0].X, sv[1].X, sv[2].X))))
	minY := int(math.Max(0, math.Floor(min3(sv[0].Y, sv[1].Y, sv[2].Y))))
	maxY := int(math.Min(float64(r.Height()-1), math.Ceil(max3(sv[0].Y, sv[1].Y, sv[2].Y))))

	if minX > maxX || minY > maxY {
		return
	}

	// Compute edge function coefficients for each edge
	// Edge 0: v1 -> v2, Edge 1: v2 -> v0, Edge 2: v0 -> v1
	A0, B0, C0 := edgeCoeffs(sv[1].X, sv[1].Y, sv[2].X, sv[2].Y)
	A1, B1, C1 := edgeCoeffs(sv[2].X, sv[2].Y, sv[0].X, sv[0].Y)
	A2, B2, C2 := edgeCoeffs(sv[0].X, sv[0].Y, sv[1].X, sv[1].Y)

	// Normalize for barycentric: divide by 2*triangle_area
	area2 := cross // This is 2 * signed area
	if area2 == 0 {
		return
	}
	invArea := 1.0 / area2

	// Pre-compute depth deltas
	dZ0 := sv[0].Z
	dZ1 := sv[1].Z
	dZ2 := sv[2].Z

	// Pre-compute color components
	r0, g0, b0 := float64(sv[0].Color.R), float64(sv[0].Color.G), float64(sv[0].Color.B)
	r1, g1, b1 := float64(sv[1].Color.R), float64(sv[1].Color.G), float64(sv[1].Color.B)
	r2, g2, b2 := float64(sv[2].Color.R), float64(sv[2].Color.G), float64(sv[2].Color.B)

	// Evaluate edge functions at top-left corner of bounding box
	px := float64(minX) + 0.5
	py := float64(minY) + 0.5

	w0Row := edgeFunc(A0, B0, C0, px, py)
	w1Row := edgeFunc(A1, B1, C1, px, py)
	w2Row := edgeFunc(A2, B2, C2, px, py)

	width := r.Width()
	zbuffer := r.zbuffer
	fb := r.fb

	// Rasterize using incremental edge functions
	for y := minY; y <= maxY; y++ {
		w0 := w0Row
		w1 := w1Row
		w2 := w2Row
		rowOffset := y * width

		for x := minX; x <= maxX; x++ {
			// Check if inside triangle (all edge functions >= 0)
			if w0 >= 0 && w1 >= 0 && w2 >= 0 {
				// Compute barycentric coordinates
				bc0 := w0 * invArea
				bc1 := w1 * invArea
				bc2 := w2 * invArea

				// Interpolate depth
				z := bc0*dZ0 + bc1*dZ1 + bc2*dZ2

				// Z-buffer test (no bounds check - we're within clamped bounds)
				idx := rowOffset + x
				if z < zbuffer[idx] {
					// Interpolate color
					cr := uint8(r0*bc0 + r1*bc1 + r2*bc2)
					cg := uint8(g0*bc0 + g1*bc1 + g2*bc2)
					cb := uint8(b0*bc0 + b1*bc1 + b2*bc2)

					zbuffer[idx] = z
					fb.SetPixel(x, y, RGB(cr, cg, cb))
				}
			}

			// Step in X direction
			w0 += A0
			w1 += A1
			w2 += A2
		}

		// Step in Y direction
		w0Row += B0
		w1Row += B1
		w2Row += B2
	}
}

// DrawMeshGouraudOpt renders a mesh with optimized Gouraud shading.
func (r *Rasterizer) DrawMeshGouraudOpt(mesh MeshRenderer, transform math3d.Mat4, color Color, lightDir math3d.Vec3) {
	if r.tryFrustumCull(mesh, transform) {
		return
	}

	for i := 0; i < mesh.TriangleCount(); i++ {
		face := mesh.GetFace(i)

		p0, n0, _ := mesh.GetVertex(face[0])
		p1, n1, _ := mesh.GetVertex(face[1])
		p2, n2, _ := mesh.GetVertex(face[2])

		v0 := transform.MulVec3(p0)
		v1 := transform.MulVec3(p1)
		v2 := transform.MulVec3(p2)

		wn0 := transform.MulVec3Dir(n0).Normalize()
		wn1 := transform.MulVec3Dir(n1).Normalize()
		wn2 := transform.MulVec3Dir(n2).Normalize()

		tri := Triangle{
			V: [3]Vertex{
				{Position: v0, Normal: wn0, Color: color},
				{Position: v1, Normal: wn1, Color: color},
				{Position: v2, Normal: wn2, Color: color},
			},
		}

		r.DrawTriangleGouraudOpt(tri, lightDir)
	}
}

// DrawTriangleTexturedOpt is an optimized textured triangle rasterizer with Gouraud shading.
func (r *Rasterizer) DrawTriangleTexturedOpt(tri Triangle, tex *Texture, lightDir math3d.Vec3) {
	var sv [3]screenVertex
	var vertexIntensity [3]float64
	allBehind := true

	viewProj := r.camera.ViewProjectionMatrix()
	normLight := lightDir.Normalize()

	for i := range 3 {
		clipPos := viewProj.MulVec4(math3d.V4FromV3(tri.V[i].Position, 1))

		if clipPos.W > 0 {
			allBehind = false
		}

		if clipPos.W != 0 {
			invW := 1.0 / clipPos.W
			sv[i].X = clipPos.X * invW
			sv[i].Y = clipPos.Y * invW
			sv[i].Z = clipPos.Z * invW
		}
		sv[i].W = clipPos.W

		sv[i].X = (sv[i].X + 1) * 0.5 * float64(r.Width())
		sv[i].Y = (1 - sv[i].Y) * 0.5 * float64(r.Height())
		sv[i].UV = tri.V[i].UV

		// Per-vertex lighting (Gouraud)
		intensity := math.Max(0, tri.V[i].Normal.Dot(normLight))
		vertexIntensity[i] = 0.3 + 0.7*intensity
	}

	if allBehind {
		return
	}

	// Backface culling
	edge1X := sv[1].X - sv[0].X
	edge1Y := sv[1].Y - sv[0].Y
	edge2X := sv[2].X - sv[0].X
	edge2Y := sv[2].Y - sv[0].Y
	cross := edge1X*edge2Y - edge1Y*edge2X
	if cross < 0 && !r.DisableBackfaceCulling {
		return
	}

	minX := int(math.Max(0, math.Floor(min3(sv[0].X, sv[1].X, sv[2].X))))
	maxX := int(math.Min(float64(r.Width()-1), math.Ceil(max3(sv[0].X, sv[1].X, sv[2].X))))
	minY := int(math.Max(0, math.Floor(min3(sv[0].Y, sv[1].Y, sv[2].Y))))
	maxY := int(math.Min(float64(r.Height()-1), math.Ceil(max3(sv[0].Y, sv[1].Y, sv[2].Y))))

	if minX > maxX || minY > maxY {
		return
	}

	// Edge coefficients
	A0, B0, C0 := edgeCoeffs(sv[1].X, sv[1].Y, sv[2].X, sv[2].Y)
	A1, B1, C1 := edgeCoeffs(sv[2].X, sv[2].Y, sv[0].X, sv[0].Y)
	A2, B2, C2 := edgeCoeffs(sv[0].X, sv[0].Y, sv[1].X, sv[1].Y)

	area2 := cross
	if area2 == 0 {
		return
	}
	invArea := 1.0 / area2

	// Perspective-correct interpolation: precompute 1/W
	var invW [3]float64
	for i := range 3 {
		if sv[i].W != 0 {
			invW[i] = 1.0 / sv[i].W
		}
	}

	px := float64(minX) + 0.5
	py := float64(minY) + 0.5

	w0Row := edgeFunc(A0, B0, C0, px, py)
	w1Row := edgeFunc(A1, B1, C1, px, py)
	w2Row := edgeFunc(A2, B2, C2, px, py)

	width := r.Width()
	zbuffer := r.zbuffer
	fb := r.fb

	for y := minY; y <= maxY; y++ {
		w0 := w0Row
		w1 := w1Row
		w2 := w2Row
		rowOffset := y * width

		for x := minX; x <= maxX; x++ {
			if w0 >= 0 && w1 >= 0 && w2 >= 0 {
				bc0 := w0 * invArea
				bc1 := w1 * invArea
				bc2 := w2 * invArea

				z := bc0*sv[0].Z + bc1*sv[1].Z + bc2*sv[2].Z

				idx := rowOffset + x
				if idx < len(zbuffer) && z < zbuffer[idx] {
					// Perspective-correct interpolation
					pw0 := bc0 * invW[0]
					pw1 := bc1 * invW[1]
					pw2 := bc2 * invW[2]
					oneOverW := pw0 + pw1 + pw2
					if oneOverW != 0 {
						invOneOverW := 1.0 / oneOverW
						u := (pw0*sv[0].UV.X + pw1*sv[1].UV.X + pw2*sv[2].UV.X) * invOneOverW
						v := (pw0*sv[0].UV.Y + pw1*sv[1].UV.Y + pw2*sv[2].UV.Y) * invOneOverW

						// Perspective-correct lighting intensity
						intensity := (pw0*vertexIntensity[0] + pw1*vertexIntensity[1] + pw2*vertexIntensity[2]) * invOneOverW

						texColor := tex.Sample(u, v)
						litColor := MultiplyColor(texColor, intensity)

						zbuffer[idx] = z
						fb.SetPixel(x, y, litColor)
					}
				}
			}

			w0 += A0
			w1 += A1
			w2 += A2
		}

		w0Row += B0
		w1Row += B1
		w2Row += B2
	}
}

// DrawMeshTexturedOpt renders a textured mesh with optimized rasterization.
func (r *Rasterizer) DrawMeshTexturedOpt(mesh MeshRenderer, transform math3d.Mat4, tex *Texture, lightDir math3d.Vec3) {
	if r.tryFrustumCull(mesh, transform) {
		return
	}

	for i := 0; i < mesh.TriangleCount(); i++ {
		face := mesh.GetFace(i)

		p0, n0, uv0 := mesh.GetVertex(face[0])
		p1, n1, uv1 := mesh.GetVertex(face[1])
		p2, n2, uv2 := mesh.GetVertex(face[2])

		v0 := transform.MulVec3(p0)
		v1 := transform.MulVec3(p1)
		v2 := transform.MulVec3(p2)

		wn0 := transform.MulVec3Dir(n0).Normalize()
		wn1 := transform.MulVec3Dir(n1).Normalize()
		wn2 := transform.MulVec3Dir(n2).Normalize()

		tri := Triangle{
			V: [3]Vertex{
				{Position: v0, Normal: wn0, UV: uv0, Color: RGB(255, 255, 255)},
				{Position: v1, Normal: wn1, UV: uv1, Color: RGB(255, 255, 255)},
				{Position: v2, Normal: wn2, UV: uv2, Color: RGB(255, 255, 255)},
			},
		}

		r.DrawTriangleTexturedOpt(tri, tex, lightDir)
	}
}
