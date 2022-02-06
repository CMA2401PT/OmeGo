package block

import "main.go/plugins/world_mirror/server/world"

// Terracotta is a block formed from clay, with a hardness and blast resistance comparable to stone. For colouring it,
// take a look at StainedTerracotta.
type Terracotta struct {
	solid
	bassDrum
}

// SoilFor ...
func (Terracotta) SoilFor(block world.Block) bool {
	_, ok := block.(DeadBush)
	return ok
}

// BreakInfo ...
func (t Terracotta) BreakInfo() BreakInfo {
	return newBreakInfo(1.25, pickaxeHarvestable, pickaxeEffective, oneOf(t))
}

// EncodeItem ...
func (Terracotta) EncodeItem() (name string, meta int16) {
	return "minecraft:hardened_clay", meta
}

// EncodeBlock ...
func (Terracotta) EncodeBlock() (string, map[string]interface{}) {
	return "minecraft:hardened_clay", nil
}
