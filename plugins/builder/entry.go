package builder

import (
	"gopkg.in/yaml.v3"
	"main.go/plugins/define"
	"main.go/task"
	"strings"
)

type CmdSource struct {
	RegName string `yaml:"reg_name"`
	Plugin  string `yaml:"plugin"`
	Prefix  string `yaml:"prefix"`
}

type Builder struct {
	Sources   []CmdSource `yaml:"sources"`
	LogName   string      `yaml:"log_name"`
	LogPlugin string      `yaml:"log_plugin"`
	Operator  string      `yaml:"operator"`
	taskIO    *task.TaskIO
	log       func(isJson bool, data string)
	processor *Processor
}

func (o *Builder) New(config []byte) define.Plugin {
	o.Sources = make([]CmdSource, 0)
	o.LogName = ""
	o.LogPlugin = "storage"
	o.Operator = "@a[tag=fb_op,c=1]"
	err := yaml.Unmarshal(config, o)
	o.processor = &Processor{}
	if err != nil {
		panic(err)
	}
	return o
}

func (o *Builder) Compact(cmd string) []string {
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

func (o *Builder) onNewText(fromPlugin string, prefix string, data string) (bool, string) {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, prefix) {
		//catch
		cmds := o.Compact(data)
		if cmds[0] == prefix {
			o.processor.process(cmds)
			return true, ""
		}
	}
	// fall through
	return false, data
}

func (o *Builder) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
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
	o.processor.log = o.log
	o.processor.Operator = o.Operator
	o.processor.expectSpeed = 600
	o.processor.speedFactor = 1
	return o
}

func (o *Builder) Routine() {

}

func (o *Builder) Close() {
	o.processor.close()
}
