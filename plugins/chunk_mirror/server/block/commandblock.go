package block

import (
	"main.go/plugins/chunk_mirror/server/world"
)

type CommandBlock struct {
	solid
	bassDrum
	FACE            int32
	LPCommandMode   int32
	LPCondionalMode uint8
	nbt             map[string]interface{}
}

func (c CommandBlock) Hash() uint64 {
	_, meta := c.EncodeItem()
	h := uint64(0)
	if c.LPCommandMode == 0 {
		h = uint64(hashCommandBlock) | uint64(meta)<<8
	}
	if c.LPCommandMode == 1 {
		h = uint64(hashRepeatCommandBlock) | uint64(meta)<<8
	}
	if c.LPCommandMode == 2 {
		h = uint64(hashChainCommandBlock) | uint64(meta)<<8
	}

	return h
}

// 1 minecraft:repeating_command_block 0~5 conditional_bit=0, facing_direction=0~5 8~13 conditional_bit=1
// 2 minecraft:chain_command_block
// 0 minecraft:command_block

// EncodeItem ...
func (c CommandBlock) EncodeItem() (name string, meta int16) {
	name = "minecraft:command_block"
	if c.LPCommandMode == 0 {
		name = "minecraft:command_block"
	} else if c.LPCommandMode == 1 {
		name = "minecraft:repeating_command_block"
	} else if c.LPCommandMode == 2 {
		name = "minecraft:chain_command_block"
	}
	meta = 0
	if c.LPCondionalMode == 1 {
		meta = 8
	}
	meta += int16(c.FACE)
	return name, meta
}

// EncodeBlock ...
func (c CommandBlock) EncodeBlock() (name string, properties map[string]interface{}) {
	name, _ = c.EncodeItem()
	properties = make(map[string]interface{})
	properties["conditional_bit"] = c.LPCondionalMode
	properties["facing_direction"] = c.FACE
	return name, properties
}

// DecodeNBT ...
func (c CommandBlock) DecodeNBT(data map[string]interface{}) interface{} {
	c.nbt = data
	c.LPCommandMode = c.nbt["LPCommandMode"].(int32)
	c.LPCondionalMode = c.nbt["LPCondionalMode"].(uint8)
	return c
}

// EncodeNBT ...
func (c CommandBlock) EncodeNBT() map[string]interface{} {
	if c.nbt == nil {
		c.nbt = make(map[string]interface{})
	}
	c.nbt["LPCommandMode"] = c.LPCommandMode
	c.nbt["LPCondionalMode"] = c.LPCondionalMode
	return c.nbt
}

func allCommandBlock() []world.Block {
	b := make([]world.Block, 0)
	for f := int32(0); f < int32(6); f++ {
		for m := int32(0); m <= int32(2); m++ {
			for cm := uint8(0); cm <= uint8(1); cm++ {
				b = append(b, CommandBlock{FACE: f, LPCommandMode: m, LPCondionalMode: cm})
			}
		}
	}
	return b
}
