package main

import (
	"os"

	"github.com/tdewolff/canvas"
)

func main() {
	pdfFile, err := os.Create("test.pdf")
	if err != nil {
		panic(err)
	}
	defer pdfFile.Close()

	c := canvas.New()
	Draw(c)
	c.WritePDF(pdfFile)
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

	c.DrawPath(0, 80, path)
	path.Scale(-1.0, 1.0)
	c.DrawPath(150, 80, path)
	path.Scale(1.0, -1.0)
	c.DrawPath(150, 80, path)
	path.Scale(-1.0, 1.0)
	c.DrawPath(0, 80, path)
}
