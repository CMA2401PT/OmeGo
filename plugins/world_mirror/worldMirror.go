package world_mirror

import (
	"bytes"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	reflect_protocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	reflect_packet "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"main.go/minecraft"
	"main.go/minecraft/protocol"
	"main.go/minecraft/protocol/login"
	"main.go/minecraft/protocol/packet"
	"main.go/plugins/chunk_mirror"
	provider_world "main.go/plugins/chunk_mirror/server/world"
	reflect_chunk "main.go/plugins/chunk_mirror/server/world/chunk"
	"main.go/plugins/world_mirror/server"
	"main.go/plugins/world_mirror/server/session"
	"main.go/plugins/world_mirror/server/world"
	"main.go/task"
	"time"
)

type Processor struct {
	cm                *chunk_mirror.ChunkMirror
	taskIO            *task.TaskIO
	log               func(isJson bool, data string)
	reflect           bool
	closeLock         chan int
	gameData          minecraft.GameData
	clientData        login.ClientData
	RuntimeID         uint64
	ServerHandle      *server.ServerHandle
	cachedPackets     *CachedPackets
	currentPos        mgl32.Vec3
	needUpdate        bool
	StartTime         time.Time
	TickOffest        uint64
	Tick              uint64
	PacketsToTransfer []reflect_packet.Packet
	TeleportingCount  int
	updateAtTick      uint64
	currentSession    *session.Session
}

func (p *Processor) process(cmd []string) {
	if cmd[0] == "reflect" {
		if p.reflect {
			fmt.Println("Already Reflecting")
		} else {
			p.beginReflect()
		}
	}
}

type PlayerAuthInputBridgeHandler struct {
	taskIO         *task.TaskIO
	p              *Processor
	posUpdataFn    func(pos mgl32.Vec3)
	defaultHandler session.PacketHandler
}

