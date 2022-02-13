package oct

import (
	. "main.go/plugins/builder/define"
	"main.go/plugins/builder/ir/subChunk"
)

type Oct struct {
	Major BLOCKID
	//MajorCount uint8
	SpecialCase uint8
	MajorData   [8]BLOCKID
}

func (c *Oct) Finish(Thres uint8) {
	var hist [8]uint8
	i := uint8(0)
	j := uint8(0)
	MajorCount := uint8(0)
	c.Major = c.MajorData[0]
	for i = 1; i < uint8(8); i++ {
		if c.Major != c.MajorData[i] {
			j = 1
			break
		}
	}
	if j == 0 {
		if c.Major != SKIPBLK {
			c.SpecialCase = ALLSAME
		} else {
			c.SpecialCase = NotSpecial
		}
		return
	}
	for i = 0; i < uint8(8); i++ {
		compVal := c.MajorData[i]
		for j = uint8(0); j <= i; j++ {
			if c.MajorData[j] == compVal {
				hist[j] += 1
			}
		}
	}
	for i = 0; i < uint8(7); i++ {
		if hist[i] > MajorCount {
			MajorCount = hist[i]
			c.Major = c.MajorData[i]
		}
	}
	//switch MajorCount {
	//case 8:
	//	c.SpecialCase = ALLSAME
	//	break
	//case 7:
	//	c.SpecialCase = OneAnother
	//	for i = 0; i < uint8(8); i++ {
	//		if c.MajorData[i] != c.Major {
	//			c.SpecialVal = uint64(c.MajorData[i])<<4 | uint64(i)
	//		}
	//	}
	//	break
	//default:
	//	if MajorCount > Thres {
	//		c.SpecialCase = CanUseFill
	//	} else {
	//		c.SpecialCase = NotSpecial
	//	}
	//}
	//if MajorCount == 8 {
	//	c.SpecialCase = ALLSAME
	//} else if MajorCount > Thres {
	//	c.SpecialCase = CanUseFill
	//} else {
	//	c.SpecialCase = NotSpecial
	//}

	if MajorCount < Thres {
		c.SpecialCase = NotSpecial
		c.Major = SKIPBLK
	} else {
		if c.Major != SKIPBLK {
			c.SpecialCase = CanUseFill
		} else {
			c.SpecialCase = NotSpecial
		}
	}
}

