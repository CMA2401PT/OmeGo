package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// Chain is a model used by chain blocks.
type Chain struct {
	// Axis is the axis which the chain faces.
	Axis cube.Axis
}

// AABB ...
func (c Chain) AABB(cube.Pos, *world.World) []physics.AABB {
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{0.40625, 0.40625, 0.40625}, mgl64.Vec3{0.59375, 0.59375, 0.59375}).Stretch(c.Axis, 0.40625)}
}

// FaceSolid ...
func (Chain) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return false
}
