package world_mirror

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
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
	cm            *chunk_mirror.ChunkMirror
	taskIO        *task.TaskIO
	log           func(isJson bool, data string)
	reflect       bool
	closeLock     chan int
	gameData      minecraft.GameData
	clientData    login.ClientData
	RuntimeID     uint64
	ServerHandle  *server.ServerHandle
	cachedPackets *CachedPackets
	currentPos    mgl32.Vec3
	needUpdate    bool
	StartTime     time.Time
	Time          int64
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
	taskIO *task.TaskIO
	p      *Processor
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
	tick := uint64(time.Now().Sub(h.p.StartTime).Milliseconds() / 50)
	fmt.Println(tick)
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
		Tick:                uint64(tick),
		Delta:               pk.Delta,
		ItemInteractionData: protocol.UseItemTransactionData{},
		ItemStackRequest:    protocol.ItemStackRequest{},
		BlockActions:        blockActions,
	}
	fmt.Println(spk)
	//fmt.Println(spk.Tick)
	//s.Conn.WritePacket(packet.Respawn{
	//	Position:        mgl32.Vec3{},
	//	State:           0,
	//	EntityRuntimeID: 0,
	//})
	h.taskIO.ShieldIO.SendNoLock(spk)
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
			SessionInjectFns: &session.InjectFns{PlayerAuthInputHandler: &PlayerAuthInputBridgeHandler{
				p.taskIO,
				p,
			}},
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
	p.cachedPackets = &CachedPackets{
		AvailableActorIdentifiers: make([]*packet.AvailableActorIdentifiers, 0),
		PlayerList:                make([]*packet.PlayerList, 0),
	}
	p.taskIO.ShieldIO.AddNewPacketCallback(p.handleNeteasePacket)
	p.taskIO.ShieldIO.AddInitCallBack(func(conn *minecraft.Conn) {
		p.currentPos = conn.GameData().PlayerPosition
		p.StartTime = conn.GameData().StartTime
	})
}
