package tetra3d

// sortingTriangle is used specifically for sorting triangles when rendering. Less data means more data fits in cache,
// which means sorting is faster.
type sortingTriangle struct {
	// Triangle *Triangle
	TriangleID    int
	depth         float32
	vertexIndices []int
}

type sortingTriangleBin struct {
	triangles     []sortingTriangle
	triangleIndex int
}

type sortingTriangleBucket struct {
	bins          []sortingTriangleBin
	unsetTris     []sortingTriangle
	unsetTriIndex int
	sortMode      int
}

func newSortingTriangleBucket() *sortingTriangleBucket {
	bucket := &sortingTriangleBucket{}
	return bucket
}

func (s *sortingTriangleBucket) AddTriangle(triID int, depth float32, vertexIndices []int) {
	s.unsetTris[s.unsetTriIndex].TriangleID = triID
	s.unsetTris[s.unsetTriIndex].depth = depth
	s.unsetTris[s.unsetTriIndex].vertexIndices = vertexIndices
	s.unsetTriIndex++
}

func (s *sortingTriangleBucket) Sort(minRange, maxRange float32) {

	binCount := len(s.bins)
	rangeDiff := maxRange - minRange

	if rangeDiff == 0 {
		maxRange += 0.001
		rangeDiff += 0.001
	}

	for i := 0; i < s.unsetTriIndex; i++ {

		targetBin := 0

		if s.sortMode != TriangleSortModeNone && binCount > 1 {
			depth := (s.unsetTris[i].depth - minRange) / rangeDiff * float32(binCount)
			t := clamp(depth, 0, float32(binCount-1))
			targetBin = int(t)
		}

		s.bins[targetBin].triangles[s.bins[targetBin].triangleIndex].depth = s.unsetTris[i].depth
		s.bins[targetBin].triangles[s.bins[targetBin].triangleIndex].TriangleID = s.unsetTris[i].TriangleID
		s.bins[targetBin].triangles[s.bins[targetBin].triangleIndex].vertexIndices = s.unsetTris[i].vertexIndices
		s.bins[targetBin].triangleIndex++

	}

}

func (s *sortingTriangleBucket) Initialize(binCount int) {
	s.bins = make([]sortingTriangleBin, binCount)
	for i := range s.bins {
		s.bins[i].triangles = make([]sortingTriangle, MaxTriangleCount)
	}
	s.unsetTris = make([]sortingTriangle, MaxTriangleCount)
	s.Clear()
}

func (s *sortingTriangleBucket) Clear() {
	for i := 0; i < len(s.bins); i++ {
		s.bins[i].triangleIndex = 0
	}
	s.unsetTriIndex = 0
}

func (s *sortingTriangleBucket) ForEach(forEach func(triIndex, triID int, vertexIndices []int)) {

	if s.IsEmpty() {
		return
	}

	triIndex := 0

	if s.sortMode == TriangleSortModeBackToFront {
		for binIndex := len(s.bins) - 1; binIndex >= 0; binIndex-- {
			bin := s.bins[binIndex]
			for ti := range bin.triangles {
				if ti >= bin.triangleIndex {
					break
				}
				forEach(triIndex, bin.triangles[ti].TriangleID, bin.triangles[ti].vertexIndices)
				triIndex++
			}
		}
	} else {
		for binIndex := 0; binIndex < len(s.bins); binIndex++ {
			bin := s.bins[binIndex]
			for ti := range bin.triangles {
				if ti >= bin.triangleIndex {
					break
				}
				forEach(triIndex, bin.triangles[ti].TriangleID, bin.triangles[ti].vertexIndices)
				triIndex++
			}
		}
	}

}

// func (s *sortingTriangleBucket) ForEach(forEach func(triIndex, triID int)) {

// 	if s.IsEmpty() {
// 		return
// 	}

// 	triIndex := 0

// 	if s.sortMode == TriangleSortModeBackToFront {
// 		for binIndex := len(s.bins) - 1; binIndex >= 0; binIndex-- {
// 			for ti, tri := range s.bins[binIndex].triangles {
// 				// fmt.Println("bin index:", i, len(s.bins))
// 				if ti >= s.bins[binIndex].triangleIndex {
// 					break
// 				}
// 				forEach(triIndex, tri.TriangleID)
// 				triIndex++
// 			}
// 		}
// 	} else {
// 		for i := 0; i < len(s.bins); i++ {
// 			for ti, tri := range s.bins[i].triangles {
// 				if ti >= s.bins[i].triangleIndex {
// 					break
// 				}
// 				forEach(triIndex, tri.TriangleID)
// 				triIndex++
// 			}
// 		}
// 	}

// }

func (s *sortingTriangleBucket) IsEmpty() bool {
	return s.unsetTriIndex == 0
}

var globalSortingTriangleBucket = newSortingTriangleBucket()

func init() {
	globalSortingTriangleBucket.Initialize(512)
}
