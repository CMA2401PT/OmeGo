package chunk_mirror

import (
	"bytes"
	"fmt"
	"image/color"
	"main.go/minecraft/nbt"
	"main.go/minecraft/protocol/packet"
	reflect_provider "main.go/plugins/chunk_mirror/server/world/mcdb"
)

// 以下字段全部是猜的
// oh,原来问题不是出在这里
type NbtMapDecorationsData struct {
	Rot  int32 `nbt:"rot"`  // 在那个放地图的方块上，旋转角度
	Type int32 `nbt:"type"` // 放地图的方块的类型
	X    int32 `nbt:"x"`    // 地图起点X与放地图的方块的偏移
	Y    int32 `nbt:"y"`    // 地图起点Z（Y）与放地图的方块的偏移
}
type NbtMapDecorationsKey struct {
	BlockX int32 `nbt:"blockX"` // 放地图方块的位置
	BlockY int32 `nbt:"blockY"`
	BlockZ int32 `nbt:"blockZ"`
	Type   int32 `nbt:"type"` // 类型
}
type NbtMapDecoration struct {
	Data NbtMapDecorationsData `nbt:"data"`
	Key  NbtMapDecorationsKey  `nbt:"key"`
}

type NbtMAP struct {
	MapID       int64        `nbt:"mapId"`
	ParentMapID int64        `nbt:"parentMapId"` // itself
	Colors      [65536]uint8 `nbt:"colors"`      // should be computed
	//Colors            interface{} `nbt:"colors"`

	LockedMap         byte               `nbt:"mapLocked"`
	Scale             byte               `nbt:"scale"`
	Dimension         byte               `nbt:"dimension"`
	FullyExplored     byte               `nbt:"fullyExplored"`     //1
	UnlimitedTracking byte               `nbt:"unlimitedTracking"` //0
	XCenter           int32              `nbt:"xCenter"`
	ZCenter           int32              `nbt:"zCenter"`
	Height            int16              `nbt:"height"`
	Width             int16              `nbt:"width"`
	Decorations       []NbtMapDecoration `nbt:"decorations"`
	pixels            [][]color.RGBA
	offsetX           int32
	offsetZ           int32
}

func (m *NbtMAP) Color2Pixels() {
	offset := 0
	m.pixels = make([][]color.RGBA, m.Height)
	for y := int16(0); y < m.Height; y++ {
		m.pixels[y] = make([]color.RGBA, m.Width)
		for x := int16(0); x < m.Width; x++ {
			m.pixels[y][x] = color.RGBA{
				R: m.Colors[offset*4],
				G: m.Colors[offset*4+1],
				B: m.Colors[offset*4+2],
				A: m.Colors[offset*4+3],
			}
			offset++
		}
	}
}
func (m *NbtMAP) Pixels2Color() {
	offset := 0
	var p color.RGBA
	for y := int16(0); y < m.Height; y++ {
		for x := int16(0); x < m.Width; x++ {
			p = m.pixels[y][x]
			m.Colors[offset*4] = p.R
			m.Colors[offset*4+1] = p.G
			m.Colors[offset*4+2] = p.B
			m.Colors[offset*4+3] = p.A
			offset++
		}
	}
}

func (m *NbtMAP) FromPacket(pk *packet.ClientBoundMapItemData) error {
	if pk.Height != 128 || pk.Width != 128 {
		return fmt.Errorf("unacceptable Size %v %v", m.Height, m.Width)
	}
	m.MapID = pk.MapID
	m.ParentMapID = pk.MapID + 1
	m.pixels = pk.Pixels
	m.Dimension = pk.Dimension
	if pk.LockedMap {
		m.LockedMap = 1
	} else {
		m.LockedMap = 0
	}
	m.Scale = pk.Scale
	//m.Decorations = make([]interface{}, 0)
	////not work, don't know why
	//for _, dec := range pk.Decorations {
	//	m.Decorations = append(m.Decorations, dec)
	//}
	m.Height = int16(pk.Height)
	m.Width = int16(pk.Width)

	m.FullyExplored = 1
	m.UnlimitedTracking = 0
	//m.XCenter = -64
	//m.ZCenter = -64
	m.XCenter = m.offsetX - 64 + (128 << m.Scale)
	m.ZCenter = m.offsetZ - 64 + (128 << m.Scale)
	m.Pixels2Color()
	return nil
}
func (m *NbtMAP) CreateParentMap() []*NbtMAP {
	maps := make([]*NbtMAP, 0)
	mc := m
	for mc.Scale < 4 {
		nm := &NbtMAP{
			MapID:             mc.ParentMapID,
			ParentMapID:       mc.ParentMapID + 1,
			Colors:            [65536]uint8{},
			LockedMap:         0,
			Scale:             mc.Scale + 1,
			Dimension:         0,
			FullyExplored:     0,
			UnlimitedTracking: 0,
			XCenter:           mc.offsetX - 64 + (128 << (mc.Scale + 1)),
			ZCenter:           mc.offsetX - 64 + (128 << (mc.Scale + 1)),
			Height:            128,
			Width:             128,
			Decorations:       nil,
			pixels:            nil,
			offsetX:           mc.offsetX,
			offsetZ:           mc.offsetZ,
		}
		maps = append(maps, nm)
		mc = nm
	}
	mc.ParentMapID = -1
	return maps
}

func (m *NbtMAP) Save(p *reflect_provider.Provider) error {
	buf := bytes.NewBuffer(nil)
	enc := nbt.NewEncoderWithEncoding(buf, nbt.LittleEndian)
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("save entities: error encoding NBT: %w", err)
	}
	mapKey := []byte(fmt.Sprintf("map_%v", m.MapID))
	return p.DB.Put(mapKey, buf.Bytes(), nil)
}
