package plugins

import (
	"fmt"
	"main.go/plugins/define"
	"main.go/task"
	"time"
)

type AskForOP struct {
	taskIO   *task.TaskIO
	initLock chan int
}

func (a *AskForOP) New(config []byte) define.Plugin {
	a.initLock = make(chan int)
	return a
}

func (a *AskForOP) WaitOP() {
	<-a.initLock
}

func (a *AskForOP) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	a.taskIO = taskIO
	//a.taskIO.ShieldIO.AddInitCallBack(func(conn *minecraft.Conn) {
	//	go a.AskForOP()
	//})
	return a
}

func (a *AskForOP) AskForOP() {
	for {
		cheatOn := a.taskIO.Status.CmdEnabled()
		if !cheatOn {
			fmt.Println("need cheat mode")
		} else {
			fmt.Println("Cheat mode is on")
			break
		}
		time.Sleep(3 * time.Second)
	}
	for {
		isop := a.taskIO.Status.IsOP()
		if !isop {
			fmt.Println("need OP")
		} else {
			fmt.Println("Op getted")
			close(a.initLock)
			return
		}
		time.Sleep(3 * time.Second)
	}
}

func (a *AskForOP) Routine() {
	a.taskIO.WaitInit()
	a.AskForOP()
}

func (a *AskForOP) Close() {

}
