package enchantment

import (
	"math/rand"

	"main.go/plugins/chunk_mirror/server/item"
	"main.go/plugins/chunk_mirror/server/world"
)

// Unbreaking is an enchantment that gives a chance for an item to avoid durability reduction when it
// is used, effectively increasing the item's durability.
type Unbreaking struct{ enchantment }

// Reduce returns the amount of damage that should be reduced with unbreaking.
func (e Unbreaking) Reduce(it world.Item, level, amount int) int {
	after := amount

	_, ok := it.(item.Armour)
	for i := 0; i < amount; i++ {
		if (!ok || rand.Float64() >= 0.6) && rand.Intn(level+1) > 0 {
			after--
		}
	}

	return after
}

// Name ...
func (e Unbreaking) Name() string {
	return "Unbreaking"
}

// MaxLevel ...
func (e Unbreaking) MaxLevel() int {
	return 3
}

// WithLevel ...
func (e Unbreaking) WithLevel(level int) item.Enchantment {
	return Unbreaking{e.withLevel(level, e)}
}

// CompatibleWith ...
func (e Unbreaking) CompatibleWith(s item.Stack) bool {
	_, ok := s.Item().(item.Durable)
	return ok
}
