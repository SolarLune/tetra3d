package tetra3d

// Broadphase is a utility object specifically created to assist with quickly ruling out triangles
// for collision detection or mesh rendering. This works largely automatically; you should generally
// not have to tweak this too much.
// The general idea is that the broadphase is composed of AABBs in a grid layout, completely covering
// the owning mesh. When you, say, check for a collision against a BoundingTriangles object, it first
// checks to see if the colliding object is within the BoundingTriangles' overall AABB bounding box.
// If so, it proceeds to the broadphase check, where it sees which set(s) of triangles the colliding
// object could be colliding with, and then returns that set for finer examination.
type Broadphase struct {
	GridCellCount int           // The number of cells in the broadphase grid
	cellSize      float32       // The size of each cell in the grid
	center        *Node         // The center node of the broadphase
	workingAABB   *BoundingAABB // The working-process AABB used to determine which triangles belong in which cells
	TriSets       [][][][]int   // The sets of triangles
	allTriSet     Set[int]      // A set containing all triangles
	mesh          *Mesh         // The mesh used for the Broadphase object
}

// NewBroadphase returns a new Broadphase object.
// gridCellCount is number of cells in the grid.
// worldPosition is the world position of the broadphase object; this should be a zero vector if
// it's being used for a mesh, and should be the position of the BoundingTriangles if it's being used for
// collision testing.
// mesh is the Mesh to be used for broadphase testing.
func NewBroadphase(gridCellCount int, worldPosition Vector3, mesh *Mesh) *Broadphase {

	b := &Broadphase{
		mesh:        mesh,
		center:      NewNode("broadphase center"),
		workingAABB: NewBoundingAABB("bounding aabb", 1, 1, 1),
	}

	b.center.AddChildren(b.workingAABB)
	b.center.SetWorldPositionVec(worldPosition.Add(mesh.Dimensions.Center()))
	b.center.Transform() // Update the transform so that when resizing, the aabb can properly check all triangles
	b.Resize(gridCellCount)

	return b
}

// Clone clones the Broadphase.
func (b *Broadphase) Clone() *Broadphase {
	newBroadphase := NewBroadphase(0, b.center.WorldPosition(), b.mesh)
	newBroadphase.center = b.center.Clone().(*Node)
	newBroadphase.workingAABB = newBroadphase.center.children[0].(*BoundingAABB)
	newBroadphase.GridCellCount = b.GridCellCount
	newBroadphase.cellSize = b.cellSize

	ts := make([][][][]int, len(b.TriSets))

	for i := range b.TriSets {
		ts[i] = make([][][]int, len(b.TriSets[i]))
		for j := range b.TriSets[i] {
			ts[i][j] = make([][]int, len(b.TriSets[i][j]))
			for k := range b.TriSets[i][j] {
				ts[i][j][k] = make([]int, len(b.TriSets[i][j][k]))
				copy(ts[i][j][k], b.TriSets[i][j][k])
			}
		}
	}

	newBroadphase.TriSets = ts
	newBroadphase.allTriSet = b.allTriSet.Clone()

	return newBroadphase
}

// Resize resizes the Broadphase struct, using the new gridSize for how big in units each cell in the grid should be.
func (b *Broadphase) Resize(gridSize int) {
	b.GridCellCount = gridSize

	if gridSize <= 0 {
		return
	}

	maxDim := b.mesh.Dimensions.MaxDimension()
	cellSize := (maxDim / float32(gridSize)) + 1 // The +1 adds a little extra room
	b.cellSize = cellSize

	b.workingAABB.internalSize.X = cellSize
	b.workingAABB.internalSize.Y = cellSize
	b.workingAABB.internalSize.Z = cellSize
	b.workingAABB.updateSize()

	b.TriSets = make([][][][]int, gridSize)
	b.allTriSet = newSet[int]()

	hg := float32(b.GridCellCount) / 2

	for i := 0; i < gridSize; i++ {
		b.TriSets[i] = make([][][]int, gridSize)
		for j := 0; j < gridSize; j++ {
			b.TriSets[i][j] = make([][]int, gridSize)
			for k := 0; k < gridSize; k++ {

				b.TriSets[i][j][k] = []int{}

				b.workingAABB.SetLocalPositionVec(Vector3{
					(float32(i) - hg + 0.5) * b.cellSize,
					(float32(j) - hg + 0.5) * b.cellSize,
					(float32(k) - hg + 0.5) * b.cellSize,
				})

				for triID, tri := range b.mesh.Triangles {

					// Triangles should not be able to be in multiple bins. This allows us
					// to ensure checking triangles only if they could conceivably be struck by a ray,
					// for example, without needing to use sets to ensure we check each triangle once.
					if b.workingAABB.PointInside(tri.Center) {
						b.TriSets[i][j][k] = append(b.TriSets[i][j][k], triID)
					}
					b.allTriSet.Add(triID)

				}

			}

		}

	}

}

