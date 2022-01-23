package plugins

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"main.go/define"
	"main.go/minecraft/protocol/packet"
	"main.go/task"
	"strings"
)

type CmdSource struct {
	RegName string `yaml:"reg_name"`
	Plugin  string `yaml:"plugin"`
	Prefix  string `yaml:"prefix"`
}

type SendCmdLine struct {
	Sources   []CmdSource `yaml:"sources"`
	LogName   string      `yaml:"log_name"`
	LogPlugin string      `yaml:"log_plugin"`
	taskIO    *task.TaskIO
	log       func(isJson bool, data string)
}

func (o *SendCmdLine) New(config []byte) define.Plugin {
	o.Sources = make([]CmdSource, 0)
	o.LogName = ""
	o.LogPlugin = "storage"
	err := yaml.Unmarshal(config, o)
	if err != nil {
		panic(err)
	}
	return o
}

func (o *SendCmdLine) onNewText(fromPlugin string, prefix string, data string) (bool, string) {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, prefix) {
		//catch
		fmt.Println("cmd [" + fromPlugin + "]: " + data)
		o.taskIO.LockCMDAndFBOn().
			SendCmdWithFeedBack(strings.TrimLeft(data, prefix), func(respPk *packet.CommandOutput) {
				if o.log != nil {
					o.log(false, fromPlugin+": "+data+" -> "+fmt.Sprintf("%v", respPk.OutputMessages))
				}
			}).UnlockAndRestore()
		return true, ""
	} else {
		// fall through
		return false, data
	}

}

func (o *SendCmdLine) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
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
	return o
}

func (o *SendCmdLine) Routine() {

}

func (o *SendCmdLine) Close() {

}
