package tetra3d

import (
	"github.com/kvartborg/vector"
)

type Broadphase struct {
	GridSize          int
	CellSize          float64
	Limits            Dimensions
	Center            *Node
	aabb              *BoundingAABB
	TriSets           [][][][]int
	BoundingTriangles *BoundingTriangles
}

func NewBroadphase(gridSize int, cellSize float64, bt *BoundingTriangles) *Broadphase {
	bp := &Broadphase{
		BoundingTriangles: bt,
		Center:            NewNode("broadphase center"),
		aabb:              NewBoundingAABB("bounding aabb", cellSize, cellSize, cellSize),
	}
	bp.Center.AddChildren(bp.aabb)
	bp.Resize(gridSize, cellSize)
	return bp
}

func (bp *Broadphase) Resize(gridSize int, cellSize float64) {
	bp.GridSize = gridSize
	bp.CellSize = cellSize

	if gridSize <= 0 || cellSize <= 0 {
		return
	}

	bp.TriSets = make([][][][]int, gridSize)

	bp.Limits = Dimensions{
		{-float64(gridSize) * cellSize, -float64(gridSize) * cellSize, -float64(gridSize) * cellSize},
		{float64(gridSize) * cellSize, float64(gridSize) * cellSize, float64(gridSize) * cellSize},
	}

	for i := 0; i < gridSize; i++ {
		bp.TriSets[i] = make([][][]int, gridSize)
		for j := 0; j < gridSize; j++ {
			bp.TriSets[i][j] = make([][]int, gridSize)
			for k := 0; k < gridSize; k++ {

				bp.TriSets[i][j][k] = []int{}

				bp.aabb.SetLocalPosition(vector.Vector{
					(-float64(gridSize)/2+float64(i))*cellSize + (cellSize / 2),
					(-float64(gridSize)/2+float64(j))*cellSize + (cellSize / 2),
					(-float64(gridSize)/2+float64(k))*cellSize + (cellSize / 2),
				})

				for _, tri := range bp.BoundingTriangles.Mesh.Triangles {
					v0 := bp.BoundingTriangles.Mesh.VertexPositions[tri.ID*3]
					v1 := bp.BoundingTriangles.Mesh.VertexPositions[tri.ID*3+1]
					v2 := bp.BoundingTriangles.Mesh.VertexPositions[tri.ID*3+2]
					closestOnAABB := bp.aabb.ClosestPoint(tri.Center)
					closestOnTri := closestPointOnTri(closestOnAABB, v0, v1, v2)
					// if bp.AABBs[i][j][k].PointInside(v0) || bp.AABBs[i][j][k].PointInside(v1) || bp.AABBs[i][j][k].PointInside(v2) {
					if bp.aabb.PointInside(closestOnTri) {
						bp.TriSets[i][j][k] = append(bp.TriSets[i][j][k], tri.ID)
					}
				}
			}
		}
	}

}

func (bp *Broadphase) GetTrianglesFromBounding(boundingObject BoundingObject) []*Triangle {

	if bp.CellSize <= 0 || bp.GridSize <= 0 {
		return bp.BoundingTriangles.Mesh.Triangles
	}

	trianglesSet := make(map[*Triangle]bool, len(bp.TriSets))

	// We're brute forcing it here; this isn't ideal, but it works alright for now
	for i := 0; i < bp.GridSize; i++ {
		for j := 0; j < bp.GridSize; j++ {
			for k := 0; k < bp.GridSize; k++ {

				bp.aabb.SetLocalPosition(vector.Vector{
					(-float64(bp.GridSize)/2+float64(i))*bp.CellSize + (bp.CellSize / 2),
					(-float64(bp.GridSize)/2+float64(j))*bp.CellSize + (bp.CellSize / 2),
					(-float64(bp.GridSize)/2+float64(k))*bp.CellSize + (bp.CellSize / 2),
				})

				if bp.aabb.Colliding(boundingObject) {
					// triangles = append(triangles, bp.TriSets[i][j][k]...)
					for _, triID := range bp.TriSets[i][j][k] {
						trianglesSet[bp.BoundingTriangles.Mesh.Triangles[triID]] = true
					}
				}
			}
		}
	}

	tris := make([]*Triangle, len(trianglesSet))

	i := 0
	for t := range trianglesSet {
		tris[i] = t
		i++
	}

	return tris

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
// 				b.SetLocalPosition(vector.Vector{
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
