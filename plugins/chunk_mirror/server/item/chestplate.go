package item

import (
	"main.go/plugins/chunk_mirror/server/world"
)

// Chestplate is a defensive item that may be equipped in the chestplate slot. Generally, chestplates provide
// the most defence of all armour items.
type Chestplate struct {
	// Tier is the tier of the chestplate.
	Tier ArmourTier
}

// Use handles the using of a chestplate to auto-equip it in the designated armour slot.
func (c Chestplate) Use(_ *world.World, _ User, ctx *UseContext) bool {
	ctx.SwapHeldWithArmour(1)
	return false
}

// MaxCount always returns 1.
func (c Chestplate) MaxCount() int {
	return 1
}

// DefencePoints ...
func (c Chestplate) DefencePoints() float64 {
	switch c.Tier {
	case ArmourTierLeather:
		return 3
	case ArmourTierGold, ArmourTierChain:
		return 5
	case ArmourTierIron:
		return 6
	case ArmourTierDiamond, ArmourTierNetherite:
		return 8
	}
	panic("invalid chestplate tier")
}

// KnockBackResistance ...
func (c Chestplate) KnockBackResistance() float64 {
	return c.Tier.KnockBackResistance
}

// DurabilityInfo ...
func (c Chestplate) DurabilityInfo() DurabilityInfo {
	return DurabilityInfo{
		MaxDurability: int(c.Tier.BaseDurability + c.Tier.BaseDurability/2.2),
		BrokenItem:    simpleItem(Stack{}),
	}
}

// Chestplate ...
func (c Chestplate) Chestplate() bool {
	return true
}

// EncodeItem ...
func (c Chestplate) EncodeItem() (name string, meta int16) {
	return "minecraft:" + c.Tier.Name + "_chestplate", 0
}
