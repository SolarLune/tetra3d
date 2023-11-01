package tetra3d

type SectorDetectionType int

const (
	SectorDetectionTypeVertices = iota // If sector neighbors are detected by sharing vertex positions
	SectorDetectionTypeAABB            // If sector neighbors are detected by AABB
)

// Sector represents an area in a game (a sector), and is used for sector-based rendering.
// When a camera renders a scene with sector-based rendering enabled, the Camera will render
// only the objects within the current sector and any neighboring sectors, up to a customizeable
// depth.
// A Sector is, spatially, an AABB, which sits next to or within other Sectors (AABBs).
// Logically, a Sector is determined to be a neighbor of another Sector if they either intersect,
// or share vertex positions. Which of these is the case depends on the Sectors' SectorDetectionType.
type Sector struct {
	Model               *Model              // The owning Model that forms the Sector
	AABB                *BoundingAABB       // The AABB used to search for neighbors if the SectorDetectionType is set to SectorDetectionTypeAABB
	Neighbors           Set[*Sector]        // The Sector's neighbors
	SectorDetectionType SectorDetectionType // How the Sector is detected
	sectorVisible       bool
}

// NewSector creates a new Sector for the provided Model.
func NewSector(model *Model) *Sector {

	mesh := model.Mesh
	margin := 0.01
	sectorAABB := NewBoundingAABB("sector", mesh.Dimensions.Width()+margin, mesh.Dimensions.Height()+margin, mesh.Dimensions.Depth()+margin)
	sectorAABB.SetLocalPositionVec(model.WorldPosition().Add(mesh.Dimensions.Center()))

	return &Sector{
		Model:     model,
		AABB:      sectorAABB,
		Neighbors: newSet[*Sector](),
	}

}

// Clone clones a Sector.
func (sector *Sector) Clone() *Sector {

	newSector := &Sector{
		Model:     sector.Model,
		AABB:      sector.AABB.Clone().(*BoundingAABB),
		Neighbors: make(Set[*Sector], len(sector.Neighbors)),
	}
	for n := range sector.Neighbors {
		newSector.Neighbors[n] = struct{}{}
	}

	return newSector

}

// UpdateNeighbors updates the Sector's neighbors to refresh it, in case it moves. The Neighbors set is updated, not replaced,
// by this process (so clear the Sector's NeighborSet first if you need to do so).
func (sector *Sector) UpdateNeighbors(otherModels ...*Model) {

	sector.AABB.updateSize()

	for _, otherModel := range otherModels {

		if otherModel == sector.Model || otherModel.sector == nil || sector.Neighbors.Contains(otherModel.sector) {
			continue
		}

		otherModel.sector.AABB.updateSize()

		switch sector.SectorDetectionType {

		case SectorDetectionTypeVertices:

			modelMatrix := sector.Model.Transform()
			otherMatrix := otherModel.Transform()

		exit:

			for _, v := range sector.Model.Mesh.VertexPositions {

				transformedV := modelMatrix.MultVec(v)

				for _, v2 := range otherModel.Mesh.VertexPositions {

					transformedV2 := otherMatrix.MultVec(v2)

					if transformedV.Equals(transformedV2) {

						sector.Neighbors.Add(otherModel.sector)
						otherModel.sector.Neighbors.Add(sector)
						break exit

					}
				}

			}

		case SectorDetectionTypeAABB:

			if sector.AABB.Colliding(otherModel.sector.AABB) {
				sector.Neighbors.Add(otherModel.sector)
				otherModel.sector.Neighbors.Add(sector)
			}

		}

	}

}

// NeighborsWithinRange returns the neighboring Sectors within a certain search range. For example,
// given the following example:
//
// # A - B - C - D - E - F
//
// If you were to check NeighborsWithinRange(2) from E, it would return F, D, and C.
func (sector *Sector) NeighborsWithinRange(searchRange int) Set[*Sector] {

	out := newSet[*Sector]()

	if searchRange > 0 {

		out.Combine(sector.Neighbors)

		for next := range sector.Neighbors {
			out.Combine(next.NeighborsWithinRange(searchRange - 1))
		}

	}

	// The sector itself is not a neighbor of itself
	out.Remove(sector)

	return out

}
