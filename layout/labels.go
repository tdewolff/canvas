package layout

// OptimizeLabelPlacement uses simulated annealing to move the labels so that none are overlapping, they are close to their original position (anchor), and they don't leave the bounds rectangle.
//func OptimizeLabelPlacement(bounds canvas.Rect, labels, others []canvas.Rect, maxDistance float64) []canvas.Rect {
//	if len(labels) == 0 {
//		return labels
//	}
//
//	N := 50                  // new temperatures
//	MinM := 5 * len(labels)  // min iterations per temperature
//	MaxM := 30 * len(labels) // max iterations per temperature
//	Temperature := maxDistance / math.Log(3.0)
//
//	// get anchors
//	// move labels inside bounds
//	current := make([]canvas.Rect, len(labels))
//	copy(current, labels)
//	anchors := make([]canvas.Point, len(labels))
//	for i, label := range labels {
//		anchors[i] = label.Center()
//		if !bounds.Contains(label) {
//			if label.X0 < bounds.X0 {
//				current[i].X0 += bounds.X0 - label.X0
//				current[i].X1 += bounds.X0 - label.X0
//			} else if bounds.X1 < label.X1 {
//				current[i].X0 -= label.X1 - bounds.X1
//				current[i].X1 -= label.X1 - bounds.X1
//			}
//			if label.Y0 < bounds.Y0 {
//				current[i].Y0 += bounds.Y0 - label.Y0
//				current[i].Y1 += bounds.Y0 - label.Y0
//			} else if bounds.Y1 < label.Y1 {
//				current[i].Y0 -= label.Y1 - bounds.Y1
//				current[i].Y1 -= label.Y1 - bounds.Y1
//			}
//		}
//	}
//
//	// define energy function
//	energy := func(current []canvas.Rect, Es []float64) float64 {
//		E := 0.0
//		for i, label := range current {
//			// outside bounds is highly penalised
//			if !bounds.Contains(label) {
//				return math.Inf(1.0) // never picked as a solution
//			} else {
//				overlap := 0.0
//				for j, other := range labels {
//					if i != j {
//						overlap += label.And(other).Area()
//					}
//				}
//				for _, other := range others {
//					overlap += label.And(other).Area()
//				}
//				if 0.0 < overlap {
//					// overlap with other labels and other objects
//					Es[i] = 1.0
//					E += 1.0 // + overlap/maxDistance/maxDistance
//				} else {
//					Es[i] = 0.0
//					// distance from original position
//					//dist := label.DistanceToPoint(anchors[i])
//					//dist = math.Min(dist, maxDistance)
//					//Es[i] = 1.0
//					//E += dist / maxDistance
//				}
//			}
//		}
//		return E
//	}
//
//	// define update function
//	update := func(labels []canvas.Rect, Es []float64, T float64) {
//		// select label to optimize based on "wrongness"
//		var index int
//		var sum, cum float64
//		for _, e := range Es {
//			sum += e
//		}
//		x := rand.Float64() * sum
//		for i, e := range Es {
//			cum += e
//			if x < cum {
//				index = i
//				break
//			}
//		}
//
//		// stepSize is the standard deviation
//		stepSize := 0.1 * maxDistance * T / Temperature
//		dx := (2.0*rand.Float64() - 1.0) * stepSize
//		dy := (2.0*rand.Float64() - 1.0) * stepSize
//		//dir := rand.Float64() * 2.0 * math.Pi
//		//d := canvas.Point{stepSize, 0.0}.Rot(dir, canvas.Point{})
//		labels[index] = labels[index].Translate(dx, dy)
//	}
//
//	// initial solution
//	currentEs := make([]float64, len(labels))
//	currentE := energy(current, currentEs)
//
//	// best solution
//	best := make([]canvas.Rect, len(labels))
//	copy(best, labels)
//	bestE := currentE
//
//	// simulated annealing
//	candidate := make([]canvas.Rect, len(labels))
//	candidateEs := make([]float64, len(labels))
//
//	T := Temperature
//	for n := 0; n < N; n++ {
//		accepted, consecAccepted := 0, 0
//		for i := 0; i < MaxM; i++ {
//			// generate candidate solution
//			copy(candidate, current)
//			update(candidate, currentEs, T)
//			candidateE := energy(candidate, candidateEs)
//
//			// check to keep the new solution or not
//			fmt.Println(currentE, candidateE)
//			if candidateE < bestE || rand.Float64() < math.Exp((currentE-candidateE)/T) {
//				if candidateE < bestE {
//					copy(best, candidate)
//					bestE = candidateE
//				}
//				copy(current, candidate)
//				copy(currentEs, candidateEs)
//				currentE = candidateE
//
//				accepted++
//				consecAccepted++
//				if consecAccepted == MinM {
//					// decrease temperature immediately
//					break
//				}
//			} else {
//				consecAccepted = 0
//			}
//		}
//		if accepted == 0 {
//			// found optimal
//			break
//		}
//		T -= 0.1 * T
//	}
//	for i := range labels {
//		fmt.Println(i, canvas.Point{labels[i].X0, labels[i].Y0}.Sub(canvas.Point{best[i].X0, best[i].Y0}).Length(), bestE)
//	}
//	return best
//}
