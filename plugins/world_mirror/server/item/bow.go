package item

import (
	"math"
	"time"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/world_mirror/server/item/potion"
	"main.go/plugins/world_mirror/server/world"
	"main.go/plugins/world_mirror/server/world/sound"
)

// Bow is a ranged weapon that fires arrows.
type Bow struct{}

// MaxCount always returns 1.
func (Bow) MaxCount() int {
	return 1
}

// DurabilityInfo ...
func (Bow) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability: 385,
		BrokenItem:    simpleItem(Stack{}),
	}
}

// Release ...
func (Bow) Release(releaser Releaser, duration time.Duration, ctx *UseContext) {
	ticks := duration.Milliseconds() / 50
	if ticks < 3 {
		return
	}

	t := float64(ticks) / 20
	force := math.Min((t*t+t*2)/3, 1)
	if force < 0.1 {
		return
	}

	var tip potion.Potion
	creative := releaser.GameMode().CreativeInventory()
	if arrow, ok := ctx.FirstFunc(func(stack Stack) bool {
		_, ok := stack.Item().(Arrow)
		return ok
	}); ok {
		arr := arrow.Item().(Arrow)
		tip = arr.Tip

		if !creative {
			ctx.DamageItem(1)
			ctx.Consume(arrow.Grow(-arrow.Count() + 1))
		}
	} else {
		if !creative {
			return
		}
	}

	rYaw, rPitch := releaser.Rotation()
	yaw, pitch := -rYaw, -rPitch
	if rYaw > 180 {
		yaw = 360 - rYaw
	}

	proj, ok := world.EntityByName("minecraft:arrow")
	if !ok {
		return
	}

	if p, ok := proj.(interface {
		New(pos, vel mgl64.Vec3, yaw, pitch float64, owner world.Entity, critical, disallowPickup, obtainArrowOnPickup bool, tip potion.Potion) world.Entity
	}); ok {
		releaser.PlaySound(sound.BowShoot{})
		player := releaser.EncodeEntity() == "minecraft:player"
		a := p.New(eyePosition(releaser), directionVector(releaser).Mul(force*3), yaw, pitch, releaser, force >= 1, !player, !creative, tip)
		releaser.World().AddEntity(a)
	}
}

// Requirements returns the required items to release this item.
func (Bow) Requirements() []Stack {
	return []Stack{NewStack(Arrow{}, 1)}
}

// EncodeItem ...
func (Bow) EncodeItem() (name string, meta int16) {
	return "minecraft:bow", 0
}
