package task

import (
	"fmt"
	"main.go/minecraft/protocol/packet"
	"time"

	"main.go/minecraft/protocol/login"
	"main.go/shield"
)

type TaskIO struct {
	ShieldIO *shield.ShieldIO
	cbs      *CallBacks
	Status   *HoldedStatus

	requestID string

	initLock       chan int
	identityData   login.IdentityData
	EntityUniqueID int64
	//cmdSendMu           sync.Mutex
	packetTypes map[uint32]packet.Packet
}

type TaskIOWithLock struct {
	origTaskIO        *TaskIO
	beforeSwitchCMDFB bool
	ShieldIOWithLock  *shield.ShieldIOWithLock
}

func NewTaskIO(shieldIO *shield.ShieldIO) *TaskIO {
	taskIO := TaskIO{
		ShieldIO:    shieldIO,
		cbs:         newCallbacks(),
		Status:      newHolder(),
		requestID:   "96045347-a6a3-4114-94c0-1bc4cc561694",
		initLock:    make(chan int),
		packetTypes: make(map[uint32]packet.Packet),
		//cmdSendMu:           sync.Mutex{},
	}
	shieldIO.AddInitCallBack(taskIO.onMCInit)
	shieldIO.AddSessionTerminateCallBack(taskIO.onMCSessionTerminate)
	_, err := shieldIO.AddNewPacketCallback(taskIO.newPacketFn)
	if err != nil {
		panic(fmt.Sprintf("Task IO; init fail (%v)", err))
	}
	return &taskIO
}

// this could happen only once, and each task has it's own goruntine, so we use chan
func (io *TaskIO) WaitInit() {
	<-io.initLock
}

func (io *TaskIO) RequestID() string {
	io.WaitInit()
	return io.requestID
}

func (io *TaskIO) IdentityData() login.IdentityData {
	io.WaitInit()
	return io.identityData
}

// schedule
func (io *TaskIO) DelayExec(delay time.Duration, fn func()) {
	time.AfterFunc(delay, fn)
}
