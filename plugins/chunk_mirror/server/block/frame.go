package block

import (
	"main.go/plugins/chunk_mirror/server/world"
)

type Frame struct {
	solid
	item_frame_map_bit   uint8
	item_frame_photo_bit uint8
	facing_direction     int32
	nbt                  map[string]interface{}
}

func (c Frame) Hash() uint64 {
	h := uint64(0)
	//_, meta := c.EncodeItem()
	//h = uint64(hashFrame) | uint64(meta)<<8
	return h
}

func (c Frame) EncodeItem() (name string, meta int16) {
	meta = int16(c.facing_direction) + int16(c.item_frame_map_bit)*8
	return "minecraft:frame", meta
}

// EncodeBlock ...
func (c Frame) EncodeBlock() (name string, properties map[string]interface{}) {
	properties = make(map[string]interface{})
	properties["facing_direction"] = c.facing_direction
	properties["item_frame_map_bit"] = c.item_frame_map_bit
	properties["item_frame_photo_bit"] = c.item_frame_photo_bit
	return "minecraft:frame", properties
}

//// DecodeNBT ...
func (c Frame) DecodeNBT(data map[string]interface{}) interface{} {
	c.nbt = data
	return c
}

func (c Frame) EncodeNBT() map[string]interface{} {
	if c.nbt == nil {
		c.nbt = make(map[string]interface{})
	}
	return c.nbt
}

func allFrame() []world.Block {
	b := make([]world.Block, 0)
	for f := int32(0); f < int32(6); f++ {
		for m := uint8(0); m <= uint8(1); m++ {
			b = append(b, Frame{item_frame_map_bit: m, facing_direction: f, item_frame_photo_bit: 0})
		}
	}
	return b
}
