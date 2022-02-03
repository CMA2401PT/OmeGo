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

type ChunkMirror struct {
	WorldDir             string `yaml:"world_dir"`
	ConfigExpiredTimeStr string `yaml:"expired_time"`
	WorldMin             int    `yaml:"world_min"`
	WorldMax             int    `yaml:"world_max"`
	NeteaseAirRID        int
	MirrorAirRID         uint32
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
}

func (cm *ChunkMirror) LoadCacheRecordFile() {
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

func (cm *ChunkMirror) DumpCacheRecordFile() {
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
	cm.doCache = true
	var err error
	cm.WorldDir = "TmpWorld"
	cm.ConfigExpiredTimeStr = ""
	cm.WorldMin = -64
	cm.WorldMax = 320
	cm.worldRange = cube.Range{cm.WorldMin, cm.WorldMax}

	err = yaml.Unmarshal(config, &cm)
	if err != nil {
		panic("Chunk Mirror: cannot handle config")
	}
	richBlocks := RichBlocks{}
	err = json.Unmarshal(richBlocksData, &richBlocks)
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
	if err != nil {
		panic(fmt.Sprintf("Chunk Mirror: Load/Create Chunk @ %v fail (%v)", cm.WorldDir, err))
	}
	cm.cacheRecordFile = path.Join(cm.WorldDir, "cache_log.txt")
	cm.cacheMap = make(map[reflect_world.ChunkPos]time.Time)
	cm.providerMu = sync.Mutex{}
	cm.LoadCacheRecordFile()
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
	cm.DumpCacheRecordFile()
}

func (cm *ChunkMirror) Routine() {
	//cm.taskIO.WaitInit()
	//time.AfterFunc(time.Second*10, func() {
	//	fmt.Println("begin mirror chunk")
	//	cm.doCache = true
	//})
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

func (cm *ChunkMirror) GetCachedChunk(pos reflect_world.ChunkPos) (c *reflect_chunk.Chunk, err error) {
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	if !cm.hasCache(pos) {
		return nil, fmt.Errorf("chunk Mirror: Try Getting an non-cached Chunk")
	}
	loadChunk, e, err := cm.WorldProvider.LoadChunk(pos)
	if err != nil {
		fmt.Printf("Chunk Mirror: on Cached Chunk Load, an error occured (%v)", err)
		return nil, err
	}
	if !e {
		fmt.Printf("Chunk Mirror: on Get Cached Chunk Load, an error occured: Map Info say this chunk exist, but provider cannot find it %v", pos)
		return nil, fmt.Errorf("chunk Mirror: on Get Cached Chunk Load, an error occured: Map Info say this chunk exist, but provider cannot find it %v", pos)
	}
	return loadChunk, nil
}
func (cm *ChunkMirror) HasCache(pos reflect_world.ChunkPos) bool {
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	return cm.hasCache(pos)
}
func (cm *ChunkMirror) hasCache(pos reflect_world.ChunkPos) bool {
	T, hasK := cm.cacheMap[pos]
	if hasK {
		if T.After(cm.expiredTime) {
			return true
		}
	}
	return false
}

func (cm *ChunkMirror) saveChunk(pos reflect_world.ChunkPos, c *reflect_world.ChunkData) {
	// We allocate a new map for all block entities.
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
	if err := cm.WorldProvider.SaveChunk(pos, c.Chunk); err != nil {
		fmt.Printf("Chunk Mirror: error saving chunk %v to provider: %v", pos, err)
	}
	if err := cm.WorldProvider.SaveBlockNBT(pos, m); err != nil {
		fmt.Printf("Chunk Mirror: error saving block NBT in chunk %v to provider: %v", pos, err)
	}
}

func (cm *ChunkMirror) reflectChunk(pos reflect_world.ChunkPos, c *chunk.Chunk) error {
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
		nbtBlockRid := c.RuntimeID(uint8(blockPos.X()), int16(blockPos.Y()), uint8(blockPos.Z()), 0)
		reflectRid = cm.blockReflectMapping[nbtBlockRid]
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
				fmt.Printf("Chunk Mirror: Cannot Convert Nbt Block (%v -> %v) @ %v nbt: %v to a normal block! %v\n", nbtBlockRid, reflectRid, blockPos, nbt, b)
				continue
			}
			reflectChunkData.E[cube.Pos{blockPos.X(), blockPos.Y(), blockPos.Z()}] = wb
		} else {
			fmt.Printf("Chunk Mirror: Block (%v -> %v) @ %v nbt: %v cannot be a Nbt Block! %b\n", nbtBlockRid, reflectRid, blockPos, nbt, b)
		}
	}
	cm.saveChunk(pos, reflectChunkData)
	return nil
}

func (cm *ChunkMirror) onNewChunk(p *packet.LevelChunk) {
	c, err := chunk.NetworkDecode(block_define.AirRuntimeId, p.RawPayload, int(p.SubChunkCount))
	if !cm.doCache {
		fmt.Printf("Skip Chunk @ (%v,%v)", p.ChunkX, p.ChunkZ)
		return
	}
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	if err != nil {
		fmt.Printf("Chunk Mirror: onNewChunk, an error occur when decode network package (%v)\n", err)
		return
	}
	chunkX, chunkZ := p.ChunkX, p.ChunkZ
	pos := reflect_world.ChunkPos{chunkX, chunkZ}

	if !cm.doCache {
		fmt.Printf("Skip Chunk @ (%v,%v)", p.ChunkX, p.ChunkZ)
		return
	}
	if err != nil {
		fmt.Printf("Chunk Mirror: onNewChunk, an error occur when decode network package (%v)\n", err)
		return
	}
	err = cm.reflectChunk(pos, c)
	if err != nil {
		fmt.Println("Chunk Mirror: onNewChunk, an error occur when cache Chunk")
		return
	}
	if cm.hasCache(pos) {
		fmt.Printf("Chunk Mirror: Update Cache Chunk (%v,%v)\n", chunkX, chunkZ)
	} else {
		fmt.Printf("Chunk Mirror: New Cache Chunk (%v,%v)\n", chunkX, chunkZ)
	}
	cm.cacheMap[pos] = time.Now()
}