func (h *PlayerAuthInputBridgeHandler) Handle(p reflect_packet.Packet, s *session.Session) error {
	h.p.currentSession = s
	pk := p.(*reflect_packet.PlayerAuthInput)
	isTeleporting := false
	if h.p.needUpdate && h.p.TeleportingCount == 0 && pk.Tick == h.p.updateAtTick {
		if pk.Position.Sub(h.p.currentPos).Len() > 16 {
			h.p.TeleportingCount = 20
			isTeleporting = true
			time.Sleep(time.Millisecond * 100)
			fmt.Printf("Teleporting!")
		}
	}

	blockActions := make([]protocol.PlayerBlockAction, 0)
	for _, a := range pk.BlockActions {
		blockActions = append(blockActions, protocol.PlayerBlockAction{
			a.Action,
			protocol.BlockPos(a.BlockPos),
			a.Face})
	}
	//fmt.Println(pk.Tick)
	if h.p.TickOffest == 0 {
		h.p.TickOffest = uint64(time.Now().Sub(h.p.StartTime).Milliseconds()/50) - pk.Tick
	}
	h.p.Tick = pk.Tick

	//fmt.Println(tick)
	spk := &packet.PlayerAuthInput{
		Pitch:               pk.Pitch,
		Yaw:                 pk.Yaw,
		Position:            pk.Position,
		MoveVector:          pk.MoveVector,
		HeadYaw:             pk.HeadYaw,
		InputData:           pk.InputData,
		InputMode:           pk.InputMode,
		PlayMode:            pk.PlayMode,
		GazeDirection:       pk.GazeDirection,
		Tick:                h.p.TickOffest + pk.Tick,
		Delta:               pk.Delta,
		ItemInteractionData: protocol.UseItemTransactionData{},
		ItemStackRequest:    protocol.ItemStackRequest{},
		BlockActions:        blockActions,
	}

	//fmt.Println(spk)
	//fmt.Println(spk.Tick)
	//s.Conn.WritePacket(packet.Respawn{
	//	Position:        mgl32.Vec3{},
	//	State:           0,
	//	EntityRuntimeID: 0,
	////})
	//if h.p.TeleportingCount > 3 {
	//	fmt.Println("Freezing ", h.p.TeleportingCount)
	//	// 欺骗网易服务器自已经被tp完成
	//	// 模拟服务器暂时不更新位置
	//	spk.Position = h.p.currentPos
	//	h.p.TeleportingCount -= 1
	//} else if h.p.TeleportingCount <= 3 && h.p.TeleportingCount != 0 {
	//	fmt.Println("Server Update! ", h.p.TeleportingCount)
	//	spk.Position = h.p.currentPos
	//	h.p.TeleportingCount -= 1
	//	// 模拟服务器更新位置
	//	pk.Position = h.p.currentPos
	//	for _, p := range h.p.ServerHandle.Server.Players() {
	//		p.Teleport(mgl64.Vec3{
	//			float64(h.p.currentPos.X()),
	//			float64(h.p.currentPos.Y()),
	//			float64(h.p.currentPos.Z()),
	//		})
	//	}
	//} else {
	//	h.defaultHandler.Handle(pk, s)
	//}
	//if h.p.TeleportingCount <= 3 {
	//	// 模拟客户端更新位置
	//	pks := h.p.PacketsToTransfer
	//	h.p.PacketsToTransfer = make([]reflect_packet.Packet, 0)
	//	for _, pk := range pks {
	//		s.WritePacket(pk)
	//	}
	//}
	//if h.p.TeleportingCount != 0 {
	//	fmt.Println("Teleporting Count ", h.p.TeleportingCount)
	//}

	if !h.p.needUpdate {
		h.defaultHandler.Handle(pk, s)
		h.taskIO.ShieldIO.SendNoLock(spk)
	} else {
		fmt.Printf("Pos Update! %v\n", h.p.currentPos)
		h.p.needUpdate = false
		pk.Position = h.p.currentPos
		h.defaultHandler.Handle(pk, s)
		if isTeleporting {
			for _, p := range h.p.ServerHandle.Server.Players() {
				p.Teleport(mgl64.Vec3{
					float64(h.p.currentPos.X()),
					float64(h.p.currentPos.Y()),
					float64(h.p.currentPos.Z()),
				})
			}
		}

		//h.defaultHandler.Handle(pk, s)
		//h.posUpdataFn(h.p.currentPos)
		//spk.Position = h.p.currentPos
		//h.taskIO.ShieldIO.SendNoLock(spk)
	}

	pks := h.p.PacketsToTransfer
	h.p.PacketsToTransfer = make([]reflect_packet.Packet, 0)
	for _, pk := range pks {
		s.WritePacket(pk)
	}

	return nil
}

type PlayerActionHandler struct {
	taskIO *task.TaskIO
	p      *Processor
}

func (h *PlayerActionHandler) Handle(p reflect_packet.Packet, s *session.Session) error {
	pk := p.(*reflect_packet.PlayerAction)
	h.taskIO.ShieldIO.SendNoLock(&packet.PlayerAction{
		EntityRuntimeID: pk.EntityRuntimeID,
		ActionType:      pk.ActionType,
		BlockPosition:   protocol.BlockPos(pk.BlockPosition),
		BlockFace:       pk.BlockFace,
	})
	return nil
}

type AnimateHandler struct {
	taskIO *task.TaskIO
	p      *Processor
}

func (h *AnimateHandler) Handle(p reflect_packet.Packet, s *session.Session) error {
	pk := p.(*reflect_packet.Animate)
	h.taskIO.ShieldIO.SendNoLock(&packet.Animate{
		ActionType:      pk.ActionType,
		EntityRuntimeID: h.p.RuntimeID,
		BoatRowingTime:  pk.BoatRowingTime,
	})
	return nil
}

type InventoryTransactionHandler struct {
	taskIO *task.TaskIO
	p      *Processor
}

