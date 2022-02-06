package effect

import (
	"image/color"
	"time"

	"main.go/plugins/world_mirror/server/world"
)

// Saturation is a lasting effect that causes the affected player to regain food and saturation.
type Saturation struct {
	nopLasting
}

// Apply ...
func (Saturation) Apply(e world.Entity, lvl int, _ time.Duration) {
	if i, ok := e.(interface {
		Saturate(food int, saturation float64)
	}); ok {
		i.Saturate(lvl, 2*float64(lvl))
	}
}

// RGBA ...
func (Saturation) RGBA() color.RGBA {
	return color.RGBA{R: 0xf8, G: 0x24, B: 0x23, A: 0xff}
}
