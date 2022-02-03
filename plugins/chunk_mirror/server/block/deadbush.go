package block

import (
	"math/rand"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/item"
	"main.go/plugins/chunk_mirror/server/world"
)

// DeadBush is a transparent block in the form of an aesthetic plant.
type DeadBush struct {
	empty
	replaceable
	transparent
}

// NeighbourUpdateTick ...
func (d DeadBush) NeighbourUpdateTick(pos, _ cube.Pos, w *world.World) {
	if !supportsVegetation(d, w.Block(pos.Side(cube.FaceDown))) {
		w.BreakBlock(pos)
	}
}

// UseOnBlock ...
func (d DeadBush) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, w *world.World, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(w, pos, face, d)
	if !used {
		return false
	}
	if !supportsVegetation(d, w.Block(pos.Side(cube.FaceDown))) {
		return false
	}

	place(w, pos, d, user, ctx)
	return placed(ctx)
}

// CanDisplace ...
func (d DeadBush) CanDisplace(b world.Liquid) bool {
	_, ok := b.(Water)
	return ok
}

// SideClosed ...
func (d DeadBush) SideClosed(cube.Pos, cube.Pos, *world.World) bool {
	return false
}

// HasLiquidDrops ...
func (d DeadBush) HasLiquidDrops() bool {
	return true
}

// FlammabilityInfo ...
func (d DeadBush) FlammabilityInfo() FlammabilityInfo {
	return newFlammabilityInfo(60, 100, true)
}

// BreakInfo ...
func (d DeadBush) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, func(t item.Tool, enchantments []item.Enchantment) []item.Stack {
		if t.ToolType() == item.TypeShears {
			return []item.Stack{item.NewStack(d, 1)}
		}
		if amount := rand.Intn(3); amount != 0 {
			return []item.Stack{item.NewStack(item.Stick{}, amount)}
		}
		return nil
	})
}

// EncodeItem ...
func (d DeadBush) EncodeItem() (name string, meta int16) {
	return "minecraft:deadbush", 0
}

// EncodeBlock ...
func (d DeadBush) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:deadbush", nil
}
