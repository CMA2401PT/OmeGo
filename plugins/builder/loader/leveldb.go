package loader

import (
	"main.go/plugins/builder/ir"
	"main.go/plugins/chunk_mirror/server/block/cube"
	reflect_world "main.go/plugins/chunk_mirror/server/world"
	reflect_provider "main.go/plugins/chunk_mirror/server/world/mcdb"
)

func LoadLevelDB(WorldDir string) (*reflect_provider.Provider, error) {
	return reflect_provider.New(WorldDir, reflect_world.Overworld)
}

func RangeToIR(p *reflect_provider.Provider, ir *ir.IR, start cube.Pos, end cube.Pos) {

}
