package layout

import (
	"math"
	"math/rand"

	"github.com/tdewolff/canvas"
)

// OptimizeLabelPlacement uses simulated annealing to move the labels so that none are overlapping, they are close to their original position (anchor), and they don't leave the bounds rectangle.
func OptimizeLabelPlacement(bounds canvas.Rect, labels, others []canvas.Rect) []canvas.Rect {
	if len(labels) == 0 {
		return labels
	}

	N := 250
	Temperature := 100.0
	MaxStepSize := math.Max(bounds.W(), bounds.H()) * 10.0

	// ensure bounds encompasses all labels
	// get anchors
	anchors := make([]canvas.Point, len(labels))
	for i, label := range labels {
		anchors[i] = label.Center()
	}

	// define energy function
	energy := func(current []canvas.Rect, Es []float64) float64 {
		E := 0.0
		for i, label := range current {
			// outside bounds is highly penalised
			if !bounds.Contains(label) {
				Es[i] = 1e6
				E = math.Inf(1.0)
				continue
			}

			// distance from original position
			distAnchor := label.DistanceToPoint(anchors[i])

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
			Es[i] = distAnchor + 1e6*overlapArea
			E += Es[i]
		}
		return E
	}

	// define update function
	update := func(labels []canvas.Rect, Es []float64, T float64) {
		// select label to optimize based on "wrongness"
		var index int
		var sum, cum float64
		for _, e := range Es {
			sum += e
		}
		x := rand.Float64() * sum
		for i, e := range Es {
			cum += e
			if x < cum {
				index = i
				break
			}
		}

		// stepSize is the standard deviation
		stepSize := T * MaxStepSize / Temperature
		dx := rand.NormFloat64() * stepSize
		dy := rand.NormFloat64() * stepSize
		labels[index] = labels[index].Translate(dx, dy)
	}

	// initial solution
	current := make([]canvas.Rect, len(labels))
	currentEs := make([]float64, len(labels))
	copy(current, labels)
	currentE := energy(current, currentEs)

	// best solution
	best := make([]canvas.Rect, len(labels))
	copy(best, labels)
	bestE := currentE

	// simulated annealing
	candidate := make([]canvas.Rect, len(labels))
	candidateEs := make([]float64, len(labels))
	for i := 0; i < N; i++ {
		T := Temperature / float64(i+1)

		// generate candidate solution
		copy(candidate, current)
		update(candidate, currentEs, T)
		candidateE := energy(candidate, candidateEs)

		// check to keep the new solution or not
		if candidateE < bestE || rand.Float64() < math.Exp((currentE-candidateE)/T) {
			if candidateE < bestE {
				copy(best, candidate)
				if math.IsInf(bestE, 0.0) && !math.IsInf(candidateE, 0.0) {
					// reset iterations if this is the first viable candidate
					N += i
				}
				bestE = candidateE
			}
			copy(current, candidate)
			copy(currentEs, candidateEs)
			currentE = candidateE
		}
	}
	return best
}
