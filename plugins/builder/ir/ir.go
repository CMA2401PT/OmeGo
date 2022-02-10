package ir

import (
	"main.go/plugins/builder/define"
	"main.go/plugins/builder/ir/subChunk"
)

type IR struct {
	Template  subChunk.SubChunk
	Chunks    map[[2]define.PE]*Chunk
	Block2ID  define.BlockDescribe2BlockIDMapping
	ID2Block  define.BlockID2BlockDescribeMapping
	MaxHeight uint16
}

func NewIR(MaxHeight uint16, Template subChunk.SubChunk) *IR {
	ir := &IR{
		Template:  Template,
		Chunks:    make(map[[2]define.PE]*Chunk),
		Block2ID:  define.NewBlock2IDMapping(),
		ID2Block:  define.NewID2BlockMapping(),
		MaxHeight: MaxHeight,
	}
	return ir
}

func (ir *IR) SetBlockByID(X, Y, Z define.PE, blk define.BLOCKID) {
	ChunkX, ChunkZ := X>>4, Z>>4
	c, hasK := ir.Chunks[[2]define.PE{ChunkX, ChunkZ}]
	if !hasK {
		c = NewChunk(ir.MaxHeight, ChunkX, ChunkZ, ir.Template)
		ir.Chunks[[2]define.PE{ChunkX, ChunkZ}] = c
	}
	c.SetBlockByID(X, Y, Z, blk)
}

func (ir *IR) SetNbtBlock(X, Y, Z define.PE, blk define.BlockDescribe, nbt define.Nbt) {
	ChunkX, ChunkZ := X>>4, Z>>4
	c, hasK := ir.Chunks[[2]define.PE{ChunkX, ChunkZ}]
	if !hasK {
		c = NewChunk(ir.MaxHeight, ChunkX, ChunkZ, ir.Template)
		ir.Chunks[[2]define.PE{ChunkX, ChunkZ}] = c
	}
	c.SetNbtBlock(X, Y, Z, blk, nbt)
}

func (ir *IR) BlockID(blk define.BlockDescribe) define.BLOCKID {
	blkID, hasK := ir.Block2ID[blk]
	if !hasK {
		blkID = define.BLOCKID(len(ir.ID2Block))
		ir.ID2Block = append(ir.ID2Block, blk)
		ir.Block2ID[blk] = blkID
	}
	return blkID
}

func (ir *IR) SetBlock(X, Y, Z define.PE, blk define.BlockDescribe) {
	ChunkX, ChunkZ := X>>4, Z>>4
	c, hasK := ir.Chunks[[2]define.PE{ChunkX, ChunkZ}]
	if !hasK {
		c = NewChunk(ir.MaxHeight, ChunkX, ChunkZ, ir.Template)
		ir.Chunks[[2]define.PE{ChunkX, ChunkZ}] = c
	}
	c.SetBlockByID(X, Y, Z, ir.BlockID(blk))
}

func (ir *IR) IterOpsGroup(iter func(opsGroup *define.OpsGroup), move func(X, Z define.PE)) {
	chunk16s := make(Chunk16S)
	for _, c := range ir.Chunks {
		chunk16s.AddChunk(c)
	}
	for _, cs := range chunk16s.Order() {
		move(cs.X16, cs.Z16)
		for _, c := range cs.Order() {
			iter(c.GetOps(ir.ID2Block))
		}
	}
}

type AnchoredChunk struct {
	C       *Chunk
	MovePos [2]define.PE
}

func (ir *IR) GetAnchoredChunk() []AnchoredChunk {
	chunk16s := make(Chunk16S)
	for _, c := range ir.Chunks {
		chunk16s.AddChunk(c)
	}
	anchoredChunks := make([]AnchoredChunk, 0)
	for _, cs := range chunk16s.Order() {
		anchoredChunks = append(anchoredChunks, AnchoredChunk{
			MovePos: [2]define.PE{cs.X16, cs.Z16},
		})
		for _, c := range cs.Order() {
			anchoredChunks = append(anchoredChunks, AnchoredChunk{
				C:       c,
				MovePos: [2]define.PE{cs.X16, cs.Z16},
			})
		}
	}
	return anchoredChunks
}
