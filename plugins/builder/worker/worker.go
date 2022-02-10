package worker

import (
	"fmt"
	"main.go/plugins/builder/define"
	"main.go/task"
	"sync"
)

type DebugWorker struct {
	Mu           sync.Mutex
	taskIO       *task.TaskIO
	OpCounter    int
	BlockCounter int
}

func (w *DebugWorker) NotifyStart() {
	fmt.Println("Task Start")
	w.BlockCounter = 0
	w.OpCounter = 0
	w.Mu.Lock()
}

func (w *DebugWorker) Notify(info string) {
	fmt.Println("Notify: ", info)
	w.Mu.Lock()
}

func (w *DebugWorker) NotifyEnd() {
	fmt.Println("Task Accomplished")
	w.Mu.Lock()
}

func (w *DebugWorker) Move(X, Y, Z define.PE) {
	fmt.Println(fmt.Sprintf("tp @s %v %v %v", X, Y, Z), task.AfterResponseFeedBackOff)
}

func (w *DebugWorker) LaunchOpsGroup(opsGroup *define.OpsGroup) {
	var blk *define.BlockDescribe
	var Xr, Yr, Zr define.PE
	var cmd string
	for _, op := range *opsGroup.NormalOps {
		w.OpCounter += 1
		blk = &opsGroup.Palette[op.BlockID]
		if op.Spawn == 0 && op.Level == define.BuildLevelBlock {
			cmd = fmt.Sprintf("setblock %d %d %d %v %d", op.Pos[0], op.Pos[1], op.Pos[2], blk.Name, blk.Meta)
			w.BlockCounter += 1
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
			cmd = fmt.Sprintf("[%d^%d] fill %d %d %d %d %d %d %v %d", op.Level, op.Spawn, op.Pos[0], op.Pos[1], op.Pos[2], op.Pos[0]+Xr-1, op.Pos[1]+Yr-1, op.Pos[2]+Zr-1, blk.Name, blk.Meta)

			w.BlockCounter += int(Xr * Yr * Zr)
		}
		fmt.Sprintf(cmd)
	}
}

type Worker interface {
	Move(X, Y, Z define.PE)
	LaunchOpsGroup(opsGroup *define.OpsGroup)
	NotifyStart()
	Notify(info string)
	NotifyEnd()
}
