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

type sortingTriangleBin struct {
	triangleIndex int
	triangles     []sortingTriangle
}

type sortingTriangleBucket struct {
	unsetTriIndex int
	sortMode      int
	bins          []sortingTriangleBin
	unsetTris     []sortingTriangle
}

func newSortingTriangleBucket() *sortingTriangleBucket {
	bucket := &sortingTriangleBucket{}
	bucket.unsetTris = make([]sortingTriangle, startingDisplayListSize)
	return bucket
}

func (s *sortingTriangleBucket) AddTriangle(tri *Triangle, depth float32) {
	s.unsetTris[s.unsetTriIndex].Triangle = tri
	s.unsetTris[s.unsetTriIndex].depth = depth
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
			t := math32.Clamp(depth, 0, float32(binCount-1))
			targetBin = int(t)
		}

		s.bins[targetBin].triangles[s.bins[targetBin].triangleIndex].depth = s.unsetTris[i].depth
		s.bins[targetBin].triangles[s.bins[targetBin].triangleIndex].Triangle = s.unsetTris[i].Triangle
		s.bins[targetBin].triangleIndex++

	}

}

func (s *sortingTriangleBucket) Resize(binCount int) {
	s.bins = make([]sortingTriangleBin, binCount)
	for i := range s.bins {
		s.bins[i].triangles = make([]sortingTriangle, startingDisplayListSize)
	}
	s.Clear()
}

func (s *sortingTriangleBucket) resizeTriangleCount(triCount int) {
	for range triCount - len(s.unsetTris) {
		s.unsetTris = append(s.unsetTris, sortingTriangle{})
	}

	for _, bin := range s.bins {
		for range triCount - len(bin.triangles) {
			bin.triangles = append(bin.triangles, sortingTriangle{})
		}
	}
}

func (s *sortingTriangleBucket) Clear() {
	for i := 0; i < len(s.bins); i++ {
		s.bins[i].triangleIndex = 0
	}
	s.unsetTriIndex = 0
}

func (s *sortingTriangleBucket) ForEach(forEach func(triIndex int, triangle *Triangle)) {

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
				forEach(triIndex, bin.triangles[ti].Triangle)
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
				forEach(triIndex, bin.triangles[ti].Triangle)
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
	globalSortingTriangleBucket.Resize(512)
}
