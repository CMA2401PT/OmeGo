package block

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/item"
	"main.go/plugins/chunk_mirror/server/world"
)

// NetherSprouts are a non-solid plant block that generate in warped forests.
type NetherSprouts struct {
	transparent
	replaceable
	empty
}

// NeighbourUpdateTick ...
func (n NetherSprouts) NeighbourUpdateTick(pos, _ cube.Pos, w *world.World) {
	if !supportsVegetation(n, w.Block(pos.Side(cube.FaceDown))) {
		w.BreakBlock(pos) //TODO: Nylium & mycelium
	}
}

// UseOnBlock ...
func (n NetherSprouts) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, w *world.World, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(w, pos, face, n)
	if !used {
		return false
	}
	if !supportsVegetation(n, w.Block(pos.Side(cube.FaceDown))) {
		return false //TODO: Nylium & mycelium
	}

	place(w, pos, n, user, ctx)
	return placed(ctx)
}

// HasLiquidDrops ...
func (n NetherSprouts) HasLiquidDrops() bool {
	return false
}

// FlammabilityInfo ...
func (n NetherSprouts) FlammabilityInfo() FlammabilityInfo {
	return newFlammabilityInfo(0, 0, true)
}

// BreakInfo ...
func (n NetherSprouts) BreakInfo() BreakInfo {
	return newBreakInfo(0, func(t item.Tool) bool {
		return t.ToolType() == item.TypeShears
	}, nothingEffective, oneOf(n))
}

// EncodeItem ...
func (n NetherSprouts) EncodeItem() (name string, meta int16) {
	return "minecraft:nether_sprouts", 0
}

// EncodeBlock ...
func (n NetherSprouts) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:nether_sprouts", nil
}
