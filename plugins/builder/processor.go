package builder

import (
	"fmt"
	"main.go/plugins/builder/define"
	"main.go/plugins/builder/ir"
	"main.go/plugins/builder/ir/subChunk/plain"
	"main.go/plugins/builder/loader/bdump"
	"main.go/task"
	"strconv"
	"strings"
	"time"
)

type Processor struct {
	taskIO       *task.TaskIO
	log          func(isJson bool, data string)
	lastIr       *ir.IR
	BlockCounter int
	expectSpeed  float64
	startTime    time.Time
	Operator     string
	Busy         bool
	X, Y, Z      define.PE
	posSetted    bool
	speedFactor  float64
	stop         bool
}

func (p *Processor) getPos() (int, int, int, error) {
	resp := <-p.taskIO.SendCmdAndWaitForResponse(fmt.Sprintf("execute %v ~~~ testforblock ~~~ air", p.Operator), task.AfterResponseFeedBackRestore)
	fmt.Println(resp)
	for _, r := range resp.OutputMessages {
		if r.Success {
			X, _ := strconv.Atoi(r.Parameters[0])
			Y, _ := strconv.Atoi(r.Parameters[1])
			Z, _ := strconv.Atoi(r.Parameters[2])
			return X, Y, Z, nil
		}
	}
	return 0, 0, 0, fmt.Errorf("get Response: %v", resp)
}

func (p *Processor) AutoFix() {
	if p.Busy {
		fmt.Println("Bot Busy!")
		return
	} else {
		p.Busy = true
	}
	if p.lastIr == nil {
		p.Busy = false
		fmt.Println("Ir Not Found!")
		return
	}
	anchoredChunks := p.lastIr.GetAnchoredChunk()
	X, _, Z, err := p.getPos()
	if err != nil {
		fmt.Printf("无法自动获取坐标 (%v)\n", err)
	}
	chunkX := define.PE((X >> 4) << 4)
	chunkZ := define.PE((Z >> 4) << 4)
	for _, c := range anchoredChunks {
		if c.C == nil {
			continue
		}
		if c.C.X == chunkX && c.C.Z == chunkZ {
			p.ResetCounter()
			fmt.Println("Fix Start")
			p.taskIO.SendChat("Fix Start")
			p.Move(chunkX, chunkZ)
			opsGroup := c.C.GetOps(p.lastIr.ID2Block)
			p.LaunchOpsGroup(opsGroup)
		}
	}
	p.Busy = false
}

func (p *Processor) setSpeed(cmds []string) {
	var speedStr string
	if len(cmds) == 1 {
		speedStr = cmds[0]
		speed, err := strconv.Atoi(speedStr)
		if err != nil {
			fmt.Println("期望速度 ", speedStr, " 不是一个整数")
			return
		}
		p.expectSpeed = float64(speed)
		fmt.Printf("期望速度=%v block/s\n", p.expectSpeed)
	}

}

func (p *Processor) set(cmds []string) {
	var X, Y, Z int
	var err error
	if len(cmds) < 3 {
		X, Y, Z, err = p.getPos()
		if err != nil {
			fmt.Printf("无法自动获取坐标 (%v)\n", err)
			return
		}
	} else {
		X, err = strconv.Atoi(cmds[0])
		if err != nil {
			fmt.Println("X 坐标 ", cmds[0], " 不是一个整数")
			return
		}
		Y, err = strconv.Atoi(cmds[1])
		if err != nil {
			fmt.Println("Y 坐标 ", cmds[1], " 不是一个整数")
			return
		}
		Z, err = strconv.Atoi(cmds[2])
		if err != nil {
			fmt.Println("Z 坐标 ", cmds[2], " 不是一个整数")
			return
		}
	}
	var speedStr string
	if len(cmds) == 1 {
		speedStr = cmds[0]
	} else if len(cmds) == 4 {
		speedStr = cmds[3]
	}
	if speedStr != "" {
		speed, err := strconv.Atoi(speedStr)
		if err != nil {
			fmt.Println("期望速度 ", speedStr, " 不是一个整数")
			return
		}
		p.expectSpeed = float64(speed)
	}

	p.X = define.PE(X)
	p.Y = define.PE(Y)
	p.Z = define.PE(Z)
	p.posSetted = true
	fmt.Printf("起始坐标 X=%v, Y=%v, Z=%v, 期望速度=%v block/s\n", p.X, p.Y, p.Z, p.expectSpeed)
}

func (p *Processor) GetPos() (X, Y, Z define.PE, err error) {
	if !p.posSetted {
		return 0, 0, 0, fmt.Errorf("起始坐标未设定!")
	}
	return p.X, p.Y, p.Z, nil
}

func (p *Processor) buildStructure(cmds []string) {
	if len(cmds) < 1 {
		fmt.Println("Need File Name")
		return
	}
	filePath := cmds[0]
	irStructure := ir.NewIR(255, &plain.Storage{})

	if strings.HasSuffix(filePath, ".bdx") {
		X, Y, Z, err := p.GetPos()
		if err != nil {
			fmt.Printf("Get Pos error! (%v)\n", err)
			return
		}
		err = bdump.LoadBDX(filePath, X, Y, Z, false, func(s string) {
			fmt.Println(s)
		}, irStructure)
		if err != nil {
			fmt.Println("BDX decode error! ", err)
			return
		}
	}
	fmt.Printf("起始坐标 X=%v, Y=%v, Z=%v, 期望速度=%v block/s\n", p.X, p.Y, p.Z, p.expectSpeed)
	go p.BuildfromIR(irStructure)
}

