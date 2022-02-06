package session

import (
	"image/color"
	"time"

	"main.go/plugins/world_mirror/server/entity/effect"
	"main.go/plugins/world_mirror/server/internal/nbtconv"
	"main.go/plugins/world_mirror/server/item/potion"
	"main.go/plugins/world_mirror/server/world"
)

// entityMetadata represents a map that holds metadata associated with an entity. The data held in the map
// depends on the entity and varies on a per-entity basis.
type entityMetadata map[uint32]interface{}

// parseEntityMetadata returns an entity metadata object with default values. It is equivalent to setting
// all properties to their default values and disabling all flags.
func parseEntityMetadata(e world.Entity) entityMetadata {
	bb := e.AABB()
	m := entityMetadata{
		dataKeyBoundingBoxWidth:  float32(bb.Width()),
		dataKeyBoundingBoxHeight: float32(bb.Height()),
		dataKeyPotionColour:      int32(0),
		dataKeyPotionAmbient:     byte(0),
		dataKeyColour:            byte(0),
	}

	m.setFlag(dataKeyFlags, dataFlagAffectedByGravity)
	m.setFlag(dataKeyFlags, dataFlagCanClimb)
	if s, ok := e.(sneaker); ok && s.Sneaking() {
		m.setFlag(dataKeyFlags, dataFlagSneaking)
	}
	if s, ok := e.(sprinter); ok && s.Sprinting() {
		m.setFlag(dataKeyFlags, dataFlagSprinting)
	}
	if s, ok := e.(swimmer); ok && s.Swimming() {
		m.setFlag(dataKeyFlags, dataFlagSwimming)
	}
	if s, ok := e.(breather); ok && s.Breathing() {
		m.setFlag(dataKeyFlags, dataFlagBreathing)
	}
	if i, ok := e.(invisible); ok && i.Invisible() {
		m.setFlag(dataKeyFlags, dataFlagInvisible)
	}
	if i, ok := e.(immobile); ok && i.Immobile() {
		m.setFlag(dataKeyFlags, dataFlagNoAI)
	}
	if o, ok := e.(onFire); ok && o.OnFireDuration() > 0 {
		m.setFlag(dataKeyFlags, dataFlagOnFire)
	}
	if u, ok := e.(using); ok && u.UsingItem() {
		m.setFlag(dataKeyFlags, dataFlagUsingItem)
	}
	if c, ok := e.(arrow); ok && c.Critical() {
		m.setFlag(dataKeyFlags, dataFlagCritical)
	}
	if s, ok := e.(scaled); ok {
		m[dataKeyScale] = float32(s.Scale())
	}
	if n, ok := e.(named); ok {
		m[dataKeyNameTag] = n.NameTag()
		m[dataKeyAlwaysShowNameTag] = uint8(1)
		m.setFlag(dataKeyFlags, dataFlagAlwaysShowNameTag)
		m.setFlag(dataKeyFlags, dataFlagCanShowNameTag)
	}
	if s, ok := e.(scoreTag); ok {
		m[dataKeyScoreTag] = s.ScoreTag()
	}
	if s, ok := e.(splash); ok {
		pot := s.Type()
		m[dataKeyPotionAuxValue] = int16(pot.Uint8())
		if len(pot.Effects()) > 0 {
			m.setFlag(dataKeyFlags, dataFlagEnchanted)
		}
	}
	if t, ok := e.(tipped); ok {
		if tip := t.Tip().Uint8(); tip > 4 {
			m[dataKeyCustomDisplay] = tip + 1
		}
	}
	if eff, ok := e.(effectBearer); ok && len(eff.Effects()) > 0 {
		colour, am := effect.ResultingColour(eff.Effects())
		if (colour != color.RGBA{}) {
			m[dataKeyPotionColour] = nbtconv.Int32FromRGBA(colour)
			if am {
				m[dataKeyPotionAmbient] = byte(1)
			} else {
				m[dataKeyPotionAmbient] = byte(0)
			}
		}
	}
	return m
}

// setFlag sets a flag with a specific index in the int64 stored in the entity metadata map to the value
// passed. It is typically used for entity metadata flags.
func (m entityMetadata) setFlag(key uint32, index uint8) {
	if v, ok := m[key]; !ok {
		m[key] = int64(0) ^ (1 << uint64(index))
	} else {
		m[key] = v.(int64) ^ (1 << uint64(index))
	}
}

//noinspection GoUnusedConst
const (
	dataKeyFlags = iota
	dataKeyHealth
	dataKeyVariant
	dataKeyColour
	dataKeyNameTag
	dataKeyOwnerRuntimeID
	dataKeyTargetRuntimeID
	dataKeyAir
	dataKeyPotionColour
	dataKeyPotionAmbient
	dataKeyCustomDisplay     = 18
	dataKeyPotionAuxValue    = 36
	dataKeyScale             = 38
	dataKeyBoundingBoxWidth  = 53
	dataKeyBoundingBoxHeight = 54
	dataKeyAlwaysShowNameTag = 81
	dataKeyScoreTag          = 84
)

//noinspection GoUnusedConst
const (
	dataFlagOnFire = iota
	dataFlagSneaking
	dataFlagRiding
	dataFlagSprinting
	dataFlagUsingItem
	dataFlagInvisible
	dataFlagCritical          = 13
	dataFlagCanShowNameTag    = 14
	dataFlagAlwaysShowNameTag = 15
	dataFlagNoAI              = 16
	dataFlagCanClimb          = 19
	dataFlagBreathing         = 35
	dataFlagAffectedByGravity = 48
	dataFlagEnchanted         = 51
	dataFlagSwimming          = 56
)

type sneaker interface {
	Sneaking() bool
}

type sprinter interface {
	Sprinting() bool
}

type swimmer interface {
	Swimming() bool
}

type breather interface {
	Breathing() bool
}

type immobile interface {
	Immobile() bool
}

type invisible interface {
	Invisible() bool
}

type scaled interface {
	Scale() float64
}

type named interface {
	NameTag() string
}

type scoreTag interface {
	ScoreTag() string
}

type splash interface {
	Type() potion.Potion
}

type onFire interface {
	OnFireDuration() time.Duration
}

type effectBearer interface {
	Effects() []effect.Effect
}

type tipped interface {
	Tip() potion.Potion
}

type using interface {
	UsingItem() bool
}

type arrow interface {
	Critical() bool
}
