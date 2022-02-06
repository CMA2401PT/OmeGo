package block

import (
	"main.go/plugins/world_mirror/server/item"
)

// NetheriteBlock is a precious mineral block made from 9 netherite ingots.
type NetheriteBlock struct {
	solid
	bassDrum
}

// BreakInfo ...
func (n NetheriteBlock) BreakInfo() BreakInfo {
	return newBreakInfo(50, func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierDiamond.HarvestLevel
	}, pickaxeEffective, oneOf(n))
}

// PowersBeacon ...
func (NetheriteBlock) PowersBeacon() bool {
	return true
}

// EncodeItem ...
func (NetheriteBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:netherite_block", 0
}

// EncodeBlock ...
func (NetheriteBlock) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:netherite_block", nil
}
