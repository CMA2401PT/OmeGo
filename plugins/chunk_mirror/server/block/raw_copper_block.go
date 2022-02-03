package block

import (
	"main.go/plugins/chunk_mirror/server/item"
)

// RawCopperBlock is a raw metal block equivalent to nine raw copper.
type RawCopperBlock struct {
	solid
	bassDrum
}

// BreakInfo ...
func (r RawCopperBlock) BreakInfo() BreakInfo {
	return newBreakInfo(5, func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
	}, pickaxeEffective, oneOf(r))
}

// EncodeItem ...
func (RawCopperBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:raw_copper_block", 0
}

// EncodeBlock ...
func (RawCopperBlock) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:raw_copper_block", nil
}
