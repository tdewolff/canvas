package main

import (
	"log"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/svg"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func main() {
	p, err := plot.New()
	if err != nil {
		log.Fatalf("could not create plot: %v", err)
	}
	p.Title.Text = "Scatter plot"
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"

	scatter, err := plotter.NewScatter(plotter.XYs{{X: 1, Y: 1}, {X: 0, Y: 1}, {X: 0, Y: 0}})
	if err != nil {
		log.Fatalf("could not create scatter: %v", err)
	}
	p.Add(scatter)

	err = p.Save(5*vg.Centimeter, 5*vg.Centimeter, "target.svg")
	if err != nil {
		log.Fatalf("could not save SVG plot: %v", err)
	}

	c := canvas.New(50.0, 50.0)
	gonumCanvas := canvas.NewGonumPlot(c)
	p.Draw(gonumCanvas)

	c.WriteFile("output.svg", svg.Writer)
}
