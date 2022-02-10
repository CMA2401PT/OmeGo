package builder

import (
	"fmt"
	"main.go/plugins/builder/define"
	"main.go/plugins/builder/ir"
	"main.go/plugins/builder/ir/subChunk/oct"
	"main.go/plugins/builder/loader/bdump"
	"main.go/task"
	"strconv"
	"time"
)

type Processor struct {
	taskIO        *task.TaskIO
	log           func(isJson bool, data string)
	lastStructure *[]ir.AnchoredChunk
	lastIr        *ir.IR
	OpCounter     int
	BlockCounter  int
	expectSpeed   float32
	startTime     time.Time
}

func (p *Processor) AutoFix(player string) {
	if p.lastStructure == nil {
		fmt.Println("Structure Not Found!")
		return
	}
	resp := <-p.taskIO.SendCmdAndWaitForResponse(fmt.Sprintf("execute %v ~~~ testforblock ~~~ air", player), task.AfterResponseFeedBackOff)
	fmt.Println(resp)
	for _, r := range resp.OutputMessages {
		if r.Success {
			X, _ := strconv.Atoi(r.Parameters[0])
			Z, _ := strconv.Atoi(r.Parameters[1])
			chunkX := define.PE((X >> 4) << 4)
			chunkZ := define.PE((Z >> 4) << 4)
			for _, c := range *p.lastStructure {
				if c.C == nil {
					continue
				}
				if c.C.X == chunkX && c.C.Z == chunkZ {
					fmt.Println("Fix Start")
					p.Move(chunkX, chunkZ)
					p.ResetCounter()
					opsGroup := c.C.GetOps(p.lastIr.ID2Block)
					p.LaunchOpsGroup(opsGroup)
				}
			}
		}
	}
}

func (p *Processor) ResetCounter() {
	p.startTime = time.Now()
	p.BlockCounter = 0
	p.OpCounter = 0
}

func (p *Processor) process(cmds []string) {
	if len(cmds) < 5 {
		fmt.Println("Format Error!:  build [name] x y z")
		return
	}
	p.expectSpeed = 800
	bdxPath := cmds[1]
	//if bdxPath == "fix" {
	//	p.AutoFix("2401PT")
	//	return
	//}
	oX, err := strconv.Atoi(cmds[2])
	if err != nil {
		fmt.Println("X 坐标 ", cmds[2], " 不是一个整数")
		return
	}
	oY, err := strconv.Atoi(cmds[2])
	if err != nil {
		fmt.Println("Y 坐标 ", cmds[2], " 不是一个整数")
		return
	}
	oZ, err := strconv.Atoi(cmds[2])
	if err != nil {
		fmt.Println("Z 坐标 ", cmds[2], " 不是一个整数")
		return
	}
	irStructure := ir.NewIR(255, &oct.SubChunkStorage{})
	p.lastIr = irStructure
	err = bdump.LoadBDX(bdxPath, define.PE(oX), define.PE(oY), define.PE(oZ), false, func(s string) {
		fmt.Println(s)
	}, irStructure)
	if err != nil {
		fmt.Println(err)
		return
	}
	anchoredChunks := irStructure.GetAnchoredChunk()
	p.lastStructure = &anchoredChunks
	p.ResetCounter()
	//return
	for i, c := range anchoredChunks {
		p.taskIO.SendChat(fmt.Sprintf("Progress: [%v]/[%v]", i+1, len(anchoredChunks)))
		if c.C == nil {
			fmt.Println("Move to Next Anchor")
			p.Move(c.MovePos[0], c.MovePos[1])
		} else {
			fmt.Println("Launch Next Group")
			opsGroup := c.C.GetOps(irStructure.ID2Block)
			p.LaunchOpsGroup(opsGroup)
			fmt.Printf("[%v]/[%v] Ops:    %v\n", i, len(anchoredChunks), p.OpCounter)
			fmt.Printf("[%v]/[%v] Blocks: %v\n", i, len(anchoredChunks), p.BlockCounter)
		}
	}
	fmt.Println("Construction Complete!")
	fmt.Println("Ops:    ", p.OpCounter)
	fmt.Println("Blocks: ", p.BlockCounter)
}

func (p *Processor) Move(X, Z define.PE) {
	p.taskIO.SendCmd(fmt.Sprintf("tp @s %v 255 %v", X, Z))
	//time.Sleep(time.Second * 3)
}

func (p *Processor) LaunchOpsGroup(group *define.OpsGroup) {
	var blk *define.BlockDescribe
	var Xr, Yr, Zr define.PE
	var cmd string

	tmpBlocks := p.BlockCounter
	for _, op := range *group.NormalOps {
		//fmt.Println(op)

		blk = &group.Palette[op.BlockID]
		if op.Spawn == 0 && op.Level == define.BuildLevelBlock {
			cmd = fmt.Sprintf("setblock %d %d %d %v %d", op.Pos[0], op.Pos[1], op.Pos[2], blk.Name, blk.Meta)
			p.BlockCounter += 1
		} else {
			Xr = 1 << op.Level
			if (op.Spawn & define.SPAWN_X) != 0 {
				Xr <<= 1
			}
			Yr = 1 << op.Level
			if (op.Spawn & define.SPAWN_Y) != 0 {
				Yr <<= 1
			}
			Zr = 1 << op.Level
			if (op.Spawn & define.SPAWN_Z) != 0 {
				Zr <<= 1
			}
			cmd = fmt.Sprintf("fill %d %d %d %d %d %d %v %d", op.Pos[0], op.Pos[1], op.Pos[2], op.Pos[0]+Xr-1, op.Pos[1]+Yr-1, op.Pos[2]+Zr-1, blk.Name, blk.Meta)

			p.BlockCounter += int(Xr * Yr * Zr)
		}
		p.OpCounter += 1

		if p.BlockCounter-tmpBlocks > 64 {
			tmpBlocks = p.BlockCounter
			expectTime := p.startTime.Add(time.Duration(float32(p.BlockCounter)/p.expectSpeed) * time.Second)
			if expectTime.After(time.Now()) {
				sleepTime := expectTime.Sub(time.Now())
				fmt.Printf("Delay %v\n", sleepTime)
				time.Sleep(sleepTime)
			}
		}
		//fmt.Println(cmd)
		p.taskIO.SendCmdNoLock(cmd)
	}
}

func (p *Processor) close() {

}
