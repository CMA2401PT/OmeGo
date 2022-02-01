package chunk_mirror

import (
	"encoding/csv"
	"fmt"
	"gopkg.in/yaml.v3"
	"main.go/dragonfly/server/world"
	"main.go/dragonfly/server/world/chunk"
	"main.go/minecraft/protocol/packet"
	"main.go/plugins/chunk_mirror/provider"
	"main.go/plugins/define"
	"main.go/task"
	block_define "main.go/world/define"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

type ChunkMirror struct {
	WorldDir             string `yaml:"world_dir"`
	ConfigExpiredTimeStr string `yaml:"expired_time"`
	taskIO               *task.TaskIO
	WorldProvider        *provider.Provider
	providerMu           sync.Mutex
	cacheRecordFile      string
	listenerDestroyFn    func()
	cacheMap             map[world.ChunkPos]time.Time
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
			cm.cacheMap[world.ChunkPos{int32(X), int32(Z)}] = T
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
	err = yaml.Unmarshal(config, &cm)
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
	cm.WorldProvider, err = provider.New(cm.WorldDir)
	if err != nil {
		panic(fmt.Sprintf("Chunk Mirror: Load/Create Chunk @ %v fail (%v)", cm.WorldDir, err))
	}
	cm.cacheRecordFile = path.Join(cm.WorldDir, "cache_log.txt")
	cm.cacheMap = make(map[world.ChunkPos]time.Time)
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

func (cm *ChunkMirror) GetCachedChunk(pos world.ChunkPos) (c *chunk.Chunk, err error) {
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
func (cm *ChunkMirror) HasCache(pos world.ChunkPos) bool {
	cm.providerMu.Lock()
	defer cm.providerMu.Unlock()
	return cm.hasCache(pos)
}
func (cm *ChunkMirror) hasCache(pos world.ChunkPos) bool {
	T, hasK := cm.cacheMap[pos]
	if hasK {
		if T.After(cm.expiredTime) {
			return true
		}
	}
	return false
}

func (cm *ChunkMirror) onNewChunk(p *packet.LevelChunk) {
	defer func() {
		if info := recover(); info != nil {
			fmt.Printf("Chunk Mirror: onNewChunk, panic happen (%v) @ (%v,%v)\n", info, p.ChunkX, p.ChunkZ)
			c, err := chunk.NetworkDecode(block_define.AirRuntimeId, p.RawPayload, int(p.SubChunkCount))
			if !cm.doCache {
				fmt.Printf("Skip Chunk @ (%v,%v)", p.ChunkX, p.ChunkZ)
				return
			}
			if err != nil {
				fmt.Printf("Chunk Mirror: onNewChunk, an error occur when decode network package (%v)\n", err)
				return
			}
			chunkX, chunkZ := p.ChunkX, p.ChunkZ
			pos := world.ChunkPos{chunkX, chunkZ}

			err = cm.WorldProvider.SaveChunk(pos, c)
			fmt.Println(err)
		}
	}()
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
	pos := world.ChunkPos{chunkX, chunkZ}

	err = cm.WorldProvider.SaveChunk(pos, c)
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
