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
	TriSets [][]int // The sets of triangles

	boundingTriangles *BoundingTriangles
	dimensions        Dimensions
	enabled           bool

	cellSizeX float32
	cellSizeY float32
	cellSizeZ float32

	cellCountX int
	cellCountY int
	cellCountZ int
}

// NewBroadphase returns a new Broadphase object that performs broadphase triangle detection for a Mesh.
func NewBroadphase(boundingTris *BoundingTriangles, cellSizeX, cellSizeY, cellSizeZ float32) *Broadphase {

	b := &Broadphase{

		boundingTriangles: boundingTris,
		dimensions:        boundingTris.Mesh.Dimensions, // Specifically the untransformed dimensions

		cellSizeX: cellSizeX,
		cellSizeY: cellSizeY,
		cellSizeZ: cellSizeZ,

		enabled: true,
	}

	if cellSizeX > 0 && cellSizeY > 0 && cellSizeZ > 0 {
		b.Resize(cellSizeX, cellSizeY, cellSizeZ)
	}

	return b
}

// Clone clones the Broadphase.
func (b *Broadphase) Clone(newBoundingTriangles *BoundingTriangles) *Broadphase {

	// Create a new broadphase with a cell size of 0 so it specifically does NOT resize again
	newBroadphase := NewBroadphase(newBoundingTriangles, 0, 0, 0)

	ts := make([][]int, 0, len(b.TriSets))

	for _, set := range b.TriSets {
		newSet := make([]int, 0, len(set))
		newSet = append(newSet, set...)
		ts = append(ts, set)
	}

	newBroadphase.TriSets = ts

	newBroadphase.enabled = b.enabled
	newBroadphase.dimensions = b.dimensions

	newBroadphase.cellSizeX = b.cellSizeX
	newBroadphase.cellSizeY = b.cellSizeY
	newBroadphase.cellSizeZ = b.cellSizeZ

	newBroadphase.cellCountX = b.cellCountX
	newBroadphase.cellCountY = b.cellCountY
	newBroadphase.cellCountZ = b.cellCountZ

	return newBroadphase
}

// Resize resizes the Broadphase struct, using the new gridSize for how big in units each cell in the grid should be.
func (b *Broadphase) Resize(cellSizeX, cellSizeY, cellSizeZ float32) {

	if cellSizeX <= 0 {
		cellSizeX = 1
	}

	if cellSizeY <= 0 {
		cellSizeY = 1
	}

	if cellSizeZ <= 0 {
		cellSizeZ = 1
	}

	// maxDim := b.mesh.Dimensions.MaxDimension()

	cellCountX := 0
	cellCountY := 0
	cellCountZ := 0

	mesh := b.boundingTriangles.Mesh

	cellCountZ = 0

	dimMin := b.dimensions.Min
	dimMax := b.dimensions.Max

	if dimMax.X-dimMin.X < cellSizeX {
		dimMax.X = dimMin.X + cellSizeX
	}

	if dimMax.Y-dimMin.Y < cellSizeY {
		dimMax.Y = dimMin.Y + cellSizeY
	}

	if dimMax.Z-dimMin.Z < cellSizeZ {
		dimMax.Z = dimMin.Z + cellSizeZ
	}
	margin := max(cellSizeX, cellSizeY, cellSizeZ)

	dimDiffZ := b.dimensions.Depth() / cellSizeZ
	dimDiffY := b.dimensions.Height() / cellSizeY
	dimDiffX := b.dimensions.Width() / cellSizeX

	b.TriSets = make([][]int, 0, int(dimDiffX)*int(dimDiffY)*int(dimDiffZ))

	for z := dimMin.Z; z < dimMax.Z; z += cellSizeZ {
		cellCountY = 0

		for y := dimMin.Y; y < dimMax.Y; y += cellSizeY {
			cellCountX = 0

			for x := dimMin.X; x < dimMax.X; x += cellSizeX {
				nd := Dimensions{
					Min: NewVector3(x-margin, y-margin, z-margin),
					Max: NewVector3(x+cellSizeX+margin, y+cellSizeY+margin, z+cellSizeZ+margin),
				}

				set := []int{}

				for triIndex, tri := range b.boundingTriangles.Mesh.Triangles {

					point := closestPointOnTri(
						NewVector3(x, y, z),
						mesh.VertexPositions[tri.VertexIndexA],
						mesh.VertexPositions[tri.VertexIndexB],
						mesh.VertexPositions[tri.VertexIndexC],
					)

					if nd.Contains(point) {
						set = append(set, triIndex)
					}

				}

				b.TriSets = append(b.TriSets, set)
				cellCountX++
			}

			cellCountY++

		}

		cellCountZ++

	}

	b.cellCountX = cellCountX
	b.cellCountY = cellCountY
	b.cellCountZ = cellCountZ

}

func (b *Broadphase) convertToXYZIndices(wx, wy, wz float32) (x, y, z int) {

	x = int((wx - b.dimensions.Min.X) / b.cellSizeX)
	y = int((wy - b.dimensions.Min.Y) / b.cellSizeY)
	z = int((wz - b.dimensions.Min.Z) / b.cellSizeZ)

	return
}

func (b *Broadphase) convertToXYZIndicesVec(position Vector3) (x, y, z int) {
	return b.convertToXYZIndices(position.X, position.Y, position.Z)
}

func (b *Broadphase) xyzIndicesInsideVolume(x, y, z int) bool {
	return x >= 0 && y >= 0 && z >= 0 && x < b.cellCountX && y < b.cellCountY && z < b.cellCountZ
}

