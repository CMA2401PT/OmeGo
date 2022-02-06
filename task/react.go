package task

import (
	"fmt"
	"github.com/google/uuid"
	"main.go/minecraft"
	"main.go/minecraft/protocol/packet"
)

type PacketCallBack struct {
	cbs   map[int]func(pk packet.Packet)
	count int
}

type CallBacks struct {
	packetsCBS             map[uint32]PacketCallBack
	onCmdFeedbackTmpCbS    map[uuid.UUID]func(output *packet.CommandOutput)
	onCmdFeedbackOnTmpCbs  []func()
	onCmdFeedbackOffTmpCbs []func()
}

func newCallbacks() *CallBacks {
	cbs := CallBacks{
		packetsCBS:             make(map[uint32]PacketCallBack),
		onCmdFeedbackTmpCbS:    make(map[uuid.UUID]func(output *packet.CommandOutput)),
		onCmdFeedbackOnTmpCbs:  make([]func(), 0),
		onCmdFeedbackOffTmpCbs: make([]func(), 0),
	}
	for packetID := range packet.NewPool() {
		cbs.packetsCBS[packetID] = PacketCallBack{cbs: make(map[int]func(pk packet.Packet)), count: 0}
	}
	return &cbs
}

// handle callbacks
func (io *TaskIO) AddPacketTypeCallback(packetID uint32, cb func(packet.Packet)) (func(), error) {
	packetCBS, ok := io.cbs.packetsCBS[packetID]
	if ok {
		c := packetCBS.count + 1
		_, hasK := packetCBS.cbs[c]
		for hasK {
			c += 1
			_, hasK = packetCBS.cbs[c]
		}
		packetCBS.count = c
		packetCBS.cbs[packetCBS.count] = cb
		removed := false
		deleteFn := func() {
			if removed {
				return
			}
			removed = true
			packetCBS, _ := io.cbs.packetsCBS[packetID]
			delete(packetCBS.cbs, c)
		}
		return deleteFn, nil
	} else {
		return nil, fmt.Errorf("do not have such packet ID (%v) for call back ", packetID)
	}
}

func (io *TaskIO) AddOnCmdFeedBackOnCb(cb func()) {
	io.cbs.onCmdFeedbackOnTmpCbs = append(io.cbs.onCmdFeedbackOnTmpCbs, cb)
}

func (io *TaskIO) AddOnCmdFeedBackOffCb(cb func()) {
	io.cbs.onCmdFeedbackOffTmpCbs = append(io.cbs.onCmdFeedbackOffTmpCbs, cb)
}

func (taskIO *TaskIO) onGameRuleChanged(pk *packet.GameRulesChanged) {
	for _, rule := range pk.GameRules {
		if rule.Name == "sendcommandfeedback" {
			sendCommandFeedBack := rule.Value.(bool)
			taskIO.Status.setCmdFB(sendCommandFeedBack)
			if sendCommandFeedBack && (len(taskIO.cbs.onCmdFeedbackOnTmpCbs) != 0) {
				cbs := taskIO.cbs.onCmdFeedbackOnTmpCbs
				taskIO.cbs.onCmdFeedbackOnTmpCbs = make([]func(), 0)
				for _, cb := range cbs {
					cb()
				}
			} else if len(taskIO.cbs.onCmdFeedbackOffTmpCbs) != 0 {
				cbs := taskIO.cbs.onCmdFeedbackOffTmpCbs
				taskIO.cbs.onCmdFeedbackOffTmpCbs = make([]func(), 0)
				for _, cb := range cbs {
					cb()
				}
			}
		}
	}
}

func (taskIO *TaskIO) onMCInit(conn *minecraft.Conn) {
	fmt.Println("Reactor: Start OnMCInit Tasks")
	gameData := conn.GameData()
	taskIO.EntityUniqueID = gameData.EntityUniqueID
	taskIO.identityData = conn.IdentityData()
	for _, rule := range gameData.GameRules {
		if rule.Name == "sendcommandfeedback" {
			sendCommandFeedBack := rule.Value.(bool)
			taskIO.Status.setCmdFB(sendCommandFeedBack)
			close(taskIO.initLock)
		}
	}
}

