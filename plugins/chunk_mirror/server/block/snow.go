package block

import "main.go/plugins/chunk_mirror/server/item"

// Snow is a full-sized block of snow.
type Snow struct {
	solid
}

// BreakInfo ...
func (s Snow) BreakInfo() BreakInfo {
	return newBreakInfo(0.2, alwaysHarvestable, shovelEffective, silkTouchDrop(item.NewStack(item.Snowball{}, 4), item.NewStack(s, 1)))
}

// EncodeItem ...
func (Snow) EncodeItem() (name string, meta int16) {
	return "minecraft:snow", 0
}

// EncodeBlock ...
func (Snow) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:snow", nil
}
