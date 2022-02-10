package main

import (
	"fmt"
	ir "main.go/plugins/builder/ir"
	"main.go/plugins/builder/loader/bdump"
	"main.go/plugins/builder/worker"
)

func main() {
	bdxPath := "C:\\projects\\OmeGo\\t.bdx"
	ir_structure := ir.NewIR(255)
	bdump.LoadBDX(bdxPath, 0, 0, 0, false, func(s string) {
		fmt.Println(s)
	}, ir_structure)
	anchored_chunks := ir_structure.GetAnchoredChunk()
	w := worker.DebugWorker{}
	for _, c := range anchored_chunks {
		if c.C == nil {
			fmt.Println("Move to ", c.MovePos)
		} else {
			opsGroup := c.C.GetOps(ir_structure.ID2Block)
			w.LaunchOpsGroup(opsGroup)
		}
	}
	fmt.Println(w.BlockCounter)
	fmt.Println(w.OpCounter)
}
