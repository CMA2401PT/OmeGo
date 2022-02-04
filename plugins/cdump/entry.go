package cdump

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"main.go/plugins/chunk_mirror"
	"main.go/plugins/define"
	"main.go/task"
	"os"
	"strings"
)

type CmdSource struct {
	RegName string `yaml:"reg_name"`
	Plugin  string `yaml:"plugin"`
	Prefix  string `yaml:"prefix"`
}

type CDump struct {
	Sources   []CmdSource `yaml:"sources"`
	LogName   string      `yaml:"log_name"`
	LogPlugin string      `yaml:"log_plugin"`
	taskIO    *task.TaskIO
	log       func(isJson bool, data string)
	processor *Processor
}

func (o *CDump) New(config []byte) define.Plugin {
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

func (o *CDump) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
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
	return o
}

func (o *CDump) Compact(cmd string) []string {
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

func (o *CDump) onNewText(fromPlugin string, prefix string, data string) (bool, string) {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, prefix) {
		//catch
		cmds := o.Compact(data)
		if cmds[0] == prefix {
			// cdump sx:sz ex:ez savedir
			o.processor.process(cmds)
			return true, ""
		} else if cmds[0] == prefix+"s" {
			if len(cmds) == 1 {
				fmt.Println("no file specific!")
			} else {
				fp, err := os.Open(cmds[1])
				if err != nil {
					return false, ""
				}
				tasks := make([]string, 0)
				json.NewDecoder(fp).Decode(&tasks)
				for _, t := range tasks {
					o.onNewText(fromPlugin, prefix, t)
				}
			}

		}
	}
	// fall through
	return false, data
}

func (o *CDump) Routine() {

}

func (o *CDump) Close() {

}