func (p *Processor) BuildfromIR(ir *ir.IR) {
	if p.Busy {
		fmt.Println("Build ir fail: Bot Busy!")
	} else {
		p.Busy = true
	}
	p.lastIr = ir
	anchoredChunks := ir.GetAnchoredChunk()
	p.ResetCounter()
	lastTime := time.Now()
	lastBlocks := p.BlockCounter
	for i, c := range anchoredChunks {

		if c.C == nil {
			fmt.Println("Move to Next Anchor")
			p.Move(c.MovePos[0], c.MovePos[1])
		} else {
			opsGroup := c.C.GetOps(ir.ID2Block)
			p.LaunchOpsGroup(opsGroup)
			realSpeed := float64(p.BlockCounter-lastBlocks) / time.Now().Sub(lastTime).Seconds()
			if realSpeed < p.expectSpeed*0.9 {
				p.speedFactor = realSpeed / p.expectSpeed
			}
			lastTime = time.Now()
			lastBlocks = p.BlockCounter
			hint := fmt.Sprintf("Progress: Chunk [%v]/[%v] Total Blocks: %v Speed: %.2f Block/s", i+1, len(anchoredChunks), p.BlockCounter, float64(p.BlockCounter)/time.Now().Sub(p.startTime).Seconds())
			fmt.Println(hint)
			p.taskIO.SendChat(hint)
		}
		if p.stop {
			p.stop = false
			return
		}
	}
	p.Busy = false
	fmt.Println("Construction Complete!")
	fmt.Println("Blocks: ", p.BlockCounter)
	p.taskIO.SendChat("Construction Complete!")
}

func (p *Processor) ResetCounter() {
	p.startTime = time.Now()
	p.BlockCounter = 0
	//p.OpCounter = 0
}

func (p *Processor) process(cmds []string) {
	if len(cmds) < 2 {
		fmt.Println("Insufficient args")
		fmt.Println("build set")
		fmt.Println("build [file_name]")
		fmt.Println("build fix")
		fmt.Println("build speed")
	}
	if cmds[1] == "fix" {
		p.AutoFix()
	} else if cmds[1] == "set" {
		p.set(cmds[2:])
	} else if cmds[1] == "speed" {
		p.setSpeed(cmds[2:])
	} else if cmds[1] == "stop" {
		p.stop = true
	} else {
		p.buildStructure(cmds[1:])
	}
}

func (p *Processor) Move(X, Z define.PE) {
	<-p.taskIO.SendCmdAndWaitForResponse(fmt.Sprintf("tp @s %v 128 %v", X, Z), task.AfterResponseFeedBackRestore)
	time.Sleep(time.Second * 1)
}

func (p *Processor) LaunchOpsGroup(group *define.OpsGroup) {
	var blk *define.BlockDescribe
	//var Xr, Yr, Zr define.PE
	var cmd string
	//tmpBlocks := p.BlockCounter
	for _, op := range *group.NormalOps {
		if p.stop {
			return
		}
		//fmt.Println(op)

		blk = &group.Palette[op.BlockID]
		p.BlockCounter += 1
		cmd = fmt.Sprintf("setblock %d %d %d %v %d", op.Pos[0], op.Pos[1], op.Pos[2], blk.Name, blk.Meta)
		//if op.Spawn == 0 && op.Level == define.BuildLevelBlock {
		//	cmd = fmt.Sprintf("setblock %d %d %d %v %d", op.Pos[0], op.Pos[1], op.Pos[2], blk.Name, blk.Meta)
		//	p.BlockCounter += 1
		//} else {
		//	Xr = 1 << op.Level
		//	if (op.Spawn & define.SPAWN_X) != 0 {
		//		Xr <<= 1
		//	}
		//	Yr = 1 << op.Level
		//	if (op.Spawn & define.SPAWN_Y) != 0 {
		//		Yr <<= 1
		//	}
		//	Zr = 1 << op.Level
		//	if (op.Spawn & define.SPAWN_Z) != 0 {
		//		Zr <<= 1
		//	}
		//	cmd = fmt.Sprintf("fill %d %d %d %d %d %d %v %d", op.Pos[0], op.Pos[1], op.Pos[2], op.Pos[0]+Xr-1, op.Pos[1]+Yr-1, op.Pos[2]+Zr-1, blk.Name, blk.Meta)
		//
		//	p.BlockCounter += int(Xr * Yr * Zr)
		//}
		//p.OpCounter += 1

		//if p.BlockCounter-tmpBlocks > 32 {
		//
		//	delayMS := float32(p.BlockCounter-tmpBlocks) / (p.expectSpeed / 1000)
		//	sleepTime := time.Duration(delayMS) * time.Millisecond
		//	fmt.Println(sleepTime)
		//	time.Sleep(sleepTime)
		//	//expectTime := p.startTime.Add(time.Duration(float32(p.BlockCounter)/p.expectSpeed) * time.Second)
		//	//if expectTime.After(time.Now()) {
		//	//	sleepTime := expectTime.Sub(time.Now())
		//	//	//fmt.Println(sleepTime)
		//	//	time.Sleep(sleepTime)
		//	//}
		//	tmpBlocks = p.BlockCounter
		//}
		//fmt.Println(cmd)
		p.taskIO.SendCmdNoLock(cmd)
		time.Sleep(time.Duration((p.speedFactor/p.expectSpeed)*1000*1000) * time.Microsecond)
	}
}

func (p *Processor) close() {

}
