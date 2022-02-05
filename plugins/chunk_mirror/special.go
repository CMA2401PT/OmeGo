package chunk_mirror

import (
	"fmt"
	"main.go/minecraft/protocol/packet"
	reflect_block "main.go/plugins/chunk_mirror/server/block"
	reflect_world "main.go/plugins/chunk_mirror/server/world"
	reflect_provider "main.go/plugins/chunk_mirror/server/world/mcdb"
	"sync"
)

type SpecialData struct {
	cm   *ChunkMirror
	maps map[int64]*MapPair
	Mu   sync.Mutex
}

type MapPair struct {
	m *NbtMAP
	//nbt map[string]interface{}
	obj      *reflect_block.ItemFrame
	p        reflect_world.ChunkPos
	decItems []*NbtMapDecoration
}

func (s *SpecialData) New(cm *ChunkMirror) {
	s.cm = cm
	s.Mu = sync.Mutex{}
	s.maps = make(map[int64]*MapPair)
	cm.taskIO.AddPacketTypeCallback(packet.IDClientBoundMapItemData, s.handledMapItemData)
}

func (s *SpecialData) hasPath(nbt interface{}, ps []string) interface{} {
	if len(ps) == 0 {
		return nbt
	}
	nbtMap, ok := nbt.(map[string]interface{})
	if !ok {
		return nil
	}
	childNbt, hasK := nbtMap[ps[0]]
	if !hasK {
		return nil
	}
	return s.hasPath(childNbt, ps[1:])
}

func (s *SpecialData) handledMapItemData(p packet.Packet) {
	pk, ok := p.(*packet.ClientBoundMapItemData)
	if !ok {
		return
	}
	if pk.Height != 128 || pk.Width != 128 {
		return
	}
	uuid := pk.MapID
	s.Mu.Lock()
	defer s.Mu.Unlock()
	_, hasK := s.maps[uuid]
	if !hasK {
		fmt.Printf("Get Map from Unknown %v\n", uuid)
		return
	} else {
		//fmt.Printf("Get Map Data of %v\n", uuid)
	}
	//if pk.Height != 0 {
	//	fmt.Println(pk)
	//}
	//fmt.Println(pk)

	nbtMap := &NbtMAP{offsetX: 128 * (10000 + 2401 - 49), offsetZ: 128 * (10000 - 2401 + 49)}
	err := nbtMap.FromPacket(pk)
	if err != nil {
		return
	}
	//halfWidth := int32(64 << (nbtMap.Scale - 1))
	//Items := make([]NbtMapDecoration, 0)
	//for _, decItem := range s.maps[uuid].decItems {
	//	decItem.Data.X = nbtMap.XCenter - halfWidth - 63 + decItem.Key.BlockX
	//	decItem.Data.Y = nbtMap.ZCenter - halfWidth - 63 + decItem.Key.BlockZ
	//	Items = append(Items, *decItem)
	//}

	nbtMap.Decorations = nil

	s.maps[uuid].m = nbtMap
}

func (s *SpecialData) SaveMapDataToProvider(provider *reflect_provider.Provider) {
	for _, m := range s.maps {
		if m.m != nil {
			parentMaps := m.m.CreateParentMap()
			for i := len(parentMaps) - 1; i >= 0; i-- {
				ms := parentMaps[i]
				if err := ms.Save(provider); err != nil {
					fmt.Printf("Mirror-Chunk Special Data: on Save map (%v) @ (%v) (parent %v) to world (%v), an error occour (%v)\n", m.m.MapID, m.p, i, provider, err)
					return
				}
			}
			if err := m.m.Save(provider); err != nil {
				fmt.Printf("Mirror-Chunk Special Data: on Save map (%v) @ (%v) to world (%v), an error occour (%v)\n", m.m.MapID, m.p, provider, err)
			}
		}
	}
}

func (s *SpecialData) CheckBeeData(obj *reflect_block.BeeContainer, pos reflect_world.ChunkPos) {
	nbt := obj.EncodeNBT()
	//sth := s.hasPath(nbt, []string{"Occupants"})
	//if sth == nil {
	//	return
	//}
	//_nbt, ok := nbt["Occupants"].([]interface{})
	//if !ok {
	//	nbt["Occupants"] = make([]interface{}, 0)
	//	obj.DecodeNBT(nbt)
	//	return
	//}
	//fail := false
	//for _, bee := range _nbt {
	//	_beeNbt, ok := bee.(map[string]interface{})
	//	if !ok {
	//		fail = true
	//		break
	//	}
	//	_beeNbtSaveData := _beeNbt["SaveData"]
	//	__beeNbtSaveData, ok := _beeNbtSaveData.(map[string]interface{})
	//	if !ok {
	//		fail = true
	//		break
	//	}
	//	//_beeNbt["TicksLeftToStay"] = 20
	//	//uuid, ok := __beeNbtSaveData["UniqueID"].(int64)
	//	//if !ok {
	//	//	fail = true
	//	//	break
	//	//}
	//	//fmt.Println(uuid)
	//}
	//if fail {
	//	nbt["Occupants"] = make([]interface{}, 0)
	//	obj.DecodeNBT(nbt)
	//	return
	//}
	obj.DecodeNBT(nbt)
}

func (s *SpecialData) CheckMapData(obj *reflect_block.ItemFrame, pos reflect_world.ChunkPos) {
	nbt := obj.EncodeNBT()
	sth := s.hasPath(nbt, []string{"Item", "tag", "map_uuid"})
	if sth == nil {
		return
	}
	_nbt, ok := nbt["Item"].(map[string]interface{})
	if ok {
		__nbt, ok := _nbt["tag"].(map[string]interface{})
		if ok {
			__nbt["map_is_init"] = uint8(1)
		}
	}
	//posKey:=make(map[string]interface{})
	//posKey[]
	//
	//cube.Pos{nbt["x"],nbt["y"],nbt["z"]}
	//obj.Facing
	decItem := &NbtMapDecoration{
		Data: NbtMapDecorationsData{
			Rot:  int32(nbt["ItemRotation"].(float32)),
			Type: 1,
			X:    0,
			Y:    0,
		},
		Key: NbtMapDecorationsKey{
			BlockX: nbt["x"].(int32),
			BlockY: nbt["y"].(int32),
			BlockZ: nbt["z"].(int32),
			Type:   1,
		}}
	obj.DecodeNBT(nbt)
	uuid, ok := sth.(int64)
	if !ok {
		return
	}
	_, hasK := s.maps[uuid]
	if !hasK {
		s.maps[uuid] = &MapPair{m: nil, p: pos, obj: obj}
		s.cm.taskIO.ShieldIO.SendNoLock(&packet.MapInfoRequest{MapID: uuid})
		s.maps[uuid].decItems = make([]*NbtMapDecoration, 0)
	}
	s.maps[uuid].decItems = append(s.maps[uuid].decItems, decItem)
	//fmt.Printf("Get Map UUId: %v\n", uuid)
}
