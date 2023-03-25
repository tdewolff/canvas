package rasterizer

import (
	"image"
	"image/color"

	"github.com/tdewolff/canvas"
)

type PatternImage struct {
	pattern  canvas.Pattern
	zp, size image.Point
	dpmm     float64
}

func NewPatternImage(pattern canvas.Pattern, zp, size image.Point, res canvas.Resolution, colorSpace canvas.ColorSpace) *PatternImage {
	return &PatternImage{
		pattern: pattern.SetColorSpace(colorSpace),
		zp:      zp,   // zero-point in dst
		size:    size, // dst size
		dpmm:    res.DPMM(),
	}
}

func (g *PatternImage) ColorModel() color.Model {
	return color.RGBAModel
}

func (g *PatternImage) Bounds() image.Rectangle {
	return image.Rectangle{image.Point{-1e9, -1e9}, image.Point{1e9, 1e9}}
}

func (g *PatternImage) At(x, y int) color.Color {
	return g.pattern.At(float64(g.zp.X+x)/g.dpmm, float64(g.size.Y-g.zp.Y-y)/g.dpmm)
}
