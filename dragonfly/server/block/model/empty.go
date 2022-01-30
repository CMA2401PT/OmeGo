package model

import (
	"main.go/dragonfly/server/block/cube"
	"main.go/dragonfly/server/entity/physics"
	"main.go/dragonfly/server/world"
)

// Empty is a model that is completely empty. It has no collision boxes or solid faces.
type Empty struct{}

// AABB ...
func (Empty) AABB(cube.Pos, *world.World) []physics.AABB {
	return nil
}

// FaceSolid ...
func (Empty) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return false
}