// Mesh returns the mesh that is associated with the Broadphase object.
func (b *Broadphase) Mesh() *Mesh {
	return b.mesh
}

// TrianglesFromBoundingObject returns a set of triangle IDs, based on where the BoundingObject is
// in relation to the Broadphase owning BoundingTriangles instance. The returned set contains each triangle only
// once, of course.
func (b *Broadphase) ForEachTriangleFromBoundingObject(boundingObject IBoundingObject, forEach func(triID int) bool) {

	// Previously, this used sets; this no longer does for optimization's sake.
	// This works because triangles used to be able to be in multiple sets; this should no longer be the case.
	if b.GridCellCount <= 1 {
		for t := range b.allTriSet {
			if !forEach(t) {
				return
			}
		}
	}

	margin := float32(1.2)
	b.workingAABB.internalSize.X = b.cellSize * margin
	b.workingAABB.internalSize.Y = b.cellSize * margin
	b.workingAABB.internalSize.Z = b.cellSize * margin
	b.workingAABB.updateSize()

	hg := float32(b.GridCellCount) / 2

	// We're brute forcing it here; this isn't ideal, but it works alright for now
	for i := 0; i < b.GridCellCount; i++ {
		for j := 0; j < b.GridCellCount; j++ {
			for k := 0; k < b.GridCellCount; k++ {

				// We can skip empty sets
				if len(b.TriSets[i][j][k]) == 0 {
					continue
				}

				b.workingAABB.SetLocalPositionVec(Vector3{
					(float32(i) - hg + 0.5) * b.cellSize,
					(float32(j) - hg + 0.5) * b.cellSize,
					(float32(k) - hg + 0.5) * b.cellSize,
				})

				if b.workingAABB.Colliding(boundingObject) {
					for _, triID := range b.TriSets[i][j][k] {
						if !forEach(triID) {
							break
						}
					}
				}
			}
		}
	}

}

func (b *Broadphase) ForEachTriangleInRange(pos Vector3, radius float32, forEach func(triID int) bool) {

	if b.GridCellCount <= 1 {
		for t := range b.allTriSet {
			if !forEach(t) {
				return
			}
		}
	}

	hg := float32(b.GridCellCount) / 2

	transform := b.center.Transform()

	rr2 := (radius + b.cellSize) * (radius + b.cellSize)

	for i := 0; i < b.GridCellCount; i++ {
		for j := 0; j < b.GridCellCount; j++ {
			for k := 0; k < b.GridCellCount; k++ {

				// We can skip empty sets
				if len(b.TriSets[i][j][k]) == 0 {
					continue
				}

				vp := transform.MultVec(Vector3{
					(float32(i) - hg + 0.5) * b.cellSize,
					(float32(j) - hg + 0.5) * b.cellSize,
					(float32(k) - hg + 0.5) * b.cellSize,
				})

				if vp.DistanceSquaredTo(pos) <= rr2 {
					for _, triID := range b.TriSets[i][j][k] {
						if !forEach(triID) {
							return
						}
					}
				}
			}
		}
	}

}

func (b *Broadphase) allAABBPositions() []*BoundingAABB {

	aabbs := []*BoundingAABB{}

	if b.GridCellCount <= 0 || b.cellSize <= 0 {
		return aabbs
	}

	hg := float32(b.GridCellCount) / 2

	b.workingAABB.SetLocalPositionVec(Vector3{})

	for i := 0; i < b.GridCellCount; i++ {
		for j := 0; j < b.GridCellCount; j++ {
			for k := 0; k < b.GridCellCount; k++ {

				clone := b.workingAABB.Clone().(*BoundingAABB)

				clone.SetLocalPositionVec(Vector3{
					(float32(i) - hg + 0.5) * b.cellSize,
					(float32(j) - hg + 0.5) * b.cellSize,
					(float32(k) - hg + 0.5) * b.cellSize,
				})

				clone.SetWorldTransform(b.workingAABB.Transform().Mult(clone.Transform()))

				clone.Transform()

				aabbs = append(aabbs, clone)

			}
		}
	}

	return aabbs
}
