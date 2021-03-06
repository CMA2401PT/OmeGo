package shield

import (
	"fmt"
	"sync"
	"time"

	"main.go/minecraft"
	"main.go/minecraft/protocol/login"
	"main.go/minecraft/protocol/packet"
)

type ConnDataResponseChans struct {
	GameDataChain   chan minecraft.GameData
	ClientDataChain chan login.ClientData
}

type ShieldIO struct {
	newPacketCBCount            int
	newPacketCallbacks          map[int]func(pk packet.Packet)
	packetGroupWriteChan        chan []packet.Packet
	currentlyWritingGroup       []packet.Packet
	currentlyWritingPacketIndex int
	connDataRequestFlag         chan int
	connDataResponseChans       ConnDataResponseChans
	beforeInitCallBacks         []func()
	initCallBacks               []func(conn *minecraft.Conn)
	beforeReInitCallBacks       []func()
	reInitCallBacks             []func(conn *minecraft.Conn)
	sessionTerminateCallBacks   []func()
	sendMu                      sync.Mutex
	conn                        *minecraft.Conn
	connInited                  bool
	connInitWaiter              chan int
}

type ShieldIOWithLock struct {
	o *ShieldIO
}

func (io *ShieldIO) onInit(conn *minecraft.Conn) {
	io.conn = conn
	close(io.connInitWaiter)
	io.connInited = true
}

func (io *ShieldIO) onTerminate() {
	io.connInitWaiter = make(chan int)
	io.connInited = false
	io.conn = nil
}

func (io *ShieldIO) GetConn() *minecraft.Conn {
	if io.connInited {
		return io.conn
	} else {
		<-io.connInitWaiter
		return io.conn
	}
}

func (io *ShieldIO) GameData() minecraft.GameData {
	io.connDataRequestFlag <- 0
	return <-io.connDataResponseChans.GameDataChain
}
func (io *ShieldIO) ClientData() login.ClientData {
	io.connDataRequestFlag <- 1
	return <-io.connDataResponseChans.ClientDataChain
}

func (io *ShieldIO) AddNewPacketCallback(cb func(pk packet.Packet)) (func(), error) {
	io.newPacketCBCount += 1
	c := io.newPacketCBCount
	if c == 0 {
		return nil, fmt.Errorf("number of packet callback fn over limit")
	}
	io.newPacketCallbacks[c] = cb
	removed := false
	removeFn := func() {
		if removed {
			return
		}
		removed = true
		delete(io.newPacketCallbacks, c)
	}
	return removeFn, nil
}

func (io *ShieldIO) Lock() *ShieldIOWithLock {
	io.sendMu.Lock()
	return &ShieldIOWithLock{o: io}
}

func (io *ShieldIOWithLock) SendPacketsGroup(pks []packet.Packet) *ShieldIOWithLock {
	io.o.packetGroupWriteChan <- pks
	return io
}
func (io *ShieldIOWithLock) SendPacket(pk packet.Packet) *ShieldIOWithLock {
	io.o.packetGroupWriteChan <- []packet.Packet{pk}
	return io
}

func (io *ShieldIOWithLock) UnLock() *ShieldIO {
	io.o.sendMu.Unlock()
	return io.o
}

func (io *ShieldIO) SendPacketsGroup(pks []packet.Packet) *ShieldIO {
	io.sendMu.Lock()
	defer io.sendMu.Unlock()
	io.packetGroupWriteChan <- pks
	return io
}
func (io *ShieldIO) SendPacket(pk packet.Packet) *ShieldIO {
	io.sendMu.Lock()
	defer io.sendMu.Unlock()
	io.packetGroupWriteChan <- []packet.Packet{pk}
	return io
}

func (io *ShieldIO) SendNoLock(pk packet.Packet) *ShieldIO {
	io.packetGroupWriteChan <- []packet.Packet{pk}
	return io
}

func (io *ShieldIO) EmptySendSequence() {
	io.currentlyWritingPacketIndex = 0
	io.currentlyWritingGroup = make([]packet.Packet, 0)
}

func (io *ShieldIO) AddBeforeInitCallBack(cb func()) {
	io.beforeInitCallBacks = append(io.beforeInitCallBacks, cb)
}
func (io *ShieldIO) AddInitCallBack(cb func(conn *minecraft.Conn)) {
	io.initCallBacks = append(io.initCallBacks, cb)
}
func (io *ShieldIO) AddBeforeReInitCallBack(cb func()) {
	io.beforeReInitCallBacks = append(io.beforeReInitCallBacks, cb)
}
func (io *ShieldIO) AddReInitCallBack(cb func(conn *minecraft.Conn)) {
	io.reInitCallBacks = append(io.reInitCallBacks, cb)
}
func (io *ShieldIO) AddSessionTerminateCallBack(cb func()) {
	io.sessionTerminateCallBacks = append(io.sessionTerminateCallBacks, cb)
}

type ShieldConfig struct {
	Respawn         bool `json:"respawn"`
	MaxRetryTimes   int  `json:"max_restart_retry"`
	MaxDelaySeconds int  `json:"max_delay_seconds"`
}

type Shield struct {
	Respawn             bool
	RespawnTimes        int
	RetryTimes          int
	MaxRetryTimes       int
	DelayFactor         time.Duration
	MaxDelay            time.Duration
	isInit              bool
	IO                  *ShieldIO
	LoginTokenGenerator func() (*minecraft.LoginToken, error)
	PacketInterceptor   PacketInterceptor
	Variant             int
	LoginClientData     login.ClientData
	LoginIdentityData   login.IdentityData
}

func NewShield(config *ShieldConfig) *Shield {
	if config.MaxDelaySeconds < 1 {
		config.MaxDelaySeconds = 1
	}
	shield := &Shield{
		Respawn:       config.Respawn,
		RespawnTimes:  0,
		RetryTimes:    0,
		MaxRetryTimes: config.MaxRetryTimes,
		DelayFactor:   time.Second,
		MaxDelay:      time.Duration(config.MaxDelaySeconds) * time.Second,
		isInit:        false,
		IO: &ShieldIO{
			newPacketCBCount:            0,
			newPacketCallbacks:          make(map[int]func(pk packet.Packet), 0),
			packetGroupWriteChan:        make(chan []packet.Packet),
			currentlyWritingGroup:       make([]packet.Packet, 0),
			currentlyWritingPacketIndex: 0,
			connDataRequestFlag:         make(chan int),
			connDataResponseChans: ConnDataResponseChans{
				GameDataChain:   make(chan minecraft.GameData),
				ClientDataChain: make(chan login.ClientData),
			},
			beforeInitCallBacks:   make([]func(), 0),
			initCallBacks:         make([]func(conn *minecraft.Conn), 0),
			beforeReInitCallBacks: make([]func(), 0),
			reInitCallBacks:       make([]func(conn *minecraft.Conn), 0),

			sessionTerminateCallBacks: make([]func(), 0),
			sendMu:                    sync.Mutex{},
			connInited:                false,
			connInitWaiter:            make(chan int),
		},
		LoginTokenGenerator: func() (*minecraft.LoginToken, error) { return &minecraft.LoginToken{}, nil },
		PacketInterceptor:   func(conn *minecraft.Conn, pk packet.Packet) (packet.Packet, error) { return pk, nil },
	}

	shield.IO.AddInitCallBack(shield.IO.onInit)
	shield.IO.AddSessionTerminateCallBack(shield.IO.onTerminate)
	return shield
}
