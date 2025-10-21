package rasterizer

import (
	"image"
	"image/color"
	"math"

	"github.com/srwiley/rasterx"
	"github.com/srwiley/scanx"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"

	"github.com/tdewolff/canvas"
)

// TODO: add ASM optimized version for NRGBA images, since those are much faster to write as PNG

// Draw draws the canvas on a new image with given resolution (in dots-per-millimeter). Higher resolution will result in larger images.
func Draw(c *canvas.Canvas, resolution canvas.Resolution, colorSpace canvas.ColorSpace) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(c.W*resolution.DPMM()+0.5), int(c.H*resolution.DPMM()+0.5)))
	ras := FromImage(img, resolution, colorSpace)
	c.RenderTo(ras)
	ras.Close()
	return img
}

// Rasterizer is a rasterizing renderer.
type Rasterizer struct {
	draw.Image
	resolution canvas.Resolution
	colorSpace canvas.ColorSpace

	spanner *scanx.ImgSpanner
	scanner *scanx.Scanner
}

// New returns a renderer that draws to a rasterized image. The final width and height of the image is the width and height (mm) multiplied by the resolution (px/mm), thus a higher resolution results in larger images. By default the linear color space is used, which assumes input and output colors are in linearRGB. If the sRGB color space is used for drawing with an average of gamma=2.2, the input and output colors are assumed to be in sRGB (a common assumption) and blending happens in linearRGB. Be aware that for text this results in thin stems for black-on-white (but wide stems for white-on-black).
func New(width, height float64, resolution canvas.Resolution, colorSpace canvas.ColorSpace) *Rasterizer {
	img := image.NewRGBA(image.Rect(0, 0, int(width*resolution.DPMM()+0.5), int(height*resolution.DPMM()+0.5)))
	return FromImage(img, resolution, colorSpace)
}

// FromImage returns a renderer that draws to an existing image. Resolution is in pixels per unit of canvas coordinates (millimeters). A higher resolution will give a larger and more detailed image.
func FromImage(img draw.Image, resolution canvas.Resolution, colorSpace canvas.ColorSpace) *Rasterizer {
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		panic("raster size is zero, increase resolution")
	} else if math.MaxInt32/bounds.Dx() < bounds.Dy() {
		panic("raster size overflow, decrease resolution")
	}

	if colorSpace == nil {
		colorSpace = canvas.DefaultColorSpace
	}
	spanner := scanx.NewImgSpanner(img)
	return &Rasterizer{
		Image:      img,
		resolution: resolution,
		colorSpace: colorSpace,

		spanner: spanner,
		scanner: scanx.NewScanner(spanner, bounds.Dx(), bounds.Dy()),
	}
}

func (r *Rasterizer) Close() {
	if _, ok := r.colorSpace.(canvas.LinearColorSpace); !ok {
		// gamma compress
		changeColorSpace(r.Image, r.Image, r.colorSpace.FromLinear)
	}
}

