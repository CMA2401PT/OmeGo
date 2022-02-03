package mcdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"main.go/minecraft/nbt"
	"main.go/minecraft/protocol"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/world"
	"main.go/plugins/chunk_mirror/server/world/chunk"
)

// Provider implements a world provider for the Minecraft world format, which is based on a leveldb database.
type Provider struct {
	DB  *leveldb.DB
	dim world.Dimension
	dir string
	D   data
}

// chunkVersion is the current version of chunks.
const chunkVersion = 27

// New creates a new provider reading and writing from/to files under the path passed. If a world is present
// at the path, New will parse its data and initialise the world with it. If the data cannot be parsed, an
// error is returned.
func New(dir string, d world.Dimension) (*Provider, error) {
	_ = os.MkdirAll(filepath.Join(dir, "db"), 0777)

	p := &Provider{dir: dir, dim: d}
	if _, err := os.Stat(filepath.Join(dir, "level.dat")); os.IsNotExist(err) {
		// A level.dat was not currently present for the world.
		p.initDefaultLevelDat()
	} else {
		f, err := ioutil.ReadFile(filepath.Join(dir, "level.dat"))
		if err != nil {
			return nil, fmt.Errorf("error opening level.dat file: %w", err)
		}
		// The first 8 bytes are a useless header (version and length): We don't need it.
		if len(f) < 8 {
			// The file did not have enough content, meaning it is corrupted. We return an error.
			return nil, fmt.Errorf("level.dat exists but has no data")
		}
		if err := nbt.UnmarshalEncoding(f[8:], &p.D, nbt.LittleEndian); err != nil {
			return nil, fmt.Errorf("error decoding level.dat NBT: %w", err)
		}
		p.D.WorldStartCount++
	}
	db, ok := cacheLoad(dir)
	if !ok {
		var err error
		if db, err = leveldb.OpenFile(filepath.Join(dir, "db"), &opt.Options{
			Compression: opt.FlateCompression,
			BlockSize:   16 * opt.KiB,
		}); err != nil {
			return nil, fmt.Errorf("error opening leveldb database: %w", err)
		}
		cacheStore(dir, db)
	}

	p.DB = db
	return p, nil
}

// initDefaultLevelDat initialises a default level.dat file.
func (p *Provider) initDefaultLevelDat() {
	p.D.DoDayLightCycle = true
	p.D.DoWeatherCycle = true
	p.D.BaseGameVersion = protocol.CurrentVersion
	p.D.NetworkVersion = protocol.CurrentProtocol
	p.D.LastOpenedWithVersion = minimumCompatibleClientVersion
	p.D.MinimumCompatibleClientVersion = minimumCompatibleClientVersion
	p.D.LevelName = "World"
	p.D.GameType = 1
	p.D.StorageVersion = 8
	p.D.Generator = 1
	p.D.Abilities.WalkSpeed = 0.1
	p.D.PVP = true
	p.D.WorldStartCount = 1
	p.D.RandomTickSpeed = 1
	p.D.FallDamage = true
	p.D.FireDamage = true
	p.D.DrowningDamage = true
	p.D.CommandsEnabled = true
	p.D.MultiPlayerGame = true
	p.D.SpawnY = math.MaxInt32
	p.D.Difficulty = 2
	p.D.DoWeatherCycle = true
	p.D.RainLevel = 1.0
	p.D.LightningLevel = 1.0
	p.D.ServerChunkTickRange = 6
	p.D.NetherScale = 8
}

// Settings returns the world.Settings of the world loaded by the Provider.
func (p *Provider) Settings(s *world.Settings) {
	s.Name = p.D.LevelName
	s.Spawn = cube.Pos{int(p.D.SpawnX), int(p.D.SpawnY), int(p.D.SpawnZ)}
	s.Time = p.D.Time
	s.TimeCycle = p.D.DoDayLightCycle
	s.WeatherCycle = p.D.DoWeatherCycle
	s.RainTime = int64(p.D.RainTime)
	s.Raining = p.D.RainLevel > 0
	s.ThunderTime = int64(p.D.LightningTime)
	s.Thundering = p.D.LightningLevel > 0
	s.CurrentTick = p.D.CurrentTick
	s.DefaultGameMode = p.loadDefaultGameMode()
	s.Difficulty = p.loadDifficulty()
	s.TickRange = p.D.ServerChunkTickRange
}

