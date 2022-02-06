package world_mirror

import (
	"gopkg.in/yaml.v3"
	"main.go/plugins/chunk_mirror"
	"main.go/plugins/define"
	"main.go/task"
	"strings"
)

type CmdSource struct {
	RegName string `yaml:"reg_name"`
	Plugin  string `yaml:"plugin"`
	Prefix  string `yaml:"prefix"`
}

type WorldMirror struct {
	Sources   []CmdSource `yaml:"sources"`
	LogName   string      `yaml:"log_name"`
	LogPlugin string      `yaml:"log_plugin"`
	taskIO    *task.TaskIO
	log       func(isJson bool, data string)
	processor *Processor
}

func (o *WorldMirror) New(config []byte) define.Plugin {
	o.Sources = make([]CmdSource, 0)
	o.LogName = ""
	o.LogPlugin = "storage"
	err := yaml.Unmarshal(config, o)
	o.processor = &Processor{}
	if err != nil {
		panic(err)
	}
	return o
}

func (o *WorldMirror) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	o.taskIO = taskIO
	if o.LogName != "" {
		o.log = collaborationContext[o.LogPlugin].(define.StringWriteInterface).RegStringSender(o.LogName)
	}
	for _, s := range o.Sources {
		src := collaborationContext[s.Plugin].(define.StringReadInterface)
		src.RegStringInterceptor(s.RegName, func(isJson bool, data string) (bool, string) {
			return o.onNewText(s.Plugin, s.Prefix, data)
		})
	}
	o.processor.taskIO = taskIO
	o.processor.cm = collaborationContext["chunk_mirror"].(*chunk_mirror.ChunkMirror)
	o.processor.log = o.log
	o.processor.inject()
	return o
}

func (o *WorldMirror) Compact(cmd string) []string {
	cmds := strings.Split(cmd, " ")
	compactCmds := make([]string, 0)
	for _, cmd := range cmds {
		frag := strings.TrimSpace(cmd)
		if len(frag) > 0 {
			compactCmds = append(compactCmds, frag)
		}
	}
	return compactCmds
}

func (o *WorldMirror) onNewText(fromPlugin string, prefix string, data string) (bool, string) {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, prefix) {
		//catch
		cmds := o.Compact(data)
		if cmds[0] == prefix {
			// cdump sx:sz ex:ez savedir
			o.processor.process(cmds)
			return true, ""
		}
	}
	// fall through
	return false, data
}

func (o *WorldMirror) Routine() {
	//o.taskIO.WaitInit()
	//i := int32(0)
	//f := int32(5)
	//for {
	//	time.Sleep(time.Second * 3)
	//	fmt.Println("Send ", i, f)
	//	o.taskIO.ShieldIO.SendPacket(&packet.PlayerAction{
	//		EntityRuntimeID: o.taskIO.ShieldIO.GameData().EntityRuntimeID,
	//		ActionType:      i,
	//		BlockPosition:   protocol.BlockPos{-5, 1, 7},
	//		BlockFace:       f,
	//	})
	//	i += 1
	//	if i > 30 {
	//		i = 0
	//		f -= 1
	//		if f == 0 {
	//			f = 5
	//		}
	//	}
	//}

}

func (o *WorldMirror) Close() {
	o.processor.close()
}
