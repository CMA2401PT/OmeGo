package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/df-mc/goleveldb/leveldb"
	"github.com/df-mc/goleveldb/leveldb/opt"
	"io/ioutil"
	"main.go/dragonfly/server/world"
	"main.go/minecraft/nbt"
	"os"
	"path/filepath"
)

// Provider implements a world provider for the Minecraft world format, which is based on a leveldb database.
type Provider struct {
	dim Dimension
	DB  *leveldb.DB
	dir string
	D   interface{}
}

// chunkVersion is the current version of chunks.
const chunkVersion = 19

// New creates a new provider reading and writing files to files under the path passed. If a world is present
// at the path, New will parse its data and initialise the world with it. If the data cannot be parsed, an
// error is returned.
func New(dir string) (*Provider, error) {
	_ = os.MkdirAll(filepath.Join(dir, "db"), 0777)

	p := &Provider{dir: dir}
	p.dim = Overworld
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

	db, err := leveldb.OpenFile(filepath.Join(dir, "db"), &opt.Options{
		Compression: opt.FlateCompression,
		BlockSize:   16 * opt.KiB,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening leveldb database: %w", err)
	}
	p.DB = db
	return p, nil
}

//
//// LoadChunk loads a chunk at the position passed from the leveldb database. If it doesn't exist, exists is
//// false. If an error is returned, exists is always assumed to be true.
//func (p *Provider) LoadChunk(position world.ChunkPos) (c *chunk.Chunk, exists bool, err error) {
//	data := chunk.SerialisedData{}
//	key := index(position)
//
//	// This key is where the version of a chunk resides. The chunk version has changed many times, without any
//	// actual substantial changes, so we don't check this.
//	version, err := p.DB.Get(append(key, keyVersion), nil)
//	fmt.Println("version", version)
//	if err == leveldb.ErrNotFound {
//		// The new key was not found, so we try the old key.
//		if _, err = p.DB.Get(append(key, keyVersionOld), nil); err != nil {
//			return nil, false, nil
//		}
//	} else if err != nil {
//		return nil, true, fmt.Errorf("error reading version: %w", err)
//	}
//
//	data.Data2D, err = p.DB.Get(append(key, key2DData), nil)
//	if err == leveldb.ErrNotFound {
//		return nil, false, nil
//	} else if err != nil {
//		return nil, true, fmt.Errorf("error reading 2D data: %w", err)
//	}
//
//	data.BlockNBT, err = p.DB.Get(append(key, keyBlockEntities), nil)
//	// Block entities aren't present when there aren't any, so it's okay if we can't find the key.
//	if err != nil && err != leveldb.ErrNotFound {
//		return nil, true, fmt.Errorf("error reading block entities: %w", err)
//	}
//
//	for y := byte(0); y < 16; y++ {
//		data.SubChunks[y], err = p.DB.Get(append(key, keySubChunkData, y), nil)
//		if err == leveldb.ErrNotFound {
//			// No sub chunk present at this Y level. We skip this one and move to the next, which might still
//			// be present.
//			continue
//		} else if err != nil {
//			return nil, true, fmt.Errorf("error reading 2D sub chunk %v: %w", y, err)
//		}
//	}
//	c, err = chunk.DiskDecode(data)
//	return c, true, err
//}
type ChunkInfo struct {
	Version   []byte
	Biomes    []byte
	BlockNBT  []byte
	SubChunks [][]byte
}

func DiskDecode(data ChunkInfo, r Range) {
	airRID := 314

	c := New(airRID, r)

	err := decodeBiomes(bytes.NewBuffer(data.Biomes), c, DiskEncoding)
	if err != nil {
		return nil, err
	}
	for i, sub := range data.SubChunks {
		if len(sub) == 0 {
			// No data for this sub chunk.
			continue
		}
		index := uint8(i)
		if c.sub[index], err = decodeSubChunk(bytes.NewBuffer(sub), c, &index, DiskEncoding); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (p *Provider) LoadChunk(position world.ChunkPos) (c *ChunkInfo, exists bool, err error) {
	key := p.index(position)
	chunkInfo := &ChunkInfo{}

	// This key is where the version of a chunk resides. The chunk version has changed many times, without any
	// actual substantial changes, so we don't check this.
	chunkInfo.Version, err = p.DB.Get(append(key, keyVersion), nil)
	if err == leveldb.ErrNotFound {
		// The new key was not found, so we try the old key.
		if _, err = p.DB.Get(append(key, keyVersionOld), nil); err != nil {
			return nil, false, nil
		}
	} else if err != nil {
		return nil, true, fmt.Errorf("error reading version: %w", err)
	}

	chunkInfo.Biomes, err = p.DB.Get(append(key, key3DData), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, false, fmt.Errorf("error reading 3D data: %w", err)
	}
	if len(chunkInfo.Biomes) > 512 {
		// Strip the heightmap from the biomes.
		chunkInfo.Biomes = chunkInfo.Biomes[512:]
	}

	chunkInfo.BlockNBT, err = p.DB.Get(append(key, keyBlockEntities), nil)
	// Block entities aren't present when there aren't any, so it's okay if we can't find the key.
	if err != nil && err != leveldb.ErrNotFound {
		return nil, true, fmt.Errorf("error reading block entities: %w", err)
	}
	chunkInfo.SubChunks = make([][]byte, (p.dim.Range().Height()>>4)+1)
	for i := range chunkInfo.SubChunks {
		chunkInfo.SubChunks[i], err = p.DB.Get(append(key, keySubChunkData, uint8(i+(p.dim.Range()[0]>>4))), nil)
		if err == leveldb.ErrNotFound {
			// No sub chunk present at this Y level. We skip this one and move to the next, which might still
			// be present.
			continue
		} else if err != nil {
			return nil, true, fmt.Errorf("error reading sub chunk data %v: %w", i, err)
		}
	}
	//c, err = chunk.DiskDecode(data, p.dim.Range())
	return chunkInfo, true, err
}

//// LoadEntities loads all entities from the chunk position passed.
//func (p *Provider) LoadEntities(pos world.ChunkPos) ([]world.SaveableEntity, error) {
//	data, err := p.DB.Get(append(index(pos), keyEntities), nil)
//	if err != leveldb.ErrNotFound && err != nil {
//		return nil, err
//	}
//	var a []world.SaveableEntity
//
//	buf := bytes.NewBuffer(data)
//	dec := nbt.NewDecoderWithEncoding(buf, nbt.LittleEndian)
//
//	for buf.Len() != 0 {
//		var m map[string]interface{}
//		if err := dec.Decode(&m); err != nil {
//			return nil, fmt.Errorf("error decoding block NBT: %w", err)
//		}
//		id, ok := m["identifier"]
//		if !ok {
//			return nil, fmt.Errorf("entity has no ID but data (%v)", m)
//		}
//		name, _ := id.(string)
//		e, ok := world.EntityByName(name)
//		if !ok {
//			// Entity was not registered: This can only be expected sometimes, so the best we can do is to just
//			// ignore this and proceed.
//			continue
//		}
//		if v := e.DecodeNBT(m); v != nil {
//			a = append(a, v.(world.SaveableEntity))
//		}
//	}
//	return a, nil
//}

//// LoadBlockNBT loads all block entities from the chunk position passed.
//func (p *Provider) LoadBlockNBT(position world.ChunkPos) ([]map[string]interface{}, error) {
//	data, err := p.DB.Get(append(index(position), keyBlockEntities), nil)
//	if err != leveldb.ErrNotFound && err != nil {
//		return nil, err
//	}
//	var a []map[string]interface{}
//
//	buf := bytes.NewBuffer(data)
//	dec := nbt.NewDecoderWithEncoding(buf, nbt.LittleEndian)
//
//	for buf.Len() != 0 {
//		var m map[string]interface{}
//		if err := dec.Decode(&m); err != nil {
//			return nil, fmt.Errorf("error decoding block NBT: %w", err)
//		}
//		a = append(a, m)
//	}
//	return a, nil
//}

//// index returns a byte buffer holding the written index of the chunk position passed.
//func index(position world.ChunkPos) []byte {
//	x, z := uint32(position[0]), uint32(position[1])
//	return []byte{
//		byte(x), byte(x >> 8), byte(x >> 16), byte(x >> 24),
//		byte(z), byte(z >> 8), byte(z >> 16), byte(z >> 24),
//	}
//}

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

func main() {
	save_dir := "C:\\Users\\daiji\\AppData\\Local\\Packages\\Microsoft.MinecraftUWP_8wekyb3d8bbwe\\LocalState\\games\\com.mojang\\minecraftWorlds\\26z6YfCVBwA="
	p, err := New(save_dir)
	if err != nil {
		fmt.Println(err)
	}
	//pDc, _ := p.D.(map[string]interface{})
	//for key, v := range pDc {
	//	fmt.Println(key, " ", v)
	//}
	c, e, err := p.LoadChunk(world.ChunkPos{0, 0})
	if !e {
		fmt.Println("Chunk not exist")
	}
	if err != nil {
		fmt.Println("Cannot Load Chunk %v", err)
	}
	subChunks := c.SubChunks
	fmt.Println(len(subChunks))
}
