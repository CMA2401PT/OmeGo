package WorldMirror

import (
	"fmt"
	"main.go/minecraft"
	"main.go/minecraft/protocol/login"
	"main.go/minecraft/protocol/packet"
	"main.go/plugins/WorldMirror/simulator"
	"main.go/plugins/chunk_mirror"
	"main.go/task"
)

type Processor struct {
	cm           *chunk_mirror.ChunkMirror
	taskIO       *task.TaskIO
	log          func(isJson bool, data string)
	reflect      bool
	closeLock    chan int
	gameData     minecraft.GameData
	clientData   login.ClientData
	RuntimeID    uint64
	ServerHandle *simulator.ServerHandle
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
	startData := &simulator.StartData{}
	p.ServerHandle = simulator.StartWithData(startData)
	p.reflect = true

	//defer p.close()
}

func (p *Processor) handleNeteasePacket(pk packet.Packet) {
	fmt.Println(pk.ID())
}

func (p *Processor) close() {
	p.reflect = false
	close(p.closeLock)
	p.closeLock = make(chan int)
}

func (p *Processor) inject() {
	p.reflect = false
	p.closeLock = make(chan int)
	p.taskIO.ShieldIO.AddSessionTerminateCallBack(p.close)
	p.taskIO.ShieldIO.AddNewPacketCallback(func(pk packet.Packet) {
		if p.reflect {
			p.handleNeteasePacket(pk)
		}
	})
}
