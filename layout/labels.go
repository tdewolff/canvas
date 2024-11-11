package layout

import (
	"math"
	"math/rand"

	"github.com/tdewolff/canvas"
)

// OptimizeLabelPlacement uses simulated annealing to move the labels so that none are overlapping, they are close to their original position (anchor), and they don't leave the bounds rectangle.
func OptimizeLabelPlacement(bounds canvas.Rect, labels, others []canvas.Rect) []canvas.Rect {
	N := 100
	Temperature := 100.0
	StepSize := math.Max(bounds.X1-bounds.X0, bounds.Y1-bounds.Y0) / 100.0

	// ensure bounds encompasses all labels
	for _, label := range labels {
		bounds = bounds.Add(label)
	}

	// define energy function
	energy := func(current []canvas.Rect) float64 {
		E := 0.0
		for i, label := range current {
			// outside bounds is highly penalised
			if !bounds.Contains(label) {
				return math.Inf(1.0)
			}

			// distance from original position
			dx := label.X0 - labels[i].X0
			dy := label.Y0 - labels[i].Y0
			distAnchor := canvas.Point{dx, dy}.Length()

			// overlap with other labels and other objects
			overlapArea := 0.0
			for j, other := range labels {
				if i != j {
					overlapArea += label.And(other).Area()
				}
			}
			for _, other := range others {
				overlapArea += label.And(other).Area()
			}
			E += distAnchor + 100.0*overlapArea
		}
		return E
	}

	// define update function
	update := func(labels []canvas.Rect) {
		// StepSize is the standard deviation
		dx := rand.NormFloat64() * StepSize
		dy := rand.NormFloat64() * StepSize
		index := rand.Int() % len(labels)
		labels[index] = labels[index].Translate(dx, dy)
	}

	// initial solution
	current := make([]canvas.Rect, len(labels))
	copy(current, labels)
	currentE := energy(current)

	best := make([]canvas.Rect, len(labels))
	copy(best, labels)
	bestE := currentE

	candidate := make([]canvas.Rect, len(labels))
	for i := 0; i < N; i++ {
		T := Temperature / float64(i+1)

		// generate candidate solution
		copy(candidate, current)
		update(candidate)
		candidateE := energy(candidate)

		// check to keep the new solution or not
		if candidateE < bestE || rand.Float64() < math.Exp((currentE-candidateE)/T) {
			if candidateE < bestE {
				copy(best, candidate)
				bestE = candidateE
			}
			copy(current, candidate)
			currentE = candidateE
		}
	}
	return best
}
