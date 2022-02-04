package cdump

import (
	"fmt"
	"main.go/plugins/chunk_mirror"
	reflect_world "main.go/plugins/chunk_mirror/server/world"
	"main.go/plugins/chunk_mirror/server/world/mcdb"
	"main.go/task"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Processor struct {
	cm     *chunk_mirror.ChunkMirror
	taskIO *task.TaskIO
	log    func(isJson bool, data string)
}

func (p *Processor) process(cmd []string) {
	if len(cmd) < 4 {
		fmt.Println("not enough arguments... more arguments")
		return
	}
	startStr := cmd[1]
	endStr := cmd[2]
	dstStr := cmd[3]
	name := "#NONAME#"
	if len(cmd) > 4 {
		name = cmd[4]
	}

	startPosStr := strings.Split(startStr, ":")
	if len(startPosStr) != 2 {
		fmt.Println("Src format error")
		return
	}
	startX, err := strconv.Atoi(startPosStr[0])
	if err != nil {
		fmt.Println("Src [X] format error")
	}
	startZ, err := strconv.Atoi(startPosStr[1])
	if err != nil {
		fmt.Println("Src [Z] format error")
	}
	endPosStr := strings.Split(endStr, ":")
	if len(startPosStr) != 2 {
		fmt.Println("Src format error")
		return
	}
	endX, err := strconv.Atoi(endPosStr[0])
	if err != nil {
		fmt.Println("End [X] format error")
	}
	endZ, err := strconv.Atoi(endPosStr[1])
	if err != nil {
		fmt.Println("End [Z] format error")
	}
	dstPosStr := strings.Split(dstStr, ":")
	if len(dstPosStr) != 3 && len(dstPosStr) != 1 {
		fmt.Println("Dst format error")
		return
	}
	dstDir := dstPosStr[0]
	dstX, dstZ := startX, startZ
	if len(dstPosStr) == 3 {
		dstX, err = strconv.Atoi(dstPosStr[1])
		if err != nil {
			fmt.Println("Dst [X] format error")
		}
		dstZ, err = strconv.Atoi(dstPosStr[2])
		if err != nil {
			fmt.Println("Dst [Z] format error")
		}
	}
	provider, err := mcdb.New(dstDir, reflect_world.Overworld)
	if err != nil {
		fmt.Printf("CDump: Load/Create World @ %v fail (%v)\n", dstDir, err)
		return
	}

	operateLogFile := path.Join(dstDir, "operate_log.txt")
	logFile, err := os.OpenFile(operateLogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 644)
	if err != nil && os.IsNotExist(err) {
		fmt.Printf(fmt.Sprintf("CDump: cannot create or append operate log %v (%v)", operateLogFile, err))
		return
	}
	startX, startZ, endX, endZ, dstX, dstZ = startX>>4, startZ>>4, endX>>4, endZ>>4, dstX>>4, dstZ>>4
	offsetX, offsetZ := dstX-startX, dstZ-startZ
	var SX, EX, SZ, EZ int
	if startX > endX {
		SX = endX
		EX = startX
	} else {
		EX = endX
		SX = startX
	}
	if startZ > endZ {
		SZ = endZ
		EZ = startZ
	} else {
		EZ = endZ
		SZ = startZ
	}

	fmt.Fprintf(logFile, "#Record %v %v:%v:%v %v:%v %v:%v [fmt NAME FILE:StartX:StratZ (FILE)EndX:EndZ (Origin):StartX:StartZ]\n", name, dstDir, (SX+offsetX)*16, (SZ+offsetZ)*16, (EX+offsetX+1)*16, (EZ+offsetZ+1)*16, SX*16, SZ*16)
	fmt.Fprintf(logFile, "#CMD %v @ %v\n", cmd, time.Now().Format("[2006-01-02-15:04:05]"))

	logFile.Close()
	p.dump(SX, EX, SZ, EZ, offsetX, offsetZ, provider)
	provider.Close()
}

func (p *Processor) dump(SX, EX, SZ, EZ, offsetX, offsetZ int, provider *mcdb.Provider) {
	cachedStatus := make(map[reflect_world.ChunkPos]bool)

	expiredTime := time.Now()
	<-p.cm.WaitIdle(time.Second * 2)
	fmt.Println("Begin Cdumping ...")
	rev := true
	nextChunk := func(x, z int) {
		pos := reflect_world.ChunkPos{int32(x + offsetX), int32(z + offsetZ)}
		hasCache := p.cm.HasCache(pos, expiredTime)
		cd := <-p.cm.RequireChunk(&chunk_mirror.ChunkReq{
			Dry:             false,
			X:               x,
			Z:               z,
			AllowCacheAfter: expiredTime,
			Active:          true,
			GetTimeOut:      time.Second * 30,
			FarPoint:        nil,
		})
		if !hasCache {
			<-p.cm.WaitIdle(time.Second * 1)
		}
		if cd != nil {
			cachedStatus[pos] = true
			chunk_mirror.SaveChunk(reflect_world.ChunkPos{int32(x), int32(z)}, cd, provider)
		} else {
			cachedStatus[pos] = false
		}
	}

	for x := SX; x <= EX; x++ {
		rev = !rev
		if rev {
			for z := SZ; z <= EZ; z++ {
				nextChunk(x, z)
			}
		} else {
			for z := EZ; z >= SZ; z-- {
				nextChunk(x, z)
			}
		}

	}
	for pos, succ := range cachedStatus {
		if !succ {
			cd := <-p.cm.RequireChunk(&chunk_mirror.ChunkReq{
				Dry:             false,
				X:               int(pos.X()),
				Z:               int(pos.Z()),
				AllowCacheAfter: expiredTime,
				Active:          true,
				GetTimeOut:      time.Second * 30,
				FarPoint:        nil,
			})
			if cd != nil {
				chunk_mirror.SaveChunk(reflect_world.ChunkPos{pos.X() + int32(offsetX), pos.Z() + int32(offsetZ)}, cd, provider)
				cachedStatus[pos] = true
			}
		}
	}
	for pos, succ := range cachedStatus {
		if !succ {
			fmt.Printf("CDump: Miss Chunk @ (%v %v)\n", pos.X(), pos.Z())
		}
	}
	p.cm.WriteSpecial(provider)
	fmt.Println("CDump Completed!")
}

// 主城区 cdump 19250:19250 20750:20750 cma
// 仓库  cdump  -30000:30000 28000:28000 cma
// 工会1 cdump  -21050:10050 -19950:11450 cma
// 2会   cdump -22000:16650 -21000:17650 cma
// 7hui cdump  -20450:12050 -20050:12950 cma
