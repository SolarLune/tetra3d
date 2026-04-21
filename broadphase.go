package tetra3d

// Broadphase is a utility object specifically created to assist with quickly ruling out triangles
// for collision detection or mesh rendering. This works largely automatically; you should generally
// not have to tweak this too much.
// The general idea is that the broadphase is composed of rectangles in a grid layout, completely covering
// the owning mesh. When you, say, check for a collision against a BoundingTriangles object, it first
// checks the Broadphase object to determine which set(s) of triangles the colliding
// object could be colliding with for finer examination.
type Broadphase struct {
	triSets [][]int // The sets of triangles

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

	ts := make([][]int, 0, len(b.triSets))

	for _, set := range b.triSets {
		newSet := make([]int, 0, len(set))
		newSet = append(newSet, set...)
		ts = append(ts, set)
	}

	newBroadphase.triSets = ts

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

	b.triSets = make([][]int, 0, int(dimDiffX)*int(dimDiffY)*int(dimDiffZ))

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

				b.triSets = append(b.triSets, set)
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

type BroadphaseTestFunction func(tri *Triangle) bool

var broadphaseForEachTriangleDepth = -1

// Performs a function on each triangle within the given Dimensions object.
// The function returns a boolean indicating if any triangle was found in the given bounds.
func (b *Broadphase) ForEachTriangleInDimensions(dim Dimensions, forEach BroadphaseTestFunction) bool {

	// OK, so the concept here is as follows:
	//
	// Overall, this is a fairly straightforward function to get the triangles from a given set of triangle sets that overlap with
	// a given Dimensions object. However, we also need to handle duplicate entries, since triangles can exist in mulitple
	// sets at once (e.g. in a 3x3 broadphase grid, a given triangle may span any number of cells).
	//
	// Previously, I just used a set to ensure the triangle is only added once, and then looped through the set at the end.
	// However, traversing the map was slow. So for an alternative, I'll just set booleans on the Triangles themselves.
	//
	// Triangles can belong to multiple tri sets in a Broadphase grid. We want to avoid checking them more than once,
	// both because it's wasteful if we already checked them once before, and because that would create duplicate
	// results in collision and raycast calls.
	//
	// To get around this, we put a slice of booleans on Triangles that indicate if they've been checked in a broadphase
	// ForEachTriangleInDimensionsSet call.
	// At the beginning of the function call, we increment the depth and set the boolean indicating that the triangle has
	// been checked to false.

	if dim.Size().IsZero() {
		return false
	}

	tris := b.boundingTriangles.Mesh.Triangles

	if !b.enabled || len(b.triSets) == 0 {
		foundResult := false
		for _, t := range tris {
			foundResult = true
			if forEach != nil && !forEach(t) {
				break
			}
		}
		return foundResult
	}

	broadphaseForEachTriangleDepth++

	margin := float32(max(b.cellSizeX, b.cellSizeY, b.cellSizeZ))

	invertedTransform := b.boundingTriangles.Transform().Inverted()

	center := dim.Center()

	// Expand the dimensions to add in some margin
	dim.Max = dim.Max.Sub(center).Expand(margin, 0).Add(center)
	dim.Min = dim.Min.Sub(center).Expand(margin, 0).Add(center)

	// Inverted transform the dimensions so we don't have to transform the cellular dimensions
	dim.Max = invertedTransform.MultVec(dim.Max)
	dim.Min = invertedTransform.MultVec(dim.Min)

	dim = dim.Canon()

	foundOne := false

earlyExit:

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

				if triSetIndex >= 0 && triSetIndex < len(b.triSets) {
					if len(b.triSets[triSetIndex]) > 0 {
						foundOne = true
					}
					if forEach != nil {
						for _, t := range b.triSets[triSetIndex] {
							tri := tris[t]
							if !tri.broadphaseTriCheck[broadphaseForEachTriangleDepth] {
								tris[t].broadphaseTriCheck[broadphaseForEachTriangleDepth] = true
								if !forEach(tris[t]) {
									break earlyExit
								}
							}
						}
					}
				}

			}

		}

	}

	for _, t := range tris {
		t.broadphaseTriCheck[broadphaseForEachTriangleDepth] = false
	}

	broadphaseForEachTriangleDepth--

	return foundOne

}

// Performs a function on each triangle within the bounds of the given BoundingObject.
// The function returns a boolean indicating if any triangle was found in the given bounds.
func (b *Broadphase) ForEachTriangleFromBoundingObject(boundingObject IBoundingObject, forEach func(tri *Triangle) bool) bool {
	return b.ForEachTriangleInDimensions(boundingObject.Dimensions(), forEach)
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
	// TODO: Remove this, it's not necessary.

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
