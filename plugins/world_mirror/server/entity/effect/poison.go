package effect

import (
	"image/color"
	"time"

	"main.go/plugins/world_mirror/server/entity/damage"
	"main.go/plugins/world_mirror/server/world"
)

// Poison is a lasting effect that causes the affected entity to lose health gradually. Poison cannot kill,
// unlike FatalPoison.
type Poison struct {
	nopLasting
}

// Apply ...
func (Poison) Apply(e world.Entity, lvl int, d time.Duration) {
	interval := 50 >> lvl
	if tickDuration(d)%interval == 0 {
		if l, ok := e.(living); ok {
			l.Hurt(1, damage.SourcePoisonEffect{})
		}
	}
}

// RGBA ...
func (Poison) RGBA() color.RGBA {
	return color.RGBA{R: 0x4e, G: 0x93, B: 0x31, A: 0xff}
}
