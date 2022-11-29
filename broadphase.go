package tetra3d

import (
	"github.com/kvartborg/vector"
)

// Broadphase is a utility object specifically created to assist with quickly ruling out
// triangles when doing Triangle-Sphere/Capsule/AABB collision detection. This works
// largely automatically; you should generally not have to tweak this too much. The general
// idea is that the broadphase is composed of AABBs in a grid layout, completely covering
// the owning BoundingTriangles instance. When you check for a collision against the
// BoundingTriangles instance, it first checks to see if the colliding object is within the
// BoundingTriangles' overall AABB bounding box. If so, it proceeds to the broadphase check,
// where it sees which set(s) of triangles the colliding object could be colliding with, and
// then returns that set for finer examination.
type Broadphase struct {
	GridSize          int
	cellSize          float64
	Center            *Node
	aabb              *BoundingAABB
	TriSets           [][][][]uint16
	BoundingTriangles *BoundingTriangles
}

// NewBroadphase returns a new Broadphase instance.
func NewBroadphase(gridSize int, bt *BoundingTriangles) *Broadphase {
	bp := &Broadphase{
		BoundingTriangles: bt,
		Center:            NewNode("broadphase center"),
		aabb:              NewBoundingAABB("bounding aabb", 1, 1, 1),
	}

	bp.Center.AddChildren(bp.aabb)

	bp.Center.SetWorldPositionVec(bt.WorldPosition().Add(bt.Mesh.Dimensions.Center()))
	bp.Center.Transform() // Update the transform so that when resizing, the aabb can properly check all triangles

	bp.Resize(gridSize)
	return bp
}

// Clone clones the Broadphase.
func (bp *Broadphase) Clone() *Broadphase {
	newBroadphase := NewBroadphase(0, bp.BoundingTriangles)
	newBroadphase.Center = bp.Center.Clone().(*Node)
	newBroadphase.aabb = newBroadphase.Center.children[0].(*BoundingAABB)
	newBroadphase.GridSize = bp.GridSize
	newBroadphase.cellSize = bp.cellSize

	ts := make([][][][]uint16, len(bp.TriSets))

	for i := range bp.TriSets {
		ts[i] = make([][][]uint16, len(bp.TriSets[i]))
		for j := range bp.TriSets[i] {
			ts[i][j] = make([][]uint16, len(bp.TriSets[i][j]))
			for k := range bp.TriSets[i][j] {
				ts[i][j][k] = make([]uint16, len(bp.TriSets[i][j][k]))
				copy(ts[i][j][k], bp.TriSets[i][j][k])
			}
		}
	}

	newBroadphase.TriSets = ts

	return newBroadphase
}

// Resize resizes the Broadphase struct, using the new gridSize for how big in units each cell in the grid should be.
func (bp *Broadphase) Resize(gridSize int) {
	bp.GridSize = gridSize

	if gridSize <= 0 {
		return
	}

	maxDim := bp.BoundingTriangles.Mesh.Dimensions.MaxDimension()
	cellSize := (maxDim / float64(gridSize)) + 1 // The +1 adds a little extra room
	bp.cellSize = cellSize

	bp.aabb.internalSize[0] = cellSize
	bp.aabb.internalSize[1] = cellSize
	bp.aabb.internalSize[2] = cellSize
	bp.aabb.updateSize()

	bp.TriSets = make([][][][]uint16, gridSize)

	hg := float64(bp.GridSize) / 2

	for i := 0; i < gridSize; i++ {
		bp.TriSets[i] = make([][][]uint16, gridSize)
		for j := 0; j < gridSize; j++ {
			bp.TriSets[i][j] = make([][]uint16, gridSize)
			for k := 0; k < gridSize; k++ {

				bp.TriSets[i][j][k] = []uint16{}

				bp.aabb.SetLocalPositionVec(vector.Vector{
					(float64(i) - hg + 0.5) * bp.cellSize,
					(float64(j) - hg + 0.5) * bp.cellSize,
					(float64(k) - hg + 0.5) * bp.cellSize,
				})

				center := bp.aabb.WorldPosition()

				for _, tri := range bp.BoundingTriangles.Mesh.Triangles {

					v0 := bp.BoundingTriangles.Mesh.VertexPositions[tri.VertexIndices[0]]
					v1 := bp.BoundingTriangles.Mesh.VertexPositions[tri.VertexIndices[1]]
					v2 := bp.BoundingTriangles.Mesh.VertexPositions[tri.VertexIndices[2]]
					closestOnTri := closestPointOnTri(center, v0, v1, v2)
					if bp.aabb.PointInside(closestOnTri) {
						bp.TriSets[i][j][k] = append(bp.TriSets[i][j][k], tri.ID)
					}

				}

			}

		}

	}

}

