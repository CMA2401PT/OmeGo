package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// Door is a model used for doors. It has no solid faces and a bounding box that changes depending on
// the direction of the door, whether it is open, and the side of its hinge.
type Door struct {
	// Facing is the direction that the door is facing when closed.
	Facing cube.Direction
	// Open specifies if the Door is open. The direction it opens towards depends on the Right field.
	Open bool
	// Right specifies the attachment side of the door and, with that, the direction it opens in.
	Right bool
}

// AABB returns a physics.AABB that depends on if the Door is open, what direction it is facing and whether it is
// attached to the right/left side of a block.
func (d Door) AABB(cube.Pos, *world.World) []physics.AABB {
	if d.Open {
		if d.Right {
			return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 1, 1}).ExtendTowards(d.Facing.RotateLeft().Face(), -0.8125)}
		}
		return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 1, 1}).ExtendTowards(d.Facing.RotateRight().Face(), -0.8125)}
	}
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 1, 1}).ExtendTowards(d.Facing.Face(), -0.8125)}
}

// FaceSolid always returns false.
func (d Door) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return false
}
