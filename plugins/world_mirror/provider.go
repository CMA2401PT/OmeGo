package world_mirror

import (
	"fmt"
	"main.go/plugins/chunk_mirror"
	provider_world "main.go/plugins/chunk_mirror/server/world"
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/world"
	"main.go/plugins/world_mirror/server/world/chunk"
)

// NoIOProvider implements a Provider while not performing any disk I/O. It generates values on the run and
// dynamically, instead of reading and writing data, and returns otherwise empty values.
type ReflectIOProvider struct {
	cm         *chunk_mirror.ChunkMirror
	isWaiting  bool
	waitingPos provider_world.ChunkPos
	waitChan   chan *provider_world.ChunkData
}

func cvtPos(pos provider_world.ChunkPos) world.ChunkPos {
	return world.ChunkPos{pos.X(), pos.Z()}
}

// Settings ...
func (p *ReflectIOProvider) Settings(settings *world.Settings) {}

// SaveSettings ...
func (p *ReflectIOProvider) SaveSettings(*world.Settings) {}

// LoadEntities ...
func (p *ReflectIOProvider) LoadEntities(world.ChunkPos) ([]world.SaveableEntity, error) {
	return nil, nil
}

// SaveEntities ...
func (p *ReflectIOProvider) SaveEntities(world.ChunkPos, []world.SaveableEntity) error {
	return nil
}

// LoadBlockNBT ...
func (p *ReflectIOProvider) LoadBlockNBT(pos world.ChunkPos) ([]map[string]interface{}, error) {
	_pos := provider_world.ChunkPos{pos.X(), pos.Z()}
	chunkData, _ := p.cm.GetCachedChunk(_pos)
	if chunkData == nil {
		p.isWaiting = true
		p.waitingPos = _pos
		fmt.Println("waiting For ChunkData to Load Chunk")
		chunkData = <-p.waitChan
	}
	c := chunkData

	m := make([]map[string]interface{}, 0, len(c.E))
	for pos, b := range c.E {
		if n, ok := b.(provider_world.NBTer); ok {
			// Encode the block entities and add the 'x', 'y' and 'z' tags to it.
			data := n.EncodeNBT()
			data["x"], data["y"], data["z"] = int32(pos[0]), int32(pos[1]), int32(pos[2])
			m = append(m, data)
		}
	}
	return m, nil
}

// SaveBlockNBT ...
func (p *ReflectIOProvider) SaveBlockNBT(world.ChunkPos, []map[string]interface{}) error {
	return nil
}

// SaveChunk ...
func (p *ReflectIOProvider) SaveChunk(world.ChunkPos, *chunk.Chunk) error {
	return nil
}

// LoadChunk ...
func (p *ReflectIOProvider) LoadChunk(pos world.ChunkPos) (*chunk.Chunk, bool, error) {
	_pos := provider_world.ChunkPos{pos.X(), pos.Z()}
	chunkData, _ := p.cm.GetCachedChunk(_pos)
	if chunkData == nil {
		p.isWaiting = true
		p.waitingPos = _pos
		fmt.Println("waiting For ChunkData to Load Chunk")
		chunkData = <-p.waitChan
	}
	c := chunkData

	subs := make([]*chunk.SubChunk, 0)
	biomes := make([]*chunk.PalettedStorage, 0)
	for _, subc := range c.Sub() {
		subs = append(subs, chunk.InitFromReflect(subc))
	}

	newChunk := chunk.NewChunk(cube.Range(c.Range()), c.Air(), subs, biomes)
	return newChunk, true, nil
}

// Close ...
func (p *ReflectIOProvider) Close() error {
	return nil
}

func (p *ReflectIOProvider) onNewChunk(pos provider_world.ChunkPos, cdata *provider_world.ChunkData) {
	if p.isWaiting && p.waitingPos == pos {
		p.waitChan <- cdata
		p.isWaiting = false
	}
}

func NewReflectProvider(cm *chunk_mirror.ChunkMirror) *ReflectIOProvider {
	p := &ReflectIOProvider{cm: cm}
	p.waitChan = make(chan *provider_world.ChunkData)
	cm.RegChunkListener(func(X, Z int) bool {
		return true
	}, p.onNewChunk)
	return p
}
