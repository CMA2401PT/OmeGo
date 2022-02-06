package block

import (
	"main.go/plugins/world_mirror/server/world/sound"
)

// HoneycombBlock is a decorative blocks crafted from honeycombs.
type HoneycombBlock struct {
	solid
}

// Instrument ...
func (h HoneycombBlock) Instrument() sound.Instrument {
	return sound.Flute()
}

// BreakInfo ...
func (h HoneycombBlock) BreakInfo() BreakInfo {
	return newBreakInfo(0.6, alwaysHarvestable, nothingEffective, oneOf(h))
}

// EncodeItem ...
func (HoneycombBlock) EncodeItem() (name string, meta int16) {
	return "minecraft:honeycomb_block", 0
}

// EncodeBlock ...
func (HoneycombBlock) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:honeycomb_block", nil
}
