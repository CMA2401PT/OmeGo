package model

import (
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/entity/physics"
	"main.go/plugins/world_mirror/server/world"
)

// Empty is a model that is completely empty. It has no collision boxes or solid faces.
type Empty struct{}

// AABB returns an empty slice.
func (Empty) AABB(cube.Pos, *world.World) []physics.AABB {
	return nil
}

// FaceSolid always returns false.
func (Empty) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return false
}
