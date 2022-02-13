package define

type BLOCKID int32
type PE int32
type Pos [3]PE
type Nbt map[string]interface{}
type BlockDescribe struct {
	Name string
	Meta uint16
}
type BlockDescribe2BlockIDMapping map[BlockDescribe]BLOCKID

func NewBlock2IDMapping() BlockDescribe2BlockIDMapping {
	r := make(BlockDescribe2BlockIDMapping)
	r[BlockDescribe{"air", 0}] = 0
	return r
}

type BlockID2BlockDescribeMapping []BlockDescribe

func NewID2BlockMapping() BlockID2BlockDescribeMapping {
	r := make(BlockID2BlockDescribeMapping, 1, 32)
	r[0] = BlockDescribe{"air", 0}
	return r
}

type NbtBlock struct {
	Pos   Pos
	Nbt   Nbt
	Block BlockDescribe
}

const (
	AIRBLK  = BLOCKID(0)
	SKIPBLK = BLOCKID(-1)
)

const (
	BuildLevelBlock   = iota // 1 block
	BuildLevelCell           // 8 (2x2x2) block
	BuildLevelGroup          // 64 (4x4x4)
	BuildLevelSection        // 512 (8x8x8)
)
const (
	SPAWN_X = uint8(1)
	SPAWN_Y = uint8(2)
	SPAWN_Z = uint8(4)
)

//type Op struct {
//	Level   uint8
//	Pos     Pos
//	Spawn   uint8
//	BlockID BLOCKID
//}

type BlockOp struct {
	Pos     Pos
	BlockID BLOCKID
}

type OpsGroup struct {
	NormalOps *[]*BlockOp
	Palette   BlockID2BlockDescribeMapping
	NbtOps    []NbtBlock
}

const (
	NotSpecial = uint8(0)
	ALLSAME    = uint8(1)
	CanUseFill = uint8(2)
	//OneAnother     = uint8(2)
)

const (
	CellCanUseFillThres     = 8 // at lest 5
	GroupCanUseFillThres    = 8 // 6
	SectionCanUseFillThres  = 8 // 7
	SubChunkCanUseFillThres = 8 // 8
)

const (
// CASE01         = uint8(9)
// CASE23         = uint8(11)
// CASE45         = uint8(13)
// CASE67         = uint8(15)
// CASE02         = uint8(18)
// CASE13         = uint8(19)
// CASE46         = uint8(22)
// CASE57         = uint8(23)
// CASE04         = uint8(36)
// CASE15         = uint8(37)
// CASE26         = uint8(38)
// CASE37         = uint8(39)
)

func SpecialCase(a, b uint8) uint8 {
	axorb := a ^ b
	if axorb == 1 || axorb == 2 || axorb == 4 {
		return axorb<<3 | a | b
	}
	return 0
}
