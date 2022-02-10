package world_mirror

import (
	"fmt"
	reflect_protocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	reflect_packet "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"main.go/minecraft/protocol/packet"
)

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
