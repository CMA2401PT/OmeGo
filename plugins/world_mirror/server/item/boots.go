package item

import (
	"main.go/plugins/world_mirror/server/world"
)

// Boots are a defensive item that may be equipped in the boots armour slot. They come in several tiers, like
// leather, gold, chain, iron and diamond.
type Boots struct {
	// Tier is the tier of the boots.
	Tier ArmourTier
}

// Use handles the auto-equipping of boots in the armour slot when using it.
func (b Boots) Use(_ *world.World, _ User, ctx *UseContext) bool {
	ctx.SwapHeldWithArmour(3)
	return false
}

// MaxCount always returns 1.
func (b Boots) MaxCount() int {
	return 1
}

// DurabilityInfo ...
func (b Boots) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability: int(b.Tier.BaseDurability + b.Tier.BaseDurability/5.5),
		BrokenItem:    simpleItem(Stack{}),
	}
}

// DefencePoints ...
func (b Boots) DefencePoints() float64 {
	switch b.Tier {
	case ArmourTierLeather, ArmourTierGold, ArmourTierChain:
		return 1
	case ArmourTierIron:
		return 2
	case ArmourTierDiamond, ArmourTierNetherite:
		return 3
	}
	panic("invalid boots tier")
}

// KnockBackResistance ...
func (b Boots) KnockBackResistance() float64 {
	return b.Tier.KnockBackResistance
}

// Boots ...
func (b Boots) Boots() bool {
	return true
}

// EncodeItem ...
func (b Boots) EncodeItem() (name string, meta int16) {
	return "minecraft:" + b.Tier.Name + "_boots", 0
}
