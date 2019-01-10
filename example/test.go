package main

import (
	"fmt"
	"os"

	"github.com/tdewolff/canvas"
)

func main() {
	svgFile, err := os.Create("test.svg")
	if err != nil {
		panic(err)
	}
	defer svgFile.Close()

	c := canvas.New(72.0)
	Draw(c)
	c.WriteSVG(svgFile)
}

func Draw(c *canvas.C) {
	c.Open(300, 300)

	path, err := canvas.ParseSVGPath("M10 -10L10 -50C10 -70 50 -70 50 -50A20 10 45 1 0 50 -30z")
	if err != nil {
		panic(err)
	}

	psPath := path.Copy()
	psPath.Scale(2.0, -2.0)
	psPath.Translate(0, 200)
	fmt.Println(psPath.ToPS(), "fill")

	c.DrawPath(0, 80, path)
	path.Scale(-1.0, 1.0)
	c.DrawPath(150, 80, path)
	path.Scale(1.0, -1.0)
	c.DrawPath(150, 80, path)
	path.Scale(-1.0, 1.0)
	c.DrawPath(0, 80, path)
}
