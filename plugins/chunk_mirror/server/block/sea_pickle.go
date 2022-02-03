package block

import (
	"math"
	"math/rand"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/item"
	"main.go/plugins/chunk_mirror/server/world"
)

// SeaPickle is a small stationary underwater block that emits light, and is typically found in colonies of up to
// four sea pickles.
type SeaPickle struct {
	empty
	transparent

	// AdditionalCount is the amount of additional sea pickles clustered together.
	AdditionalCount int
	// Dead is whether the sea pickles are not alive. Sea pickles are only considered alive when inside of water. While
	// alive, sea pickles emit light & can be grown with bone meal.
	Dead bool
}

// canSurvive ...
func (SeaPickle) canSurvive(pos cube.Pos, w *world.World) bool {
	below := w.Block(pos.Side(cube.FaceDown))
	if !below.Model().FaceSolid(pos.Side(cube.FaceDown), cube.FaceUp, w) {
		return false
	}
	if emitter, ok := below.(LightDiffuser); ok && emitter.LightDiffusionLevel() != 15 {
		return false
	}
	return true
}

// BoneMeal ...
func (s SeaPickle) BoneMeal(pos cube.Pos, w *world.World) bool {
	if s.Dead {
		return false
	}
	if coral, ok := w.Block(pos.Side(cube.FaceDown)).(CoralBlock); !ok || coral.Dead {
		return false
	}

	if s.AdditionalCount != 3 {
		s.AdditionalCount = 3
		w.PlaceBlock(pos, s)
	}

	for x := -2; x <= 2; x++ {
		distance := -int(math.Abs(float64(x))) + 2
		for z := -distance; z <= distance; z++ {
			for y := -1; y < 1; y++ {
				if (x == 0 && y == 0 && z == 0) || rand.Intn(6) != 0 {
					continue
				}
				newPos := pos.Add(cube.Pos{x, y, z})

				if _, ok := w.Block(newPos).(Water); !ok {
					continue
				}
				if coral, ok := w.Block(newPos.Side(cube.FaceDown)).(CoralBlock); !ok || coral.Dead {
					continue
				}
				w.PlaceBlock(newPos, SeaPickle{AdditionalCount: rand.Intn(3) + 1})
			}
		}
	}

	return true
}

// UseOnBlock ...
func (s SeaPickle) UseOnBlock(pos cube.Pos, face cube.Face, _ mgl64.Vec3, w *world.World, user item.User, ctx *item.UseContext) bool {
	if existing, ok := w.Block(pos).(SeaPickle); ok {
		if existing.AdditionalCount >= 3 {
			return false
		}

		existing.AdditionalCount++
		w.PlaceBlock(pos, existing)
		ctx.CountSub = 1
		return true
	}

	pos, _, used := firstReplaceable(w, pos, face, s)
	if !used {
		return false
	}
	if !s.canSurvive(pos, w) {
		return false
	}

	s.Dead = true
	if liquid, ok := w.Liquid(pos); ok {
		_, ok = liquid.(Water)
		s.Dead = !ok
	}

	place(w, pos, s, user, ctx)
	return placed(ctx)
}

// NeighbourUpdateTick ...
func (s SeaPickle) NeighbourUpdateTick(pos, _ cube.Pos, w *world.World) {
	if !s.canSurvive(pos, w) {
		w.BreakBlock(pos)
		return
	}

	alive := false
	if liquid, ok := w.Liquid(pos); ok {
		_, alive = liquid.(Water)
	}
	if s.Dead == alive {
		s.Dead = !alive
		w.PlaceBlock(pos, s)
	}
}

// HasLiquidDrops ...
func (SeaPickle) HasLiquidDrops() bool {
	return true
}

// CanDisplace ...
func (SeaPickle) CanDisplace(b world.Liquid) bool {
	_, ok := b.(Water)
	return ok
}

// SideClosed ...
func (SeaPickle) SideClosed(cube.Pos, cube.Pos, *world.World) bool {
	return false
}

// LightEmissionLevel ...
func (s SeaPickle) LightEmissionLevel() uint8 {
	if s.Dead {
		return 0
	}
	return uint8(6 + s.AdditionalCount*3)
}

// BreakInfo ...
func (s SeaPickle) BreakInfo() BreakInfo {
	return newBreakInfo(0, alwaysHarvestable, nothingEffective, simpleDrops(item.NewStack(s, s.AdditionalCount+1)))
}

// EncodeItem ...
func (SeaPickle) EncodeItem() (name string, meta int16) {
	return "minecraft:sea_pickle", 0
}

// EncodeBlock ...
func (s SeaPickle) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:sea_pickle", map[string]interface{}{"cluster_count": int32(s.AdditionalCount), "dead_bit": s.Dead}
}

// allSeaPickles ...
func allSeaPickles() (b []world.Block) {
	for i := 0; i <= 3; i++ {
		b = append(b, SeaPickle{AdditionalCount: i})
		b = append(b, SeaPickle{AdditionalCount: i, Dead: true})
	}
	return
}