func (h *InventoryTransactionHandler) Handle(p reflect_packet.Packet, s *session.Session) error {
	LegacySetItemSlots := make([]protocol.LegacySetItemSlot, 0)
	//Actions := make([]protocol.InventoryAction, 0)

	pk := p.(*reflect_packet.InventoryTransaction)
	for _, s := range pk.LegacySetItemSlots {
		LegacySetItemSlots = append(LegacySetItemSlots, protocol.LegacySetItemSlot{
			ContainerID: s.ContainerID,
			Slots:       s.Slots,
		})
	}

	origTransactionData := pk.TransactionData
	var DeReflectTransactionData protocol.InventoryTransactionData
	switch origTransactionData.(type) {
	case nil, *reflect_protocol.NormalTransactionData:
		DeReflectTransactionData = &protocol.NormalTransactionData{}
	case *reflect_protocol.MismatchTransactionData:
		DeReflectTransactionData = &protocol.MismatchTransactionData{}
	case *reflect_protocol.UseItemTransactionData:
		oT := origTransactionData.(*reflect_protocol.UseItemTransactionData)
		DeReflectTransactionData = &protocol.UseItemTransactionData{
			LegacyRequestID:    oT.LegacyRequestID,
			LegacySetItemSlots: DeReflectLegacySetItemSlots(oT.LegacySetItemSlots),
			Actions:            DeReflectInventoryActions(oT.Actions),
			ActionType:         oT.ActionType,
			BlockPosition:      protocol.BlockPos(oT.BlockPosition),
			BlockFace:          oT.BlockFace,
			HotBarSlot:         oT.HotBarSlot,
			HeldItem:           DeReflectItemInstance(oT.HeldItem),
			Position:           oT.Position,
			ClickedPosition:    oT.ClickedPosition,
			BlockRuntimeID:     chunk_mirror.BlockDeReflectMapping[uint32(oT.BlockRuntimeID)],
		}
	case *reflect_protocol.UseItemOnEntityTransactionData:
		oT := origTransactionData.(*reflect_protocol.UseItemOnEntityTransactionData)
		DeReflectTransactionData = &protocol.UseItemOnEntityTransactionData{
			TargetEntityRuntimeID: oT.TargetEntityRuntimeID,
			ActionType:            oT.ActionType,
			HotBarSlot:            oT.HotBarSlot,
			HeldItem:              DeReflectItemInstance(oT.HeldItem),
			Position:              oT.Position,
			ClickedPosition:       oT.ClickedPosition,
		}
	case *reflect_protocol.ReleaseItemTransactionData:
		oT := origTransactionData.(*reflect_protocol.ReleaseItemTransactionData)
		DeReflectTransactionData = &protocol.ReleaseItemTransactionData{
			ActionType:   oT.ActionType,
			HotBarSlot:   oT.HotBarSlot,
			HeldItem:     DeReflectItemInstance(oT.HeldItem),
			HeadPosition: oT.HeadPosition,
		}
	}

	//for _, s := range pk.Actions {
	//	Actions = append(Actions, protocol.InventoryAction{
	//		SourceType:    s.SourceType,
	//		WindowID:      s.WindowID,
	//		SourceFlags:   s.SourceFlags,
	//		InventorySlot: s.InventorySlot,
	//		OldItem:       s.OldItem{},
	//		NewItem:       s.NewItem,
	//	})
	//}
	h.taskIO.ShieldIO.SendNoLock(&packet.InventoryTransaction{
		LegacyRequestID:    pk.LegacyRequestID,
		LegacySetItemSlots: LegacySetItemSlots,
		Actions:            DeReflectInventoryActions(pk.Actions),
		TransactionData:    DeReflectTransactionData,
	})
	return nil
}

