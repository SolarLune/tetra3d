package tetra3d

import (
	"github.com/solarlune/tetra3d/math32"
)

// sortingTriangle is used specifically for sorting triangles when rendering. Less data means more data fits in cache,
// which means sorting is faster.
type sortingTriangle struct {
	Triangle *Triangle
	depth    float32
}

// sortingTriangleBucket sorts the triangles of a mesh part into depth bins
// with a stable counting sort, so that the triangles of one bin keep the order
// in which they were added.
type sortingTriangleBucket struct {
	unsetTriIndex int
	sortMode      int
	binCount      int
	unsetTris     []sortingTriangle // The triangles in the order of AddTriangle.
	binOf         []int32           // The bin of each triangle in unsetTris.
	binStarts     []int             // The count, then the next place, of each bin.
	sortedTris    []sortingTriangle // The buffer for sorted.

	// sorted holds the triangles in draw order after Sort.
	sorted []sortingTriangle
}

func newSortingTriangleBucket() *sortingTriangleBucket {
	bucket := &sortingTriangleBucket{}
	bucket.unsetTris = make([]sortingTriangle, startingDisplayListSize)
	return bucket
}

func (s *sortingTriangleBucket) AddTriangle(tri *Triangle, depth float32) {
	if s.unsetTriIndex >= len(s.unsetTris) {
		s.unsetTris = append(s.unsetTris, sortingTriangle{})
		s.unsetTris = s.unsetTris[:cap(s.unsetTris)]
	}
	s.unsetTris[s.unsetTriIndex].Triangle = tri
	s.unsetTris[s.unsetTriIndex].depth = depth
	s.unsetTriIndex++
}

func (s *sortingTriangleBucket) Sort(minRange, maxRange float32) {

	count := s.unsetTriIndex
	binCount := s.binCount

	if s.sortMode == TriangleSortModeNone || binCount <= 1 {
		s.sorted = s.unsetTris[:count]
		return
	}

	rangeDiff := maxRange - minRange

	if rangeDiff == 0 {
		maxRange += 0.001
		rangeDiff += 0.001
	}

	if len(s.binOf) < count {
		s.binOf = make([]int32, len(s.unsetTris))
		s.sortedTris = make([]sortingTriangle, len(s.unsetTris))
	}

	starts := s.binStarts
	clear(starts)

	for i := 0; i < count; i++ {
		depth := (s.unsetTris[i].depth - minRange) / rangeDiff * float32(binCount)
		t := math32.Clamp(depth, 0, float32(binCount-1))
		targetBin := int32(t)
		s.binOf[i] = targetBin
		starts[targetBin]++
	}

	// Turn the counts into the first place of each bin, in draw order.
	place := 0
	if s.sortMode == TriangleSortModeBackToFront {
		for bin := binCount - 1; bin >= 0; bin-- {
			n := starts[bin]
			starts[bin] = place
			place += n
		}
	} else {
		for bin := 0; bin < binCount; bin++ {
			n := starts[bin]
			starts[bin] = place
			place += n
		}
	}

	for i := 0; i < count; i++ {
		bin := s.binOf[i]
		s.sortedTris[starts[bin]] = s.unsetTris[i]
		starts[bin]++
	}

	s.sorted = s.sortedTris[:count]

}

// Resize sets the number of depth bins.
func (s *sortingTriangleBucket) Resize(binCount int) {
	s.binCount = binCount
	s.binStarts = make([]int, binCount)
	s.Clear()
}

func (s *sortingTriangleBucket) resizeTriangleCount(triCount int) {
	for range triCount - len(s.unsetTris) {
		s.unsetTris = append(s.unsetTris, sortingTriangle{})
	}
}

func (s *sortingTriangleBucket) Clear() {
	s.unsetTriIndex = 0
	s.sorted = s.sorted[:0]
}

// ForEach calls forEach for each triangle in draw order, after Sort.
func (s *sortingTriangleBucket) ForEach(forEach func(triIndex int, triangle *Triangle)) {
	for triIndex, tri := range s.sorted {
		forEach(triIndex, tri.Triangle)
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
	globalSortingTriangleBucket.Resize(512)
}
