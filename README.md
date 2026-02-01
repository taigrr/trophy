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

| Input        | Action         |
| ------------ | -------------- |
| Mouse drag   | Rotate model   |
| Scroll wheel | Zoom in/out    |
| W/S          | Pitch up/down  |
| A/D          | Yaw left/right |
| Q/E          | Roll           |
| Space        | Random spin    |
| +/-          | Zoom           |
| R            | Reset view     |
| Esc          | Quit           |

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

// Render
rasterizer.DrawMeshTexturedGouraud(mesh, transform, texture, lightDir)
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
| Mat4Mul        | 32.61 |    0 |         0 |
| Mat4MulVec4    |  2.81 |    0 |         0 |
| Mat4MulVec3    |  2.64 |    0 |         0 |
| Mat4Inverse    | 18.58 |    0 |         0 |
| Vec3Normalize  |  1.62 |    0 |         0 |
| Vec3Cross      |  1.58 |    0 |         0 |
| Vec3Dot        |  1.57 |    0 |         0 |
| Perspective    |  6.34 |    0 |         0 |
| LookAt         |  6.26 |    0 |         0 |
| ViewProjection | 32.96 |    0 |         0 |

### Rendering (pkg/render)

| Benchmark                  | ns/op | B/op | allocs/op |
| -------------------------- | ----: | ---: | --------: |
| FrustumExtract             | 37.02 |    0 |         0 |
| AABBIntersection (visible) |  9.89 |    0 |         0 |
| AABBIntersection (culled)  |  7.77 |    0 |         0 |
| TransformAABB              | 129.8 |    0 |         0 |
| FrustumIntersectAABB       |  7.34 |    0 |         0 |
| FrustumIntersectsSphere    |  4.91 |    0 |         0 |
| DrawTriangleGouraud        |  4696 |    0 |         0 |
| DrawMeshGouraud            | 55400 |    0 |         0 |
| DrawTriangleGouraudOpt     |  3975 |    0 |         0 |
| DrawMeshGouraudOpt         | 26654 |    0 |         0 |

### Culling Performance

| Benchmark                       |  ns/op | B/op | allocs/op |
| ------------------------------- | -----: | ---: | --------: |
| MeshRendering (with culling)    | 112726 |    0 |         0 |
| MeshRendering (without culling) | 139805 |    0 |         0 |

_Benchmarks run on Apple M4 Max, darwin/arm64_

## Credits

Built with [Ultraviolet](https://github.com/charmbracelet/ultraviolet) for terminal rendering.
