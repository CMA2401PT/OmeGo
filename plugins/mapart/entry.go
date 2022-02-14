package mapart

import (
	"fmt"
	"gopkg.in/yaml.v3"
	builder "main.go/plugins/builder"
	builder_define "main.go/plugins/builder/define"
	"main.go/plugins/define"
	"main.go/task"
	"main.go/utils"
	"strings"
)

type CmdSource struct {
	RegName string `yaml:"reg_name"`
	Plugin  string `yaml:"plugin"`
	Prefix  string `yaml:"prefix"`
}

type MapArt struct {
	Sources []CmdSource `yaml:"sources"`
	taskIO  *task.TaskIO
	Builder *builder.Builder
	blockFN func(X, Y, Z int, blockName string, blockData int)
}

func (o *MapArt) New(config []byte) define.Plugin {
	o.Sources = make([]CmdSource, 0)
	err := yaml.Unmarshal(config, o)
	if err != nil {
		panic(err)
	}
	return o
}

func (o *MapArt) Routine() {

}

func (o *MapArt) Close() {

}

func (o *MapArt) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	o.taskIO = taskIO
	for _, s := range o.Sources {
		src := collaborationContext[s.Plugin].(define.StringReadInterface)
		src.RegStringInterceptor(s.RegName, func(isJson bool, data string) (bool, string) {
			return o.onNewText(s.Plugin, s.Prefix, data)
		})
	}
	o.Builder = collaborationContext["builder"].(*builder.Builder)
	return o
}

func (o *MapArt) onNewText(fromPlugin string, prefix string, data string) (bool, string) {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, prefix) {
		//catch
		cmds := utils.Compact(data)
		if cmds[0] == prefix {
			o.invoke(cmds[1:])
			return true, ""
		}
	}
	// fall through
	return false, data
}

func (o *MapArt) invoke(cmds []string) {
	filePath := ""
	var MapX, MapY, MapZ int
	MapX = 1
	MapY = 0
	MapZ = 1
	if len(cmds) < 1 {
		fmt.Println("Insufficient Args: MapArt [filePath] -mapX x -mapY y -mapZ z")
		return
	}
	filePath = cmds[0]
	cmds = cmds[1:]
	if len(cmds) > 1 {

	}
	utils.SimplePrase(&cmds, []string{"-mapX", "-X"}, &MapX, true)
	utils.SimplePrase(&cmds, []string{"-mapY", "-Y"}, &MapY, true)
	utils.SimplePrase(&cmds, []string{"-mapZ", "-Z"}, &MapZ, true)
	ir, err := o.Builder.GetIR()
	if err != nil {
		fmt.Printf("MapArt: Get IR Error: (%v)\n", err)
		return
	}
	fmt.Printf("MapArt: File %v X=%v, Y=%v Z=%v\n", filePath, MapX, MapY, MapZ)
	blocFn := func(X, Y, Z int, blockName string, blockData int) {
		ir.SetBlock(builder_define.PE(X), builder_define.PE(Y), builder_define.PE(Z), builder_define.BlockDescribe{
			Name: blockName,
			Meta: uint16(blockData),
		})
	}
	if err := mapArt(filePath, MapX, MapY, MapZ, blocFn); err != nil {
		fmt.Printf("MapArt: Dither Error: (%v)\n", err)
		return
	}
	go o.Builder.BuildIR(ir)
}
