package world_mirror

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	reflect_packet "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"main.go/minecraft"
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

//func (p *Processor) onNewChunk(pos provider_world.ChunkPos, cdata *provider_world.ChunkData) {
//	if p.currentSession != nil {
//		chunkBuf := &bytes.Buffer{}
//		data := reflect_chunk.Encode(cdata.Chunk, reflect_chunk.NetworkEncoding)
//		for i := range data.SubChunks {
//			_, _ = chunkBuf.Write(data.SubChunks[i])
//		}
//		_, _ = chunkBuf.Write(append(make([]byte, 512), data.Biomes...))
//		chunkBuf.WriteByte(0)
//		enc := nbt.NewEncoderWithEncoding(chunkBuf, nbt.NetworkLittleEndian)
//		for bp, b := range cdata.E {
//			if n, ok := b.(world.NBTer); ok {
//				d := n.EncodeNBT()
//				d["x"], d["y"], d["z"] = int32(bp[0]), int32(bp[1]), int32(bp[2])
//				_ = enc.Encode(d)
//			}
//		}
//		p.currentSession.WritePacket(&reflect_packet.LevelChunk{
//			ChunkX:        pos[0],
//			ChunkZ:        pos[1],
//			SubChunkCount: uint32(len(data.SubChunks)),
//			RawPayload:    append([]byte(nil), chunkBuf.Bytes()...),
//		})
//	}
//}

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
