package subChunk

import . "main.go/plugins/builder/define"

type SubChunk interface {
	// a sub chunk storage 16*16*16 blocks
	New() SubChunk
	Set(X, Y, Z uint8, blk BLOCKID)
	GetOps(X, Y, Z PE, Ops *[]*BlockOp)
	Finish()
}
