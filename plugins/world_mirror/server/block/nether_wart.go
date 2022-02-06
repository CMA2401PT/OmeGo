package block

import (
	"math/rand"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/item"
	"main.go/plugins/world_mirror/server/world"
)

// NetherWart is a fungus found in the Nether that is vital in the creation of potions.
type NetherWart struct {
	transparent
	empty

	// Age is the age of the nether wart block. 3 is fully grown.
	Age int
}

// HasLiquidDrops ...
func (n NetherWart) HasLiquidDrops() bool {
	return true
}

// RandomTick ...
func (n NetherWart) RandomTick(pos cube.Pos, w *world.World, r *rand.Rand) {
	if n.Age < 3 && r.Float64() < 0.1 {
		n.Age++
		w.PlaceBlock(pos, n)
	}
}

// UseOnBlock ...
func (n NetherWart) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, w *world.World, user item.User, ctx *item.UseContext) bool {
	pos, _, used := firstReplaceable(w, pos, face, n)
	if !used {
		return false
	}
	if _, ok := w.Block(pos.Side(cube.FaceDown)).(SoulSand); !ok {
		return false
	}

	place(w, pos, n, user, ctx)
	return placed(ctx)
}

// NeighbourUpdateTick ...
func (n NetherWart) NeighbourUpdateTick(pos, _ cube.Pos, w *world.World) {
	if _, ok := w.Block(pos.Side(cube.FaceDown)).(SoulSand); !ok {
		w.BreakBlockWithoutParticles(pos)
	}
}

// BreakInfo ...
func (n NetherWart) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, func(item.Tool, []item.Enchantment) []item.Stack {
		if n.Age == 3 {
			return []item.Stack{item.NewStack(n, rand.Intn(3)+2)}
		}
		return []item.Stack{item.NewStack(n, 1)}
	})
}

// EncodeItem ...
func (NetherWart) EncodeItem() (name string, meta int16) {
	return "minecraft:nether_wart", 0
}

// EncodeBlock ...
func (n NetherWart) EncodeBlock() (name string, properties map[string]interface{}) {
	return "minecraft:nether_wart", map[string]interface{}{"age": int32(n.Age)}
}

// allNetherWart ...
func allNetherWart() (wart []world.Block) {
	for i := 0; i < 4; i++ {
		wart = append(wart, NetherWart{Age: i})
	}
	return
}