func (b *Broadphase) convertToWorldPosition(x, y, z int) Vector3 {

	return NewVector3(
		(float32(x)*b.cellSizeX)+b.dimensions.Min.X,
		(float32(y)*b.cellSizeY)+b.dimensions.Min.Y,
		(float32(z)*b.cellSizeZ)+b.dimensions.Min.Z,
	)

}

// Mesh returns the mesh that is associated with the Broadphase object.
func (b *Broadphase) Mesh() *Mesh {
	return b.boundingTriangles.Mesh
}

var broadphaseTestingTriIndices = Set[int]{}

// TrianglesFromBoundingObject returns a set of triangle IDs, based on where the BoundingObject is
// in relation to the Broadphase owning BoundingTriangles instance. The returned set contains each triangle only
// once, of course.
func (b *Broadphase) ForEachTriangleFromBoundingObject(boundingObject IBoundingObject, forEach func(triID int) bool) {

	if !b.enabled || len(b.TriSets) == 0 {
		for t := range b.boundingTriangles.Mesh.Triangles {
			if !forEach(t) {
				return
			}
		}
	}

	broadphaseTestingTriIndices.Clear()

	margin := float32(max(b.cellSizeX, b.cellSizeY, b.cellSizeZ))

	invertedTransform := b.boundingTriangles.Transform().Inverted()

	dim := boundingObject.Dimensions()

	center := dim.Center()

	// Expand the dimensions to add in some margin
	dim.Max = dim.Max.Sub(center).Expand(margin, 0).Add(center)
	dim.Min = dim.Min.Sub(center).Expand(margin, 0).Add(center)

	// Inverted transform the dimensions so we don't have to transform the cellular dimensions
	dim.Max = invertedTransform.MultVec(dim.Max)
	dim.Min = invertedTransform.MultVec(dim.Min)

	dim = dim.Canon()

	for z := 0; z < b.cellCountZ; z++ {
		for y := 0; y < b.cellCountY; y++ {
			for x := 0; x < b.cellCountX; x++ {

				pos := b.convertToWorldPosition(x, y, z)

				nd := Dimensions{
					Min: NewVector3(
						pos.X-margin,
						pos.Y-margin,
						pos.Z-margin,
					),
					Max: NewVector3(
						pos.X+b.cellSizeX+margin,
						pos.Y+b.cellSizeY+margin,
						pos.Z+b.cellSizeZ+margin,
					),
				}

				if !nd.Intersects(dim) {
					continue
				}

				triSetIndex := x + (y * b.cellCountX) + (z * b.cellCountY * b.cellCountX)

				if triSetIndex >= 0 && triSetIndex < len(b.TriSets) {
					for _, t := range b.TriSets[triSetIndex] {
						broadphaseTestingTriIndices.Add(t)
					}
				}

			}

		}

	}

	broadphaseTestingTriIndices.ForEach(func(element int) bool {
		return forEach(element)
	})

}

// func (b *Broadphase) ForEachTriangleInRange(pos Vector3, checkRadius float32, forEach func(triID int) bool) {

// 	// Previously, this used sets; this no longer does for optimization's sake.
// 	// This works because triangles used to be able to be in multiple sets; this should no longer be the case.
// 	if !b.enabled {
// 		for t := range b.boundingTriangles.Mesh.Triangles {
// 			if !forEach(t) {
// 				return
// 			}
// 		}
// 	}

// 	checkRadius *= checkRadius

// 	// We're brute forcing it here; this isn't ideal, but it works alright for now
// 	for z := 0; z < b.cellCountZ; z++ {
// 		for y := 0; y < b.cellCountY; y++ {
// 			for x := 0; x < b.cellCountX; x++ {

// 				// We can skip empty sets
// 				// if len(b.TriSets[z][y][x]) == 0 {
// 				// 	continue
// 				// }

// 				// cellPos := b.boundingTriangles.Transform().MultVec(b.convertToWorldPosition(x, y, z))

// 				// if cellPos.DistanceSquaredTo(pos) > checkRadius {
// 				// 	continue
// 				// }

// 				// for _, triID := range b.TriSets[z][y][x] {
// 				// 	if !forEach(triID) {
// 				// 		break
// 				// 	}
// 				// }

// 			}

// 		}

// 	}

// }

func (b *Broadphase) allAABBPositions() []*BoundingAABB {

	aabbs := []*BoundingAABB{}

	dimMin := b.dimensions.Min
	dimMax := b.dimensions.Max

	if dimMax.X-dimMin.X < b.cellSizeX {
		dimMax.X = dimMin.X + b.cellSizeX
	}

	if dimMax.Y-dimMin.Y < b.cellSizeY {
		dimMax.Y = dimMin.Y + b.cellSizeY
	}

	if dimMax.Z-dimMin.Z < b.cellSizeZ {
		dimMax.Z = dimMin.Z + b.cellSizeZ
	}

	for z := dimMin.Z; z < dimMax.Z; z += b.cellSizeZ {

		for y := dimMin.Y; y < dimMax.Y; y += b.cellSizeY {

			for x := dimMin.X; x < dimMax.X; x += b.cellSizeX {
				aabb := NewBoundingAABB("", b.cellSizeX, b.cellSizeY, b.cellSizeZ)
				aabb.SetWorldPosition(x, y, z)
				aabbs = append(aabbs, aabb)
			}

		}

	}

	return aabbs
}
