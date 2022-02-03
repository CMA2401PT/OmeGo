package chunk_mirror

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"main.go/dragonfly/server/world/chunk"
	"main.go/minecraft/protocol/packet"
	reflect_block "main.go/plugins/chunk_mirror/server/block"
	"main.go/plugins/chunk_mirror/server/block/cube"
	reflect_world "main.go/plugins/chunk_mirror/server/world"
	reflect_chunk "main.go/plugins/chunk_mirror/server/world/chunk"
	reflect_provider "main.go/plugins/chunk_mirror/server/world/mcdb"
	"main.go/plugins/define"
	"main.go/task"
	block_define "main.go/world/define"
	"os"
	"path"
	"reflect"
	"strconv"
	"sync"
	"time"
)

//go:embed richBlock.json
var richBlocksData []byte

type RichBlock struct {
	Name       string
	Val        int
	NeteaseRID int
	ReflectRID int
	Props      map[string]interface{}
}

type RichBlocks struct {
	ReflectAirRID int
	NeteaseAirRID int
	RichBlocks    []RichBlock
}

type ChunkListenerCb func(pos reflect_world.ChunkPos, cdata *reflect_world.ChunkData)
type ListenRuleFn func(X, Z int) bool
type ChunkListener struct {
	cb     ChunkListenerCb
	ruleFN ListenRuleFn
}

type ChunkMirror struct {
	AutoCacheByDefault   bool   `yaml:"auto_cache_by_default"`
	WorldDir             string `yaml:"world_dir"`
	ConfigExpiredTimeStr string `yaml:"expired_time"`
	WorldMin             int    `yaml:"world_min"`
	WorldMax             int    `yaml:"world_max"`
	FarPointX            int    `yaml:"far_point_x"`
	FarPointZ            int    `yaml:"far_point_z"`
	NeteaseAirRID        int
	MirrorAirRID         uint32
	richBlocks           *RichBlocks
	worldRange           cube.Range
	taskIO               *task.TaskIO
	WorldProvider        *reflect_provider.Provider
	blockReflectMapping  []uint32
	providerMu           sync.Mutex
	cacheRecordFile      string
	listenerDestroyFn    func()
	cacheMap             map[reflect_world.ChunkPos]time.Time
	expiredTime          time.Time
	doCache              bool
	ChunkListeners       map[*ChunkListener]*ChunkListener
	lastChunkTime        time.Time
	chunkReqs            chan *ChunkReq
	isWaiting            bool
	waitLock             chan int
}

func (cm *ChunkMirror) loadCacheRecordFile() {
	if _, err := os.Stat(cm.cacheRecordFile); os.IsNotExist(err) {
		fmt.Println("Chunk Mirror: Cache Record File Not Exist")
	} else {
		fi, err := os.Open(cm.cacheRecordFile)
		if err != nil {
			panic(fmt.Sprintf("Chunk Mirror: Open Cache Record File Fail (%v)", err))
		}
		defer fi.Close()
		fmt.Println("Chunk Mirror: Reading Cache Record File...")
		reader := csv.NewReader(fi)
		records, err := reader.ReadAll()
		if err != nil {
			panic(fmt.Sprintf("Chunk Mirror: Read Cache Record File Fail (%v)", err))
		}
		var X, Z int
		var T time.Time

		for _, record := range records {
			X, err = strconv.Atoi(record[0])
			if err != nil {
				panic(fmt.Sprintf("Chunk Mirror: Convert Cache Record (X) Fail @(%v) (%v)", record, err))
			}
			Z, err = strconv.Atoi(record[1])
			if err != nil {
				panic(fmt.Sprintf("Chunk Mirror: Convert Cache Record (Z) Fail @(%v) (%v)", record, err))
			}
			T, err = time.ParseInLocation("2006-01-02 15:04:05", record[2], time.Local)
			if err != nil {
				panic(fmt.Sprintf("Chunk Mirror: Convert Cache Record (Time) Fail @(%v) (%v)", record, err))
			}
			cm.cacheMap[reflect_world.ChunkPos{int32(X), int32(Z)}] = T
		}
		fmt.Println("Chunk Mirror: Cache Record Load Successfully!")
	}
}

