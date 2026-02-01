# Trophy 🏆

Terminal 3D Model Viewer - View OBJ and GLB files directly in your terminal.

![Trophy Demo](docs/demo.gif)

## Features

- **OBJ & GLB Support** - Load standard 3D model formats
- **Embedded Textures** - Automatically extracts and applies GLB textures
- **Interactive Controls** - Rotate, zoom, and spin models with mouse/keyboard
- **Software Rendering** - No GPU required, works over SSH
- **Springy Physics** - Smooth, satisfying rotation with momentum

## Installation

```bash
go install github.com/taigrr/trophy/cmd/trophy@latest
```

## Usage

```bash
trophy model.glb              # View a GLB model
trophy model.obj              # View an OBJ model
trophy -texture tex.png model.obj  # Apply custom texture
trophy -bg 0,0,0 model.glb    # Black background
trophy -fps 60 model.glb      # Higher framerate
```

## Controls

| Input        | Action             |
| ------------ | ------------------ |
| Mouse drag   | Rotate model       |
| Scroll wheel | Zoom in/out        |
| W/S          | Pitch up/down      |
| A/D          | Yaw left/right     |
| Q/E          | Roll               |
| Space        | Toggle spin mode   |
| +/-          | Zoom               |
| R            | Reset view         |
| T            | Toggle texture     |
| X            | Toggle wireframe   |
| L            | Position light     |
| ?            | Toggle HUD overlay |
| Esc          | Quit               |

## Lighting

Press `L` to enter lighting mode and drag to reposition the light source in real-time:

![Lighting Demo](docs/lighting-demo.gif)

## Library Usage

Trophy's rendering packages can be used as a library:

```go
import (
    "github.com/taigrr/trophy/pkg/models"
    "github.com/taigrr/trophy/pkg/render"
    "github.com/taigrr/trophy/pkg/math3d"
)

// Load a model
mesh, texture, _ := models.LoadGLBWithTexture("model.glb")

// Create renderer
fb := render.NewFramebuffer(320, 200)
camera := render.NewCamera()
rasterizer := render.NewRasterizer(camera, fb)

// Render (uses optimized edge-function rasterizer)
rasterizer.DrawMeshTexturedOpt(mesh, transform, texture, lightDir)
```

## Packages

- `pkg/math3d` - 3D math (Vec2, Vec3, Vec4, Mat4)
- `pkg/models` - Model loaders (OBJ, GLB/GLTF)
- `pkg/render` - Software rasterizer, camera, textures

## Benchmarks

Run with `go test -bench=. -benchmem ./...`

### Math (pkg/math3d)

| Benchmark      | ns/op | B/op | allocs/op |
| -------------- | ----: | ---: | --------: |
| Mat4Mul        | 74.86 |    0 |         0 |
| Mat4MulVec4    |  6.84 |    0 |         0 |
| Mat4MulVec3    |  8.24 |    0 |         0 |
| Mat4Inverse    | 62.13 |    0 |         0 |
| Vec3Normalize  |  7.41 |    0 |         0 |
| Vec3Cross      |  2.47 |    0 |         0 |
| Vec3Dot        |  2.48 |    0 |         0 |
| Perspective    | 24.15 |    0 |         0 |
| LookAt         | 33.66 |    0 |         0 |
| ViewProjection | 71.32 |    0 |         0 |

### Rendering (pkg/render)

| Benchmark                  |  ns/op | B/op | allocs/op |
| -------------------------- | -----: | ---: | --------: |
| FrustumExtract             |  90.31 |    0 |         0 |
| AABBIntersection (visible) |  25.67 |    0 |         0 |
| AABBIntersection (culled)  |  15.37 |    0 |         0 |
| TransformAABB              |  246.5 |    0 |         0 |
| FrustumIntersectAABB       |  20.84 |    0 |         0 |
| FrustumIntersectsSphere    |  12.35 |    0 |         0 |
| DrawTriangleGouraud        |  10175 |    0 |         0 |
| DrawTriangleGouraudOpt     |   8012 |    0 |         0 |
| DrawMeshGouraud            | 185440 |    0 |         0 |
| DrawMeshGouraudOpt         |  79676 |    0 |         0 |

### Culling Performance

| Benchmark                       |  ns/op | B/op | allocs/op |
| ------------------------------- | -----: | ---: | --------: |
| MeshRendering (with culling)    | 285282 |    0 |         0 |
| MeshRendering (without culling) | 362384 |    0 |         0 |

The optimized rasterizer (`*Opt` variants) uses incremental edge functions instead of per-pixel barycentric recomputation, yielding **~21% speedup** on triangles and **~57% speedup** on full mesh rendering.

_Benchmarks run on AMD EPYC 7642 48-Core, linux/amd64_

## Credits

Built with [Ultraviolet](https://github.com/charmbracelet/ultraviolet) for terminal rendering.
