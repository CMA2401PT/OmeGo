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

type Chunk16 struct {
	X16    define.PE
	Z16    define.PE
	Chunks []*Chunk
}

func (cs *Chunk16) Len() int {
	return len(cs.Chunks)
}

func (cs *Chunk16) Less(i, j int) bool {
	d1 := math.Pow(float64(cs.Chunks[i].X-cs.X16), 2) + math.Pow(float64(cs.Chunks[i].Z-cs.Z16), 2)
	d2 := math.Pow(float64(cs.Chunks[j].X-cs.X16), 2) + math.Pow(float64(cs.Chunks[j].Z-cs.Z16), 2)
	return d1 < d2
}

func (cs *Chunk16) Swap(i, j int) {
	t := cs.Chunks[i]
	cs.Chunks[i] = cs.Chunks[j]
	cs.Chunks[j] = t
}

func (cs *Chunk16) Order() []*Chunk {
	sort.Sort(cs)
	return cs.Chunks
}

type Chunk16S map[[2]define.PE]*Chunk16

func (s Chunk16S) AddChunk(c *Chunk) {
	X16 := c.X >> 5
	Z16 := c.Z >> 5
	c16, hasK := s[[2]define.PE{X16, Z16}]
	if !hasK {
		c16 = &Chunk16{
			X16:    X16<<5 + 1<<4,
			Z16:    Z16<<5 + 1<<4,
			Chunks: make([]*Chunk, 0),
		}
		s[[2]define.PE{X16, Z16}] = c16
	}
	c16.Chunks = append(c16.Chunks, c)
}

func (s Chunk16S) Order() []*Chunk16 {
	chunk16s := make([]*Chunk16, 0)
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