// TrianglesFromBounding returns a set (a map[int]bool) of triangle IDs, based on where the BoundingObject is
// in relation to the Broadphase owning BoundingTriangles instance. The returned set contains each triangle only
// once, of course.
func (bp *Broadphase) TrianglesFromBounding(boundingObject IBoundingObject) map[uint16]bool {

	if bp.GridSize <= 0 {
		trianglesSet := make(map[uint16]bool, len(bp.BoundingTriangles.Mesh.Triangles))
		for _, tri := range bp.BoundingTriangles.Mesh.Triangles {
			trianglesSet[tri.ID] = true
		}
		return trianglesSet
	}

	trianglesSet := make(map[uint16]bool, len(bp.TriSets))

	hg := float64(bp.GridSize) / 2

	// We're brute forcing it here; this isn't ideal, but it works alright for now
	for i := 0; i < bp.GridSize; i++ {
		for j := 0; j < bp.GridSize; j++ {
			for k := 0; k < bp.GridSize; k++ {

				// We can skip empty sets
				if len(bp.TriSets[i][j][k]) == 0 {
					continue
				}

				bp.aabb.SetLocalPositionVec(vector.Vector{
					(float64(i) - hg + 0.5) * bp.cellSize,
					(float64(j) - hg + 0.5) * bp.cellSize,
					(float64(k) - hg + 0.5) * bp.cellSize,
				})

				if bp.aabb.Colliding(boundingObject) {
					// triangles = append(triangles, bp.TriSets[i][j][k]...)
					for _, triID := range bp.TriSets[i][j][k] {
						trianglesSet[triID] = true
					}
				}
			}
		}
	}

	return trianglesSet

}

func (bp *Broadphase) allAABBPositions() []*BoundingAABB {

	aabbs := []*BoundingAABB{}

	if bp.GridSize <= 0 || bp.cellSize <= 0 {
		return aabbs
	}

	hg := float64(bp.GridSize) / 2

	bp.aabb.SetLocalPositionVec(vector.Vector{0, 0, 0})

	for i := 0; i < bp.GridSize; i++ {
		for j := 0; j < bp.GridSize; j++ {
			for k := 0; k < bp.GridSize; k++ {

				clone := bp.aabb.Clone().(*BoundingAABB)

				clone.SetLocalPositionVec(vector.Vector{
					// (-float64(bp.GridSize)/2+float64(i))*bp.CellSize + (bp.CellSize / 2),
					// (-float64(bp.GridSize)/2+float64(j))*bp.CellSize + (bp.CellSize / 2),
					// (-float64(bp.GridSize)/2+float64(k))*bp.CellSize + (bp.CellSize / 2),
					(float64(i) - hg + 0.5) * bp.cellSize,
					(float64(j) - hg + 0.5) * bp.cellSize,
					(float64(k) - hg + 0.5) * bp.cellSize,
					// 0, 0, 0,
				})

				clone.SetWorldTransform(bp.aabb.Transform().Mult(clone.Transform()))

				clone.Transform()

				aabbs = append(aabbs, clone)

			}
		}
	}

	return aabbs
}

// package tetra3d

// import (
// 	"github.com/kvartborg/vector"
// )

// type Broadphase struct {
// 	GridSize          int
// 	CellSize          float64
// 	Limits            Dimensions
// 	Center            *Node
// 	AABBs             [][][]*BoundingAABB
// 	allAABBs          []*BoundingAABB
// 	TriSets           [][][][]*Triangle
// 	BoundingTriangles *BoundingTriangles
// }