func (p *Processor) beginReflect() {
	fmt.Println("Reflecting Start")
	p.gameData = p.taskIO.ShieldIO.GameData()
	p.clientData = p.taskIO.ShieldIO.ClientData()
	p.RuntimeID = p.gameData.EntityRuntimeID
	startData := &server.StartData{
		InjectFns: &server.InjectFns{
			ChatSubscriber: &server.ChatSubscriber{ChatCb: func(a ...interface{}) {}},
			GetNetEaseGameData: func() minecraft.GameData {
				return p.taskIO.ShieldIO.GameData()
			},
			GetNetEaseClientData: func() login.ClientData {
				return p.taskIO.ShieldIO.ClientData()
			},
			GetNetEaseIdentityData: func() login.IdentityData {
				return p.taskIO.ShieldIO.GetConn().IdentityData()
			},
			GetProvider: p.GetProvider,
			GetPos: func() mgl32.Vec3 {
				return p.currentPos
			},
			SessionInjectFns: &session.InjectFns{
				PlayerAuthInputHandler: &PlayerAuthInputBridgeHandler{
					p.taskIO,
					p,
					func(pos mgl32.Vec3) { fmt.Println("Update Fn Not filled!") },
					&session.PlayerAuthInputHandler{},
				},
				PlayerActionInputHandler: &PlayerActionHandler{
					taskIO: p.taskIO,
					p:      p,
				},
				InventoryTransactionHandler: &InventoryTransactionHandler{
					taskIO: p.taskIO,
					p:      p,
				},
				AnimateHandler: &AnimateHandler{
					taskIO: p.taskIO,
					p:      p,
				},
				RuntimeID: p.RuntimeID,
			},
		},
	}
	p.ServerHandle = server.StartWithData(startData)
	p.reflect = true

	//defer p.close()
}

func (o *Processor) handleNeteasePacket(pk packet.Packet) {
	switch p := pk.(type) {
	case *packet.AvailableActorIdentifiers:
		o.cachedPackets.AvailableActorIdentifiers = append(o.cachedPackets.AvailableActorIdentifiers, p)
	case *packet.PlayerList:
		o.cachedPackets.PlayerList = append(o.cachedPackets.PlayerList, p)
	case *packet.MovePlayer:
		//fmt.Printf("Player Move ! %v %v %v\n ", p.Position, p.EntityRuntimeID, o.RuntimeID, p.Tick)
		if p.EntityRuntimeID == o.RuntimeID {
			fmt.Printf("Player Me Move ! %v %v %v\n ", p.Position, p.Tick)
			o.currentPos = p.Position
			o.needUpdate = true
			o.PacketsToTransfer = append(o.PacketsToTransfer, &reflect_packet.MovePlayer{
				EntityRuntimeID:          1,
				Position:                 p.Position,
				Pitch:                    p.Pitch,
				Yaw:                      p.Yaw,
				HeadYaw:                  p.HeadYaw,
				Mode:                     p.Mode,
				OnGround:                 p.OnGround,
				RiddenEntityRuntimeID:    p.RiddenEntityRuntimeID,
				TeleportCause:            p.TeleportCause,
				TeleportSourceEntityType: p.TeleportSourceEntityType,
				Tick:                     o.Tick + 1,
			})
			//o.Time = int64(p.Tick)
			//o.StartTime = time.Now()
			//o.currentSession
		}
	case *packet.Respawn:
		//fmt.Printf("Player Respawn ! %v %v %v\n ", p.Position, p.EntityRuntimeID, o.RuntimeID)
		if p.EntityRuntimeID == o.RuntimeID {
			fmt.Printf("Player Me Respawn ! \n")
			o.currentPos = p.Position
			o.updateAtTick = o.Tick + 1
			o.needUpdate = true
			//o.PacketsToTransfer = append(o.PacketsToTransfer, &reflect_packet.Respawn{
			//	Position:        p.Position,
			//	State:           p.State,
			//	EntityRuntimeID: 1,
			//})
		}
	case *packet.CorrectPlayerMovePrediction:
		fmt.Printf("Correct Move Prediction ! \n")
		o.currentPos = p.Position
		o.updateAtTick = o.Tick + 1
		o.needUpdate = true
		//o.StartTime = time.Now()
		//o.TickOffest = p.Tick - o.Tick
		o.TickOffest = p.Tick - o.Tick
		//o.TickOffest = p.Tick - o.Tick
		o.PacketsToTransfer = append(o.PacketsToTransfer, &reflect_packet.CorrectPlayerMovePrediction{
			Position: p.Position,
			Delta:    p.Delta,
			OnGround: p.OnGround,
			Tick:     o.Tick,
		})
	case *packet.UpdateAttributes:
		if p.EntityRuntimeID == o.RuntimeID {
			attrs := make([]reflect_protocol.Attribute, 0)
			for _, a := range p.Attributes {
				attrs = append(attrs, reflect_protocol.Attribute{
					Name:    a.Name,
					Value:   a.Value,
					Max:     a.Max,
					Min:     a.Min,
					Default: a.Default,
				})
			}
			fmt.Printf("Player UpdateAttributes %v! \n", p)
			//T := uint64(10)
			//if o.Tick != 0 {
			//	T = o.Tick
			//}
			//o.PacketsToTransfer = append(o.PacketsToTransfer, &reflect_packet.UpdateAttributes{
			//	EntityRuntimeID: 1,
			//	Attributes:      attrs,
			//	Tick:            T,
			//})
		}

	}
}

