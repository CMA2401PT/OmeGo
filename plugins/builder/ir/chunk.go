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
	T    subChunk.SubChunk
}

func NewChunk(ChunkX, ChunkZ PE, Template subChunk.SubChunk) *Chunk {
	sub := make([]subChunk.SubChunk, 0)
	c := &Chunk{Sub: sub, X: ChunkX << 4, Z: ChunkZ << 4, NbtBlocks: make([]NbtBlock, 0), T: Template}
	//c.Block2ID = NewBlock2IDMapping()
	//c.ID2Block = NewID2BlockMapping()
	return c
}

func (chunk *Chunk) SetBlockByID(X, Y, Z PE, blk BLOCKID) {
	layerI := Y >> 4
	for len(chunk.Sub) <= int(layerI) {
		chunk.Sub = append(chunk.Sub, chunk.T.New())
	}
	sub := chunk.Sub[layerI]
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
	Ops := make([]*BlockOp, 0, 4096)
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
