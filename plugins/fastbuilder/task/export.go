package task

import (
	"fmt"
	"main.go/dragonfly/server/block/cube"
	"main.go/dragonfly/server/world"
	"main.go/dragonfly/server/world/chunk"
	"main.go/minecraft/protocol/packet"
	"main.go/plugins/fastbuilder/bdump"
	"main.go/plugins/fastbuilder/bridge"
	"main.go/plugins/fastbuilder/configuration"
	"main.go/plugins/fastbuilder/parsing"
	"main.go/plugins/fastbuilder/types"
	"main.go/plugins/fastbuilder/world_provider"
	world_logger "main.go/world/logger"
	"main.go/world/provider"
	"time"

	//"main.go/plugins/fastbuilder/world_provider"
	//"main.go/world/provider"
	"runtime"
	"strings"
)

type SolidSimplePos struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	Z int64 `json:"z"`
}

type SolidRet struct {
	BlockName  string         `json:"blockName"`
	Position   SolidSimplePos `json:"position"`
	StatusCode int64          `json:"statusCode"`
}

var ExportWaiter chan map[string]interface{}

func checkChunkInCache(pos world.ChunkPos) (c *chunk.Chunk, exists bool, err error) {
	var cacheitem *packet.LevelChunk
	hascacheitem := false
	bridge.BypassedTaskIO.Status.AccessChunkCache(func(m map[world.ChunkPos]*packet.LevelChunk) {
		cacheitem, hascacheitem = m[pos]
	})
	if hascacheitem {
		c, err := chunk.NetworkDecode(world_provider.AirRuntimeId, cacheitem.RawPayload, int(cacheitem.SubChunkCount))
		if err != nil {
			fmt.Printf("Failed to decode chunk: (%v)\n", err)
			return nil, true, err
		}
		//fmt.Printf("Hit Chunk: (%v,%v)", pos.X(), pos.Z())
		return c, true, nil
	}
	return nil, false, nil
}
func GetCacheChunkInterceptor() provider.LoadChunkInterceptorFn {
	return func(position world.ChunkPos) (c *chunk.Chunk, exists bool, err error) {
		return checkChunkInCache(position)
	}
}

func GetActiveChunkInterceptor() (func(), provider.LoadChunkInterceptorFn, error) {
	waitLock := make(chan int)
	isWaiting := true
	destroyFn, err := bridge.BypassedTaskIO.AddPacketTypeCallback(packet.IDLevelChunk, func(p packet.Packet) {
		if isWaiting {
			fmt.Println("Notify New Chunk Arrival")
			close(waitLock)
			isWaiting = false
		}
	})
	if err != nil {
		return nil, nil, err
	}
	interceptFN := func(position world.ChunkPos) (c *chunk.Chunk, exists bool, err error) {
		fmt.Printf("Activate Require Chunk @ (%v,%v)\n", position.X(), position.Z())
		bridge.BypassedTaskIO.SendCmd(fmt.Sprintf("tp @s %d 127 %d", position.X()*16, position.Z()*16))
		time.Sleep(time.Second * 1)
		for {
			c, e, r := checkChunkInCache(position)
			if e {
				return c, e, r
			}
			select {
			case <-waitLock:
			case <-time.After(time.Second):
				fmt.Printf("Wait Time out")
				if !isWaiting {
					waitLock = make(chan int)
					isWaiting = true
				}
				//fmt.Printf("Retry Require Chunk @ (%v,%v)\n", position.X(), position.Z())
				//bridge.BypassedTaskIO.SendCmdWithFeedBack(fmt.Sprintf("tp @s %d 127 %d", position.X()*16+1000, position.Z()*16+1000), func(respPk *packet.CommandOutput) {
				//	fmt.Println()
				//})
			}
			//
			//time.Sleep(2 * time.Second)
			//bridge.BypassedTaskIO.SendCmd(fmt.Sprintf("tp @s %d 127 %d", position.X()*16, position.Z()*16))

		}

	}
	return destroyFn, interceptFN, nil
}

