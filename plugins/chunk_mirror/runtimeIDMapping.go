package chunk_mirror

import (
	_ "embed"
	"encoding/json"
	reflect_block "main.go/plugins/chunk_mirror/server/block"
	"main.go/plugins/chunk_mirror/server/block/cube"
	reflect_world "main.go/plugins/chunk_mirror/server/world"
)

type RichBlock struct {
	Name       string
	Val        int
	NeteaseRID int
	ReflectRID int
	Props      map[string]interface{}
}

//go:embed richBlock.json
var richBlocksData []byte

var RichBlocks struct {
	ReflectAirRID int
	NeteaseAirRID int
	RichBlocks    []RichBlock
}

// NetEase to Inc
var BlockReflectMapping []uint32

// Inc to Netease
var BlockDeReflectMapping map[uint32]uint32

var NeteaseAirRID int
var MirrorAirRID uint32
var WorldRange cube.Range

func init() {
	err := json.Unmarshal(richBlocksData, &RichBlocks)
	if err != nil {
		panic("Chunk Mirror: cannot read remapping info")
	}
	NeteaseAirRID = RichBlocks.NeteaseAirRID
	// check runtime id of air is same with that in dragonfly
	MirrorAirRID, _ = reflect_world.BlockRuntimeID(reflect_block.Air{})
	if MirrorAirRID != uint32(RichBlocks.ReflectAirRID) {
		panic("Reflect World not properly init!")
	}
	BlockReflectMapping = make([]uint32, len(RichBlocks.RichBlocks))
	BlockDeReflectMapping = make(map[uint32]uint32)
	for _, richBlocks := range RichBlocks.RichBlocks {
		BlockReflectMapping[richBlocks.NeteaseRID] = uint32(richBlocks.ReflectRID)
		BlockDeReflectMapping[uint32(richBlocks.ReflectRID)] = uint32(richBlocks.NeteaseRID)
	}
}
