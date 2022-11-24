package tetra3d

import "github.com/quartercastle/vector"

type FinishMode int

const (
	FinishModeLoop     FinishMode = iota // Loop on animation completion
	FinishModePingPong                   // Reverse on animation completion; if this is the case, the OnFinish() callback is called after two loops (one reversal)
	FinishModeStop                       // Stop on animation completion
)

// Distance returns the distance between two vector.Vectors.
func Distance(posOne, posTwo vector.Vector) float64 {
	return posOne.Sub(posTwo).Magnitude()
}
