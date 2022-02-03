package block

import (
	"main.go/plugins/chunk_mirror/server/item"
)

// DiamondOre is a rare ore that generates underground.
type DiamondOre struct {
	solid
	bassDrum

	// Type is the type of diamond ore.
	Type OreType
}

// BreakInfo ...
func (d DiamondOre) BreakInfo() BreakInfo {
	i := newBreakInfo(d.Type.Hardness(), func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierIron.HarvestLevel
	}, pickaxeEffective, silkTouchOneOf(item.Diamond{}, d))
	i.XPDrops = XPDropRange{3, 7}
	return i
}

// EncodeItem ...
func (d DiamondOre) EncodeItem() (name string, meta int16) {
	return "minecraft:" + d.Type.Prefix() + "diamond_ore", 0
}

// EncodeBlock ...
func (d DiamondOre) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:" + d.Type.Prefix() + "diamond_ore", nil
}