func (c *Oct) Spawn(defaultBlock BLOCKID) (*[8]BLOCKID, *[8]uint8) {
	var SpawnData [8]uint8
	TreeData := c.MajorData

	var xASpawn, yASpawn, zASpawn, xBSpawn, yBSpawn, zBSpawn bool
	var cA, cB, lB BLOCKID

	// block @ 0
	cA = TreeData[0]
	noFace0 := true
	noFace1 := true
	noFace4 := true
	noFace2 := true
	if cA != defaultBlock {
		xASpawn = TreeData[1] == cA
		yASpawn = TreeData[2] == cA
		zASpawn = TreeData[4] == cA
		if xASpawn {
			if yASpawn && TreeData[3] == cA {
				SpawnData[0] = SPAWN_X | SPAWN_Y
				TreeData[1] = defaultBlock
				TreeData[2] = defaultBlock
				TreeData[3] = defaultBlock
				noFace0 = false
			} else if zASpawn && TreeData[5] == cA {
				SpawnData[0] = SPAWN_X | SPAWN_Z
				TreeData[1] = defaultBlock
				TreeData[4] = defaultBlock
				TreeData[5] = defaultBlock
				noFace0 = false
			}
		}
		if SpawnData[0] == 0 {
			if yASpawn && zASpawn && TreeData[6] == cA {
				SpawnData[0] = SPAWN_Y | SPAWN_Z
				TreeData[2] = defaultBlock
				TreeData[4] = defaultBlock
				TreeData[6] = defaultBlock
				noFace0 = false
			}
		}
	}
	cB = TreeData[7]
	if cB != defaultBlock {
		xBSpawn = TreeData[6] == cB
		yBSpawn = TreeData[5] == cB
		zBSpawn = TreeData[3] == cB
		if xBSpawn {
			if yBSpawn && TreeData[4] == cB {
				SpawnData[4] = SPAWN_X | SPAWN_Y
				TreeData[5] = defaultBlock
				TreeData[6] = defaultBlock
				TreeData[7] = defaultBlock
				noFace4 = false
			} else if zBSpawn && TreeData[2] == cB {
				SpawnData[2] = SPAWN_X | SPAWN_Z
				TreeData[3] = defaultBlock
				TreeData[6] = defaultBlock
				TreeData[7] = defaultBlock
				noFace2 = false
			}
		}
		if TreeData[7] != defaultBlock {
			if yASpawn && zASpawn && TreeData[1] == cB {
				SpawnData[1] = SPAWN_Y | SPAWN_Z
				TreeData[3] = defaultBlock
				TreeData[5] = defaultBlock
				TreeData[7] = defaultBlock
				noFace1 = false
			}
		}
	}
	if noFace0 {
		if xASpawn && noFace1 {
			SpawnData[0] = SPAWN_X
			TreeData[1] = defaultBlock
		} else if yASpawn && noFace2 {
			SpawnData[0] = SPAWN_Y
			TreeData[2] = defaultBlock
		} else if zASpawn && noFace4 {
			SpawnData[0] = SPAWN_Z
			TreeData[4] = defaultBlock
		}
	}
	if TreeData[7] != defaultBlock {
		if xBSpawn && SpawnData[6] == 0 {
			SpawnData[6] = SPAWN_X
			TreeData[7] = defaultBlock
		}
		if yBSpawn && SpawnData[3] == 0 {
			SpawnData[3] = SPAWN_Y
			TreeData[7] = defaultBlock
		}
		if zBSpawn && SpawnData[5] == 0 {
			SpawnData[5] = SPAWN_X
			TreeData[7] = defaultBlock
		}
	}
	lB = TreeData[1]
	if lB != defaultBlock && SpawnData[1] == 0 {
		if lB == TreeData[3] && SpawnData[3] == 0 {
			SpawnData[1] = SPAWN_Y
			TreeData[3] = defaultBlock
		}
		if lB == TreeData[5] && SpawnData[5] == 0 {
			SpawnData[1] = SPAWN_Z
			TreeData[5] = defaultBlock
		}
	}
	lB = TreeData[2]
	if lB != defaultBlock && SpawnData[2] == 0 {
		if lB == TreeData[3] && SpawnData[3] == 0 {
			SpawnData[2] = SPAWN_X
			TreeData[3] = defaultBlock
		}
		if lB == TreeData[6] && SpawnData[6] == 0 {
			SpawnData[2] = SPAWN_Z
			TreeData[6] = defaultBlock
		}
	}
	lB = TreeData[4]
	if lB != defaultBlock && SpawnData[4] == 0 {
		lB = TreeData[4]
		if lB == TreeData[5] && SpawnData[5] == 0 {
			SpawnData[4] = SPAWN_X
			TreeData[5] = defaultBlock
		}
		if lB == TreeData[6] && SpawnData[6] == 0 {
			SpawnData[4] = SPAWN_Y
			TreeData[6] = defaultBlock
		}
	}
	return &TreeData, &SpawnData
}

// 8 blocks
type Cell struct {
	Oct
}

func (c *Cell) Set(b uint8, blk BLOCKID) {
	c.MajorData[b] = blk
}

func (c *Cell) Finish() {
	c.Oct.Finish(CellCanUseFillThres)
}

