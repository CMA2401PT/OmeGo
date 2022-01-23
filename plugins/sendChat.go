package plugins

import (
	"gopkg.in/yaml.v3"
	"main.go/define"
	"main.go/task"
	"strings"
)

// this plugin is just for test
//    {
//      "name": "game_chat",
//      "as": "",
//      "file": "internal",
//      "require": ["interface"],
//      "configs": {
//        "log_name": "chat-sent",
//        "sources": [
//          {
//            "perfix": "[backend] ",
//            "plugin": "interface",
//            "reg_name": "game-chat"
//          }
//        ]
//      }
//    }

type Source struct {
	Perfix  string `yaml:"perfix"`
	RegName string `yaml:"reg_name"`
	Plugin  string `yaml:"plugin"`
}

type SendChat struct {
	Sources   []Source `yaml:"sources"`
	LogName   string   `yaml:"log_name"`
	LogPlugin string   `yaml:"log_plugin"`
	taskIO    *task.TaskIO
}

func (o *SendChat) New(config []byte) define.Plugin {
	o.Sources = make([]Source, 0)
	o.LogName = ""
	o.LogPlugin = "storage"
	err := yaml.Unmarshal(config, o)
	if err != nil {
		panic(err)
	}
	return o
}

func (o *SendChat) onNewText(isJson bool, perfix string, data string) (bool, string) {
	data = strings.TrimSpace(data)
	if isJson {
		o.taskIO.Say(true, data)
	} else {
		o.taskIO.Say(false, perfix+data)
	}
	// fall through
	return true, data
}

func (o *SendChat) Inject(taskIO *task.TaskIO, collaborationContext map[string]define.Plugin) define.Plugin {
	o.taskIO = taskIO
	var log func(isJson bool, data string)
	if o.LogName != "" {
		log = collaborationContext[o.LogPlugin].(StringWriteInterface).RegStringSender(o.LogName)
	}
	for _, s := range o.Sources {
		src := collaborationContext[s.Plugin].(StringReadInterface)
		src.RegStringInterceptor(s.RegName, func(isJson bool, data string) (bool, string) {
			if log != nil {
				log(false, s.Perfix+data)
			}
			return o.onNewText(isJson, s.Perfix, data)
		})
	}
	return o
}

func (o *SendChat) Routine() {

}

func (o *SendChat) Close() {

}
