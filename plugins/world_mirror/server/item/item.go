package item

import (
	"math"
	"time"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/entity/effect"
	"main.go/plugins/world_mirror/server/world"
)

// MaxCounter represents an item that has a specific max count. By default, each item will be expected to have
// a maximum count of 64. MaxCounter may be implemented to change this behaviour.
type MaxCounter interface {
	// MaxCount returns the maximum number of items that a stack may be composed of. The number returned must
	// be positive.
	MaxCount() int
}

// UsableOnBlock represents an item that may be used on a block. If an item implements this interface, the
// UseOnBlock method is called whenever the item is used on a block.
type UsableOnBlock interface {
	// UseOnBlock is called when an item is used on a block. The world passed is the world that the item was
	// used in. The user passed is the entity that used the item. Usually this entity is a player.
	// The position of the block that was clicked, along with the clicked face and the position clicked
	// relative to the corner of the block are passed.
	// UseOnBlock returns a bool indicating if the item was used successfully.
	UseOnBlock(pos cube.Pos, face cube.Face, clickPos mgl64.Vec3, w *world.World, user User, ctx *UseContext) bool
}

// UsableOnEntity represents an item that may be used on an entity. If an item implements this interface, the
// UseOnEntity method is called whenever the item is used on an entity.
type UsableOnEntity interface {
	// UseOnEntity is called when an item is used on an entity. The world passed is the world that the item is
	// used in, and the entity clicked and the user of the item are also passed.
	// UseOnEntity returns a bool indicating if the item was used successfully.
	UseOnEntity(e world.Entity, w *world.World, user User, ctx *UseContext) bool
}

// Usable represents an item that may be used 'in the air'. If an item implements this interface, the Use
// method is called whenever the item is used while pointing at the air. (For example, when throwing an egg.)
type Usable interface {
	// Use is called when the item is used in the air. The user that used the item and the world that the item
	// was used in are passed to the method.
	// Use returns a bool indicating if the item was used successfully.
	Use(w *world.World, user User, ctx *UseContext) bool
}

// Consumable represents an item that may be consumed by a player. If an item implements this interface, a player
// may use and hold the item to consume it.
type Consumable interface {
	// AlwaysConsumable specifies if the item is always consumable. Normal food can generally only be consumed
	// when the food bar is not full or when in creative mode. Returning true here means the item can always
	// be consumed, like golden apples or potions.
	AlwaysConsumable() bool
	// ConsumeDuration is the duration consuming the item takes. If the player is using the item for at least
	// this duration, the item will be consumed and have its Consume method called.
	ConsumeDuration() time.Duration
	// Consume consumes one item of the Stack that the Consumable is in. The Stack returned is added back to
	// the inventory after consuming the item. For potions, for example, an empty bottle is returned.
	Consume(w *world.World, c Consumer) Stack
}

// Consumer represents a User that is able to consume Consumable items.
type Consumer interface {
	User
	// Saturate saturates the Consumer's food bar by the amount of food points passed and the saturation by
	// up to as many saturation points as passed. The final saturation will never exceed the final food level.
	Saturate(food int, saturation float64)
	// AddEffect adds an effect.Effect to the Consumer. If the effect is instant, it is applied to the Consumer
	// immediately. If not, the effect is applied to the consumer every time the Tick method is called.
	// AddEffect will overwrite any effects present if the level of the effect is higher than the existing one, or
	// if the effects' levels are equal and the new effect has a longer duration.
	AddEffect(e effect.Effect)
}

// DefaultConsumeDuration is the default duration that consuming an item takes. Dried kelp takes half this
// time to be consumed.
const DefaultConsumeDuration = (time.Second * 161) / 100

// Weapon is an item that may be used as a weapon. It has an attack damage which may be different to the 2
// damage that attacking with an empty hand deals.
type Weapon interface {
	// AttackDamage returns the custom attack damage to the weapon. The damage returned must not be negative.
	AttackDamage() float64
}

// Cooldown represents an item that has a cooldown.
type Cooldown interface {
	// Cooldown is the duration of the cooldown.
	Cooldown() time.Duration
}

// nameable represents a block that may be named. These are often containers such as chests, which have a
// name displayed in their interface.
type nameable interface {
	// WithName returns the block itself, except with a custom name applied to it.
	WithName(a ...interface{}) world.Item
}

// Releaser represents an entity that can release items, such as bows.
type Releaser interface {
	User
	// GameMode returns the gamemode of the releaser.
	GameMode() world.GameMode
	// PlaySound plays a world.Sound that only this Releaser can hear.
	PlaySound(sound world.Sound)
}

// Releasable represents an item that can be released.
type Releasable interface {
	// Release is called when an item is released.
	Release(releaser Releaser, duration time.Duration, ctx *UseContext)
	// Requirements returns the required items to release this item.
	Requirements() []Stack
}

// User represents an entity that is able to use an item in the world, typically entities such as players,
// which interact with the world using an item.
type User interface {
	Carrier
	// Facing returns the direction that the user is facing.
	Facing() cube.Direction
	SetHeldItems(mainHand, offHand Stack)
}

// Carrier represents an entity that is able to carry an item.
type Carrier interface {
	world.Entity
	// HeldItems returns the items currently held by the entity. Viewers of the entity will be able to see
	// these items.
	HeldItems() (mainHand, offHand Stack)
}

// owned represents an entity that is "owned" by another entity. Entities like projectiles typically are "owned".
type owned interface {
	world.Entity
	Owner() world.Entity
	Own(owner world.Entity)
}

// BeaconPayment represents an item that may be used as payment for a beacon to select effects to be broadcast
// to surrounding players.
type BeaconPayment interface {
	PayableForBeacon() bool
}

// defaultFood represents a consumable item with a default consumption duration.
type defaultFood struct{}

// AlwaysConsumable ...
func (defaultFood) AlwaysConsumable() bool {
	return false
}

// ConsumeDuration ...
func (d defaultFood) ConsumeDuration() time.Duration {
	return DefaultConsumeDuration
}

// directionVector returns a vector that describes the direction of the entity passed. The length of the Vec3
// returned is always 1.
func directionVector(e world.Entity) mgl64.Vec3 {
	yaw, pitch := e.Rotation()
	yawRad, pitchRad := mgl64.DegToRad(yaw), mgl64.DegToRad(pitch)
	m := math.Cos(pitchRad)

	return mgl64.Vec3{
		-m * math.Sin(yawRad),
		-math.Sin(pitchRad),
		m * math.Cos(yawRad),
	}.Normalize()
}

// eyePosition returns the position of the eyes of the entity if the entity implements entity.Eyed, or the
// actual position if it doesn't.
func eyePosition(e world.Entity) mgl64.Vec3 {
	pos := e.Position()
	if eyed, ok := e.(interface{ EyeHeight() float64 }); ok {
		pos = pos.Add(mgl64.Vec3{0, eyed.EyeHeight()})
	}
	return pos
}
