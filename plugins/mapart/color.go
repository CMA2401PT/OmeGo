package mapart

import (
	"github.com/lucasb-eyer/go-colorful"
)

type ConstBlock struct {
	Name string
	Data int
}

type ColorBlock struct {
	Block *ConstBlock
	Color colorful.Color
}

var ColorTable = []ColorBlock{
	{Block: &ConstBlock{Name: "stone", Data: 0}, Color: colorful.Color{89, 89, 89}},
	{Block: &ConstBlock{Name: "stone", Data: 1}, Color: colorful.Color{135, 102, 76}},
	{Block: &ConstBlock{Name: "stone", Data: 3}, Color: colorful.Color{237, 235, 229}},
	{Block: &ConstBlock{Name: "stone", Data: 5}, Color: colorful.Color{104, 104, 104}},
	{Block: &ConstBlock{Name: "grass", Data: 0}, Color: colorful.Color{144, 174, 94}},
	{Block: &ConstBlock{Name: "planks", Data: 0}, Color: colorful.Color{129, 112, 73}},
	{Block: &ConstBlock{Name: "planks", Data: 1}, Color: colorful.Color{114, 81, 51}},
	{Block: &ConstBlock{Name: "planks", Data: 2}, Color: colorful.Color{228, 217, 159}},
	{Block: &ConstBlock{Name: "planks", Data: 4}, Color: colorful.Color{71, 71, 71}},
	{Block: &ConstBlock{Name: "planks", Data: 5}, Color: colorful.Color{91, 72, 50}},
	{Block: &ConstBlock{Name: "leaves", Data: 0}, Color: colorful.Color{64, 85, 32}},
	{Block: &ConstBlock{Name: "leaves", Data: 1}, Color: colorful.Color{54, 75, 50}},
	{Block: &ConstBlock{Name: "leaves", Data: 2}, Color: colorful.Color{68, 83, 47}},
	{Block: &ConstBlock{Name: "leaves", Data: 14}, Color: colorful.Color{58, 71, 40}},
	{Block: &ConstBlock{Name: "leaves", Data: 15}, Color: colorful.Color{55, 73, 28}},
	{Block: &ConstBlock{Name: "sponge", Data: 0}, Color: colorful.Color{183, 183, 70}},
	{Block: &ConstBlock{Name: "lapis_block", Data: 0}, Color: colorful.Color{69, 101, 198}},
	{Block: &ConstBlock{Name: "noteblock", Data: 0}, Color: colorful.Color{111, 95, 63}},
	{Block: &ConstBlock{Name: "web", Data: 0}, Color: colorful.Color{159, 159, 159}},
	{Block: &ConstBlock{Name: "wool", Data: 0}, Color: colorful.Color{205, 205, 205}},
	{Block: &ConstBlock{Name: "wool", Data: 1}, Color: colorful.Color{163, 104, 54}},
	{Block: &ConstBlock{Name: "wool", Data: 2}, Color: colorful.Color{132, 65, 167}},
	{Block: &ConstBlock{Name: "wool", Data: 3}, Color: colorful.Color{91, 122, 169}},
	{Block: &ConstBlock{Name: "wool", Data: 5}, Color: colorful.Color{115, 162, 53}},
	{Block: &ConstBlock{Name: "wool", Data: 6}, Color: colorful.Color{182, 106, 131}},
	{Block: &ConstBlock{Name: "wool", Data: 7}, Color: colorful.Color{60, 60, 60}},
	{Block: &ConstBlock{Name: "wool", Data: 8}, Color: colorful.Color{123, 123, 123}},
	{Block: &ConstBlock{Name: "wool", Data: 9}, Color: colorful.Color{69, 100, 121}},
	{Block: &ConstBlock{Name: "wool", Data: 10}, Color: colorful.Color{94, 52, 137}},
	{Block: &ConstBlock{Name: "wool", Data: 11}, Color: colorful.Color{45, 59, 137}},
	{Block: &ConstBlock{Name: "wool", Data: 12}, Color: colorful.Color{78, 61, 43}},
	{Block: &ConstBlock{Name: "wool", Data: 13}, Color: colorful.Color{85, 100, 49}},
	{Block: &ConstBlock{Name: "wool", Data: 14}, Color: colorful.Color{113, 46, 44}},
	{Block: &ConstBlock{Name: "wool", Data: 15}, Color: colorful.Color{20, 20, 20}},
	{Block: &ConstBlock{Name: "gold_block", Data: 0}, Color: colorful.Color{198, 191, 84}},
	{Block: &ConstBlock{Name: "iron_block", Data: 0}, Color: colorful.Color{134, 134, 134}},
	{Block: &ConstBlock{Name: "double_stone_slab", Data: 1}, Color: colorful.Color{196, 187, 136}},
	{Block: &ConstBlock{Name: "double_stone_slab", Data: 6}, Color: colorful.Color{204, 202, 196}},
	{Block: &ConstBlock{Name: "double_stone_slab", Data: 7}, Color: colorful.Color{81, 11, 5}},
	{Block: &ConstBlock{Name: "redstone_block", Data: 0}, Color: colorful.Color{188, 39, 26}},
	{Block: &ConstBlock{Name: "mossy_cobblestone", Data: 0}, Color: colorful.Color{131, 134, 146}},
	{Block: &ConstBlock{Name: "diamond_block", Data: 0}, Color: colorful.Color{102, 173, 169}},
	{Block: &ConstBlock{Name: "farmland", Data: 0}, Color: colorful.Color{116, 88, 65}},
	{Block: &ConstBlock{Name: "ice", Data: 0}, Color: colorful.Color{149, 149, 231}},
	{Block: &ConstBlock{Name: "pumpkin", Data: 1}, Color: colorful.Color{189, 122, 62}},
	{Block: &ConstBlock{Name: "monster_egg", Data: 1}, Color: colorful.Color{153, 156, 169}},
	{Block: &ConstBlock{Name: "red_mushroom_block", Data: 0}, Color: colorful.Color{131, 53, 50}},
	{Block: &ConstBlock{Name: "vine", Data: 1}, Color: colorful.Color{68, 89, 34}},
	{Block: &ConstBlock{Name: "brewing_stand", Data: 6}, Color: colorful.Color{155, 155, 155}},
	{Block: &ConstBlock{Name: "double_wooden_slab", Data: 1}, Color: colorful.Color{98, 70, 44}},
	{Block: &ConstBlock{Name: "emerald_block", Data: 0}, Color: colorful.Color{77, 171, 67}},
	{Block: &ConstBlock{Name: "light_weighted_pressure_plate", Data: 7}, Color: colorful.Color{231, 221, 99}},
	{Block: &ConstBlock{Name: "stained_hardened_clay", Data: 0}, Color: colorful.Color{237, 237, 237}},
	{Block: &ConstBlock{Name: "stained_hardened_clay", Data: 2}, Color: colorful.Color{154, 76, 194}},
	{Block: &ConstBlock{Name: "stained_hardened_clay", Data: 4}, Color: colorful.Color{213, 213, 82}},
	{Block: &ConstBlock{Name: "stained_hardened_clay", Data: 6}, Color: colorful.Color{211, 123, 153}},
	{Block: &ConstBlock{Name: "stained_hardened_clay", Data: 8}, Color: colorful.Color{142, 142, 142}},
	{Block: &ConstBlock{Name: "stained_hardened_clay", Data: 10}, Color: colorful.Color{110, 62, 160}},
	{Block: &ConstBlock{Name: "slime", Data: 0}, Color: colorful.Color{109, 141, 60}},
	{Block: &ConstBlock{Name: "packed_ice", Data: 0}, Color: colorful.Color{128, 128, 199}},
	{Block: &ConstBlock{Name: "repeating_command_block", Data: 1}, Color: colorful.Color{77, 43, 112}},
	{Block: &ConstBlock{Name: "chain_command_block", Data: 1}, Color: colorful.Color{70, 82, 40}},
	{Block: &ConstBlock{Name: "nether_wart_block", Data: 0}, Color: colorful.Color{93, 38, 36}},
	{Block: &ConstBlock{Name: "bone_block", Data: 0}, Color: colorful.Color{160, 153, 112}},
}

