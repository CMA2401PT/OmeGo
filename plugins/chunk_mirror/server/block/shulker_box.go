package block

import (
	"main.go/plugins/chunk_mirror/server/world"
)

// Barrel is a fisherman's job site block, used to store items. It functions like a single chest, although
// it requires no airspace above it to be opened.
type ShulkerBox struct {
	solid
	bass
	color int
	nbt   map[string]interface{}
}

// DecodeNBT ...
func (b ShulkerBox) DecodeNBT(data map[string]interface{}) interface{} {
	b.nbt = data
	return b
}

// EncodeNBT ...
func (b ShulkerBox) EncodeNBT() map[string]interface{} {
	if b.nbt == nil {
		b.nbt = map[string]interface{}{}
	}
	return b.nbt
}

func (b ShulkerBox) Hash() uint64 {
	return hashShulkerBox | uint64(b.color)<<8
}

// EncodeBlock ...
func (b ShulkerBox) EncodeBlock() (string, map[string]interface{}) {
	props := map[string]interface{}{"color": "white"}
	name := "minecraft:shulker_box"
	if b.color == 0 {
		name = "minecraft:undyed_shulker_box"
		props = make(map[string]interface{})
	} else {
		c := "white"
		switch b.color {
		case 1:
			c = "white"
		case 2:
			c = "orange"
		case 3:
			c = "magenta"
		case 4:
			c = "light_blue"
		case 5:
			c = "yellow"
		case 6:
			c = "lime"
		case 7:
			c = "pink"
		case 8:
			c = "gray"
		case 9:
			c = "silver"
		case 10:
			c = "cyan"
		case 11:
			c = "purple"
		case 12:
			c = "blue"
		case 13:
			c = "brown"
		case 14:
			c = "green"
		case 15:
			c = "red"
		case 16:
			c = "black"
		}
		props = map[string]interface{}{"color": c}
	}
	return name, props
}

// EncodeItem ...
func (b ShulkerBox) EncodeItem() (name string, meta int16) {
	meta = int16(b.color - 1)
	name = "minecraft:shulker_box"
	if b.color == 0 {
		name = "minecraft:undyed_shulker_box"
		meta = 0
	}
	return name, meta
}

func allShulkerBoxs() (b []world.Block) {
	for i := 0; i <= 16; i++ {
		b = append(b, ShulkerBox{color: i})
	}
	return
}