// SaveSettings saves the world.Settings passed to the level.dat.
func (p *Provider) SaveSettings(s *world.Settings) {
	p.D.LevelName = s.Name
	p.D.SpawnX, p.D.SpawnY, p.D.SpawnZ = int32(s.Spawn.X()), int32(s.Spawn.Y()), int32(s.Spawn.Z())
	p.D.Time = s.Time
	p.D.DoDayLightCycle = s.TimeCycle
	p.D.DoWeatherCycle = s.WeatherCycle
	p.D.RainTime, p.D.RainLevel = int32(s.RainTime), 0
	p.D.LightningTime, p.D.LightningLevel = int32(s.ThunderTime), 0
	if s.Raining {
		p.D.RainLevel = 1
	}
	if s.Thundering {
		p.D.LightningLevel = 1
	}
	p.D.CurrentTick = s.CurrentTick
	p.D.ServerChunkTickRange = s.TickRange
	p.saveDefaultGameMode(s.DefaultGameMode)
	p.saveDifficulty(s.Difficulty)
}

// LoadChunk loads a chunk at the position passed from the leveldb database. If it doesn't exist, exists is
// false. If an error is returned, exists is always assumed to be true.
func (p *Provider) LoadChunk(position world.ChunkPos) (c *chunk.Chunk, exists bool, err error) {
	data := chunk.SerialisedData{}
	key := p.index(position)

	// This key is where the version of a chunk resides. The chunk version has changed many times, without any
	// actual substantial changes, so we don't check this.
	_, err = p.DB.Get(append(key, keyVersion), nil)
	if err == leveldb.ErrNotFound {
		// The new key was not found, so we try the old key.
		if _, err = p.DB.Get(append(key, keyVersionOld), nil); err != nil {
			return nil, false, nil
		}
	} else if err != nil {
		return nil, true, fmt.Errorf("error reading version: %w", err)
	}

	data.Biomes, err = p.DB.Get(append(key, key3DData), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, false, fmt.Errorf("error reading 3D data: %w", err)
	}
	if len(data.Biomes) > 512 {
		// Strip the heightmap from the biomes.
		data.Biomes = data.Biomes[512:]
	}

	data.BlockNBT, err = p.DB.Get(append(key, keyBlockEntities), nil)
	// Block entities aren't present when there aren't any, so it's okay if we can't find the key.
	if err != nil && err != leveldb.ErrNotFound {
		return nil, true, fmt.Errorf("error reading block entities: %w", err)
	}
	data.AuxNBTData, err = p.DB.Get(append(key, keySelfDefineAuxData), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, true, fmt.Errorf("error reading aux nbt data: %w", err)
	}
	data.SubChunks = make([][]byte, (p.dim.Range().Height()>>4)+1)
	for i := range data.SubChunks {
		data.SubChunks[i], err = p.DB.Get(append(key, keySubChunkData, uint8(i+(p.dim.Range()[0]>>4))), nil)
		if err == leveldb.ErrNotFound {
			// No sub chunk present at this Y level. We skip this one and move to the next, which might still
			// be present.
			continue
		} else if err != nil {
			return nil, true, fmt.Errorf("error reading sub chunk data %v: %w", i, err)
		}
	}
	c, err = chunk.DiskDecode(data, p.dim.Range())
	return c, true, err
}

// SaveChunk saves a chunk at the position passed to the leveldb database. Its version is written as the
// version in the chunkVersion constant.
func (p *Provider) SaveChunk(position world.ChunkPos, c *chunk.Chunk) error {
	data := chunk.Encode(c, chunk.DiskEncoding)

	key := p.index(position)
	_ = p.DB.Put(append(key, keyVersion), []byte{chunkVersion}, nil)
	// Write the heightmap by just writing 512 empty bytes.
	_ = p.DB.Put(append(key, key3DData), append(make([]byte, 512), data.Biomes...), nil)

	finalisation := make([]byte, 4)
	binary.LittleEndian.PutUint32(finalisation, 2)
	_ = p.DB.Put(append(key, keyFinalisation), finalisation, nil)

	for i, sub := range data.SubChunks {
		_ = p.DB.Put(append(key, keySubChunkData, byte(i+(c.Range()[0]>>4))), sub, nil)
	}
	return nil
}

