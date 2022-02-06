package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/entity/physics"
	"main.go/plugins/world_mirror/server/world"
)

// Solid is the model of a fully solid block. Blocks with this model, such as stone or wooden planks, have a
// 1x1x1 collision box.
type Solid struct{}

// AABB returns a physics.AABB spanning a full block.
func (Solid) AABB(cube.Pos, *world.World) []physics.AABB {
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 1, 1})}
}

// FaceSolid always returns true.
func (Solid) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return true
}
