package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// Carpet is a model for carpet-like extremely thin blocks.
type Carpet struct{}

// AABB returns a flat AABB with a width of 0.0625.
func (Carpet) AABB(cube.Pos, *world.World) []physics.AABB {
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 0.0625, 1})}
}

// FaceSolid always returns false.
func (Carpet) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return false
}
