package block

import (
	"main.go/plugins/chunk_mirror/server/world"
)

type BeeContainer struct {
	solid
	containerType uint8
	direction     int32
	honey_level   int32
	nbt           map[string]interface{}
}

func (c BeeContainer) Hash() uint64 {
	h := uint64(0)
	h = uint64(hashBeeContainer) | uint64(c.direction)<<8 | uint64(c.honey_level)<<16 | uint64(c.containerType)<<24
	return h
}

func (c BeeContainer) EncodeItem() (name string, meta int16) {
	meta = int16(c.direction)
	meta += int16(4 * c.honey_level)
	if c.containerType == 0 {
		name = "minecraft:bee_nest"
	} else {
		name = "minecraft:beehive"
	}
	return name, meta
}

// EncodeBlock ...
func (c BeeContainer) EncodeBlock() (name string, properties map[string]interface{}) {
	properties = make(map[string]interface{})
	properties["direction"] = int32(c.direction)
	properties["honey_level"] = int32(c.honey_level)
	if c.containerType == 0 {
		name = "minecraft:bee_nest"
	} else {
		name = "minecraft:beehive"
	}
	return name, properties
}

//// DecodeNBT ...
func (c BeeContainer) DecodeNBT(data map[string]interface{}) interface{} {
	c.nbt = data
	//c.Patterns = data["Patterns"].(map[string]interface{})
	return c
}

func (c BeeContainer) EncodeNBT() map[string]interface{} {
	if c.nbt == nil {
		c.nbt = make(map[string]interface{})
	}
	return c.nbt
}

func allBeeContainer() []world.Block {
	b := make([]world.Block, 0)
	for f := int32(0); f < int32(4); f++ {
		for l := int32(0); l < int32(6); l++ {
			b = append(b, BeeContainer{direction: f, honey_level: l, containerType: 0})
			b = append(b, BeeContainer{direction: f, honey_level: l, containerType: 1})
		}
	}
	return b
}