func (cm *ChunkMirror) dumpCacheRecordFile() {
	fmt.Println("Chunk Mirror: Cache Record Dumping...")
	fi, err := os.OpenFile(cm.cacheRecordFile, os.O_WRONLY|os.O_CREATE, 644)
	if err != nil {
		panic(fmt.Sprintf("Chunk Mirror: Create Cache Record File Fail (%v)", err))
	}
	writer := csv.NewWriter(fi)
	for pos, T := range cm.cacheMap {
		err := writer.Write([]string{strconv.Itoa(int(pos.X())), strconv.Itoa(int(pos.Z())), T.Format("2006-01-02 15:04:05")})
		if err != nil {
			fmt.Printf("Chunk Mirror: Write Cache Record File Fail @ %v %v (%v)\n", pos, T, err)
		}
	}
	writer.Flush()
	err = fi.Close()
	if err != nil {
		fmt.Printf("Chunk Mirror: Close Cache Record File Fail @ %v %v (%v)\n", err)
	}
	fmt.Println("Chunk Mirror: Cache Record Dump Successfully!")
}

func (cm *ChunkMirror) New(config []byte) define.Plugin {
	var err error
	cm.AutoCacheByDefault = true
	cm.WorldDir = "TmpWorld"
	cm.ConfigExpiredTimeStr = ""
	cm.WorldMin = -64
	cm.WorldMax = 320
	cm.worldRange = cube.Range{cm.WorldMin, cm.WorldMax}

	cm.FarPointX = 100000
	cm.FarPointZ = 100000

	cm.isWaiting = false
	cm.waitLock = make(chan int)

	err = yaml.Unmarshal(config, &cm)
	if err != nil {
		panic("Chunk Mirror: cannot handle config")
	}
	cm.doCache = cm.AutoCacheByDefault
	cm.chunkReqs = make(chan *ChunkReq, 4)
	richBlocks := RichBlocks{}
	err = json.Unmarshal(richBlocksData, &richBlocks)
	cm.richBlocks = &richBlocks
	if err != nil {
		panic("Chunk Mirror: cannot read remapping info")
	}
	cm.NeteaseAirRID = richBlocks.NeteaseAirRID
	cm.MirrorAirRID, _ = reflect_world.BlockRuntimeID(reflect_block.Air{})
	if cm.MirrorAirRID != uint32(richBlocks.ReflectAirRID) {
		panic("Reflect World not properly init!")
	}
	cm.blockReflectMapping = make([]uint32, len(richBlocks.RichBlocks))
	for _, richBlocks := range richBlocks.RichBlocks {
		cm.blockReflectMapping[richBlocks.NeteaseRID] = uint32(richBlocks.ReflectRID)
	}

	if cm.ConfigExpiredTimeStr == "" {
		cm.expiredTime = time.Now()
	} else {
		cm.expiredTime, err = time.ParseInLocation("2006-01-02 15:04:05", cm.ConfigExpiredTimeStr, time.Local)
		if err != nil {
			panic(fmt.Sprintf("Chunk Mirror: Read Expired Time Fail (%v)", err))
		}
	}
	if err != nil {
		panic(fmt.Sprintf("Chunk Mirror: Read Config fail (%v)", err))
	}
	cm.WorldProvider, err = reflect_provider.New(cm.WorldDir, reflect_world.Overworld)
	cm.WorldProvider.D.LevelName = "MirrorWorld"
	if err != nil {
		panic(fmt.Sprintf("Chunk Mirror: Load/Create Chunk @ %v fail (%v)", cm.WorldDir, err))
	}
	cm.cacheRecordFile = path.Join(cm.WorldDir, "cache_log.txt")
	cm.cacheMap = make(map[reflect_world.ChunkPos]time.Time)
	cm.providerMu = sync.Mutex{}
	cm.loadCacheRecordFile()
	return cm
}

