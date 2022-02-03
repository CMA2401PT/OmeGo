package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// CocoaBean is a model used by cocoa bean blocks.
type CocoaBean struct {
	// Facing is the face that the cocoa bean faces. It is the opposite of the face that the CocoaBean is attached to.
	Facing cube.Direction
	// Age is the age of the CocoaBean. The age influences the size of the CocoaBean. The maximum age value of a cocoa
	// bean is 3.
	Age int
}

// AABB returns a single physics.AABB whose size depends on the age of the CocoaBean.
func (c CocoaBean) AABB(cube.Pos, *world.World) []physics.AABB {
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 1, 1}).
		Stretch(c.Facing.RotateRight().Face().Axis(), -(6-float64(c.Age))/16).
		ExtendTowards(cube.FaceDown, -0.25).
		ExtendTowards(cube.FaceUp, -((7-float64(c.Age)*2)/16)).
		ExtendTowards(c.Facing.Face(), -0.0625).
		ExtendTowards(c.Facing.Opposite().Face(), -((11 - float64(c.Age)*2) / 16))}
}

// FaceSolid always returns false.
func (c CocoaBean) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return false
}
