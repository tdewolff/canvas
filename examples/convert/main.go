package main

import (
	"os"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

func main() {
	svgFile, err := os.Open("./age.svg")
	if err != nil {
		panic("Failed to open SVG file: " + err.Error())
	}
	defer svgFile.Close()

	c, err := canvas.ParseSVG(svgFile)
	if err != nil {
		panic("Failed to parse SVG: " + err.Error())
	}

	err = renderers.Write("./age.png", c, canvas.DPMM(3.2))
	if err != nil {
		panic("Failed to write PNG: " + err.Error())
	}

	println("✅ SVG converted successfully!")
}
