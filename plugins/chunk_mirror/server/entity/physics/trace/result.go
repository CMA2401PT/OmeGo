package trace

import (
	"math"

	"github.com/go-gl/mathgl/mgl64"
	"main.go/plugins/chunk_mirror/server/block/cube"
	"main.go/plugins/chunk_mirror/server/entity/physics"
	"main.go/plugins/chunk_mirror/server/world"
)

// Result represents the result of a ray trace collision with a bounding box.
type Result interface {
	// AABB returns the bounding box collided with.
	AABB() physics.AABB
	// Position returns where the ray first collided with the bounding box.
	Position() mgl64.Vec3
	// Face returns the face of the bounding box that was collided on.
	Face() cube.Face
}

// Perform performs a ray trace between start and end, checking if any blocks or entities collided with the
// ray. The physics.AABB that's passed is used for checking if any entity within the bounding box collided
// with the ray.
func Perform(start, end mgl64.Vec3, w *world.World, aabb physics.AABB, ignored func(world.Entity) bool) (hit Result, ok bool) {
	// Check if there's any blocks that we may collide with.
	TraverseBlocks(start, end, func(pos cube.Pos) (cont bool) {
		b := w.Block(pos)

		// Check if we collide with the block's model.
		if result, ok := BlockIntercept(pos, w, b, start, end); ok {
			hit = result
			end = hit.Position()
			return false
		}
		return true
	})

	// Now check for any entities that we may collide with.
	dist := math.MaxFloat64
	bb := aabb.Translate(start).Extend(end.Sub(start))
	for _, entity := range w.EntitiesWithin(bb.Grow(8.0), ignored) {
		if ignored != nil && ignored(entity) || !entity.AABB().Translate(entity.Position()).IntersectsWith(bb) {
			continue
		}
		// Check if we collide with the entities bounding box.
		result, ok := EntityIntercept(entity, start, end)
		if !ok {
			continue
		}

		if distance := start.Sub(result.Position()).LenSqr(); distance < dist {
			dist = distance
			hit = result
		}
	}

	return hit, hit != nil
}
