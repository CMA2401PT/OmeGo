package task

import (
	"main.go/dragonfly/server/world"
	"main.go/minecraft/protocol/packet"
	"sync"
)

type statusWaiter struct {
	isInited bool
	waiter   chan int
}

func (w *statusWaiter) wait() {
	if w.isInited {
		return
	} else {
		<-w.waiter
	}
}

func (w *statusWaiter) init() {
	if w.isInited {
		return
	}
	w.isInited = true
	close(w.waiter)
}

func newWaitor() *statusWaiter {
	return &statusWaiter{
		isInited: false,
		waiter:   make(chan int),
	}
}

type HoldedStatus struct {
	isOPWaiter       *statusWaiter
	isOP             bool
	cmdEnabledWaiter *statusWaiter
	cmdEnabled       bool
	cmdFBWaiter      *statusWaiter
	cmdFB            bool
	// 以下两个即将被弃用，以后都使用 mirror_chunk 核心插件作为区块缓存提供者
	chunkCache map[world.ChunkPos]*packet.LevelChunk
	chunkLock  sync.Mutex
}

func newHolder() *HoldedStatus {
	s := HoldedStatus{
		isOPWaiter:       newWaitor(),
		isOP:             false,
		cmdFBWaiter:      newWaitor(),
		cmdFB:            false,
		cmdEnabledWaiter: newWaitor(),
		cmdEnabled:       false,
		chunkCache:       make(map[world.ChunkPos]*packet.LevelChunk),
		chunkLock:        sync.Mutex{},
	}
	return &s
}

func (s *HoldedStatus) setCmdEnabled(v bool) {
	s.cmdEnabled = v
	s.cmdEnabledWaiter.init()
}

func (s *HoldedStatus) CmdEnabled() bool {
	s.cmdEnabledWaiter.wait()
	return s.cmdEnabled
}

func (s *HoldedStatus) setIsOP(v bool) {
	s.isOP = v
	s.isOPWaiter.init()
}

func (s *HoldedStatus) IsOP() bool {
	s.isOPWaiter.wait()
	return s.isOP
}

func (s *HoldedStatus) setCmdFB(v bool) {
	s.cmdFB = v
	s.cmdFBWaiter.init()
}

func (s *HoldedStatus) CmdFB() bool {
	s.cmdFBWaiter.wait()
	return s.cmdFB
}

//type ChunkHolder struct {
//	chunkCache       map[world.ChunkPos]*packet.LevelChunk
//	chunkLock        sync.Mutex
//	chunkCacheRange  [4]int
//	cacheRangeSetted bool
//}

// 以下3个即将被弃用，以后都使用 mirror_chunk 核心插件作为区块缓存提供者
func (s *HoldedStatus) AccessChunkCache(fn func(map[world.ChunkPos]*packet.LevelChunk)) {
	s.chunkLock.Lock()
	defer s.chunkLock.Unlock()
	fn(s.chunkCache)
}
func (s *HoldedStatus) AddChunk(p *packet.LevelChunk) {
	s.chunkLock.Lock()
	defer s.chunkLock.Unlock()
	s.chunkCache[world.ChunkPos{p.ChunkX, p.ChunkZ}] = p
}
func (s *HoldedStatus) ClearAllChunk() {
	s.chunkLock.Lock()
	defer s.chunkLock.Unlock()
	s.chunkCache = make(map[world.ChunkPos]*packet.LevelChunk)
}