// Size returns the size of the canvas in millimeters.
func (r *Rasterizer) Size() (float64, float64) {
	size := r.Bounds().Size()
	return float64(size.X) / r.resolution.DPMM(), float64(size.Y) / r.resolution.DPMM()
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *Rasterizer) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	bounds := canvas.Rect{}
	var fill, stroke *canvas.Path
	if style.HasFill() {
		fill = path.Copy().Transform(m)
		bounds = fill.FastBounds()
	}
	if style.HasStroke() {
		tolerance := canvas.PixelTolerance / r.resolution.DPMM()
		stroke = path
		if 0 < len(style.Dashes) {
			dashOffset, dashes := canvas.ScaleDash(style.StrokeWidth, style.DashOffset, style.Dashes)
			stroke = stroke.Dash(dashOffset, dashes...)
		}
		stroke = stroke.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner, tolerance)
		stroke = stroke.Transform(m)
		if style.HasFill() {
			bounds = bounds.Add(stroke.FastBounds())
		} else {
			bounds = stroke.FastBounds()
		}
	}

	r.scanner.SetWinding(style.FillRule == canvas.NonZero)

	size := r.Bounds().Size()
	if style.HasFill() {
		if style.Fill.IsPattern() {
			if hatch, ok := style.Fill.Pattern.(*canvas.HatchPattern); ok {
				style.Fill = hatch.Fill
				fill = hatch.Tile(fill)
			} else {
				pattern := style.Fill.Pattern.Transform(m).SetColorSpace(r.colorSpace)
				pattern.RenderTo(r, fill)
			}
		}
		if style.Fill.IsGradient() {
			gradient := style.Fill.Gradient.Transform(m).SetColorSpace(r.colorSpace)
			r.scanner.Clear()
			r.scanner.SetColor(rasterx.ColorFunc(func(x, y int) color.Color {
				// TODO: convert to dst color model
				return gradient.At((float64(x)+0.5)/float64(r.resolution), (float64(size.Y-y)-0.5)/float64(r.resolution))
			}))
			fill.ToScanxScanner(r.scanner, float64(size.Y), r.resolution)
			r.scanner.Draw()
		} else if style.Fill.IsColor() {
			c := r.colorSpace.ToLinear(style.Fill.Color)
			r.scanner.Clear()
			r.scanner.SetColor(color.Color(r.Image.ColorModel().Convert(c)))
			fill.ToScanxScanner(r.scanner, float64(size.Y), r.resolution)
			r.scanner.Draw()
		}
	}
	if style.HasStroke() {
		if style.Stroke.IsPattern() {
			if hatch, ok := style.Stroke.Pattern.(*canvas.HatchPattern); ok {
				style.Stroke = hatch.Fill
				stroke = hatch.Tile(stroke)
			} else {
				pattern := style.Stroke.Pattern.Transform(m).SetColorSpace(r.colorSpace)
				pattern.RenderTo(r, stroke)
			}
		}
		if style.Stroke.IsGradient() {
			gradient := style.Stroke.Gradient.Transform(m).SetColorSpace(r.colorSpace)
			r.scanner.Clear()
			r.scanner.SetColor(rasterx.ColorFunc(func(x, y int) color.Color {
				// TODO: convert to dst color model
				return gradient.At((float64(x)+0.5)/float64(r.resolution), (float64(size.Y-y)-0.5)/float64(r.resolution))
			}))
			stroke.ToScanxScanner(r.scanner, float64(size.Y), r.resolution)
			r.scanner.Draw()
		} else if style.Stroke.IsColor() {
			c := r.colorSpace.ToLinear(style.Stroke.Color)
			r.scanner.Clear()
			r.scanner.SetColor(color.Color(r.Image.ColorModel().Convert(c)))
			stroke.ToScanxScanner(r.scanner, float64(size.Y), r.resolution)
			r.scanner.Draw()
		}
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *Rasterizer) RenderText(text *canvas.Text, m canvas.Matrix) {
	text.RenderAsPath(r, m, r.resolution)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *Rasterizer) RenderImage(img image.Image, m canvas.Matrix) {
	// add transparent margin to image for smooth borders when rotating
	// TODO: optimize when transformation is only translation or stretch (if optimizing, dont overwrite original img when gamma correcting)
	margin := 0
	if (m[0][1] != 0.0 || m[1][0] != 0.0) && (m[0][0] != 0.0 || m[1][1] == 0.0) {
		// only add margin for shear transformation or rotations that are not 90/180/270 degrees
		margin = 4
		size := img.Bounds().Size()
		sp := img.Bounds().Min // starting point
		img2 := image.NewRGBA(image.Rect(0, 0, size.X+margin*2, size.Y+margin*2))
		draw.Draw(img2, image.Rect(margin, margin, size.X+margin, size.Y+margin), img, sp, draw.Over)
		img = img2
	}

	if _, ok := r.colorSpace.(canvas.LinearColorSpace); !ok {
		// gamma decompress
		changeColorSpace(img.(draw.Image), img, r.colorSpace.ToLinear)
	}

	// draw to destination image
	// note that we need to correct for the added margin in origin and m
	dpmm := r.resolution.DPMM()
	origin := m.Dot(canvas.Point{-float64(margin), float64(img.Bounds().Size().Y - margin)}).Mul(dpmm)
	m = m.Scale(dpmm, dpmm)

	h := float64(r.Bounds().Size().Y)
	aff3 := f64.Aff3{m[0][0], -m[0][1], origin.X, -m[1][0], m[1][1], h - origin.Y}
	draw.CatmullRom.Transform(r, aff3, img, img.Bounds(), draw.Over, nil)
}