// loadDefaultGameMode returns the default game mode stored in the level.dat.
func (p *Provider) loadDefaultGameMode() world.GameMode {
	switch p.D.GameType {
	default:
		return world.GameModeSurvival
	case 1:
		return world.GameModeCreative
	case 2:
		return world.GameModeAdventure
	case 3:
		return world.GameModeSpectator
	}
}

// saveDefaultGameMode changes the default game mode in the level.dat.
func (p *Provider) saveDefaultGameMode(mode world.GameMode) {
	switch mode {
	case world.GameModeSurvival:
		p.D.GameType = 0
	case world.GameModeCreative:
		p.D.GameType = 1
	case world.GameModeAdventure:
		p.D.GameType = 2
	case world.GameModeSpectator:
		p.D.GameType = 3
	}
}

// loadDifficulty loads the difficulty stored in the level.dat.
func (p *Provider) loadDifficulty() world.Difficulty {
	switch p.D.Difficulty {
	default:
		return world.DifficultyNormal
	case 0:
		return world.DifficultyPeaceful
	case 1:
		return world.DifficultyEasy
	case 3:
		return world.DifficultyHard
	}
}

// saveDifficulty saves the difficulty passed to the level.dat.
func (p *Provider) saveDifficulty(d world.Difficulty) {
	switch d {
	case world.DifficultyPeaceful:
		p.D.Difficulty = 0
	case world.DifficultyEasy:
		p.D.Difficulty = 1
	case world.DifficultyNormal:
		p.D.Difficulty = 2
	case world.DifficultyHard:
		p.D.Difficulty = 3
	}
}

// LoadEntities loads all entities from the chunk position passed.
func (p *Provider) LoadEntities(pos world.ChunkPos) ([]world.SaveableEntity, error) {
	data, err := p.DB.Get(append(p.index(pos), keyEntities), nil)
	if err != leveldb.ErrNotFound && err != nil {
		return nil, err
	}
	var a []world.SaveableEntity

	buf := bytes.NewBuffer(data)
	dec := nbt.NewDecoderWithEncoding(buf, nbt.LittleEndian)

	for buf.Len() != 0 {
		var m map[string]interface{}
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("error decoding block NBT: %w", err)
		}
		id, ok := m["identifier"]
		if !ok {
			return nil, fmt.Errorf("entity has no ID but data (%v)", m)
		}
		name, _ := id.(string)
		e, ok := world.EntityByName(name)
		if !ok {
			// Entity was not registered: This can only be expected sometimes, so the best we can do is to just
			// ignore this and proceed.
			continue
		}
		if v := e.DecodeNBT(m); v != nil {
			a = append(a, v.(world.SaveableEntity))
		}
	}
	return a, nil
}

// SaveEntities saves all entities to the chunk position passed.
func (p *Provider) SaveEntities(pos world.ChunkPos, entities []world.SaveableEntity) error {
	if len(entities) == 0 {
		return p.DB.Delete(append(p.index(pos), keyEntities), nil)
	}

	buf := bytes.NewBuffer(nil)
	enc := nbt.NewEncoderWithEncoding(buf, nbt.LittleEndian)
	for _, e := range entities {
		x := e.EncodeNBT()
		x["identifier"] = e.EncodeEntity()
		if err := enc.Encode(x); err != nil {
			return fmt.Errorf("save entities: error encoding NBT: %w", err)
		}
	}
	return p.DB.Put(append(p.index(pos), keyEntities), buf.Bytes(), nil)
}

// SaveBlockNBT saves all block NBT data to the chunk position passed.
func (p *Provider) SaveBlockAuxData(position world.ChunkPos, data []map[string]interface{}) error {
	if len(data) == 0 {
		return p.DB.Delete(append(p.index(position), keyBlockEntities), nil)
	}
	buf := bytes.NewBuffer(nil)
	enc := nbt.NewEncoderWithEncoding(buf, nbt.LittleEndian)
	for _, d := range data {
		if err := enc.Encode(d); err != nil {
			return fmt.Errorf("error encoding block NBT: (%w)", err)
		}
	}
	return p.DB.Put(append(p.index(position), keySelfDefineAuxData), buf.Bytes(), nil)
}

