package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// Leaves is a model for leaves-like blocks. These blocks have a full collision box, but none of their faces
// are solid.
type Leaves struct{}

// AABB returns a physics.AABB that spans a full block.
func (Leaves) AABB(cube.Pos, *world.World) []physics.AABB {
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 1, 1})}
}

// FaceSolid always returns false.
func (Leaves) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return false
}
