package leveldb

import (
	"bufio"
	"fmt"
	"io"
	"main.go/plugins/builder/define"
	"main.go/plugins/builder/ir"
	"main.go/plugins/chunk_mirror"
	"main.go/plugins/chunk_mirror/server/world"
	"main.go/plugins/chunk_mirror/server/world/chunk"
	"main.go/plugins/chunk_mirror/server/world/mcdb"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Structure struct {
	Name       string
	Time       time.Time
	sX, sY, sZ int
	eX, eY, eZ int
}

func loadAvailable(dir string) (*mcdb.Provider, []Structure, error) {
	if stat, err := os.Stat(dir); !(err == nil && stat.IsDir()) {
		return nil, nil, fmt.Errorf("not a vaild folder!")
	}
	provider, err := mcdb.New(dir, world.Overworld)
	if err != nil {
		return nil, nil, fmt.Errorf("create provider err (%v)", err)
	}
	operateLogFile := path.Join(dir, "Structure.txt")
	logFile, err := os.OpenFile(operateLogFile, os.O_RDONLY, 644)
	defer logFile.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("Open Structure Record Error (%v)", err)
	}
	buf := bufio.NewReader(logFile)
	structures := make([]Structure, 0)
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		s := string(line)
		s = strings.TrimSpace(s)
		if !strings.HasPrefix(s, "#> ") {
			continue
		}
		elems := strings.Split(s, " ")
		if len(elems) != 10 {
			fmt.Println("Broken Record, ", s, " format should be : #> Structure #Name# #sX# #sY# #sZ# #eX# #eY# #eZ# [#Time#]")
			continue
		}
		d := Structure{
			Name: "",
			Time: time.Time{},
			sX:   0,
			sY:   0,
			sZ:   0,
			eX:   0,
			eY:   0,
			eZ:   0,
		}
		d.Name = elems[2]
		_names := []string{"startX", "startY", "startZ", "endX", "endZ", "endZ"}
		ptrs := []*int{&d.sX, &d.sY, &d.sZ, &d.eX, &d.eY, &d.eZ}
		flag := false
		for i := 0; i < 6; i++ {
			v, err := strconv.Atoi(elems[i+3])
			if err != nil {
				fmt.Println("Broken Record, ", s, ": ", _names[i], " is not a int")
				flag = true
				break
			}
			*ptrs[i] = v
		}
		if flag {
			continue
		}
		tm, err := time.Parse("[2006-01-02-15:04:05]", elems[9])
		if err != nil {
			fmt.Println("Broken Record, ", s, ": ", elems[9], " is not valid time")
			continue
		}
		d.Time = tm
		structures = append(structures, d)
	}

	return provider, structures, err
}

func TranslatePos(x, y, z int) (world.ChunkPos, uint8, int16, uint8) {
	chunkPos := world.ChunkPos{int32(x >> 4), int32(z >> 4)}
	return chunkPos, uint8(x), int16(y), uint8(z)
}

func blockPosFromNBT(data map[string]interface{}) (int, int, int) {
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	xInterface, _ := data["x"]
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	yInterface, _ := data["y"]
	//lint:ignore S1005 Double assignment is done explicitly to prevent panics.
	zInterface, _ := data["z"]
	x, _ := xInterface.(int32)
	y, _ := yInterface.(int32)
	z, _ := zInterface.(int32)
	return int(x), int(y), int(z)
}

