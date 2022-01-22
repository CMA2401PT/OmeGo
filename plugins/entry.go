package plugins

import (
	"main.go/task"
)

type Plugin interface {
	New(config []byte) Plugin
	Inject(taskIO *task.TaskIO, collaborationContext map[string]Plugin) Plugin
	Routine()
	Close()
}

var pool map[string]func() Plugin
var isInit bool

func Pool() map[string]func() Plugin {
	if !isInit {
		pool = make(map[string]func() Plugin)

		// Registry
		pool["storage"] = func() Plugin { return &Storage{} }
		pool["cli_interface"] = func() Plugin { return &CliInterface{} }
		pool["show_game_chat"] = func() Plugin { return &ShowChat{} }
		pool["game_chat"] = func() Plugin { return &SendChat{} }

		isInit = true
	}
	return pool
}
