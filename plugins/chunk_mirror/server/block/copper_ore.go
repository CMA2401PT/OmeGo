package block

import (
	"math/rand"

	"main.go/plugins/chunk_mirror/server/item"
)

// CopperOre is a rare mineral block found underground.
type CopperOre struct {
	solid
	bassDrum

	// Type is the type of copper ore.
	Type OreType
}

// BreakInfo ...
func (c CopperOre) BreakInfo() BreakInfo {
	return newBreakInfo(c.Type.Hardness(), func(t item.Tool) bool {
		return t.ToolType() == item.TypePickaxe && t.HarvestLevel() >= item.ToolTierStone.HarvestLevel
	}, pickaxeEffective, silkTouchDrop(item.NewStack(item.RawCopper{}, rand.Intn(4)+2), item.NewStack(c, 1)))
}

// EncodeItem ...
func (c CopperOre) EncodeItem() (name string, meta int16) {
	return "minecraft:" + c.Type.Prefix() + "copper_ore", 0
}

// EncodeBlock ...
func (c CopperOre) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:" + c.Type.Prefix() + "copper_ore", nil
}
