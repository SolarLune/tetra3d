package tetra3d

// NeighborSet represents a set of Sectors that are neighbors for any given Sector.
type NeighborSet map[*Sector]bool

// Contains returns if the given sector is contained within the calling NeighborSet.
func (np NeighborSet) Contains(sector *Sector) bool {
	_, ok := np[sector]
	return ok
}

// Combine combines the given other NeighborSet with the calling set.
func (np NeighborSet) Combine(other NeighborSet) {
	for neighbor := range other {
		np[neighbor] = true
	}
}

// Remove removes the given sector from the NeighborSet.
func (np NeighborSet) Remove(sector *Sector) {
	delete(np, sector)
}

// Clear clears the NeighborSet.
func (np NeighborSet) Clear() {
	for v := range np {
		delete(np, v)
	}
}

// Sector represents an area in a game (a sector), and is used for sector-based rendering.
// When a camera renders a scene with sector-based rendering enabled, the Camera will render
// only the objects within the current sector and any neighboring sectors, up to a customizeable
// depth.
// A Sector is logically an AABB, which sits next to other Sectors (AABBs).
type Sector struct {
	Model         *Model
	AABB          *BoundingAABB
	Neighbors     NeighborSet
	sectorVisible bool
}

func newSector(model *Model) *Sector {

	mesh := model.Mesh
	margin := 0.01
	sectorAABB := NewBoundingAABB("sector", mesh.Dimensions.Width()+margin, mesh.Dimensions.Height()+margin, mesh.Dimensions.Depth()+margin)
	sectorAABB.SetLocalPositionVec(model.WorldPosition().Add(mesh.Dimensions.Center()))

	return &Sector{
		Model:     model,
		AABB:      sectorAABB,
		Neighbors: NeighborSet{},
	}

}

// Clone clones a Sector.
func (sector *Sector) Clone() *Sector {

	newSector := &Sector{
		Model:     sector.Model,
		AABB:      sector.AABB.Clone().(*BoundingAABB),
		Neighbors: make(NeighborSet, len(sector.Neighbors)),
	}
	for n := range sector.Neighbors {
		newSector.Neighbors[n] = true
	}

	return newSector

}

// UpdateNeighbors updates the Sector's neighbors to refresh it, in case it moves. The Neighbors set is updated, not replaced,
// by this process (so clear the Sector's NeighborSet first if you need to do so).
func (sector *Sector) UpdateNeighbors(otherModels ...*Model) {

	for _, otherModel := range otherModels {

		if otherModel == sector.Model {
			continue
		}

		if otherModel.Sector != nil && sector.AABB.Colliding(otherModel.Sector.AABB) {
			sector.Neighbors[otherModel.Sector] = true
			otherModel.Sector.Neighbors[sector] = true
		}

	}

}

// NeighborsWithinRange returns the neighboring Sectors within a certain search range. For example,
// given the following example:
//
// # A - B - C - D - E - F
//
// If you were to check NeighborsWithinRange(2) from E, it would return F, D, and C.
func (sector *Sector) NeighborsWithinRange(searchRange int) NeighborSet {

	out := NeighborSet{}

	if searchRange > 0 {

		out.Combine(sector.Neighbors)

		for next := range sector.Neighbors {
			out.Combine(next.NeighborsWithinRange(searchRange - 1))
		}

	}

	return out

}
