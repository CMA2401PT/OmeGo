package plain

import (
	. "main.go/plugins/builder/define"
	"main.go/plugins/builder/ir/subChunk"
)

const ConvertThres = 16 * 16

type SparseBlock struct {
	x, y, z uint8
	blk     BLOCKID
}

type Storage struct {
	elements     uint16
	SparseMatrix []*SparseBlock
	DenseMatrix  *[16 * 16 * 16]BLOCKID
}

func (s *Storage) New() subChunk.SubChunk {
	return &Storage{
		SparseMatrix: make([]*SparseBlock, 0),
	}
}

func (s *Storage) Set(X, Y, Z uint8, blk BLOCKID) {
	s.elements += 1
	if s.elements < ConvertThres {
		s.SparseMatrix = append(s.SparseMatrix, &SparseBlock{X, Y, Z, blk})
		return
	} else if s.elements == ConvertThres {
		s.DenseMatrix = &[16 * 16 * 16]BLOCKID{}
		for _, sBlk := range s.SparseMatrix {
			s.DenseMatrix[int(uint16(sBlk.x)|uint16(sBlk.y)<<8|uint16(sBlk.z)<<4)] = sBlk.blk
		}
		s.SparseMatrix = nil
	}
	s.DenseMatrix[int(uint16(X)|uint16(Y)<<8|uint16(Z)<<4)] = blk
}

func (s *Storage) Finish() {
}

func (s *Storage) GetOps(X, Y, Z PE, Ops *[]*BlockOp) {
	if s.SparseMatrix != nil {
		for _, sBlk := range s.SparseMatrix {
			*Ops = append(*Ops, &BlockOp{
				Pos:     Pos{X + PE(sBlk.x), Y + PE(sBlk.y), Z + PE(sBlk.z)},
				BlockID: sBlk.blk,
			})
		}
	}
	if s.DenseMatrix != nil {
		for pos, blk := range s.DenseMatrix {
			if blk != AIRBLK {
				*Ops = append(*Ops, &BlockOp{
					Pos: Pos{
						X + PE(pos&0xf),
						Y + PE((pos>>8)&0xf),
						Z + PE((pos>>4)&0xf),
					},
					BlockID: blk,
				})
			}
		}
	}
}
