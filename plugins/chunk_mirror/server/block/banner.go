package block

import (
	"main.go/plugins/chunk_mirror/server/world"
)

type WallBanner struct {
	solid
	facing_direction int16
	Patterns         map[string]interface{}
	nbt              map[string]interface{}
}

func (c WallBanner) Hash() uint64 {
	h := uint64(0)
	h = uint64(hashWallBanner) | uint64(c.facing_direction)<<8
	return h
}

func (c WallBanner) EncodeItem() (name string, meta int16) {
	return "minecraft:wall_banner", c.facing_direction
}

// EncodeBlock ...
func (c WallBanner) EncodeBlock() (name string, properties map[string]interface{}) {
	properties = make(map[string]interface{})
	properties["facing_direction"] = int32(c.facing_direction)
	return "minecraft:wall_banner", properties
}

//// DecodeNBT ...
func (c WallBanner) DecodeNBT(data map[string]interface{}) interface{} {
	c.nbt = data
	//c.Patterns = data["Patterns"].(map[string]interface{})
	return c
}

func (c WallBanner) EncodeNBT() map[string]interface{} {
	if c.nbt == nil {
		c.nbt = make(map[string]interface{})
	}
	return c.nbt
}

func allWallBanner() []world.Block {
	b := make([]world.Block, 0)
	for f := int16(0); f < int16(6); f++ {
		b = append(b, WallBanner{facing_direction: f})
	}
	return b
}

type StandingBanner struct {
	solid
	ground_sign_direction int16
	Patterns              map[string]interface{}
	nbt                   map[string]interface{}
}

func (c StandingBanner) Hash() uint64 {
	h := uint64(0)
	h = uint64(hashStandingBanner) | uint64(c.ground_sign_direction)<<8
	return h
}

func (c StandingBanner) EncodeItem() (name string, meta int16) {
	return "minecraft:standing_banner", c.ground_sign_direction
}

// EncodeBlock ...
func (c StandingBanner) EncodeBlock() (name string, properties map[string]interface{}) {
	properties = make(map[string]interface{})
	properties["ground_sign_direction"] = int32(c.ground_sign_direction)
	return "minecraft:standing_banner", properties
}

//// DecodeNBT ...
func (c StandingBanner) DecodeNBT(data map[string]interface{}) interface{} {
	c.nbt = data
	//c.Patterns = data["Patterns"].(map[string]interface{})
	return c
}

func (c StandingBanner) EncodeNBT() map[string]interface{} {
	if c.nbt == nil {
		c.nbt = make(map[string]interface{})
	}
	return c.nbt
}

func allStandingBanner() []world.Block {
	b := make([]world.Block, 0)
	for f := int16(0); f < int16(16); f++ {
		b = append(b, StandingBanner{ground_sign_direction: f})
	}
	return b
}

//// EncodeNBT ...
//func (c WallBanner) EncodeNBT() map[string]interface{} {
//	if c.nbt == nil {
//		c.nbt = make(map[string]interface{})
//	}
//	c.nbt["LPCommandMode"] = c.LPCommandMode
//	c.nbt["LPCondionalMode"] = c.LPCondionalMode
//	return c.nbt
//}
