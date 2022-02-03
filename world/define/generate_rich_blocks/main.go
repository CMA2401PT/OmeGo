package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"main.go/minecraft/nbt"
	"os"
)

//go:embed block_states(1.18).nbt
var blockStateData []byte

//go:embed runtimeIds.json
var runtimeJsonData []byte

type IDPairRaw [2]interface{}

func (p *IDPairRaw) Convert() *IDPair {
	s, ok := p[0].(string)
	if !ok {
		panic("fail")
	}
	i, ok := p[1].(float64)
	if !ok {
		panic("fail")
	}
	return &IDPair{Name: s, Val: int(i), Props: nil}
}

type IDPair struct {
	Name      string
	Val       int
	RuntimeID int
	Props     map[string]interface{}
	Version   int32
}

type IDGroup struct {
	Count int
	IDS   []*IDPair
}

func NewIDGroup() *IDGroup {
	return &IDGroup{
		Count: 0,
		IDS:   make([]*IDPair, 0),
	}
}
func (ig *IDGroup) AppendItem(p *IDPair) {
	ig.IDS = append(ig.IDS, p)
}

type blockState struct {
	Name       string                 `nbt:"name"`
	Properties map[string]interface{} `nbt:"states"`
	Version    int32                  `nbt:"version"`
}

func main() {
	runtimeIDData := make([]IDPairRaw, 0)
	err := json.Unmarshal(runtimeJsonData, &runtimeIDData)
	if err != nil {
		panic(err)
	}
	runtimeIDS := make([]*IDPair, 0)
	groupedBlocks := make(map[string]*IDGroup)
	var i int
	var Id IDPairRaw
	for i, Id = range runtimeIDData {
		convertedID := Id.Convert()
		convertedID.Name = "minecraft:" + convertedID.Name
		convertedID.RuntimeID = i
		runtimeIDS = append(runtimeIDS, convertedID)
		_, hasK := groupedBlocks[convertedID.Name]
		if !hasK {
			groupedBlocks[convertedID.Name] = NewIDGroup()
		}
		groupedBlocks[convertedID.Name].AppendItem(convertedID)

	}
	totalIds := i + 1

	var s blockState
	dec := nbt.NewDecoder(bytes.NewBuffer(blockStateData))
	missedInNetease := make(map[string]bool)
	for {
		if err := dec.Decode(&s); err != nil {
			break
		}
		blkName := s.Name
		_, hasK := groupedBlocks[blkName]
		if !hasK {
			_, recoredMiss := missedInNetease[blkName]
			if !recoredMiss {
				fmt.Println(blkName, "Miss in Netease MC")
				missedInNetease[blkName] = true
			}
			continue
		} else {
			g := groupedBlocks[blkName]
			if g.Count == len(g.IDS) {
				fmt.Println("Number variants Conflict! ", s, " ", g.Count)
				continue
				return
			}
			groupedBlocks[blkName].IDS[g.Count].Props = s.Properties
			groupedBlocks[blkName].IDS[g.Count].Version = s.Version
			g.Count += 1
		}
	}
	writeBackData := make([]IDPair, totalIds)
	for _, g := range groupedBlocks {
		for _, Id := range g.IDS {
			writeBackData[Id.RuntimeID] = *Id
			if Id.Props == nil {
				fmt.Println("Miss in Official MC", Id)
			}
		}
	}
	fi, err := os.Create("richBlocks.json")
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(fi)
	encoder.SetIndent("\t", "\t")
	encoder.Encode(writeBackData)
	fmt.Println("Ok!")
}
