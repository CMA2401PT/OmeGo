package chunk_mirror

import (
	"fmt"
	"main.go/plugins/chunk_mirror/server/block/cube"
	reflect_world "main.go/plugins/chunk_mirror/server/world"
	reflect_provider "main.go/plugins/chunk_mirror/server/world/mcdb"
	"time"
)

func (cm *ChunkMirror) DoAutoCache() {
	cm.doCache = true
}
func (cm *ChunkMirror) StopAutoCache() {
	cm.doCache = false
}

func (cm *ChunkMirror) RegChunkListener(ruleFn ListenRuleFn, cb ChunkListenerCb) func() {
	l := &ChunkListener{
		cb:     cb,
		ruleFN: ruleFn,
	}
	cm.ChunkListeners[l] = l
	return func() {
		delete(cm.ChunkListeners, l)
	}
}

func (cm *ChunkMirror) HasCache(pos reflect_world.ChunkPos, expireTime ...time.Time) bool {
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	return cm.hasCache(pos, expireTime[0])
}

func blockPosFromNBT(data map[string]interface{}) cube.Pos {
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	xInterface, _ := data["x"]
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	yInterface, _ := data["y"]
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	zInterface, _ := data["z"]
	x, _ := xInterface.(int32)
	y, _ := yInterface.(int32)
	z, _ := zInterface.(int32)
	return cube.Pos{int(x), int(y), int(z)}
}

func (cm *ChunkMirror) WriteSpecial(provider *reflect_provider.Provider) {
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	cm.writeSpecial(provider)
}

func (cm *ChunkMirror) getMemoryChunk(pos reflect_world.ChunkPos) (cd *reflect_world.ChunkData, hasK bool) {
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	data, hasK := cm.memoryChunks[pos]
	return data, hasK
}

func (cm *ChunkMirror) GetCachedChunk(pos reflect_world.ChunkPos, expireTime ...time.Time) (cd *reflect_world.ChunkData, err error) {
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	if !cm.hasCache(pos, expireTime...) {
		return nil, fmt.Errorf("Chunk Mirror: Try Getting an non-cached Chunk")
	}
	cd, hasK := cm.memoryChunks[pos]
	if hasK {
		return cd, nil
	}
	c, e, err := cm.WorldProvider.LoadChunk(pos)
	if err != nil {
		fmt.Printf("Chunk Mirror: on Cached Chunk Load, an error occured (%v)", err)
		return nil, err
	}
	if !e {
		fmt.Printf("Chunk Mirror: on Get Cached Chunk Load, an error occured: Map Info say this chunk exist, but provider cannot find it %v", pos)
		return nil, fmt.Errorf("chunk Mirror: on Get Cached Chunk Load, an error occured: Map Info say this chunk exist, but provider cannot find it %v", pos)
	}
	chunkData := reflect_world.NewChunkData(c)
	blockEntities, err := cm.WorldProvider.LoadBlockNBT(pos)
	if err != nil {
		fmt.Printf("Chunk Mirror: on Cached Nbt Block Load, an error occured (%v)", err)
		return nil, err
	}
	chunkData.E = make(map[cube.Pos]reflect_world.Block, len(blockEntities))
	for _, data := range blockEntities {
		pos := blockPosFromNBT(data)

		id := c.Block(uint8(pos[0]), int16(pos[1]), uint8(pos[2]), 0)
		b, ok := reflect_world.BlockByRuntimeID(id)
		if !ok {
			fmt.Printf("Chunk Mirror: on Cached Chunk Load (loading block entity data): could not find block state by runtime ID %v", id)
			continue
		}
		if nbt, ok := b.(reflect_world.NBTer); ok {
			b = nbt.DecodeNBT(data).(reflect_world.Block)
		}
		chunkData.E[pos] = b
	}
	BlockAuxData, err := cm.WorldProvider.LoadBlockAuxData(pos)
	if err != nil {
		fmt.Printf("Chunk Mirror: on Cached AuxNbtData Load, an error occured (%v)", err)
		return nil, err
	}
	for _, auxData := range BlockAuxData {
		pos := blockPosFromNBT(auxData)
		chunkData.AuxNbtInfo[pos] = auxData
	}
	return chunkData, nil
}

func (cm *ChunkMirror) IsBusy(i ...time.Duration) bool {
	if len(i) == 0 {
		// by default, idle after 1 sec
		return cm.lastChunkTime.Add(time.Second).After(time.Now())
	} else {
		return cm.lastChunkTime.Add(i[0]).After(time.Now())
	}
}

func (cm *ChunkMirror) waitMore(d time.Duration, c chan int) {
	time.AfterFunc(d, func() {
		if !cm.IsBusy() {
			close(c)
		} else {
			cm.waitMore(d, c)
		}
	})
}

func (cm *ChunkMirror) WaitIdle(i ...time.Duration) chan int {
	if !cm.IsBusy(i[0]) {
		c := make(chan int, 1)
		c <- 0
		return c
	} else {
		c := make(chan int)
		d := time.Second
		if len(i) > 0 {
			d = i[0]
		}
		cm.waitMore(d, c)
		return c
	}
}

type ChunkReq struct {
	Dry             bool // dose not load from file, just wait for cache complete (return an empty chunkData)
	X               int
	Z               int
	pos             reflect_world.ChunkPos
	AllowCacheAfter time.Time
	Active          bool // 是否允许移动机器人以主动获得区块，不允许则在无cache情况下关闭chan
	GetTimeOut      time.Duration
	hasTimeOut      bool
	deadlineTime    time.Time
	FarPoint        *[2]int
	respChan        chan *reflect_world.ChunkData
}

func (cm *ChunkMirror) fillEmptyField(req *ChunkReq) *ChunkReq {
	var defaultTime time.Time
	req.pos = reflect_world.ChunkPos{int32(req.X), int32(req.Z)}
	if req.AllowCacheAfter == defaultTime {
		req.AllowCacheAfter = cm.expiredTime
	}
	var defaultDuration time.Duration
	if req.GetTimeOut == defaultDuration {
		req.hasTimeOut = false
	} else {
		req.hasTimeOut = true
		req.deadlineTime = time.Now().Add(req.GetTimeOut)
	}
	if req.FarPoint == nil {
		req.FarPoint = &[2]int{cm.FarPointX, cm.FarPointZ}
	}
	return req
}

func (cm *ChunkMirror) RequireChunk(req *ChunkReq) chan *reflect_world.ChunkData {
	req = cm.fillEmptyField(req)
	if req.Dry && cm.HasCache(req.pos, req.AllowCacheAfter) {
		c := make(chan *reflect_world.ChunkData, 1)
		c <- &reflect_world.ChunkData{}
		return c
	}
	cd, err := cm.GetCachedChunk(req.pos, req.AllowCacheAfter)
	if err == nil {
		c := make(chan *reflect_world.ChunkData, 1)
		c <- cd
		return c
	}
	if !req.Active {
		c := make(chan *reflect_world.ChunkData, 1)
		c <- nil
		return c
	}
	c := make(chan *reflect_world.ChunkData)
	req.respChan = c
	cm.chunkReqs <- req
	return c
}