func LoadLeveldb(folderPath string, sendNotice func(string), ir *ir.IR) error {
	provider, structures, err := loadAvailable(folderPath)
	if err != nil {
		return fmt.Errorf("Can not read structures record (%v)\n", err)
	}
	if len(structures) == 0 {
		return fmt.Errorf("No Available Structures")
	}
	sendNotice("Available Structures\n")
	for i, s := range structures {
		sendNotice(fmt.Sprintf("[%d]: %v start[X,Y,Z]=[%v,%v,%v]  end[X,Y,Z]=[%v,%v,%v] time:%v\n", i+1, s.Name, s.sX, s.sY, s.sZ, s.eX, s.eY, s.eZ, s.Time.Format("[2006-01-02-15:04:05]")))
	}
	sendNotice("build structure:\n")
	i := 0
	for i == 0 {
		line, _, err := bufio.NewReader(os.Stdin).ReadLine()
		if err != nil {
			sendNotice(fmt.Sprintf("Invaliad choice: %v, can only input [1-%d]\n", err, len(structures)))
			continue
		}
		l := strings.TrimSpace(string(line))
		choice, err := strconv.Atoi(l)
		if err != nil || choice < 1 || choice > len(structures) {
			sendNotice(fmt.Sprintf("Invaliad choice: can only input [1-%d]\n", len(structures)))
			continue
		}
		i = choice
	}
	i -= 1
	s := structures[i]
	sendNotice(fmt.Sprintf("Select: [%d] %v start[X,Y,Z]=[%v,%v,%v]  end[X,Y,Z]=[%v,%v,%v] time:%v\n", i+1, s.Name, s.sX, s.sY, s.sZ, s.eX, s.eY, s.eZ, s.Time.Format("[2006-01-02-15:04:05]")))
	totalRuntimeIDs := 0
	for _, rb := range chunk_mirror.RichBlocks.RichBlocks {
		if rb.ReflectRID > totalRuntimeIDs {
			totalRuntimeIDs = rb.ReflectRID
		}
	}
	IRMapping := make(define.BlockID2BlockDescribeMapping, totalRuntimeIDs+1, totalRuntimeIDs+1)
	for _, rb := range chunk_mirror.RichBlocks.RichBlocks {
		IRMapping[rb.ReflectRID] = &define.BlockDescribe{
			Name: rb.Name,
			Meta: uint16(rb.Val),
		}
	}
	ir.SetBlockID2BlockDescribeMapping(IRMapping)
	memoryChunk := make(map[world.ChunkPos]*chunk.Chunk)
	for cX := s.sX >> 4; cX <= s.eX>>4; cX++ {
		for cZ := s.sZ >> 4; cZ <= s.eZ>>4; cZ++ {
			c, found, err := provider.LoadChunk(world.ChunkPos{int32(cX), int32(cZ)})
			if err != nil {
				return (fmt.Errorf("chunk Load Error (%v)", err))
			} else if !found {
				return (fmt.Errorf("chunk Not Found"))
			}
			memoryChunk[world.ChunkPos{int32(cX), int32(cZ)}] = c
			nbts, err := provider.LoadBlockNBT(world.ChunkPos{int32(cX), int32(cZ)})
			if err != nil {
				sendNotice(fmt.Sprintf("An error occour in loading chunk nbt blocks (%v)\n", err))
				continue
			}

			for _, nbt := range nbts {
				x, y, z := blockPosFromNBT(nbt)
				_, _x, _y, _z := TranslatePos(x, y, z)
				sub := c.SubChunk(_y)
				if sub.Empty() || uint8(len(sub.Storages)) <= 0 {
					sendNotice(fmt.Sprintf("An error occour in find the name of a nbt block! (block is air)"))
					continue
				}
				blkID := sub.Storages[0].At(_x, uint8(y), _z)
				blkDescrib := IRMapping[blkID]
				if blkDescrib == nil {
					sendNotice(fmt.Sprintf("An error occour in find the name of a nbt block! (block invalid)"))
					continue
				}
				ir.SetNbtBlock(define.PE(x), define.PE(y), define.PE(z), *blkDescrib, nbt)
			}
		}
	}
	counter := 0
	air_counter := 0
	invalid_counter := 0

	for y := s.sY; y <= s.eY; y++ {
		for x := s.sX; x <= s.eX; x++ {
			for z := s.sZ; z <= s.eZ; z++ {
				chunkPos, _x, _y, _z := TranslatePos(x, y, z)
				c := memoryChunk[chunkPos]
				sub := c.SubChunk(_y)
				if sub.Empty() || uint8(len(sub.Storages)) <= 0 {
					air_counter += 1
					continue
				}
				blkID := sub.Storages[0].At(_x, uint8(y), _z)
				if IRMapping[blkID] == nil {
					invalid_counter += 1
					continue
				}
				counter += 1
				ir.SetBlockByID(define.PE(x), define.PE(y), define.PE(z), define.BLOCKID(blkID))
			}
		}
	}
	sendNotice(fmt.Sprintf("%v Air Block", air_counter))
	sendNotice(fmt.Sprintf("%v Blocks Invalid", invalid_counter))
	sendNotice(fmt.Sprintf("%v Blocks To Build", counter))
	return nil
}
