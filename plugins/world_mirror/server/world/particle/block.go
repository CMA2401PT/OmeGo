package particle

import (
	"image/color"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/world_mirror/server/block/cube"
	"main.go/plugins/world_mirror/server/world"
	"main.go/plugins/world_mirror/server/world/sound"
)

// Flame is a particle shown around torches. It can have any colour specified with the Colour field.
type Flame struct {
	particle
	// Colour is the colour of the Flame particle.
	Colour color.RGBA
}

// BlockBreak is a particle sent when a block is broken. It represents a bunch of particles that are textured
// like the block that the particle holds.
type BlockBreak struct {
	particle
	// Block is the block of which particles should be shown. The particles will change depending on what
	// block is held.
	Block world.Block
}

// PunchBlock is a particle shown when a player is punching a block. It shows particles of a specific block
// type at a particular face of a block.
type PunchBlock struct {
	particle
	// Block is the block of which particles should be shown. The particles will change depending on what
	// block is punched.
	Block world.Block
	// Face is the face of the block that was punched. It is here that the particles will be shown.
	Face cube.Face
}

// BlockForceField is a particle that shows up as a block that turns invisible from an opaque black colour.
type BlockForceField struct{ particle }

// BoneMeal is a particle that shows up on bone meal usage.
type BoneMeal struct{ particle }

// Note is a particle that shows up on note block interactions.
type Note struct {
	particle

	// Instrument is the instrument of the note block.
	Instrument sound.Instrument
	// Pitch is the pitch of the note.
	Pitch int
}

// DragonEggTeleport is a particle that shows up when a dragon egg teleports.
type DragonEggTeleport struct {
	particle

	// Diff is a Pos with the values being the difference from the original position to the new position.
	Diff cube.Pos
}

// Evaporate is a particle that shows up when a water block evaporates
type Evaporate struct{ particle }

// particle serves as a base for all particles in this package.
type particle struct{}

// Spawn ...
func (particle) Spawn(*world.World, mgl64.Vec3) {}