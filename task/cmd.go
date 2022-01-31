package task

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"main.go/minecraft/protocol"
	"main.go/minecraft/protocol/packet"
	"main.go/shield"
	"strings"
	"sync"
	"time"
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
	}
	return &cmdRequest, UUID
}

func (io *TaskIO) AddOnCMDFeedBackCallback(UUID uuid.UUID, cb func(output *packet.CommandOutput)) {
	io.cbs.onCmdFeedbackTmpCbS[UUID] = cb
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

func turnOnCMDFB(io *TaskIO, lockedShieldIO *shield.ShieldIOWithLock, do func()) {
	pk, UUID := io.GenCMD("gamerule sendcommandfeedback true")
	io.AddOnCMDFeedBackCallback(UUID, func(respPk *packet.CommandOutput) {
		if respPk.OutputMessages[0].Success {
			do()
		} else {
			fmt.Println("Fail to set sendcommandfeedback true, do I have op?")
			time.Sleep(3 * time.Second)
			turnOnCMDFB(io, lockedShieldIO, do)
		}
	})
	lockedShieldIO.SendPacket(pk)
}

func (io *TaskIO) LockCMDAndFBOn() *TaskIOWithLock {
	beforeSwitchCMDFB := io.Status.CmdFB()
	lockedShieldIO := io.ShieldIO.Lock()
	ret := &TaskIOWithLock{origTaskIO: io, beforeSwitchCMDFB: beforeSwitchCMDFB, ShieldIOWithLock: lockedShieldIO}
	if beforeSwitchCMDFB {
		return ret
	}
	lock := sync.Mutex{}
	lock.Lock()
	turnOnCMDFB(io, lockedShieldIO, func() {
		lock.Unlock()
	})
	lock.Lock()
	return ret
}

func (io *TaskIO) Lock() *TaskIOWithLock {
	return &TaskIOWithLock{origTaskIO: io, beforeSwitchCMDFB: io.Status.CmdFB(), ShieldIOWithLock: io.ShieldIO.Lock()}
}

func (io *TaskIOWithLock) Unlock() *TaskIO {
	io.ShieldIOWithLock.UnLock()
	return io.origTaskIO
}

func (io *TaskIOWithLock) UnlockAndOff() *TaskIO {
	//fmt.Println("Switch Off")
	lock := sync.Mutex{}
	lock.Lock()
	unlocked := false
	io.origTaskIO.AddOnCmdFeedBackOffCb(func() {
		if !unlocked {
			unlocked = true
			lock.Unlock()
		}
	})
	io.SendCmdNoLock("gamerule sendcommandfeedback false")
	time.AfterFunc(time.Second, func() {
		// it seems that something went wrong, but it's ok
		if !unlocked {
			go func() {
				fmt.Println("Fail to set sendcommandfeedback false")
				for !unlocked {
					io.SendCmdNoLock("gamerule sendcommandfeedback true")
					io.SendCmdNoLock("gamerule sendcommandfeedback false")
					time.Sleep(time.Millisecond * 500)
				}
				unlocked = true
				lock.Unlock()
				fmt.Println("Retry set sendcommandfeedback false success")
				return
			}()

		}
	})
	lock.Lock()
	unlocked = true
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

// NORMAL Block Send

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

func (io *TaskIOWithLock) SendCmdNoLock(cmd string) {
	pk, _ := io.origTaskIO.GenCMD(cmd)
	io.origTaskIO.ShieldIO.SendNoLock(pk)
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
	io.ShieldIO.SendNoLock(&pk)
	return io
}

func (io *TaskIO) TalkTo(player string, content string) *TaskIO {
	cmd, _ := io.GenCMD(fmt.Sprintf(`tellraw %s {"rawtext" : [{"text":"%s"}]}`, player, content))
	io.ShieldIO.SendNoLock(cmd)
	return io
}

type TellrawItem struct {
	Text string `json:"text"`
}

type TellrawStruct struct {
	RawText []TellrawItem `json:"rawtext"`
}

func TitleRequest(player string, lines ...string) string {
	var items []TellrawItem
	for _, text := range lines {
		items = append(items, TellrawItem{Text: strings.Replace(text, "schematic", "sc***atic", -1)})
	}
	final := &TellrawStruct{
		RawText: items,
	}
	content, _ := json.Marshal(final)
	cmd := fmt.Sprintf("titleraw %v actionbar %s", player, content)
	return cmd
}

func (io *TaskIO) Title(player string, lines ...string) *TaskIO {
	cmd, _ := io.GenCMD(TitleRequest(player, lines...))
	io.ShieldIO.SendNoLock(cmd)
	return io
}

func (io *TaskIO) Say(isJson bool, content string) *TaskIO {
	if !isJson {
		cmd, _ := io.GenCMD(fmt.Sprintf(`tellraw @a {"rawtext" : [{"text":"%s"}]}`, content))
		io.ShieldIO.SendPacket(cmd)
	} else {
		cmd, _ := io.GenCMD(fmt.Sprintf(`tellraw @a {"rawtext" : %s}`, content))
		io.ShieldIO.SendNoLock(cmd)
	}
	return io
}
