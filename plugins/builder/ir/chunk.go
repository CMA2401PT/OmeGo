package ir

import (
	. "main.go/plugins/builder/define"
	"main.go/plugins/builder/ir/subChunk"
)

type Chunk struct {
	Sub       []subChunk.SubChunk
	NbtBlocks []NbtBlock
	//Block2ID  BlockDescribe2BlockIDMapping
	//ID2Block  BlockID2BlockDescribeMapping
	X, Z PE
}

func NewChunk(MaxHeight uint16, ChunkX, ChunkZ PE, Template subChunk.SubChunk) *Chunk {
	n := (MaxHeight >> 4) + 1
	sub := make([]subChunk.SubChunk, n)
	for i := uint16(0); i < n; i++ {
		sub[i] = Template.New()
	}
	c := &Chunk{Sub: sub, X: ChunkX << 4, Z: ChunkZ << 4, NbtBlocks: make([]NbtBlock, 0)}
	//c.Block2ID = NewBlock2IDMapping()
	//c.ID2Block = NewID2BlockMapping()
	return c
}

func (chunk *Chunk) SetBlockByID(X, Y, Z PE, blk BLOCKID) {
	sub := chunk.Sub[Y>>4]
	sub.Set(uint8(X&0xf), uint8(Y&0xf), uint8(Z&0xf), blk)
}

func (chunk *Chunk) SetNbtBlock(X, Y, Z PE, blk BlockDescribe, nbt Nbt) {
	chunk.NbtBlocks = append(chunk.NbtBlocks, NbtBlock{
		Pos:   Pos{X, Y, Z},
		Nbt:   nbt,
		Block: blk,
	})
}

func (chunk *Chunk) GetOps(ID2Block BlockID2BlockDescribeMapping) *OpsGroup {
	Ops := make([]*Op, 0, 4096)
	OpsPtr := &Ops
	g := &OpsGroup{
		NormalOps: OpsPtr,
		Palette:   ID2Block,
		NbtOps:    chunk.NbtBlocks,
	}
	for layerI, sub := range chunk.Sub {
		sub.Finish()
		sub.GetOps(chunk.X, PE(layerI<<4), chunk.Z, OpsPtr)
	}
	return g
}