func (cm *ChunkMirror) Close() {
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	cm.listenerDestroyFn()
	err := cm.WorldProvider.Close()
	if err != nil {
		fmt.Printf("Chunk Mirror: Close WorldProvider Fail (%v)\n", err)
	}
	cm.dumpCacheRecordFile()
}

func (cm *ChunkMirror) handleOneReq(req *ChunkReq) {
	fmt.Printf("Mirror-Chunk: Activate Require Chunk @ (%v,%v)\n", req.X, req.Z)
	cm.taskIO.SendCmd(fmt.Sprintf("tp @s %d 127 %d", req.X*16, req.Z*16))
	time.Sleep(time.Second * 1)
	retryTime := 0
	cacheAfter := req.AllowCacheAfter
	Fx, Fz := req.FarPoint[0], req.FarPoint[1]
	for {
		if req.Dry && cm.HasCache(req.pos, req.AllowCacheAfter) {
			req.respChan <- &reflect_world.ChunkData{}
			return
		}
		cd, err := cm.GetCachedChunk(req.pos, cacheAfter)
		if err == nil {
			req.respChan <- cd
			return
		}
		select {
		case <-cm.waitLock:
			//fmt.Printf("New Chunk Arrival")
		case <-time.After(time.Second):
			if req.hasTimeOut {
				if time.Now().After(req.deadlineTime) {
					close(req.respChan)
					return
				}
			} else {
				if !cm.isWaiting {
					cm.waitLock = make(chan int)
					cm.isWaiting = true
				}
				retryTime += 1
				if retryTime > 16 {
					retryTime = 16
				}
				fmt.Printf("Retry (%v) -> step1. Move Far @ (%v,%v)\n", retryTime, Fx, Fz)
				cm.taskIO.SendCmd(fmt.Sprintf("tp @s %d 127 %d", Fx, Fz))
				time.Sleep(time.Duration(retryTime) * 500 * time.Millisecond)
				fmt.Printf("Retry (%v) -> step2. Move Back @ (%v,%v)\n", retryTime, req.X*16, req.Z*16)
				cm.taskIO.SendCmd(fmt.Sprintf("tp @s %d 127 %d", req.X*16, req.Z*16))
				fmt.Printf("Retry (%v) -> step3. Delay (%v,%v)\n")
			}
		}
	}
}

func (cm *ChunkMirror) Routine() {
	for {
		req := <-cm.chunkReqs
		cm.handleOneReq(req)
	}
}

func (cm *ChunkMirror) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	cm.taskIO = taskIO
	cm.InjectListener()
	return cm
}

func (cm *ChunkMirror) InjectListener() {
	var err error
	cm.listenerDestroyFn, err = cm.taskIO.AddPacketTypeCallback(packet.IDLevelChunk, func(p packet.Packet) {
		cm.onNewChunk(p.(*packet.LevelChunk))
	})
	if err != nil {
		panic(fmt.Sprintf("Chunk Mirror: on InjectListener, an error occur(%v)", err))
	}
}

func (cm *ChunkMirror) hasCache(pos reflect_world.ChunkPos, expireTime ...time.Time) bool {
	T, hasK := cm.cacheMap[pos]
	if hasK {
		if len(expireTime) == 0 && T.After(cm.expiredTime) {
			return true
		}
		if T.After(expireTime[0]) {
			return true
		}
	}
	return false
}

