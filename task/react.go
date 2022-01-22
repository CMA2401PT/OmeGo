package task

import (
	"fmt"

	"main.go/minecraft"
	"main.go/minecraft/protocol/packet"
)

func (taskIO *TaskIO) onGameRuleChanged(pk *packet.GameRulesChanged) {
	for _, rule := range pk.GameRules {
		if rule.Name == "sendcommandfeedback" {
			taskIO.sendCommandFeedBack = rule.Value.(bool)
			taskIO.isOP = rule.CanBeModifiedByPlayer
			if taskIO.sendCommandFeedBack {
				for cbID, cb := range taskIO.cbs.onCmdFeedbackOnCbs {
					cb(cbID)
				}
			} else {
				for cbID, cb := range taskIO.cbs.onCmdFeedbackOffCbs {
					cb(cbID)
				}
			}
		}
	}
}

func (taskIO *TaskIO) onMCInit(conn *minecraft.Conn) {
	fmt.Println("Reactor: Start OnMCInit Tasks")
	gameData := conn.GameData()
	taskIO.identityData = conn.IdentityData()
	for _, rule := range gameData.GameRules {
		if rule.Name == "sendcommandfeedback" {
			taskIO.sendCommandFeedBack = rule.Value.(bool)
			taskIO.isOP = rule.CanBeModifiedByPlayer
			close(taskIO.initLock)
		}
	}
	fmt.Printf("Reactor: sendcommandfeedback is %v\n", taskIO.sendCommandFeedBack)
}

func (taskIO *TaskIO) onMCSessionTerminate() {
	fmt.Println("Reactor: Find MC Session Terminated")
}

func (taskIO *TaskIO) newPacketFn(pk packet.Packet) {
	switch p := pk.(type) {
	case *packet.GameRulesChanged:
		fmt.Println("Reactor: GameRule Update")
		taskIO.onGameRuleChanged(p)
		break
	case *packet.PlayerList:
		fmt.Println("Reactor: Recv Player List Update")

	case *packet.AddPlayer:
		fmt.Println("Reactor: New Player Nearby")
	case *packet.CommandOutput:
		cb, ok := taskIO.cbs.onCmdFeedbackTmpCbS[p.CommandOrigin.UUID]
		if ok {
			delete(taskIO.cbs.onCmdFeedbackTmpCbS, p.CommandOrigin.UUID)
			cb(p)
		}
	}
	taskIO.cbs.activatePacketCallbacks(pk)
}