func (c *Cell) GetOps(X, Y, Z PE, PlaceHolderBlock BLOCKID, Ops *[]*Op) {
	if c.SpecialCase == ALLSAME || c.SpecialCase == CanUseFill {
		if c.Major != PlaceHolderBlock {
			*Ops = append(*Ops, &Op{
				Level:   BuildLevelCell,
				BlockID: c.Major,
				Pos:     Pos{X, Y, Z},
			})
			PlaceHolderBlock = c.Major
		}
		if c.SpecialCase == ALLSAME {
			return
		}
	}

	TreeData, SpawnData := c.Oct.Spawn(PlaceHolderBlock)
	i := uint8(0)
	var BlockID BLOCKID
	for i = 0; i < uint8(8); i++ {
		BlockID = TreeData[i]
		if BlockID != PlaceHolderBlock {
			*Ops = append(*Ops, &Op{
				Level:   BuildLevelBlock,
				BlockID: BlockID,
				Pos:     Pos{X + PE(i&1), Y + PE((i&2)>>1), Z + PE((i&4)>>2)},
				Spawn:   SpawnData[i],
			})
		}
	}
}

// 64 blocks 4x4x4
type Group struct {
	Oct
	CellData [8]*Cell
}

func (g *Group) Set(c, b uint8, blk BLOCKID) {
	if g.CellData[c] == nil {
		g.CellData[c] = &Cell{}
	}
	g.CellData[c].Set(b, blk)
}

func (g *Group) Finish() {
	i := uint8(0)
	for i = 0; i < uint8(8); i++ {
		if g.CellData[i] != nil {
			g.CellData[i].Finish()
			g.MajorData[i] = g.CellData[i].Major
		} else {
			g.MajorData[i] = AIRBLK
		}
	}
	g.Oct.Finish(GroupCanUseFillThres)
}

func (g *Group) GetOps(X, Y, Z PE, PlaceHolderBlock BLOCKID, Ops *[]*Op) {
	if g.SpecialCase == ALLSAME || g.SpecialCase == CanUseFill {
		if g.Major != PlaceHolderBlock {
			*Ops = append(*Ops, &Op{
				Level:   BuildLevelGroup,
				BlockID: g.Major,
				Pos:     Pos{X, Y, Z},
			})
			PlaceHolderBlock = g.Major
		}
	}

	TreeData, SpawnData := g.Oct.Spawn(PlaceHolderBlock)
	i := uint8(0)
	var BlockID BLOCKID
	var SX, SY, SZ PE
	for i = 0; i < uint8(8); i++ {
		BlockID = TreeData[i]
		SX = X + PE((i&1)<<1)
		SY = Y + PE(i&2)
		SZ = Z + PE((i&4)>>1)
		if BlockID != PlaceHolderBlock && BlockID != SKIPBLK {
			*Ops = append(*Ops, &Op{
				Level:   BuildLevelCell,
				BlockID: BlockID,
				Pos:     Pos{SX, SY, SZ},
				Spawn:   SpawnData[i],
			})
		}
		if g.CellData[i] != nil {
			if BlockID == SKIPBLK {
				g.CellData[i].GetOps(SX, SY, SZ, PlaceHolderBlock, Ops)
			} else {
				g.CellData[i].GetOps(SX, SY, SZ, g.MajorData[i], Ops)
			}
		}
	}
}

// 512 blocks 8x8x8
type Section struct {
	Oct
	GroupData [8]*Group
}

func newSection() *Section {
	return &Section{}
}

func (s *Section) Set(g, c, b uint8, blk BLOCKID) {
	if s.GroupData[g] == nil {
		s.GroupData[g] = &Group{}
	}
	s.GroupData[g].Set(c, b, blk)
}

func (s *Section) Finish() {
	i := uint8(0)
	for i = 0; i < uint8(8); i++ {
		if s.GroupData[i] != nil {
			s.GroupData[i].Finish()
			s.MajorData[i] = s.GroupData[i].Major
		} else {
			s.MajorData[i] = AIRBLK
		}
	}
	s.Oct.Finish(SectionCanUseFillThres)
}

