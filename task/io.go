package task

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"main.go/minecraft/protocol/login"
	"main.go/minecraft/protocol/packet"
	"main.go/shield"
)

type PacketCallBack struct {
	cbs   map[int]func(pk packet.Packet, cbId int)
	count int
}

type CallBacks struct {
	packetsCBS               map[uint32]PacketCallBack
	onCmdFeedbackTmpCbS      map[uuid.UUID]func(output *packet.CommandOutput)
	onCmdFeedbackOnCbsCount  int
	onCmdFeedbackOnCbs       map[int]func(cbId int)
	onCmdFeedbackOffCbsCount int
	onCmdFeedbackOffCbs      map[int]func(cbId int)
}

func newCallbacks() *CallBacks {
	cbs := CallBacks{
		packetsCBS:               make(map[uint32]PacketCallBack),
		onCmdFeedbackTmpCbS:      make(map[uuid.UUID]func(output *packet.CommandOutput)),
		onCmdFeedbackOnCbsCount:  0,
		onCmdFeedbackOnCbs:       make(map[int]func(cbId int)),
		onCmdFeedbackOffCbsCount: 0,
		onCmdFeedbackOffCbs:      make(map[int]func(cbId int)),
	}
	for packetID := range packet.NewPool() {
		cbs.packetsCBS[packetID] = PacketCallBack{cbs: make(map[int]func(pk packet.Packet, id int)), count: 0}
	}
	return &cbs
}

func (cbs *CallBacks) activatePacketCallbacks(pk packet.Packet) {
	packetCBS, _ := cbs.packetsCBS[pk.ID()]
	for cbID, cb := range packetCBS.cbs {
		cb(pk, cbID)
	}
}

type TaskIO struct {
	ShieldIO *shield.ShieldIO
	cbs      *CallBacks

	requestID string

	initLock            chan int
	identityData        login.IdentityData
	isOP                bool
	sendCommandFeedBack bool
	//cmdSendMu           sync.Mutex
}

type TaskIOWithLock struct {
	origTaskIO        *TaskIO
	beforeSwitchCMDFB bool
	ShieldIOWithLock  *shield.ShieldIOWithLock
}

func NewTaskIO(shieldIO *shield.ShieldIO) *TaskIO {
	callBacks := newCallbacks()
	taskIO := TaskIO{
		ShieldIO:            shieldIO,
		cbs:                 callBacks,
		requestID:           "96045347-a6a3-4114-94c0-1bc4cc561694",
		initLock:            make(chan int),
		isOP:                false,
		sendCommandFeedBack: false,
		//cmdSendMu:           sync.Mutex{},
	}
	shieldIO.AddInitCallBack(taskIO.onMCInit)
	shieldIO.AddSessionTerminateCallBack(taskIO.onMCSessionTerminate)
	shieldIO.AddNewPacketCallback(taskIO.newPacketFn)
	return &taskIO
}

// this could happen only once, and each task has it's own goruntine, so we use chan
func (io *TaskIO) WaitInit() {
	<-io.initLock
}

// query Info
func (io *TaskIO) CMDFeedBack() bool {
	io.WaitInit()
	return io.sendCommandFeedBack
}

func (io *TaskIO) RequestID() string {
	io.WaitInit()
	return io.requestID
}

func (io *TaskIO) IdentityData() login.IdentityData {
	io.WaitInit()
	return io.identityData
}

// handle callbacks
func (io *TaskIO) AddPacketTypeCallback(packetID uint32, cb func(packet.Packet, int)) (int, error) {
	packetCBS, ok := io.cbs.packetsCBS[packetID]
	if ok {
		c := packetCBS.count + 1
		_, hasK := packetCBS.cbs[c]
		for hasK {
			c += 1
			_, hasK = packetCBS.cbs[c]
		}
		packetCBS.count = c
		packetCBS.cbs[packetCBS.count] = cb
		return c, nil
	} else {
		return 0, fmt.Errorf("do not have such packet ID (%v) for call back ", packetID)
	}
}

func (io *TaskIO) RemovePacketTypeCallback(packetID uint32, callBackID int) bool {
	packetCBS, ok := io.cbs.packetsCBS[packetID]
	if ok {
		delete(packetCBS.cbs, callBackID)
	}
	return ok
}

func (io *TaskIO) AddOnCmdFeedBackOnCb(cb func(int)) int {
	c := io.cbs.onCmdFeedbackOnCbsCount + 1
	_, hasK := io.cbs.onCmdFeedbackOnCbs[c]
	for hasK {
		c += 1
		_, hasK = io.cbs.onCmdFeedbackOnCbs[c]
	}
	io.cbs.onCmdFeedbackOnCbsCount = c
	io.cbs.onCmdFeedbackOnCbs[io.cbs.onCmdFeedbackOnCbsCount] = cb
	return c
}
func (io *TaskIO) RemoveOnCmdFeedBackOnCb(cbID int) bool {
	_, ok := io.cbs.onCmdFeedbackOnCbs[cbID]
	if ok {
		delete(io.cbs.onCmdFeedbackOnCbs, cbID)
	}
	return ok
}

func (io *TaskIO) AddOnCmdFeedBackOffCb(cb func(int)) int {
	c := io.cbs.onCmdFeedbackOffCbsCount + 1
	_, hasK := io.cbs.onCmdFeedbackOffCbs[c]
	for hasK {
		c += 1
		_, hasK = io.cbs.onCmdFeedbackOffCbs[c]
	}
	io.cbs.onCmdFeedbackOffCbsCount = c
	io.cbs.onCmdFeedbackOffCbs[io.cbs.onCmdFeedbackOffCbsCount] = cb
	return c
}
func (io *TaskIO) RemoveOnCmdFeedBackOffCb(cbID int) bool {
	_, ok := io.cbs.onCmdFeedbackOffCbs[cbID]
	if ok {
		delete(io.cbs.onCmdFeedbackOffCbs, cbID)
	}
	return ok
}

// schedule
func (io *TaskIO) DelayExec(delay time.Duration, fn func()) {
	time.AfterFunc(delay, fn)
}
