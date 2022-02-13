package ir

import (
	"main.go/plugins/builder/define"
	"math"
	"sort"
)

// 去重
func uniqueArr(m []int) []int {
	d := make([]int, 0)
	tempMap := make(map[int]bool, len(m))
	for _, v := range m { // 以值作为键名
		if tempMap[v] == false {
			tempMap[v] = true
			d = append(d, v)
		}
	}
	return d
}

// 升序
func ascArr(e []int) []int {
	sort.Ints(e[:])
	return e
}

// 降序
func descArr(e []int) []int {
	sort.Sort(sort.Reverse(sort.IntSlice(e)))
	return e
}

type Chunk8 struct {
	X4     define.PE
	Z2     define.PE
	Chunks []*Chunk
}

func (cs *Chunk8) Len() int {
	return len(cs.Chunks)
}

func (cs *Chunk8) Less(i, j int) bool {
	d1 := math.Pow(float64(cs.Chunks[i].X-cs.X4), 2) + math.Pow(float64(cs.Chunks[i].Z-cs.Z2), 2)
	d2 := math.Pow(float64(cs.Chunks[j].X-cs.X4), 2) + math.Pow(float64(cs.Chunks[j].Z-cs.Z2), 2)
	return d1 < d2
}

func (cs *Chunk8) Swap(i, j int) {
	t := cs.Chunks[i]
	cs.Chunks[i] = cs.Chunks[j]
	cs.Chunks[j] = t
}

func (cs *Chunk8) Order() []*Chunk {
	sort.Sort(cs)
	return cs.Chunks
}

type Chunk8S map[[2]define.PE]*Chunk8

func (s Chunk8S) AddChunk(c *Chunk) {
	X4 := c.X >> 6
	Z2 := c.Z >> 5
	c16, hasK := s[[2]define.PE{X4, Z2}]
	if !hasK {
		c16 = &Chunk8{
			X4:     X4<<6 + 1<<5,
			Z2:     Z2<<5 + 1<<4,
			Chunks: make([]*Chunk, 0),
		}
		s[[2]define.PE{X4, Z2}] = c16
	}
	c16.Chunks = append(c16.Chunks, c)
}

func (s Chunk8S) Order() []*Chunk8 {
	chunk16s := make([]*Chunk8, 0)
	XS := make([]int, 0)
	ZS := make([]int, 0)
	for pos, _ := range s {
		XS = append(XS, int(pos[0]))
		ZS = append(ZS, int(pos[1]))
	}
	XS = uniqueArr(XS)
	XS = ascArr(XS)
	ZS = uniqueArr(ZS)
	ZS = ascArr(ZS)
	for turn, X := range XS {
		if turn%2 == 0 {
			for i := 0; i < len(ZS); i++ {
				c16, hasK := s[[2]define.PE{define.PE(X), define.PE(ZS[i])}]
				if hasK {
					chunk16s = append(chunk16s, c16)
				}
			}
		} else {
			for i := len(ZS); i != 0; i-- {
				c16, hasK := s[[2]define.PE{define.PE(X), define.PE(ZS[i-1])}]
				if hasK {
					chunk16s = append(chunk16s, c16)
				}
			}
		}
	}
	return chunk16s
}