// SaveBlockNBT saves all block NBT data to the chunk position passed.
func (p *Provider) LoadBlockAuxData(position world.ChunkPos) ([]map[string]interface{}, error) {
	data, err := p.DB.Get(append(p.index(position), keySelfDefineAuxData), nil)
	if err != leveldb.ErrNotFound && err != nil {
		return nil, err
	}
	var a []map[string]interface{}

	buf := bytes.NewBuffer(data)
	dec := nbt.NewDecoderWithEncoding(buf, nbt.LittleEndian)

	for buf.Len() != 0 {
		var m map[string]interface{}
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("error decoding block NBT: %w", err)
		}
		a = append(a, m)
	}
	return a, nil
}

// LoadBlockNBT loads all block entities from the chunk position passed.
func (p *Provider) LoadBlockNBT(position world.ChunkPos) ([]map[string]interface{}, error) {
	data, err := p.DB.Get(append(p.index(position), keyBlockEntities), nil)
	if err != leveldb.ErrNotFound && err != nil {
		return nil, err
	}
	var a []map[string]interface{}

	buf := bytes.NewBuffer(data)
	dec := nbt.NewDecoderWithEncoding(buf, nbt.LittleEndian)

	for buf.Len() != 0 {
		var m map[string]interface{}
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("error decoding block NBT: %w", err)
		}
		a = append(a, m)
	}
	return a, nil
}

// SaveBlockNBT saves all block NBT data to the chunk position passed.
func (p *Provider) SaveBlockNBT(position world.ChunkPos, data []map[string]interface{}) error {
	if len(data) == 0 {
		return p.DB.Delete(append(p.index(position), keyBlockEntities), nil)
	}
	buf := bytes.NewBuffer(nil)
	enc := nbt.NewEncoderWithEncoding(buf, nbt.LittleEndian)
	for _, d := range data {
		if err := enc.Encode(d); err != nil {
			return fmt.Errorf("error encoding block NBT: %w", err)
		}
	}
	return p.DB.Put(append(p.index(position), keyBlockEntities), buf.Bytes(), nil)
}

// Close closes the provider, saving any file that might need to be saved, such as the level.dat.
func (p *Provider) Close() error {
	p.D.LastPlayed = time.Now().Unix()
	if cacheDelete(p.dir) != 0 {
		// The same provider is still alive elsewhere. Don't store the data to the level.dat and levelname.txt just yet.
		return nil
	}
	f, err := os.OpenFile(filepath.Join(p.dir, "level.dat"), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening level.dat file: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	_ = binary.Write(buf, binary.LittleEndian, int32(3))
	nbtData, err := nbt.MarshalEncoding(p.D, nbt.LittleEndian)
	if err != nil {
		return fmt.Errorf("error encoding level.dat to NBT: %w", err)
	}
	_ = binary.Write(buf, binary.LittleEndian, int32(len(nbtData)))
	_, _ = buf.Write(nbtData)

	_, _ = f.Write(buf.Bytes())

	if err := f.Close(); err != nil {
		return fmt.Errorf("error closing level.dat: %w", err)
	}
	//noinspection SpellCheckingInspection
	if err := ioutil.WriteFile(filepath.Join(p.dir, "levelname.txt"), []byte(p.D.LevelName), 0644); err != nil {
		return fmt.Errorf("error writing levelname.txt: %w", err)
	}
	return p.DB.Close()
}

// index returns a byte buffer holding the written index of the chunk position passed. If the dimension passed to New
// is not world.Overworld, the length of the index returned is 12. It is 8 otherwise.
func (p *Provider) index(position world.ChunkPos) []byte {
	x, z, dim := uint32(position[0]), uint32(position[1]), uint32(p.dim.EncodeDimension())
	b := make([]byte, 12)

	binary.LittleEndian.PutUint32(b, x)
	binary.LittleEndian.PutUint32(b[4:], z)
	if dim == 0 {
		return b[:8]
	}
	binary.LittleEndian.PutUint32(b[8:], dim)
	return b
}
