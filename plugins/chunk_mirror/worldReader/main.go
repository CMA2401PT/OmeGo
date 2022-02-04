package main

import (
	"fmt"
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

func main() {
	WorldDir := "C:\\Users\\daiji\\AppData\\Local\\Packages\\Microsoft.MinecraftUWP_8wekyb3d8bbwe\\LocalState\\games\\com.mojang\\minecraftWorlds\\rbj8YQKoAAA="
	WorldProvider, _ := reflect_provider.New(WorldDir, reflect_world.Overworld)
	blockEntities, _ := WorldProvider.LoadBlockNBT(reflect_world.ChunkPos{0, 0})
	for _, data := range blockEntities {
		pos := blockPosFromNBT(data)
		fmt.Println(pos, " ", data)
	}
}
