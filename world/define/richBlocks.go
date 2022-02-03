package define

import (
	_ "embed"
	"main.go/dragonfly/server/world"
)

//go:embed richBlocks.json
var richBlocksData []byte

func InitRichBlocks() {
	world.LoadRichBlocks(richBlocksData)
}
