package builder

import (
	"errors"
	"main.go/plugins/fastbuilder/i18n"
	"main.go/plugins/fastbuilder/types"
)

var Builder = map[string]func(config *types.MainConfig, blc chan *types.Module) error{
	"round":     Round,
	"circle":    Circle,
	"sphere":    Sphere,
	"ellipse":   Ellipse,
	"ellipsoid": Ellipsoid,
	"plot":      Paint,
	"schem":     Schematic,
	"acme":      Acme,
	"bdump":     BDump,
	"mapart":    MapArt,
}

func Generate(config *types.MainConfig, blc chan *types.Module) error {
	if config.Execute == "" {
		return errors.New(I18n.T(I18n.CommandNotFound))
	} else {
		return Builder[config.Execute](config, blc)
	}
}

func PipeGenerate(configs []*types.Config) []*types.Module {
	return nil
}
