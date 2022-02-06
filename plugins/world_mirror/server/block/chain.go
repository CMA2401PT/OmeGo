package block

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/block/model"
	"main.go/plugins/world_mirror/server/item"
	"main.go/plugins/world_mirror/server/world"
)

// Chain is a metallic decoration block.
type Chain struct {
	transparent

	// Axis is the axis which the chain faces.
	Axis cube.Axis
}

// CanDisplace ...
func (Chain) CanDisplace(b world.Liquid) bool {
	_, water := b.(Water)
	return water
}

// SideClosed ...
func (Chain) SideClosed(cube.Pos, cube.Pos, *world.World) bool {
	return false
}

// UseOnBlock ...
func (c Chain) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, w *world.World, user item.User, ctx *item.UseContext) (used bool) {
	pos, face, used = firstReplaceable(w, pos, face, c)
	if !used {
		return
	}
	c.Axis = face.Axis()

	place(w, pos, c, user, ctx)
	return placed(ctx)
}

// BreakInfo ...
func (c Chain) BreakInfo() BreakInfo {
	return newBreakInfo(5, pickaxeHarvestable, pickaxeEffective, oneOf(c))
}

// EncodeItem ...
func (Chain) EncodeItem() (name string, meta int16) {
	return "minecraft:chain", 0
}

// EncodeBlock ...
func (c Chain) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:chain", map[string]interface{}{"pillar_axis": c.Axis.String()}
}

// Model ...
func (c Chain) Model() world.BlockModel {
	return model.Chain{Axis: c.Axis}
}

// allChains ...
func allChains() (chains []world.Block) {
	for _, axis := range cube.Axes() {
		chains = append(chains, Chain{Axis: axis})
	}
	return
}