func SaveChunk(pos reflect_world.ChunkPos, c *reflect_world.ChunkData, p *reflect_provider.Provider) {
	m := make([]map[string]interface{}, 0, len(c.E))
	for pos, b := range c.E {
		if n, ok := b.(reflect_world.NBTer); ok {
			// Encode the block entities and add the 'x', 'y' and 'z' tags to it.
			data := n.EncodeNBT()
			data["x"], data["y"], data["z"] = int32(pos[0]), int32(pos[1]), int32(pos[2])
			m = append(m, data)
		}
	}
	c.Compact()
	if err := p.SaveChunk(pos, c.Chunk); err != nil {
		fmt.Printf("Chunk: error saving chunk %v to provider: %v\n", pos, err)
	}
	if err := p.SaveBlockNBT(pos, m); err != nil {
		fmt.Printf("Chunk Mirror: error saving block NBT in chunk %v to provider: %v\n", pos, err)
	}
	auxs := make([]map[string]interface{}, 0, len(c.AuxNbtInfo))
	for pos, aux := range c.AuxNbtInfo {
		// Encode the block entities and add the 'x', 'y' and 'z' tags to it.
		aux["x"], aux["y"], aux["z"] = int32(pos[0]), int32(pos[1]), int32(pos[2])
		auxs = append(auxs, aux)
	}
	if err := p.SaveBlockAuxData(pos, auxs); err != nil {
		fmt.Printf("Chunk Mirror: error saving aux NBT data in chunk %v to provider: %v\n", pos, err)
	}
}

func (cm *ChunkMirror) saveChunk(pos reflect_world.ChunkPos, c *reflect_world.ChunkData) {
	SaveChunk(pos, c, cm.WorldProvider)
}

func (cm *ChunkMirror) reflectChunk(pos reflect_world.ChunkPos, c *chunk.Chunk) *reflect_world.ChunkData {
	//reflectChunk, found, err := cm.WorldProvider.LoadChunk(pos)
	//var reflectChunkData *reflect_world.ChunkData
	//if err != nil {
	//	return fmt.Errorf("error loading chunk %v: %w", pos, err)
	//}
	//if !found {
	//	reflectChunk = reflect_chunk.New(uint32(cm.MirrorAirRID), cm.worldRange)
	//}
	reflectChunk := reflect_chunk.New(cm.MirrorAirRID, cm.worldRange)
	reflectChunkData := reflect_world.NewChunkData(reflectChunk)
	//if found{
	//	blockEntities, err := cm.WorldProvider.LoadBlockNBT(pos)
	//	if err != nil {
	//		return fmt.Errorf("error loading block entities of chunk %v: %w", pos, err)
	//	}
	//}

	var pX, pZ byte
	var subIndex, pY int16
	neteaseRid := uint32(cm.NeteaseAirRID)
	neteaseAirRid := uint32(cm.NeteaseAirRID)
	reflectRid := uint32(cm.MirrorAirRID)
	var subChunk *chunk.SubChunk

	for subIndex = 0; subIndex < 16; subIndex++ {
		subChunk = c.Sub()[subIndex]
		if subChunk == nil {
			continue
		}
		layers := subChunk.Layers()
		if uint8(len(layers)) <= 0 {
			continue
		}
		layer := layers[0]
		var sY byte
		for sY = 0; sY < 16; sY++ {
			pY = subIndex<<4 + int16(sY)
			for pX = 0; pX < 16; pX++ {
				for pZ = 0; pZ < 16; pZ++ {
					neteaseRid = layer.RuntimeID(pX, sY, pZ)
					if neteaseRid == neteaseAirRid {
						continue
					} else {
						reflectRid = cm.blockReflectMapping[neteaseRid]
						reflectChunkData.SetBlock(pX, pY, pZ, 0, reflectRid)
					}
				}
			}
		}
	}
	for blockPos, nbt := range c.BlockNBT() {
		cubePos := cube.Pos{blockPos.X(), blockPos.Y(), blockPos.Z()}
		auxBlockDefine := make(map[string]interface{})
		auxBlockDefine["nbt"] = nbt

		nbtBlockRid := c.RuntimeID(uint8(blockPos.X()), int16(blockPos.Y()), uint8(blockPos.Z()), 0)
		reflectRid = cm.blockReflectMapping[nbtBlockRid]
		rb := cm.richBlocks.RichBlocks[nbtBlockRid]
		auxBlockDefine["richBlockInfo"] = struct {
			Name             string
			Val              int32
			NeteaseRuntimeID int32
			ReflectRuntimeID int32
		}{
			Name:             rb.Name,
			Val:              int32(rb.Val),
			NeteaseRuntimeID: int32(rb.NeteaseRID),
			ReflectRuntimeID: int32(rb.ReflectRID),
		}

		reflectChunkData.AuxNbtInfo[cubePos] = auxBlockDefine

		b, found := reflect_world.BlockByRuntimeID(reflectRid)
		if !found {
			fmt.Printf("Chunk Mirror: Nbt Block not found!  (%v -> %v) @ %v nbt: %v\n", nbtBlockRid, reflectRid, blockPos, nbt)
			continue
		}
		if n, ok := b.(reflect_world.NBTer); ok {
			// Encode the block entities and add the 'x', 'y' and 'z' tags to it.
			b := n.DecodeNBT(nbt)
			wb, ok := b.(reflect_world.Block)
			if !ok {
				fmt.Printf("Chunk Mirror: Cannot Convert Nbt Block (%v -> %v) (%v) @ %v nbt: %v to a normal block! %v\n", nbtBlockRid, reflectRid, reflect.TypeOf(b), blockPos, nbt, b)
				continue
			}
			reflectChunkData.E[cubePos] = wb
		} else {
			fmt.Printf("Chunk Mirror: Block (%v -> %v) (%v) @ %v nbt=%v cannot be a Nbt Block! %v\n", nbtBlockRid, reflectRid, reflect.TypeOf(b), blockPos, nbt, b)
		}
	}
	return reflectChunkData
}

