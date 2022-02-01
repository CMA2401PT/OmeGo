package bridge

import (
	"fmt"
	"main.go/minecraft/protocol/packet"
	"main.go/plugins/fastbuilder/types"
)

func Tellraw(msg string) {
	fmt.Println(msg)
	BypassedTaskIO.TalkTo(Supervisors, msg)
}

func Title(msg string) {
	BypassedTaskIO.Title(Supervisors, msg)
}

func SendCommand(cmd string) {
	BypassedTaskIO.SendCmd(cmd)
}

func SendCommandWithCB(cmd string, cb func(p *packet.CommandOutput)) {
	BypassedTaskIO.LockCMDAndFBOn().
		SendCmdWithFeedBack(cmd, cb).
		UnlockAndRestore()
}
func SendSetBlockCommandWithCB(module *types.Module, config *types.MainConfig, cb func(p *packet.CommandOutput)) {
	Block := module.Block
	Point := module.Point
	Method := config.Method
	var cmd string
	if Block != nil {
		cmd = fmt.Sprintf("setblock %v %v %v %v %v %v", Point.X, Point.Y, Point.Z, *Block.Name, Block.Data, Method)
	} else {
		cmd = fmt.Sprintf("setblock %v %v %v %v %v %v", Point.X, Point.Y, Point.Z, config.Block.Name, config.Block.Data, Method)
	}
	SendCommandWithCB(cmd, cb)
}

func WritePacket(pk packet.Packet) {
	BypassedTaskIO.ShieldIO.SendPacket(pk)
}

type FastSender struct {
}

func CreateFastSender() *FastSender {
	s := FastSender{}
	return &s
}

func (s *FastSender) SetBlock(module *types.Module, config *types.MainConfig) {
	Block := module.Block
	Point := module.Point
	Method := config.Method
	var cmd string
	if Block != nil {
		cmd = fmt.Sprintf("setblock %v %v %v %v %v %v", Point.X, Point.Y, Point.Z, *Block.Name, Block.Data, Method)
	} else {
		cmd = fmt.Sprintf("setblock %v %v %v %v %v %v", Point.X, Point.Y, Point.Z, config.Block.Name, config.Block.Data, Method)
	}
	SendCommand(cmd)
}

func (s *FastSender) ReplaceItem(module *types.Module, cfg *types.MainConfig) {
	cmd := fmt.Sprintf("replaceitem block %d %d %d slot.container %d %s %d %d", module.Point.X, module.Point.Y, module.Point.Z, module.ChestSlot.Slot, module.ChestSlot.Name, module.ChestSlot.Count, module.ChestSlot.Damage)
	SendCommand(cmd)
}
