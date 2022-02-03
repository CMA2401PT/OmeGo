package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"main.go/plugins/chunk_mirror/server/block"
	reflect_world "main.go/plugins/chunk_mirror/server/world"
	reflect_chunk "main.go/plugins/chunk_mirror/server/world/chunk"
	"os"
)

var ReflectAirRID int
var NetEaseAirRID int

//go:embed runtimeIds.json
var runtimeJsonData []byte

type RichBlock struct {
	Name       string
	Val        int
	NeteaseRID int
	ReflectRID int
	Props      map[string]interface{}
}

type LegacyBlock [2]interface{}

func (p *LegacyBlock) Convert() *RichBlock {
	s, ok := p[0].(string)
	if !ok {
		panic("fail")
	}
	i, ok := p[1].(float64)
	if !ok {
		panic("fail")
	}
	return &RichBlock{Name: s, Val: int(i), Props: nil, ReflectRID: ReflectAirRID}
}

type IDGroup struct {
	Count int
	IDS   []*RichBlock
}

func NewIDGroup() *IDGroup {
	return &IDGroup{
		Count: 0,
		IDS:   make([]*RichBlock, 0),
	}
}
func (ig *IDGroup) AppendItem(p *RichBlock) {
	ig.IDS = append(ig.IDS, p)
}

func main() {
	_ReflectAirRID, _ := reflect_world.BlockRuntimeID(block.Air{})
	ReflectAirRID = int(_ReflectAirRID)
	NetEaseAirRID = 134
	runtimeIDData := make([]LegacyBlock, 0)
	err := json.Unmarshal(runtimeJsonData, &runtimeIDData)
	if err != nil {
		panic(err)
	}
	runtimeIDS := make([]*RichBlock, 0)
	groupedBlocks := make(map[string]*IDGroup)
	var i int
	var Id LegacyBlock
	for i, Id = range runtimeIDData {
		convertedID := Id.Convert()
		convertedID.Name = "minecraft:" + convertedID.Name
		convertedID.NeteaseRID = i
		runtimeIDS = append(runtimeIDS, convertedID)
		_, hasK := groupedBlocks[convertedID.Name]
		if !hasK {
			groupedBlocks[convertedID.Name] = NewIDGroup()
		}
		groupedBlocks[convertedID.Name].AppendItem(convertedID)

	}
	totalIds := i + 1

	notFounfName := make(map[string]bool)
	reflectRid := 0
	i = 0
	for {
		reflectRid = i
		i++
		reflectBlockName, props, found := reflect_chunk.RuntimeIDToState(uint32(reflectRid))
		if !found {
			break
		}
		_, hasK := groupedBlocks[reflectBlockName]
		if !hasK {
			_, recoredMiss := notFounfName[reflectBlockName]
			if !recoredMiss {
				fmt.Println(reflectBlockName, "Miss in Netease MC")
				notFounfName[reflectBlockName] = true
			}
			continue
		} else {
			g := groupedBlocks[reflectBlockName]
			if g.Count == len(g.IDS) {
				fmt.Println("Number variants Conflict! ", reflectBlockName, " ", g.Count)
				continue
				return
			}
			groupedBlocks[reflectBlockName].IDS[g.Count].Props = props
			groupedBlocks[reflectBlockName].IDS[g.Count].ReflectRID = reflectRid
			g.Count += 1
		}

	}
	writeBackData := make([]RichBlock, totalIds)
	notFoundInDF := 0
	for _, g := range groupedBlocks {
		for _, Id := range g.IDS {
			writeBackData[Id.NeteaseRID] = *Id
			if Id.ReflectRID == ReflectAirRID {
				notFoundInDF += 1
				fmt.Println("Miss in Official MC", Id)
			}
		}
	}
	fmt.Printf("%d Props Missed\n", notFoundInDF)
	fi, err := os.Create("richBlock.json")
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(fi)
	encoder.SetIndent("\t", "\t")
	encoder.Encode(struct {
		ReflectAirRID int
		NeteaseAirRID int
		RichBlocks    []RichBlock
	}{
		ReflectAirRID: ReflectAirRID,
		NeteaseAirRID: NetEaseAirRID,
		RichBlocks:    writeBackData,
	})
	fmt.Println("Ok!")
}
