package entity

import (
	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/world_mirror/server/entity/damage"
	"main.go/plugins/world_mirror/server/entity/physics"
	"main.go/plugins/world_mirror/server/entity/physics/trace"
	"main.go/plugins/world_mirror/server/internal/nbtconv"
	"main.go/plugins/world_mirror/server/world"
	"main.go/plugins/world_mirror/server/world/particle"
	"main.go/plugins/world_mirror/server/world/sound"
)

// EnderPearl is a smooth, greenish-blue item used to teleport and to make an eye of ender.
type EnderPearl struct {
	transform
	yaw, pitch float64

	age   int
	close bool

	owner world.Entity

	c *ProjectileComputer
}

// NewEnderPearl ...
func NewEnderPearl(pos mgl64.Vec3, yaw, pitch float64, owner world.Entity) *EnderPearl {
	e := &EnderPearl{
		yaw:   yaw,
		pitch: pitch,
		c: &ProjectileComputer{&MovementComputer{
			Gravity:           0.03,
			Drag:              0.01,
			DragBeforeGravity: true,
		}},
		owner: owner,
	}
	e.transform = newTransform(e, pos)

	return e
}

// Name ...
func (e *EnderPearl) Name() string {
	return "Ender Pearl"
}

// EncodeEntity ...
func (e *EnderPearl) EncodeEntity() string {
	return "minecraft:ender_pearl"
}

// AABB ...
func (e *EnderPearl) AABB() physics.AABB {
	return physics.NewAABB(mgl64.Vec3{-0.125, 0, -0.125}, mgl64.Vec3{0.125, 0.25, 0.125})
}

// Rotation ...
func (e *EnderPearl) Rotation() (float64, float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.yaw, e.pitch
}

// teleporter represents a living entity that can teleport.
type teleporter interface {
	// Teleport teleports the entity to the position given.
	Teleport(pos mgl64.Vec3)
	Living
}

// Tick ...
func (e *EnderPearl) Tick(w *world.World, current int64) {
	if e.close {
		_ = e.Close()
		return
	}
	e.mu.Lock()
	m, result := e.c.TickMovement(e, e.pos, e.vel, e.yaw, e.pitch, e.ignores)
	e.pos, e.vel, e.yaw, e.pitch = m.pos, m.vel, m.yaw, m.pitch
	e.mu.Unlock()

	e.age++
	m.Send()

	if m.pos[1] < float64(w.Range()[0]) && current%10 == 0 {
		e.close = true
		return
	}

	if result != nil {
		if r, ok := result.(trace.EntityResult); ok {
			if l, ok := r.Entity().(Living); ok {
				if _, vulnerable := l.Hurt(0.0, damage.SourceEntityAttack{Attacker: e}); vulnerable {
					l.KnockBack(m.pos, 0.45, 0.3608)
				}
			}
		}

		if owner := e.Owner(); owner != nil {
			if user, ok := owner.(teleporter); ok {
				w.PlaySound(user.Position(), sound.EndermanTeleport{})

				user.Teleport(m.pos)

				w.AddParticle(m.pos, particle.EndermanTeleportParticle{})
				w.PlaySound(m.pos, sound.EndermanTeleport{})

				user.Hurt(5, damage.SourceFall{})
			}
		}

		e.close = true
	}
}

// ignores returns whether the ender pearl should ignore collision with the entity passed.
func (e *EnderPearl) ignores(entity world.Entity) bool {
	_, ok := entity.(Living)
	return !ok || entity == e || (e.age < 5 && entity == e.owner)
}

// New creates an ender pearl with the position, velocity, yaw, and pitch provided. It doesn't spawn the ender pearl,
// only returns it.
func (e *EnderPearl) New(pos, vel mgl64.Vec3, yaw, pitch float64) world.Entity {
	pearl := NewEnderPearl(pos, yaw, pitch, nil)
	pearl.vel = vel
	return pearl
}

// Owner ...
func (e *EnderPearl) Owner() world.Entity {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.owner
}

// Own ...
func (e *EnderPearl) Own(owner world.Entity) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.owner = owner
}

// DecodeNBT decodes the properties in a map to a EnderPearl and returns a new EnderPearl entity.
func (e *EnderPearl) DecodeNBT(data map[string]interface{}) interface{} {
	return e.New(
		nbtconv.MapVec3(data, "Pos"),
		nbtconv.MapVec3(data, "Motion"),
		float64(nbtconv.MapFloat32(data, "Pitch")),
		float64(nbtconv.MapFloat32(data, "Yaw")),
	)
}

// EncodeNBT encodes the EnderPearl entity's properties as a map and returns it.
func (e *EnderPearl) EncodeNBT() map[string]interface{} {
	yaw, pitch := e.Rotation()
	return map[string]interface{}{
		"Pos":    nbtconv.Vec3ToFloat32Slice(e.Position()),
		"Yaw":    yaw,
		"Pitch":  pitch,
		"Motion": nbtconv.Vec3ToFloat32Slice(e.Velocity()),
		"Damage": 0.0,
	}
}
