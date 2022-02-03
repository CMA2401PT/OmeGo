package item

import (
	"time"

	"main.go/plugins/chunk_mirror/server/entity/effect"
	"main.go/plugins/chunk_mirror/server/world"
)

// SpiderEye is a poisonous food and brewing item.
type SpiderEye struct {
	defaultFood
}

// Consume ...
func (SpiderEye) Consume(_ *world.World, c Consumer) Stack {
	c.Saturate(2, 3.2)
	c.AddEffect(effect.New(effect.Poison{}, 1, time.Second*5))
	return Stack{}
}

// EncodeItem ...
func (SpiderEye) EncodeItem() (name string, meta int16) {
	return "minecraft:spider_eye", 0
}
