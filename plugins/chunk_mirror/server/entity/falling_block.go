package entity

import (
	"fmt"
	"math/rand"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/internal/nbtconv"
	"main.go/plugins/chunk_mirror/server/item"
	"main.go/plugins/chunk_mirror/server/world"
)

// FallingBlock is the entity form of a block that appears when a gravity-affected block loses its support.
type FallingBlock struct {
	transform
	block world.Block

	c *MovementComputer
}

// NewFallingBlock ...
func NewFallingBlock(block world.Block, pos mgl64.Vec3) *FallingBlock {
	b := &FallingBlock{block: block, c: &MovementComputer{
		Gravity:           0.04,
		DragBeforeGravity: true,
		Drag:              0.02,
	}}
	b.transform = newTransform(b, pos)
	return b
}

// Name ...
func (f *FallingBlock) Name() string {
	return fmt.Sprintf("%T", f.block)
}

// EncodeEntity ...
func (f *FallingBlock) EncodeEntity() string {
	return "minecraft:falling_block"
}

// AABB ...
func (f *FallingBlock) AABB() physics.AABB {
	return physics.NewAABB(mgl64.Vec3{-0.49, 0, -0.49}, mgl64.Vec3{0.49, 0.98, 0.49})
}

// Block ...
func (f *FallingBlock) Block() world.Block {
	return f.block
}

// Tick ...
func (f *FallingBlock) Tick(w *world.World, _ int64) {
	f.mu.Lock()
	m := f.c.TickMovement(f, f.pos, f.vel, 0, 0)
	f.pos, f.vel = m.pos, m.vel
	f.mu.Unlock()

	m.Send()
	pos := cube.PosFromVec3(m.pos)

	if pos[1] < w.Range()[0] {
		_ = f.Close()
	}

	if a, ok := f.block.(Solidifiable); (ok && a.Solidifies(pos, w)) || f.c.OnGround() {
		b := w.Block(pos)
		if r, ok := b.(replaceable); ok && r.ReplaceableBy(f.block) {
			w.PlaceBlock(pos, f.block)
		} else {
			if i, ok := f.block.(world.Item); ok {
				w.AddEntity(NewItem(item.NewStack(i, 1), pos.Vec3Middle()))
			}
		}

		_ = f.Close()
	}
}

// DecodeNBT decodes the relevant data from the entity NBT passed and returns a new FallingBlock entity.
func (f *FallingBlock) DecodeNBT(data map[string]interface{}) interface{} {
	b := nbtconv.MapBlock(data, "FallingBlock")
	if b == nil {
		return nil
	}
	n := NewFallingBlock(b, nbtconv.MapVec3(data, "Pos"))
	n.SetVelocity(nbtconv.MapVec3(data, "Motion"))
	return n
}

// EncodeNBT encodes the FallingBlock entity to a map that can be encoded for NBT.
func (f *FallingBlock) EncodeNBT() map[string]interface{} {
	return map[string]interface{}{
		"UniqueID":     -rand.Int63(),
		"Pos":          nbtconv.Vec3ToFloat32Slice(f.Position()),
		"Motion":       nbtconv.Vec3ToFloat32Slice(f.Velocity()),
		"FallingBlock": nbtconv.WriteBlock(f.block),
	}
}

// Solidifiable represents a block that can solidify by specific adjacent blocks. An example is concrete
// powder, which can turn into concrete by touching water.
type Solidifiable interface {
	// Solidifies returns whether the falling block can solidify at the position it is currently in. If so,
	// the block will immediately stop falling.
	Solidifies(pos cube.Pos, w *world.World) bool
}

type replaceable interface {
	ReplaceableBy(b world.Block) bool
}
