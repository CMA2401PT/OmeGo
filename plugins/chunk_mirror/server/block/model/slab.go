package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// Slab is the model of a slab-like block, which is either a half block or a full block, depending on if the
// slab is double.
type Slab struct {
	// Double and Top specify if the Slab is a double slab and if it's in the top slot respectively. If Double is true,
	// the AABB returned is always a full block.
	Double, Top bool
}

// AABB returns either a physics.AABB spanning a full block or a half block in the top/bottom part of the block,
// depending on the Double and Top fields.
func (s Slab) AABB(cube.Pos, *world.World) []physics.AABB {
	if s.Double {
		return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 1, 1})}
	}
	if s.Top {
		return []physics.AABB{physics.NewAABB(mgl64.Vec3{0, 0.5, 0}, mgl64.Vec3{1, 1, 1})}
	}
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 0.5, 1})}
}

// FaceSolid returns true if the Slab is double, or if the face is cube.FaceUp when the Top field is true, or if the
// face is cube.FaceDown when the Top field is false.
func (s Slab) FaceSolid(_ cube.Pos, face cube.Face, _ *world.World) bool {
	if s.Double {
		return true
	} else if s.Top {
		return face == cube.FaceUp
	}
	return face == cube.FaceDown
}
