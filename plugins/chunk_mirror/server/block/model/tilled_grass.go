package model

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// TilledGrass is a model used for grass that has been tilled in some way, such as dirt paths and farmland.
type TilledGrass struct{}

// AABB returns a physics.AABB that spans an entire block.
func (TilledGrass) AABB(cube.Pos, *world.World) []physics.AABB {
	// TODO: Make the max Y value 0.9375 once https://bugs.mojang.com/browse/MCPE-12109 gets fixed.
	return []physics.AABB{physics.NewAABB(mgl64.Vec3{}, mgl64.Vec3{1, 1, 1})}
}

// FaceSolid always returns true.
func (TilledGrass) FaceSolid(cube.Pos, cube.Face, *world.World) bool {
	return true
}