func (taskIO *TaskIO) onMCSessionTerminate() {
	fmt.Println("Reactor: Find MC Session Terminated")
}

func (cbs *CallBacks) activatePacketCallbacks(pk packet.Packet) {
	packetCBS, _ := cbs.packetsCBS[pk.ID()]
	for _, cb := range packetCBS.cbs {
		cb(pk)
		//delete(packetCBS.cbs, cbID)
	}
}

func (taskIO *TaskIO) newPacketFn(pk packet.Packet) {
	id := pk.ID()
	//_, hasK := taskIO.packetTypes[id]
	//if !hasK {
	//	//taskIO.packetTypes[id] = pk
	//	//fmt.Println(id)
	//	//	//fmt.Println(pk)
	//}
	switch id {
	case 143:
		//IDNetworkSettings
		break
	case 50:
		//IDInventorySlot
		break
	case 63:
		//IDPlayerList
		break
	case 10:
		//IDSetTime
		break
	case 60:
		//IDSetDifficulty
		break
	case 59:
		//IDSetCommandsEnabled
		break
	case 55:
		//IDAdventureSettings
		break
	case 72:
		//IDGameRulesChanged
		break
	case 122:
		// IDBiomeDefinitionList
		break
	case 119:
		//IDAvailableActorIdentifiers
		break
	case 160:
		//IDPlayerFog
		break
	case 29:
		//IDUpdateAttributes
		break
	case 163:
		//IDItemComponent
		break
	case 43:
		// IDSetSpawnPosition
		break
	case 145:
		// IDCreativeContent
		break
	case 49:
		// IDInventoryContent a lot
		break
	case 48:
		//IDPlayerHotBar
		break
	case 52:
		// IDCraftingData
		break
	case 76:
		// IDAvailableCommands
		break
	case 39:
		//IDSetActorData
		break
	case 121:
		//IDNetworkChunkPublisherUpdate a lot
		break
	case 58:
		// IDLevelChunk
		break

	default:
		//fmt.Println(id)
	}
	//fmt.Println(id)
	//if id != 143 && id != 49 && id != 58 && id != 111 && id != 121 && id != 40 && id != 19 && id != 27 && id != 39 {
	//	fmt.Println(id)
	//}

	switch p := pk.(type) {
	case *packet.CorrectPlayerMovePrediction:
		fmt.Println("Time Correct!")

	case *packet.SetCommandsEnabled:
		taskIO.Status.setCmdEnabled(p.Enabled)
	case *packet.GameRulesChanged:
		//fmt.Println("Reactor: GameRule Update")
		taskIO.onGameRuleChanged(p)
		break
	//case *packet.PlayerList:
	//	fmt.Println("Reactor: Recv Player List Update")
	case *packet.AdventureSettings:
		if taskIO.EntityUniqueID == p.PlayerUniqueID {
			taskIO.Status.setIsOP(p.CommandPermissionLevel > 0)
		}
	case *packet.AddPlayer:
		fmt.Println("Reactor: New Player Nearby")
	case *packet.CommandOutput:
		cb, ok := taskIO.cbs.onCmdFeedbackTmpCbS[p.CommandOrigin.UUID]
		if ok {
			delete(taskIO.cbs.onCmdFeedbackTmpCbS, p.CommandOrigin.UUID)
			cb(p)
		}
	case *packet.LevelChunk:
		// 这块即将被弃用，以后都使用 mirror_chunk 核心插件作为区块缓存提供者
		if taskIO.doCacheChunks {
			//fmt.Printf("New Chunk Arrival @ (%d,%d)\n", p.ChunkX, p.ChunkZ)
			//taskIO.Status.AddChunk(p)
		}
	}
	taskIO.cbs.activatePacketCallbacks(pk)
}
