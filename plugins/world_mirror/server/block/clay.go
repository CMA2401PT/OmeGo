package block

import (
	"main.go/plugins/world_mirror/server/item"
	"main.go/plugins/world_mirror/server/world/sound"
)

// Clay is a block that can be found underwater.
type Clay struct {
	solid
}

// Instrument ...
func (c Clay) Instrument() sound.Instrument {
	return sound.Flute()
}

// BreakInfo ...
func (c Clay) BreakInfo() BreakInfo {
	return newBreakInfo(0.6, alwaysHarvestable, shovelEffective, silkTouchDrop(item.NewStack(item.ClayBall{}, 4), item.NewStack(c, 1)))
}

// EncodeItem ...
func (c Clay) EncodeItem() (name string, meta int16) {
	return "minecraft:clay", 0
}

// EncodeBlock ...
func (c Clay) EncodeBlock() (name string, properties map[string]interface{}) {
	return "minecraft:clay", nil
}
