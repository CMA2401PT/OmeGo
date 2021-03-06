package plugins

import (
	"main.go/plugins/builder"
	"main.go/plugins/cdump"
	"main.go/plugins/chunk_mirror"
	"main.go/plugins/define"
	"main.go/plugins/fastbuilder"
	cqchat "main.go/plugins/gocq"
	"main.go/plugins/mapart"
	"main.go/plugins/world_mirror"
)

var pool map[string]func() define.Plugin
var isInit bool

func Pool() map[string]func() define.Plugin {
	if !isInit {
		pool = make(map[string]func() define.Plugin)

		// Registry
		pool["storage"] = func() define.Plugin { return &Storage{} }
		pool["cli_interface"] = func() define.Plugin { return &CliInterface{} }
		pool["read_chat"] = func() define.Plugin { return &ReadChat{} }
		pool["ask_for_op"] = func() define.Plugin { return &AskForOP{} }
		pool["show_game_chat"] = func() define.Plugin { return &ShowChat{} }
		pool["send_cmd_line"] = func() define.Plugin { return &SendCmdLine{} }
		pool["cq_interface"] = func() define.Plugin { return &cqchat.GoCQ{} }
		pool["chunk_mirror"] = func() define.Plugin { return &chunk_mirror.ChunkMirror{} }
		pool["fast_builder"] = func() define.Plugin { return &fastbuilder.FastBuilder{} }
		pool["cdump"] = func() define.Plugin { return &cdump.CDump{} }
		pool["world_mirror"] = func() define.Plugin { return &world_mirror.WorldMirror{} }
		pool["builder"] = func() define.Plugin { return &builder.Builder{} }
		pool["map_art"] = func() define.Plugin { return &mapart.MapArt{} }

		isInit = true
	}
	return pool
}