func (s *Section) GetOps(X, Y, Z PE, PlaceHolderBlock BLOCKID, Ops *[]*Op) {
	if s.SpecialCase == ALLSAME || s.SpecialCase == CanUseFill {
		if s.Major != PlaceHolderBlock {
			*Ops = append(*Ops, &Op{
				Level:   BuildLevelSection,
				BlockID: s.Major,
				Pos:     Pos{X, Y, Z},
			})
			PlaceHolderBlock = s.Major
		}
	}

	TreeData, SpawnData := s.Oct.Spawn(PlaceHolderBlock)
	i := uint8(0)
	var BlockID BLOCKID
	var SX, SY, SZ PE

	for i = 0; i < uint8(8); i++ {
		BlockID = TreeData[i]
		SX = X + PE((i&1)<<2)
		SY = Y + PE((i&2)<<1)
		SZ = Z + PE(i&4)
		if BlockID != PlaceHolderBlock && BlockID != SKIPBLK {
			*Ops = append(*Ops, &Op{
				Level:   BuildLevelGroup,
				BlockID: BlockID,
				Pos:     Pos{SX, SY, SZ},
				Spawn:   SpawnData[i],
			})
		}
		if s.GroupData[i] != nil {
			if BlockID == SKIPBLK {
				s.GroupData[i].GetOps(SX, SY, SZ, PlaceHolderBlock, Ops)
			} else {
				s.GroupData[i].GetOps(SX, SY, SZ, s.MajorData[i], Ops)
			}
		}
	}
}

// 4096 blocks 16x16x16
type SubChunkStorage struct {
	Oct
	SectionData [8]*Section
}

func (sc *SubChunkStorage) New() subChunk.SubChunk {
	return &SubChunkStorage{}
}

func (sc *SubChunkStorage) Set(X, Y, Z uint8, blk BLOCKID) {
	s := ((X & 8) >> 3) | ((Y & 8) >> 2) | ((Z & 8) >> 1)
	if sc.SectionData[s] == nil {
		sc.SectionData[s] = newSection()
	}
	sc.SectionData[s].Set(
		((X&4)>>2)|((Y&4)>>1)|(Z&4),
		((X&2)>>1)|(Y&2)|((Z&2)<<1),
		(X&1)|((Y&1)<<1)|((Z&1)<<2),
		blk,
	)
}

func (s *SubChunkStorage) Finish() {
	i := uint8(0)
	for i = 0; i < uint8(8); i++ {
		if s.SectionData[i] != nil {
			s.SectionData[i].Finish()
			s.MajorData[i] = s.SectionData[i].Major
		} else {
			s.MajorData[i] = AIRBLK
		}
	}
	s.Oct.Finish(SubChunkCanUseFillThres)
}

func (sc *SubChunkStorage) GetOps(X, Y, Z PE, Ops *[]*Op) {
	PlaceHolderBlock := AIRBLK
	if sc.SpecialCase == ALLSAME {
		if sc.Major != AIRBLK {
			*Ops = append(*Ops, &Op{
				Level:   BuildLevelSection,
				BlockID: sc.Major,
				Pos:     Pos{X, Y, Z},
				Spawn:   SPAWN_X | SPAWN_Y | SPAWN_Z,
			})
			PlaceHolderBlock = sc.Major
		}
	}

	TreeData, SpawnData := sc.Oct.Spawn(PlaceHolderBlock)
	i := uint8(0)
	var BlockID BLOCKID
	var SX, SY, SZ PE
	for i = 0; i < uint8(8); i++ {
		BlockID = TreeData[i]
		SX = X + PE((i&1)<<3)
		SY = Y + PE((i&2)<<2)
		SZ = Z + PE((i&4)<<1)
		if BlockID != PlaceHolderBlock && BlockID != SKIPBLK {
			*Ops = append(*Ops, &Op{
				Level:   BuildLevelSection,
				BlockID: BlockID,
				Pos:     Pos{SX, SY, SZ},
				Spawn:   SpawnData[i],
			})
		}
		if sc.SectionData[i] != nil {
			if BlockID == SKIPBLK {
				sc.SectionData[i].GetOps(SX, SY, SZ, PlaceHolderBlock, Ops)
			} else {
				sc.SectionData[i].GetOps(SX, SY, SZ, sc.MajorData[i], Ops)
			}
		}
	}
}