// func NewBroadphase(gridSize int, cellSize float64, bt *BoundingTriangles) *Broadphase {
// 	bp := &Broadphase{
// 		BoundingTriangles: bt,
// 		Center:            NewNode("broadphase center"),
// 	}
// 	bp.Resize(gridSize, cellSize)
// 	return bp
// }

// func (bp *Broadphase) Resize(gridSize int, cellSize float64) {
// 	bp.GridSize = gridSize
// 	bp.CellSize = cellSize
// 	bp.AABBs = make([][][]*BoundingAABB, gridSize)
// 	bp.TriSets = make([][][][]*Triangle, gridSize)

// 	bp.allAABBs = make([]*BoundingAABB, 0, gridSize*gridSize*gridSize)

// 	for i := range bp.AABBs {
// 		bp.AABBs[i] = make([][]*BoundingAABB, gridSize)
// 		bp.TriSets[i] = make([][][]*Triangle, gridSize)
// 		for j := 0; j < gridSize; j++ {
// 			bp.AABBs[i][j] = make([]*BoundingAABB, gridSize)
// 			bp.TriSets[i][j] = make([][]*Triangle, gridSize)
// 			for k := 0; k < gridSize; k++ {
// 				b := NewBoundingAABB("", cellSize, cellSize, cellSize)
// 				bp.Center.AddChildren(b)
// 				b.SetLocalPositionVec(vector.Vector{
// 					(-float64(gridSize)/2+float64(i))*cellSize + (cellSize / 2),
// 					(-float64(gridSize)/2+float64(j))*cellSize + (cellSize / 2),
// 					(-float64(gridSize)/2+float64(k))*cellSize + (cellSize / 2),
// 				})
// 				bp.allAABBs = append(bp.allAABBs, b)
// 				bp.AABBs[i][j][k] = b
// 				bp.TriSets[i][j][k] = []*Triangle{}
// 			}
// 		}
// 	}

// 	bp.Limits = Dimensions{
// 		{-float64(gridSize) * cellSize, -float64(gridSize) * cellSize, -float64(gridSize) * cellSize},
// 		{float64(gridSize) * cellSize, float64(gridSize) * cellSize, float64(gridSize) * cellSize},
// 	}

// 	for _, tri := range bp.BoundingTriangles.Mesh.Triangles {
// 		for i := range bp.AABBs {
// 			for j := range bp.AABBs[i] {
// 				for k := range bp.AABBs[i][j] {
// 					v0 := bp.BoundingTriangles.Mesh.VertexPositions[tri.ID*3]
// 					v1 := bp.BoundingTriangles.Mesh.VertexPositions[tri.ID*3+1]
// 					v2 := bp.BoundingTriangles.Mesh.VertexPositions[tri.ID*3+2]
// 					closestOnAABB := bp.AABBs[i][j][k].ClosestPoint(tri.Center)
// 					closestOnTri := closestPointOnTri(closestOnAABB, v0, v1, v2)
// 					// if bp.AABBs[i][j][k].PointInside(v0) || bp.AABBs[i][j][k].PointInside(v1) || bp.AABBs[i][j][k].PointInside(v2) {
// 					if bp.AABBs[i][j][k].PointInside(closestOnTri) {
// 						bp.TriSets[i][j][k] = append(bp.TriSets[i][j][k], tri)
// 					}
// 				}
// 			}
// 		}
// 	}

// }

// func (bp *Broadphase) GetTrianglesFromBounding(boundingObject BoundingObject) map[*Triangle]bool {

// 	// triangles := make([]*Triangle, 0, len(bp.TriSets))
// 	triangles := make(map[*Triangle]bool, len(bp.TriSets))

// 	// We're brute forcing it here; this isn't ideal, but it works alright for now
// 	for i := range bp.AABBs {
// 		for j := range bp.AABBs[i] {
// 			for k := range bp.AABBs[i][j] {
// 				if bp.AABBs[i][j][k].Colliding(boundingObject) {
// 					// triangles = append(triangles, bp.TriSets[i][j][k]...)
// 					for _, tri := range bp.TriSets[i][j][k] {
// 						triangles[tri] = true
// 					}
// 				}
// 			}
// 		}
// 	}

// 	return triangles

// }
