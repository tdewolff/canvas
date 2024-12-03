package rasterizer

import (
	"image"
	"image/color"

	"github.com/Seanld/canvas"
	"golang.org/x/image/draw"
)

type colorFunc func(color.Color) color.RGBA

func changeColorSpace(dst draw.Image, src image.Image, f colorFunc) {
	if dstRGBA, ok := dst.(*image.RGBA); ok {
		for j := 0; j < dst.Bounds().Max.Y; j++ {
			for i := 0; i < dst.Bounds().Max.X; i++ {
				// TODO: parallelize
				dstRGBA.SetRGBA(i, j, f(src.At(i, j)))
			}
		}
	} else {
		for j := 0; j < dst.Bounds().Max.Y; j++ {
			for i := 0; i < dst.Bounds().Max.X; i++ {
				// TODO: parallelize
				dst.Set(i, j, f(src.At(i, j)))
			}
		}
	}
}

type GradientImage struct {
	g        canvas.Gradient
	zp, size image.Point
	dpmm     float64
}

func NewGradientImage(g canvas.Gradient, zp, size image.Point, res canvas.Resolution) *GradientImage {
	return &GradientImage{
		g:    g,
		zp:   zp,   // zero-point in dst
		size: size, // dst size
		dpmm: res.DPMM(),
	}
}

func (img *GradientImage) ColorModel() color.Model {
	return color.RGBAModel
}

func (img *GradientImage) Bounds() image.Rectangle {
	return image.Rectangle{image.Point{-1e9, -1e9}, image.Point{1e9, 1e9}}
}

func (img *GradientImage) At(x, y int) color.Color {
	return img.g.At(float64(img.zp.X+x)/img.dpmm, float64(img.size.Y-img.zp.Y-y)/img.dpmm)
}

//func NewPatternImage(p canvas.Pattern, zp, size image.Point, res canvas.Resolution, colorSpace canvas.ColorSpace) *image.RGBA {
//	img := image.NewRGBA(image.Rect(0, 0, int(float64(size.X)*res.DPMM()+0.5), int(float64(size.Y)*res.DPMM()+0.5)))
//	ras := FromImage(img, res, colorSpace)
//	p.RenderTo(ras)
//	ras.Close()
//	return img
//}
