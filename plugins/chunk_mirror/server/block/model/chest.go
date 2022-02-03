package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// Chest is the model of a chest. It is just barely not a full block, having a slightly reduced with on all
// axes.
type Chest struct{}

// AABB returns a physics.AABB that is slightly smaller than a full block.
func (Chest) AABB(cube.Pos, *world.World) []physics.AABB {
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{0.025, 0, 0.025}, mgl64.Vec3{0.975, 0.95, 0.975})}
}

// FaceSolid always returns false.
func (Chest) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return false
}