func (p *Processor) close() {
	p.reflect = false
	close(p.closeLock)
	p.closeLock = make(chan int)
}

type CachedPackets struct {
	AvailableActorIdentifiers []*packet.AvailableActorIdentifiers
	PlayerList                []*packet.PlayerList
}

func (p *Processor) GetProvider() (world.Provider, error) {
	return NewReflectProvider(p.cm), nil
}

func (p *Processor) onNewChunk(pos provider_world.ChunkPos, cdata *provider_world.ChunkData) {
	if p.currentSession != nil {
		chunkBuf := &bytes.Buffer{}
		data := reflect_chunk.Encode(cdata.Chunk, reflect_chunk.NetworkEncoding)
		for i := range data.SubChunks {
			_, _ = chunkBuf.Write(data.SubChunks[i])
		}
		_, _ = chunkBuf.Write(append(make([]byte, 512), data.Biomes...))
		chunkBuf.WriteByte(0)
		enc := nbt.NewEncoderWithEncoding(chunkBuf, nbt.NetworkLittleEndian)
		for bp, b := range cdata.E {
			if n, ok := b.(world.NBTer); ok {
				d := n.EncodeNBT()
				d["x"], d["y"], d["z"] = int32(bp[0]), int32(bp[1]), int32(bp[2])
				_ = enc.Encode(d)
			}
		}
		p.currentSession.WritePacket(&reflect_packet.LevelChunk{
			ChunkX:        pos[0],
			ChunkZ:        pos[1],
			SubChunkCount: uint32(len(data.SubChunks)),
			RawPayload:    append([]byte(nil), chunkBuf.Bytes()...),
		})
	}
}
func (p *Processor) inject() {
	p.reflect = false
	p.closeLock = make(chan int)
	p.taskIO.ShieldIO.AddSessionTerminateCallBack(p.close)
	p.PacketsToTransfer = make([]reflect_packet.Packet, 0)
	p.cachedPackets = &CachedPackets{
		AvailableActorIdentifiers: make([]*packet.AvailableActorIdentifiers, 0),
		PlayerList:                make([]*packet.PlayerList, 0),
	}
	p.taskIO.ShieldIO.AddNewPacketCallback(p.handleNeteasePacket)
	p.taskIO.ShieldIO.AddInitCallBack(func(conn *minecraft.Conn) {
		p.currentPos = conn.GameData().PlayerPosition
		p.StartTime = conn.GameData().StartTime
		p.TickOffest = 0
		p.Tick = 0
	})
	//p.cm.RegChunkListener(func(X, Z int) bool {
	//	return true
	//}, p.onNewChunk)
}
