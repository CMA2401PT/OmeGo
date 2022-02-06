package item

import (
	"main.go/plugins/world_mirror/server/world"
)

// Leggings are a defensive item that may be equipped in the leggings armour slot. They come in several tiers,
// like leather, gold, chain, iron and diamond.
type Leggings struct {
	// Tier is the tier of the leggings.
	Tier ArmourTier
}

// Use handles the auto-equipping of leggings in an armour slot by using the item.
func (l Leggings) Use(_ *world.World, _ User, ctx *UseContext) bool {
	ctx.SwapHeldWithArmour(2)
	return false
}

// MaxCount always returns 1.
func (l Leggings) MaxCount() int {
	return 1
}

// DefencePoints ...
func (l Leggings) DefencePoints() float64 {
	switch l.Tier {
	case ArmourTierLeather:
		return 2
	case ArmourTierGold:
		return 3
	case ArmourTierChain:
		return 4
	case ArmourTierIron:
		return 5
	case ArmourTierDiamond, ArmourTierNetherite:
		return 6
	}
	panic("invalid leggings tier")
}

// KnockBackResistance ...
func (l Leggings) KnockBackResistance() float64 {
	return l.Tier.KnockBackResistance
}

// Leggings ...
func (l Leggings) Leggings() bool {
	return true
}

// DurabilityInfo ...
func (l Leggings) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability: int(l.Tier.BaseDurability + l.Tier.BaseDurability/2.5),
		BrokenItem:    simpleItem(Stack{}),
	}
}

// EncodeItem ...
func (l Leggings) EncodeItem() (name string, meta int16) {
	return "minecraft:" + l.Tier.Name + "_leggings", 0
}