func (cm *ChunkMirror) getListeners(X, Z int) []ChunkListenerCb {
	cbs := make([]ChunkListenerCb, 0)
	for _, l := range cm.ChunkListeners {
		if l.ruleFN(X, Z) {
			cbs = append(cbs, l.cb)
		}
	}
	return cbs
}

func (cm *ChunkMirror) onNewChunk(p *packet.LevelChunk) {
	cm.lastChunkTime = time.Now()
	listeners := cm.getListeners(int(p.ChunkX), int(p.ChunkZ))
	if !cm.doCache && len(listeners) == 0 && !cm.isWaiting {
		fmt.Printf("Skip Chunk @ (%v,%v)", p.ChunkX, p.ChunkZ)
		return
	}
	c, err := chunk.NetworkDecode(block_define.AirRuntimeId, p.RawPayload, int(p.SubChunkCount))
	if err != nil {
		fmt.Printf("Chunk Mirror: onNewChunk, an error occur when decode network package (%v)\n", err)
		return
	}

	chunkX, chunkZ := p.ChunkX, p.ChunkZ
	pos := reflect_world.ChunkPos{chunkX, chunkZ}

	reflectChunkData := cm.reflectChunk(pos, c)

	if cm.doCache || cm.isWaiting {
		cm.providerMu.Lock()
		cm.saveChunk(pos, reflectChunkData)

		if cm.hasCache(pos) {
			fmt.Printf("Chunk Mirror: Update Cache Chunk (%v,%v)\n", chunkX, chunkZ)
		} else {
			fmt.Printf("Chunk Mirror: New Cache Chunk (%v,%v)\n", chunkX, chunkZ)
		}
		cm.cacheMap[pos] = time.Now()
		if cm.isWaiting {
			cm.isWaiting = false
			close(cm.waitLock)
		}
		cm.providerMu.Unlock()
		//loadedChunk, err := cm.GetCachedChunk(pos)
		//if err != nil {
		//	fmt.Printf("On Chunk Loaded, an error occoured! (%v)\n", err)
		//}
		//if len(reflectChunkData.E) > 0 {
		//	fmt.Printf("%v,%v", loadedChunk.E, loadedChunk.AuxNbtInfo)
		//}
	}
	for _, listener := range listeners {
		listener(pos, reflectChunkData)
	}
}
