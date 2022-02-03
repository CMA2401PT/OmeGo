package item

import (
	"main.go/plugins/chunk_mirror/server/world"
)

// Beetroot is a food and dye ingredient.
type Beetroot struct {
	defaultFood
}

// Consume ...
func (b Beetroot) Consume(_ *world.World, c Consumer) Stack {
	c.Saturate(1, 1.2)
	return Stack{}
}

// EncodeItem ...
func (b Beetroot) EncodeItem() (name string, meta int16) {
	return "minecraft:beetroot", 0
}
