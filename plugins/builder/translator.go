package builder

import (
	"main.go/plugins/builder/define"
)

//type GroupRemapper struct {
//	Mapping   define.BlockDescribe2BlockIDMapping
//	Remapping map[define.BLOCKID]define.BLOCKID
//}

type Translator struct {
	Group   map[int]map[define.BLOCKID]define.BLOCKID
	Palette define.BlockID2BlockDescribeMapping
	LookUp  define.BlockDescribe2BlockIDMapping
}

func (t *Translator) Remap(group int, orig define.BLOCKID) define.BLOCKID {
	return t.Group[group][orig]
}

func (t *Translator) NewRemapObject(group int, blk define.BlockDescribe, blkId define.BLOCKID) {
	g, hask := t.Group[group]
	if !hask {
		g := make(map[define.BLOCKID]define.BLOCKID)
		// GroupRemapper{
		//Mapping:   make(define.BlockDescribe2BlockIDMapping),
		//Remapping: make(map[define.BLOCKID]define.BLOCKID),
		t.Group[group] = g
	}
	newID, hask := t.LookUp[blk]
	if !hask {
		newID := len(t.Palette)
		t.Palette = append(t.Palette, &blk)
		t.LookUp[blk] = define.BLOCKID(newID)
		g[blkId] = define.BLOCKID(newID)
	} else {
		g[blkId] = define.BLOCKID(newID)
	}

}

func NewTranslator() *Translator {
	ret := &Translator{
		Group:   make(map[int]map[define.BLOCKID]define.BLOCKID),
		Palette: make(define.BlockID2BlockDescribeMapping, 0),
		LookUp:  make(map[define.BlockDescribe]define.BLOCKID),
	}
	ret.LookUp[define.BlockDescribe{
		Name: "air",
		Meta: 0,
	}] = define.AIRBLK
	ret.Palette[define.AIRBLK] = &define.BlockDescribe{
		Name: "air",
		Meta: 0,
	}
	return ret
}
