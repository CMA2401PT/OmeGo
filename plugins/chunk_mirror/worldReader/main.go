package main

import (
	"bytes"
	"fmt"
	"image/color"
	"main.go/minecraft/nbt"
	"main.go/plugins/chunk_mirror/server/block/cube"
	reflect_world "main.go/plugins/chunk_mirror/server/world"
	reflect_provider "main.go/plugins/chunk_mirror/server/world/mcdb"
)

func blockPosFromNBT(data map[string]interface{}) cube.Pos {
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	xInterface, _ := data["x"]
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	yInterface, _ := data["y"]
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	zInterface, _ := data["z"]
	x, _ := xInterface.(int32)
	y, _ := yInterface.(int32)
	z, _ := zInterface.(int32)
	return cube.Pos{int(x), int(y), int(z)}
}

// 以下字段全部是猜的
type NbtMapDecorationsData struct {
	Rot  int32 `nbt:"rot"`  // 在那个放地图的方块上，旋转角度
	Type int32 `nbt:"type"` // 放地图的方块的类型
	X    int32 `nbt:"x"`    // 地图起点X与放地图的方块的偏移
	Y    int32 `nbt:"y"`    // 地图起点Z（Y）与放地图的方块的偏移
}
type NbtMapDecorationsKey struct {
	BlockX int32 `nbt:"blockX"` // 放地图方块的位置
	BlockY int32 `nbt:"blockY"`
	BlockZ int32 `nbt:"blockZ"`
	Type   int32 `nbt:"type"` // 类型
}
type NbtMapDecoration struct {
	Data NbtMapDecorationsData `nbt:"data"`
	Key  NbtMapDecorationsKey  `nbt:"key"`
}

type NbtMAP struct {
	MapID       int64        `nbt:"mapId"`
	ParentMapID int64        `nbt:"parentMapId"` // itself
	Colors      [65536]uint8 `nbt:"colors"`      // should be computed
	//Colors            interface{} `nbt:"colors"`

	LockedMap         byte               `nbt:"mapLocked"`
	Scale             byte               `nbt:"scale"`
	Dimension         byte               `nbt:"dimension"`
	FullyExplored     byte               `nbt:"fullyExplored"`     //1
	UnlimitedTracking byte               `nbt:"unlimitedTracking"` //0
	XCenter           int32              `nbt:"xCenter"`
	ZCenter           int32              `nbt:"zCenter"`
	Height            int16              `nbt:"height"`
	Width             int16              `nbt:"width"`
	Decorations       []NbtMapDecoration `nbt:"decorations"`
	pixels            [][]color.RGBA
}

func (m *NbtMAP) Save(p *reflect_provider.Provider) error {
	buf := bytes.NewBuffer(nil)
	enc := nbt.NewEncoderWithEncoding(buf, nbt.LittleEndian)
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("save entities: error encoding NBT: %w", err)
	}
	mapKey := []byte(fmt.Sprintf("map_%v", m.MapID))
	return p.DB.Put(mapKey, buf.Bytes(), nil)
}

func main() {

	//WorldDir := "C:\\Users\\daiji\\AppData\\Local\\Packages\\Microsoft.MinecraftUWP_8wekyb3d8bbwe\\LocalState\\games\\com.mojang\\minecraftWorlds\\MirrorChunk"
	//mapUUID := int64(-4294963993)
	WorldDir := "C:\\Users\\daiji\\AppData\\Local\\Packages\\Microsoft.MinecraftUWP_8wekyb3d8bbwe\\LocalState\\games\\com.mojang\\minecraftWorlds\\B2P9YcsaAQA="
	mapUUID := int64(-12884901881)
	WorldProvider, _ := reflect_provider.New(WorldDir, reflect_world.Overworld)
	//snapshot, _ := WorldProvider.DB.GetSnapshot()
	//iter := snapshot.NewIterator(nil, nil)
	//for iter.Next() {
	//	name := string(iter.Key()[:])
	//	fmt.Println(name)
	//}
	//beeUUid := -25769803767
	//WorldProvider.DB.Get()
	//fmt.Println(snapshot)
	//sth, err := WorldProvider.DB.Get([]byte("map"), nil)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(sth)
	//
	mapKey := []byte(fmt.Sprintf("map_%v", mapUUID))
	data, err := WorldProvider.DB.Get(mapKey, nil)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(data)
	dec := nbt.NewDecoderWithEncoding(buf, nbt.LittleEndian)
	m := NbtMAP{}
	err = dec.Decode(&m)
	if err != nil {
		panic(err)
	}
	//fmt.Println(m)
	err = m.Save(WorldProvider)
	if err != nil {
		panic(err)
	}
	Entity, _ := WorldProvider.LoadEntities(reflect_world.ChunkPos{0, 0})
	fmt.Println(Entity)
	blockEntities, _ := WorldProvider.LoadBlockNBT(reflect_world.ChunkPos{0, 0})
	for _, data := range blockEntities {
		pos := blockPosFromNBT(data)
		fmt.Println(pos, " ", data)
	}
}
