package block

import (
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/internal/nbtconv"
	"main.go/plugins/world_mirror/server/item"
	"main.go/plugins/world_mirror/server/world"
	"main.go/plugins/world_mirror/server/world/particle"
	"main.go/plugins/world_mirror/server/world/sound"
)

// NoteBlock is a musical block that emits sounds when powered with redstone.
type NoteBlock struct {
	solid
	bass

	// Pitch is the current pitch the note block is set to. Value ranges from 0-24.
	Pitch int
}

// playNote ...
func (n NoteBlock) playNote(pos cube.Pos, w *world.World) {
	w.PlaySound(pos.Vec3(), sound.Note{Instrument: n.instrument(pos, w), Pitch: n.Pitch})
	w.AddParticle(pos.Vec3(), particle.Note{Instrument: n.Instrument(), Pitch: n.Pitch})
}

// updateInstrument ...
func (n NoteBlock) instrument(pos cube.Pos, w *world.World) sound.Instrument {
	if instrumentBlock, ok := w.Block(pos.Side(cube.FaceDown)).(interface {
		Instrument() sound.Instrument
	}); ok {
		return instrumentBlock.Instrument()
	}
	return sound.Piano()
}

// DecodeNBT ...
func (n NoteBlock) DecodeNBT(data map[string]interface{}) interface{} {
	n.Pitch = int(nbtconv.MapByte(data, "note"))
	return n
}

// EncodeNBT ...
func (n NoteBlock) EncodeNBT() map[string]interface{} {
	return map[string]interface{}{"note": byte(n.Pitch)}
}

// Punch ...
func (n NoteBlock) Punch(pos cube.Pos, _ cube.Face, w *world.World, _ item.User) {
	if _, ok := w.Block(pos.Side(cube.FaceUp)).(Air); !ok {
		return
	}
	n.playNote(pos, w)
}

// Activate ...
func (n NoteBlock) Activate(pos cube.Pos, _ cube.Face, w *world.World, _ item.User) bool {
	if _, ok := w.Block(pos.Side(cube.FaceUp)).(Air); !ok {
		return false
	}
	n.Pitch = (n.Pitch + 1) % 25
	n.playNote(pos, w)
	w.SetBlock(pos, n)
	return true
}

// BreakInfo ...
func (n NoteBlock) BreakInfo() BreakInfo {
	return newBreakInfo(0.8, alwaysHarvestable, axeEffective, oneOf(n))
}

// EncodeItem ...
func (n NoteBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:noteblock", 0
}

// EncodeBlock ...
func (n NoteBlock) EncodeBlock() (name string, properties map[string]interface{}) {
	return "minecraft:noteblock", nil
}
