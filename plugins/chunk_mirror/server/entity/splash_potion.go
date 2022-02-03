package entity

import (
	"image/color"
	"time"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/effect"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/entity/physics/trace"
	"main.go/plugins/chunk_mirror/server/internal/nbtconv"
	"main.go/plugins/chunk_mirror/server/item/potion"
	"main.go/plugins/chunk_mirror/server/world"
	"main.go/plugins/chunk_mirror/server/world/particle"
	"main.go/plugins/chunk_mirror/server/world/sound"
)

// SplashPotion is an item that grants effects when thrown.
type SplashPotion struct {
	transform
	yaw, pitch float64

	age   int
	close bool

	owner world.Entity

	t potion.Potion
	c *ProjectileComputer
}

// NewSplashPotion ...
func NewSplashPotion(pos mgl64.Vec3, yaw, pitch float64, owner world.Entity, t potion.Potion) *SplashPotion {
	s := &SplashPotion{
		yaw:   yaw,
		pitch: pitch,
		owner: owner,

		t: t,
		c: &ProjectileComputer{&MovementComputer{
			Gravity:           0.05,
			Drag:              0.01,
			DragBeforeGravity: true,
		}},
	}
	s.transform = newTransform(s, pos)

	return s
}

// Name ...
func (s *SplashPotion) Name() string {
	return "Splash Potion"
}

// EncodeEntity ...
func (s *SplashPotion) EncodeEntity() string {
	return "minecraft:splash_potion"
}

// AABB ...
func (s *SplashPotion) AABB() physics.AABB {
	return physics.NewAABB(mgl64.Vec3{-0.125, 0, -0.125}, mgl64.Vec3{0.125, 0.25, 0.125})
}

// Rotation ...
func (s *SplashPotion) Rotation() (float64, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.yaw, s.pitch
}

// Type returns the type of potion the splash potion will grant effects for when thrown.
func (s *SplashPotion) Type() potion.Potion {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.t
}

// Tick ...
func (s *SplashPotion) Tick(w *world.World, current int64) {
	if s.close {
		_ = s.Close()
		return
	}
	s.mu.Lock()
	m, result := s.c.TickMovement(s, s.pos, s.vel, s.yaw, s.pitch, s.ignores)
	s.pos, s.vel, s.yaw, s.pitch = m.pos, m.vel, m.yaw, m.pitch
	s.mu.Unlock()

	s.age++
	m.Send()

	if m.pos[1] < float64(w.Range()[0]) && current%10 == 0 {
		s.close = true
		return
	}

	if result != nil {
		aabb := s.AABB().Translate(m.pos)

		colour := color.RGBA{R: 0x38, G: 0x5d, B: 0xc6, A: 0xff}
		if effects := s.t.Effects(); len(effects) > 0 {
			colour, _ = effect.ResultingColour(effects)

			ignore := func(entity world.Entity) bool {
				_, living := entity.(Living)
				return !living || entity == s
			}

			for _, e := range w.EntitiesWithin(aabb.GrowVec3(mgl64.Vec3{8.25, 4.25, 8.25}), ignore) {
				pos := e.Position()
				if !e.AABB().Translate(pos).IntersectsWith(aabb.GrowVec3(mgl64.Vec3{4.125, 2.125, 4.125})) {
					continue
				}

				dist := world.Distance(pos, m.pos)
				if dist > 4 {
					continue
				}

				f := 1 - dist/4
				if entityResult, ok := result.(trace.EntityResult); ok && entityResult.Entity() == e {
					f = 1
				}

				splashed := e.(Living)
				for _, eff := range effects {
					if p, ok := eff.Type().(effect.PotentType); ok {
						splashed.AddEffect(effect.NewInstant(p.WithPotency(f), eff.Level()))
						continue
					}

					dur := time.Duration(float64(eff.Duration()) * 0.75 * f)
					if dur < time.Second {
						continue
					}
					splashed.AddEffect(effect.New(eff.Type().(effect.LastingType), eff.Level(), dur))
				}
			}
		} else if s.t.Equals(potion.Water()) {
			switch result := result.(type) {
			case trace.BlockResult:
				pos := result.BlockPosition().Side(result.Face())
				if w.Block(pos) == fire() {
					w.SetBlock(pos, air())
				}

				for _, f := range cube.HorizontalFaces() {
					if h := pos.Side(f); w.Block(h) == fire() {
						w.SetBlock(h, air())
					}
				}
			case trace.EntityResult:
				// TODO: Damage endermen, blazes, striders and snow golems when implemented and rehydrate axolotls.
			}
		}

		w.AddParticle(m.pos, particle.Splash{Colour: colour})
		w.PlaySound(m.pos, sound.GlassBreak{})

		s.close = true
	}
}

// ignores returns whether the SplashPotion should ignore collision with the entity passed.
func (s *SplashPotion) ignores(entity world.Entity) bool {
	_, ok := entity.(Living)
	return !ok || entity == s || (s.age < 5 && entity == s.owner)
}

// New creates a SplashPotion with the position, velocity, yaw, and pitch provided. It doesn't spawn the SplashPotion,
// only returns it.
func (s *SplashPotion) New(pos, vel mgl64.Vec3, yaw, pitch float64, t potion.Potion) world.Entity {
	splash := NewSplashPotion(pos, yaw, pitch, nil, t)
	splash.vel = vel
	return splash
}

// Owner ...
func (s *SplashPotion) Owner() world.Entity {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.owner
}

// Own ...
func (s *SplashPotion) Own(owner world.Entity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.owner = owner
}

// DecodeNBT decodes the properties in a map to a SplashPotion and returns a new SplashPotion entity.
func (s *SplashPotion) DecodeNBT(data map[string]interface{}) interface{} {
	return s.New(
		nbtconv.MapVec3(data, "Pos"),
		nbtconv.MapVec3(data, "Motion"),
		float64(nbtconv.MapFloat32(data, "Yaw")),
		float64(nbtconv.MapFloat32(data, "Pitch")),
		potion.From(nbtconv.MapInt32(data, "PotionId")),
	)
}

// EncodeNBT encodes the SplashPotion entity's properties as a map and returns it.
func (s *SplashPotion) EncodeNBT() map[string]interface{} {
	yaw, pitch := s.Rotation()
	return map[string]interface{}{
		"Pos":      nbtconv.Vec3ToFloat32Slice(s.Position()),
		"Yaw":      yaw,
		"Pitch":    pitch,
		"Motion":   nbtconv.Vec3ToFloat32Slice(s.Velocity()),
		"Damage":   0.0,
		"PotionId": s.t.Uint8(),
	}
}

// air returns an air block.
func air() world.Block {
	f, ok := world.BlockByName("minecraft:air", map[string]interface{}{})
	if !ok {
		panic("could not find air block")
	}
	return f
}
