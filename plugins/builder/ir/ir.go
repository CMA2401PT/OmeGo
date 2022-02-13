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
	Counter   int
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
		c = NewChunk(ChunkX, ChunkZ, ir.Template)
		ir.Chunks[[2]define.PE{ChunkX, ChunkZ}] = c
	}
	c.SetBlockByID(X, Y, Z, blk)
}

func (ir *IR) SetNbtBlock(X, Y, Z define.PE, blk define.BlockDescribe, nbt define.Nbt) {
	ChunkX, ChunkZ := X>>4, Z>>4
	c, hasK := ir.Chunks[[2]define.PE{ChunkX, ChunkZ}]
	if !hasK {
		c = NewChunk(ChunkX, ChunkZ, ir.Template)
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
	ir.Counter += 1
	ChunkX, ChunkZ := X>>4, Z>>4
	c, hasK := ir.Chunks[[2]define.PE{ChunkX, ChunkZ}]
	if !hasK {
		c = NewChunk(ChunkX, ChunkZ, ir.Template)
		ir.Chunks[[2]define.PE{ChunkX, ChunkZ}] = c
	}
	c.SetBlockByID(X, Y, Z, ir.BlockID(blk))
}

func (ir *IR) SetBlockID2BlockDescribeMapping(mapping define.BlockID2BlockDescribeMapping) {
	ir.ID2Block = mapping
	ir.Block2ID = make(define.BlockDescribe2BlockIDMapping)
	for platteI, block := range ir.ID2Block {
		ir.Block2ID[block] = define.BLOCKID(platteI)
	}
}

type AnchoredChunk struct {
	C       *Chunk
	MovePos [2]define.PE
}

func (ir *IR) GetAnchoredChunk() []AnchoredChunk {
	chunk8s := make(Chunk8S)
	for _, c := range ir.Chunks {
		chunk8s.AddChunk(c)
	}
	anchoredChunks := make([]AnchoredChunk, 0)
	for _, cs := range chunk8s.Order() {
		anchoredChunks = append(anchoredChunks, AnchoredChunk{
			MovePos: [2]define.PE{cs.X4, cs.Z2},
		})
		for _, c := range cs.Order() {
			anchoredChunks = append(anchoredChunks, AnchoredChunk{
				C:       c,
				MovePos: [2]define.PE{cs.X4, cs.Z2},
			})
		}
	}
	return anchoredChunks
}
