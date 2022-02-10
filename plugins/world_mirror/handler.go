package world_mirror

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	protocol2 "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"main.go/minecraft/protocol"
	packet2 "main.go/minecraft/protocol/packet"
	"main.go/plugins/chunk_mirror"
	"main.go/plugins/world_mirror/server/session"
	"main.go/task"
	"time"
)

func (h *InventoryTransactionHandler) Handle(p packet.Packet, s *session.Session) error {
	LegacySetItemSlots := make([]protocol.LegacySetItemSlot, 0)
	//Actions := make([]protocol.InventoryAction, 0)

	pk := p.(*packet.InventoryTransaction)
	for _, s := range pk.LegacySetItemSlots {
		LegacySetItemSlots = append(LegacySetItemSlots, protocol.LegacySetItemSlot{
			ContainerID: s.ContainerID,
			Slots:       s.Slots,
		})
	}

	origTransactionData := pk.TransactionData
	var DeReflectTransactionData protocol.InventoryTransactionData
	switch origTransactionData.(type) {
	case nil, *protocol2.NormalTransactionData:
		DeReflectTransactionData = &protocol.NormalTransactionData{}
	case *protocol2.MismatchTransactionData:
		DeReflectTransactionData = &protocol.MismatchTransactionData{}
	case *protocol2.UseItemTransactionData:
		oT := origTransactionData.(*protocol2.UseItemTransactionData)
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
	case *protocol2.UseItemOnEntityTransactionData:
		oT := origTransactionData.(*protocol2.UseItemOnEntityTransactionData)
		DeReflectTransactionData = &protocol.UseItemOnEntityTransactionData{
			TargetEntityRuntimeID: oT.TargetEntityRuntimeID,
			ActionType:            oT.ActionType,
			HotBarSlot:            oT.HotBarSlot,
			HeldItem:              DeReflectItemInstance(oT.HeldItem),
			Position:              oT.Position,
			ClickedPosition:       oT.ClickedPosition,
		}
	case *protocol2.ReleaseItemTransactionData:
		oT := origTransactionData.(*protocol2.ReleaseItemTransactionData)
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
	h.taskIO.ShieldIO.SendNoLock(&packet2.InventoryTransaction{
		LegacyRequestID:    pk.LegacyRequestID,
		LegacySetItemSlots: LegacySetItemSlots,
		Actions:            DeReflectInventoryActions(pk.Actions),
		TransactionData:    DeReflectTransactionData,
	})
	return nil
}

type PlayerAuthInputBridgeHandler struct {
	taskIO         *task.TaskIO
	p              *Processor
	posUpdataFn    func(pos mgl32.Vec3)
	defaultHandler session.PacketHandler
}

func (h *PlayerAuthInputBridgeHandler) Handle(p packet.Packet, s *session.Session) error {
	h.p.currentSession = s
	pk := p.(*packet.PlayerAuthInput)
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
	spk := &packet2.PlayerAuthInput{
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
	h.p.PacketsToTransfer = make([]packet.Packet, 0)
	for _, pk := range pks {
		s.WritePacket(pk)
	}

	return nil
}

type PlayerActionHandler struct {
	taskIO *task.TaskIO
	p      *Processor
}

func (h *PlayerActionHandler) Handle(p packet.Packet, s *session.Session) error {
	pk := p.(*packet.PlayerAction)
	h.taskIO.ShieldIO.SendNoLock(&packet2.PlayerAction{
		EntityRuntimeID: pk.EntityRuntimeID,
		ActionType:      pk.ActionType,
		BlockPosition:   protocol.BlockPos(pk.BlockPosition),
		BlockFace:       pk.BlockFace,
	})
	return nil
}

func (h *AnimateHandler) Handle(p packet.Packet, s *session.Session) error {
	pk := p.(*packet.Animate)
	h.taskIO.ShieldIO.SendNoLock(&packet2.Animate{
		ActionType:      pk.ActionType,
		EntityRuntimeID: h.p.RuntimeID,
		BoatRowingTime:  pk.BoatRowingTime,
	})
	return nil
}

type AnimateHandler struct {
	taskIO *task.TaskIO
	p      *Processor
}

type InventoryTransactionHandler struct {
	taskIO *task.TaskIO
	p      *Processor
}
