package world_mirror

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	reflect_protocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	reflect_packet "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"main.go/minecraft"
	"main.go/minecraft/protocol"
	"main.go/minecraft/protocol/login"
	"main.go/minecraft/protocol/packet"
	"main.go/plugins/chunk_mirror"
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
	pk := p.(*reflect_packet.PlayerAuthInput)
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
	//})
	if !h.p.needUpdate {
		h.taskIO.ShieldIO.SendNoLock(spk)
		h.defaultHandler.Handle(pk, s)
	} else {
		fmt.Printf("Pos Update! %v\n", h.p.currentPos)
		h.p.needUpdate = false
		pk.Position = h.p.currentPos
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
		EntityRuntimeID: h.p.RuntimeID,
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

//func (h *InventoryTransactionHandler) Handle(p reflect_packet.Packet, s *session.Session) error {
//	LegacySetItemSlots := make([]protocol.LegacySetItemSlot, 0)
//	Actions := make([]protocol.InventoryAction, 0)
//
//	pk := p.(*reflect_packet.InventoryTransaction)
//	for _, s := range pk.LegacySetItemSlots {
//		LegacySetItemSlots = append(LegacySetItemSlots, protocol.LegacySetItemSlot{
//			ContainerID: s.ContainerID,
//			Slots:       s.Slots,
//		})
//	}
//	//for _, s := range pk.Actions {
//	//	LegacySetItemSlots = append(LegacySetItemSlots, protocol.InventoryAction{
//	//		SourceType:    s.SourceType,
//	//		WindowID:      s.WindowID,
//	//		SourceFlags:   s.SourceFlags,
//	//		InventorySlot: s.InventorySlot,
//	//		OldItem:       s.OldItem{},
//	//		NewItem:       s.NewItem,
//	//	})
//	//}
//	h.taskIO.ShieldIO.SendNoLock(&packet.InventoryTransaction{
//		LegacyRequestID:    pk.LegacyRequestID,
//		LegacySetItemSlots: LegacySetItemSlots,
//		Actions:            Actions,
//		TransactionData:    nil,
//	})
//	return nil
//}

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
				//InventoryTransactionHandler: &InventoryTransactionHandler{
				//	taskIO: p.taskIO,
				//	p:      p,
				//},
				AnimateHandler: &AnimateHandler{
					taskIO: p.taskIO,
					p:      p,
				},
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
		o.needUpdate = true
		//o.StartTime = time.Now()
		//o.TickOffest = p.Tick - o.Tick
		o.TickOffest = p.Tick - o.Tick
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
			T := uint64(10)
			if o.Tick != 0 {
				T = o.Tick
			}
			o.PacketsToTransfer = append(o.PacketsToTransfer, &reflect_packet.UpdateAttributes{
				EntityRuntimeID: 1,
				Attributes:      attrs,
				Tick:            T,
			})
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
}
