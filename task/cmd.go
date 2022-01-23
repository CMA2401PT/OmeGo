package task

import (
	"fmt"
	"github.com/google/uuid"
	"main.go/minecraft/protocol"
	"main.go/minecraft/protocol/packet"
	"sync"
)

func (io *TaskIO) GenCMD(command string) (*packet.CommandRequest, uuid.UUID) {
	UUID, _ := uuid.NewUUID()
	origin := protocol.CommandOrigin{
		Origin:         protocol.CommandOriginPlayer,
		UUID:           UUID,
		RequestID:      io.requestID,
		PlayerUniqueID: 0,
	}
	cmdRequest := packet.CommandRequest{
		CommandLine:   command,
		CommandOrigin: origin,
		Internal:      false,
		UnLimited:     false,
	}
	return &cmdRequest, UUID
}

func (io *TaskIO) GenSettingCMD(settingsCommand string) *packet.SettingsCommand {
	return &packet.SettingsCommand{
		CommandLine:    settingsCommand,
		SuppressOutput: true,
	}
}

func (io *TaskIO) AddOnCMDFeedBackCallback(UUID uuid.UUID, cb func(output *packet.CommandOutput)) {
	io.cbs.onCmdFeedbackTmpCbS[UUID] = cb
}

func (io *TaskIO) LockCMDAndFBOn() *TaskIOWithLock {
	beforeSwitchCMDFB := io.sendCommandFeedBack
	ret := &TaskIOWithLock{origTaskIO: io, beforeSwitchCMDFB: beforeSwitchCMDFB, ShieldIOWithLock: io.ShieldIO.Lock()}
	// previous packet maybe sendcommandfeedback false and haven't returned yet
	if beforeSwitchCMDFB {
		return ret
	}
	lock := sync.Mutex{}
	lock.Lock()
	pk, reqUUID := io.GenCMD("gamerule sendcommandfeedback true")
	ret.ShieldIOWithLock.SendPacket(pk)
	io.cbs.onCmdFeedbackTmpCbS[reqUUID] = func(respPk *packet.CommandOutput) {
		lock.Unlock()
	}
	lock.Lock()
	return ret
}

func (io *TaskIO) Lock() *TaskIOWithLock {
	return &TaskIOWithLock{origTaskIO: io, beforeSwitchCMDFB: io.sendCommandFeedBack, ShieldIOWithLock: io.ShieldIO.Lock()}
}

func (io *TaskIOWithLock) SendCmds(cmds ...string) *TaskIOWithLock {
	pks := make([]packet.Packet, 0)
	for _, cmd := range cmds {
		pk, _ := io.origTaskIO.GenCMD(cmd)
		pks = append(pks, pk)
	}
	io.ShieldIOWithLock.SendPacketsGroup(pks)
	return io
}

func (io *TaskIOWithLock) SendCmd(cmd string) *TaskIOWithLock {
	pk, _ := io.origTaskIO.GenCMD(cmd)
	io.ShieldIOWithLock.SendPacket(pk)
	return io
}

func (io *TaskIOWithLock) SendCmdWithFeedBack(cmd string, cb func(respPk *packet.CommandOutput)) *TaskIOWithLock {
	pk, reqUUID := io.origTaskIO.GenCMD(cmd)
	io.ShieldIOWithLock.SendPacket(pk)
	io.origTaskIO.cbs.onCmdFeedbackTmpCbS[reqUUID] = cb
	return io
}

func (io *TaskIOWithLock) SendSettingCMD(settingsCommand string) *TaskIOWithLock {
	io.ShieldIOWithLock.SendPacket(io.origTaskIO.GenSettingCMD(settingsCommand))
	return io
}
func (io *TaskIOWithLock) Unlock() *TaskIO {
	io.ShieldIOWithLock.UnLock()
	return io.origTaskIO
}
func (io *TaskIOWithLock) UnlockAndOff() *TaskIO {
	lock := sync.Mutex{}
	lock.Lock()
	io.origTaskIO.AddOnCmdFeedBackOffCb(func(cbID int) {
		lock.Unlock()
		io.origTaskIO.RemoveOnCmdFeedBackOffCb(cbID)
	})
	io.SendCmd("gamerule sendcommandfeedback false")
	lock.Lock()
	io.ShieldIOWithLock.UnLock()
	return io.origTaskIO
}
func (io *TaskIOWithLock) UnlockAndRestore() *TaskIO {
	if io.beforeSwitchCMDFB {
		io.ShieldIOWithLock.UnLock()
		return io.origTaskIO
	}
	return io.UnlockAndOff()
}

