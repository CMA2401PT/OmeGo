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
	name := "#NONAME_STRUCTURE#"
	//if len(cmd) > 4 {
	//	name = cmd[4]
	//}

	startPosStr := strings.Split(startStr, ":")
	if len(startPosStr) != 3 {
		fmt.Println("Src format error")
		return
	}
	startX, err := strconv.Atoi(startPosStr[0])
	if err != nil {
		fmt.Println("Src [X] format error")
		return
	}
	startY, err := strconv.Atoi(startPosStr[1])
	if err != nil {
		fmt.Println("Src [Y] format error")
		return
	}
	startZ, err := strconv.Atoi(startPosStr[2])
	if err != nil {
		fmt.Println("Src [Z] format error")
		return
	}
	endPosStr := strings.Split(endStr, ":")
	if len(startPosStr) != 3 {
		fmt.Println("Src format error")
		return
	}
	endX, err := strconv.Atoi(endPosStr[0])
	if err != nil {
		fmt.Println("End [X] format error")
		return
	}
	endY, err := strconv.Atoi(endPosStr[1])
	if err != nil {
		fmt.Println("End [Y] format error")
		return
	}
	endZ, err := strconv.Atoi(endPosStr[2])
	if err != nil {
		fmt.Println("End [Z] format error")
		return
	}

	if startX > endX {
		t := startX
		startX = endX
		endX = t
	}
	if startZ > endZ {
		t := startZ
		startZ = endZ
		endZ = t
	}

	dstPosStr := strings.Split(dstStr, ":")
	if len(dstPosStr) != 2 && len(dstPosStr) != 1 {
		fmt.Println("Dst format error")
		return
	}
	dstDir := dstPosStr[0]
	if len(dstPosStr) == 2 {
		name = dstPosStr[1]
	}
	//dstX, dstZ := startX, startZ
	//if len(dstPosStr) == 3 {
	//	dstX, err = strconv.Atoi(dstPosStr[1])
	//	if err != nil {
	//		fmt.Println("Dst [X] format error")
	//	}
	//	dstZ, err = strconv.Atoi(dstPosStr[2])
	//	if err != nil {
	//		fmt.Println("Dst [Z] format error")
	//	}
	//}
	provider, err := mcdb.New(dstDir, reflect_world.Overworld)
	provider.D.LevelName = provider.D.LevelName + "_structure_[" + name + "]"
	if err != nil {
		fmt.Printf("CDump: Load/Create World @ %v fail (%v)\n", dstDir, err)
		return
	}

	operateLogFile := path.Join(dstDir, "Structure.txt")
	logFile, err := os.OpenFile(operateLogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 644)
	if err != nil && os.IsNotExist(err) {
		fmt.Printf(fmt.Sprintf("CDump: cannot create or append operate log %v (%v)", operateLogFile, err))
		return
	}
	startCX, startCZ, endCX, endCZ := startX/16-1, startZ/16-1, endX/16+1, endZ/16+1
	startXR, startZR, endXR, endZR := startCX*16, startCZ*16, endCX*16+15, endCZ*16+15
	fmt.Fprintf(logFile, "// Structure Dump Record: by cmd %v\n", cmd)
	fmt.Fprintf(logFile, "// Structure Dump Record: dump begin @ time: %v\n", time.Now().Format("[2006-01-02-15:04:05]"))
	fmt.Fprintf(logFile, "// Structure Dump Record: structure name is: %v\n", name)
	fmt.Fprintf(logFile, "// Structure Dump Record: chunks to dump begin @ chunk [X=%v,Z=%v]\n", startCX, startCZ)
	fmt.Fprintf(logFile, "// Structure Dump Record: chunks to dump end @ chunk [X=%v,Z=%v]\n", endCX, endCZ)
	fmt.Fprintf(logFile, "// Structure Dump Record: blocks availabe in range [X=%v,Z=%v] ~ [X=%v,Z=%v]\n", startXR, startZR, endXR, endZR)
	fmt.Fprintf(logFile, "// Structure Dump Record: load anchor point is [X=%v,Z=%v]\n", startX, startZ)
	fmt.Fprintf(logFile, "#> Structure %v %v %v %v %v %v %v %v\n", name, startX, startY, startZ, endX, endY, endZ, time.Now().Format("[2006-01-02-15:04:05]"))
	fmt.Fprintf(logFile, "// Begining Dump Log:\n")

	dumpComplete := false

	lineLogger := func(fmtS string, data ...interface{}) {
		line := fmt.Sprintf(fmtS, data...)
		fmt.Fprintf(logFile, "// log: %v %v\n", line, time.Now().Format("[2006-01-02-15:04:05]"))
		fmt.Println(line)
	}

	defer func() {
		if dumpComplete {
			fmt.Fprintf(logFile, "// Structure Dump Record: Dump Complete\n")
		} else {
			fmt.Fprintf(logFile, "// Structure Dump Record: Dump Not Complete\n")
		}
		fmt.Fprintf(logFile, "// Structure Dump Record: Record Terminate\n\n")
		logFile.Close()
		provider.Close()
	}()
	p.dump(startCX, endCX, startCZ, endCZ, 0, 0, provider, lineLogger)
	dumpComplete = true
	//offsetX, offsetZ := dstX-startX, dstZ-startZ
	//var SX, EX, SZ, EZ int
	//if startX > endX {
	//	SX = endX
	//	EX = startX
	//} else {
	//	EX = endX
	//	SX = startX
	//}
	//if startZ > endZ {
	//	SZ = endZ
	//	EZ = startZ
	//} else {
	//	EZ = endZ
	//	SZ = startZ
	//}

}

func (p *Processor) dump(SX, EX, SZ, EZ, offsetX, offsetZ int, provider *mcdb.Provider, lineLogger func(fmtS string, data ...interface{})) {
	cachedStatus := make(map[reflect_world.ChunkPos]bool)

	expiredTime := time.Now()
	lineLogger("cdumping inital delay")
	<-p.cm.WaitIdle(time.Second * 2)
	lineLogger("Begin cdumping ...")
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
			lineLogger("Chunk acquire fail @ chunk %v %v", x, z)
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
			lineLogger("Chunk reacquire @ %v", pos)
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
			} else {
				lineLogger("Chunk reacquire fail @ %v", pos)
			}
		}
	}
	for pos, succ := range cachedStatus {
		if !succ {
			lineLogger("CDump: Miss Chunk @ (%v %v)", pos.X(), pos.Z())
		}
	}
	lineLogger("Writing Special Items")
	p.cm.WriteSpecial(provider)
	lineLogger("CDump Completed!")
}

// 主城区 cdump 19250:19250 20750:20750 cma
// 仓库  cdump  -30000:30000 28000:28000 cma
// 工会1 cdump  -21050:10050 -19950:11450 cma
// 2会   cdump -22000:16650 -21000:17650 cma
// 7hui cdump  -20450:12050 -20050:12950 cma
