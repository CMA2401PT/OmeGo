package block

import (
	"main.go/dragonfly/server/block/cube"
	"main.go/dragonfly/server/item"
	"main.go/dragonfly/server/world"
)

// Placer represents an entity that is able to place a block at a specific position in the world.
type Placer interface {
	item.User
	PlaceBlock(pos cube.Pos, b world.Block, ctx *item.UseContext)
}