func (io *TaskIO) SendCmds(cmds ...string) *TaskIO {
	pks := make([]packet.Packet, 0)
	for _, cmd := range cmds {
		pk, _ := io.GenCMD(cmd)
		pks = append(pks, pk)
	}
	io.ShieldIO.SendPacketsGroup(pks)
	return io
}

func (io *TaskIO) SendCmd(cmd string) *TaskIO {
	pk, _ := io.GenCMD(cmd)
	io.ShieldIO.SendPacket(pk)
	return io
}

func (io *TaskIO) SendCmdWithFeedBack(cmd string, cb func(respPk *packet.CommandOutput)) *TaskIO {
	pk, reqUUID := io.GenCMD(cmd)
	io.ShieldIO.SendPacket(pk)
	io.cbs.onCmdFeedbackTmpCbS[reqUUID] = cb
	return io
}

func (io *TaskIO) SendSettingCMD(settingsCommand string) *TaskIO {
	io.ShieldIO.SendPacket(io.GenSettingCMD(settingsCommand))
	return io
}

// cannot work since 1.17
//func (io *TaskIO) SendCmdWithEnsuredFeedBack(cmd string, cb func(respPk *packet.CommandOutput)) {
//	pk, reqUUID := io.GenCMD(cmd)
//	if !io.sendCommandFeedBack {
//		pks := make([]packet.Packet, 3)
//		pks[0] = io.turnOnFeedBackPk
//		pks[1] = pk
//		pks[2] = io.turnOffFeedBackPk
//		io.ShieldIO.SendPacketsGroup(pks)
//	} else {
//		io.ShieldIO.SendPacket(pk)
//	}
//	io.cbs.onCmdFeedbackTmpCbS[reqUUID] = cb
//}
//
//func (io *TaskIO) SendCmdsWithoutFeedBack(cmds ...string) {
//	pks := make([]packet.Packet, 0)
//	s := io.sendCommandFeedBack
//	if s {
//		pks = append(pks, io.turnOffFeedBackPk)
//	}
//	for _, cmd := range cmds {
//		pk, _ := io.GenCMD(cmd)
//		pks = append(pks, pk)
//	}
//	if s {
//		pks = append(pks, io.turnOnFeedBackPk)
//	}
//	io.ShieldIO.SendPacketsGroup(pks)
//}
//
//func (io *TaskIO) SendCmdsGroupWithoutFeedBack(cmds []string) *TaskIO {
//	io.SendCmdsWithoutFeedBack(cmds...)
//	return io
//}

func (io *TaskIO) SendChat(content string) *TaskIO {
	idd := io.identityData
	pk := packet.Text{
		TextType:         packet.TextTypeChat,
		NeedsTranslation: false,
		SourceName:       idd.DisplayName,
		Message:          content,
		XUID:             idd.XUID,
	}
	io.ShieldIO.SendPacket(&pk)
	return io
}

func (io *TaskIO) TalkTo(player string, content string) *TaskIO {
	cmd, _ := io.GenCMD(fmt.Sprintf(`tellraw %s {"rawtext" : [{"text":"%s"}]}`, player, content))
	io.ShieldIO.SendPacket(cmd)
	return io
}

func (io *TaskIO) Say(isJson bool, content string) *TaskIO {
	if !isJson {
		cmd, _ := io.GenCMD(fmt.Sprintf(`tellraw @a {"rawtext" : [{"text":"%s"}]}`, content))
		io.ShieldIO.SendPacket(cmd)
	} else {
		cmd, _ := io.GenCMD(fmt.Sprintf(`tellraw @a {"rawtext" : %s}`, content))
		io.ShieldIO.SendPacket(cmd)
	}
	return io
}