func CreateExportTask(commandLine string) *Task {
	cfg, err := parsing.Parse(commandLine, configuration.GlobalFullConfig().Main())
	if err != nil {
		bridge.Tellraw(fmt.Sprintf("Failed to parse command: %v", err))
		return nil
	}
	beginPos := cfg.Position
	endPos := cfg.End
	if endPos.X-beginPos.X < 0 {
		temp := endPos.X
		endPos.X = beginPos.X
		beginPos.X = temp
	}
	if endPos.Y-beginPos.Y < 0 {
		temp := endPos.Y
		endPos.Y = beginPos.Y
		beginPos.Y = temp
	}
	if endPos.Z-beginPos.Z < 0 {
		temp := endPos.Z
		endPos.Z = beginPos.Z
		beginPos.Z = temp
	}
	tmpWorld := world.New(&world_logger.StubLogger{}, 32)
	//bridge.SendCommandWithCB()
	worldProvider := provider.New()
	worldProvider.AddInterceptor(GetCacheChunkInterceptor())
	destroyFN, activateInterceptor, err := GetActiveChunkInterceptor()
	if err != nil {
		bridge.Tellraw(fmt.Sprintf("Failed to Create Chunk Interceptor: %v", err))
		return nil
	}
	worldProvider.AddInterceptor(activateInterceptor)
	tmpWorld.Provider(worldProvider)
	go func() {
		bridge.Tellraw("EXPORT >> Exporting...")
		V := (endPos.X - beginPos.X + 1) * (endPos.Y - beginPos.Y + 1) * (endPos.Z - beginPos.Z + 1)
		blocks := make([]*types.RuntimeModule, V)
		counter := 0
		for x := beginPos.X; x <= endPos.X; x++ {
			for z := beginPos.Z; z <= endPos.Z; z++ {
				for y := beginPos.Y; y <= endPos.Y; y++ {
					blk := tmpWorld.Block(cube.Pos{x, y, z})
					runtimeId := world.LoadRuntimeID(blk)
					if runtimeId == world_provider.AirRuntimeId {
						continue
					}
					block, item := blk.EncodeBlock()
					var cbdata *types.CommandBlockData = nil
					var chestData *types.ChestData = nil
					if block == "chest" || strings.Contains(block, "shulker_box") {
						content := item["Items"].([]interface{})
						chest := make(types.ChestData, len(content))
						for index, iface := range content {
							i := iface.(map[string]interface{})
							name := i["Name"].(string)
							count := i["Count"].(uint8)
							damage := i["Damage"].(int16)
							slot := i["Slot"].(uint8)
							name_mcnk := name[10:]
							chest[index] = types.ChestSlot{
								Name:   name_mcnk,
								Count:  count,
								Damage: uint16(int(damage)),
								Slot:   slot,
							}
						}
						chestData = &chest
					}
					if strings.Contains(block, "command_block") {
						var mode uint32
						if block == "command_block" {
							mode = packet.CommandBlockImpulse
						} else if block == "repeating_command_block" {
							mode = packet.CommandBlockRepeat
						} else if block == "chain_command_block" {
							mode = packet.CommandBlockChain
						}
						cmd := item["Command"].(string)
						cusname := item["CustomName"].(string)
						exeft := item["ExecuteOnFirstTick"].(uint8)
						tickdelay := item["TickDelay"].(int32)
						aut := item["auto"].(uint8)
						trackoutput := item["TrackOutput"].(uint8)
						lo := item["LastOutput"].(string)
						//conditionalmode:=item["conditionalMode"].(uint8)
						data := item["data"].(int32)
						var conb bool
						if (data>>3)&1 == 1 {
							conb = true
						} else {
							conb = false
						}
						var exeftb bool
						if exeft == 0 {
							exeftb = true
						} else {
							exeftb = true
						}
						var tob bool
						if trackoutput == 1 {
							tob = true
						} else {
							tob = false
						}
						var nrb bool
						if aut == 1 {
							nrb = false
							//REVERSED!!
						} else {
							nrb = true
						}
						cbdata = &types.CommandBlockData{
							Mode:               mode,
							Command:            cmd,
							CustomName:         cusname,
							ExecuteOnFirstTick: exeftb,
							LastOutput:         lo,
							TickDelay:          tickdelay,
							TrackOutput:        tob,
							Conditional:        conb,
							NeedRedstone:       nrb,
						}
					}
					blocks[counter] = &types.RuntimeModule{
						BlockRuntimeId:   runtimeId,
						CommandBlockData: cbdata,
						ChestData:        chestData,
						Point: types.Position{
							X: x,
							Y: y,
							Z: z,
						},
					}
					counter++
				}
			}
		}
		tmpWorld.Close()
		destroyFN()
		blocks = blocks[:counter]
		runtime.GC()
		out := bdump.BDump{
			Blocks: blocks,
		}
		if strings.LastIndex(cfg.Path, ".bdx") != len(cfg.Path)-4 || len(cfg.Path) < 4 {
			cfg.Path += ".bdx"
		}
		bridge.Tellraw("EXPORT >> Writing output file")
		err, signerr := out.WriteToFile(cfg.Path)
		if err != nil {
			bridge.Tellraw(fmt.Sprintf("EXPORT >> ERROR: Failed to export: %v", err))
			return
		} else if signerr != nil {
			bridge.Tellraw(fmt.Sprintf("EXPORT >> Note: The file is unsigned since the following error was trapped: %v", signerr))
		} else {
			bridge.Tellraw(fmt.Sprintf("EXPORT >> File signed successfully"))
		}
		bridge.Tellraw(fmt.Sprintf("EXPORT >> Successfully exported your structure to %v", cfg.Path))
		runtime.GC()
	}()
	return nil
}
