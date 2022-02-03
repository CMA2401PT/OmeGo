package block

import (
	"main.go/plugins/chunk_mirror/server/item"
)

// LapisBlock is a decorative mineral block that is crafted from lapis lazuli.
type LapisBlock struct {
	solid
}

// BreakInfo ...
func (l LapisBlock) BreakInfo() BreakInfo {
	return newBreakInfo(3, func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
	}, pickaxeEffective, oneOf(l))
}

// EncodeItem ...
func (LapisBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:lapis_block", 0
}

// EncodeBlock ...
func (LapisBlock) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:lapis_block", nil
}
