package plugins

import (
	"main.go/define"
	cqchat "main.go/plugins/gocq"
)

var pool map[string]func() define.Plugin
var isInit bool

func Pool() map[string]func() define.Plugin {
	if !isInit {
		pool = make(map[string]func() define.Plugin)

		// Registry
		pool["storage"] = func() define.Plugin { return &Storage{} }
		pool["cli_interface"] = func() define.Plugin { return &CliInterface{} }
		pool["show_game_chat"] = func() define.Plugin { return &ShowChat{} }
		pool["send_cmd_line"] = func() define.Plugin { return &SendCmdLine{} }
		pool["cq_interface"] = func() define.Plugin { return &cqchat.GoCQ{} }

		isInit = true
	}
	return pool
}
