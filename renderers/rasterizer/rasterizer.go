package rasterizer

import (
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"

	"github.com/tdewolff/canvas"
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/tiff"
	"golang.org/x/image/vector"
)

// PNGWriter writes the canvas as a PNG file.
// DEPRECATED
func PNGWriter(resolution canvas.Resolution) canvas.Writer {
	log.Println("WARNING: github.com/tdewolff/canvas/renderers/rasterizer.PNGWriter is deprecated, please use github.com/tdewolff/canvas/renderers.PNG")
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution, nil)
		return png.Encode(w, img)
	}
}

// JPGWriter writes the canvas as a JPG file.
// DEPRECATED
func JPGWriter(resolution canvas.Resolution, opts *jpeg.Options) canvas.Writer {
	log.Println("WARNING: github.com/tdewolff/canvas/renderers/rasterizer.JPGWriter is deprecated, please use github.com/tdewolff/canvas/renderers.JPG")
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution, nil)
		return jpeg.Encode(w, img, opts)
	}
}

// GIFWriter writes the canvas as a GIF file.
// DEPRECATED
func GIFWriter(resolution canvas.Resolution, opts *gif.Options) canvas.Writer {
	log.Println("WARNING: github.com/tdewolff/canvas/renderers/rasterizer.GIFWriter is deprecated, please use github.com/tdewolff/canvas/renderers.GIF")
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution, nil)
		return gif.Encode(w, img, opts)
	}
}

// TIFFWriter writes the canvas as a TIFF file.
// DEPRECATED
func TIFFWriter(resolution canvas.Resolution, opts *tiff.Options) canvas.Writer {
	log.Println("WARNING: github.com/tdewolff/canvas/renderers/rasterizer.TIFFWriter is deprecated, please use github.com/tdewolff/canvas/renderers.TIFF")
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution, nil)
		return tiff.Encode(w, img, opts)
	}
}

// BMPWriter writes the canvas as a BMP file.
// DEPRECATED
func BMPWriter(resolution canvas.Resolution) canvas.Writer {
	log.Println("WARNING: github.com/tdewolff/canvas/renderers/rasterizer.BMPWriter is deprecated, please use github.com/tdewolff/canvas/renderers.BMP")
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution, nil)
		return bmp.Encode(w, img)
	}
}

// Draw draws the canvas on a new image with given resolution (in dots-per-millimeter). Higher resolution will result in larger images.
func Draw(c *canvas.Canvas, resolution canvas.Resolution, colorSpace canvas.ColorSpace) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(c.W*resolution.DPMM()+0.5), int(c.H*resolution.DPMM()+0.5)))
	ras := FromImage(img, resolution, colorSpace)
	c.Render(ras)
	ras.Close()
	return img
}

// Rasterizer is a rasterizing renderer.
type Rasterizer struct {
	draw.Image
	resolution canvas.Resolution
	ColorSpace canvas.ColorSpace
}

// New returns a renderer that draws to a rasterized image. By default the linear color space is used, which assumes input and output colors are in linearRGB. If the sRGB color space is used for drawing with an average of gamma=2.2, the input and output colors are assumed to be in sRGB and blending happens in linearRGB. Be aware that for text this results in thin stems for black-on-white (but wide stems for white-on-black).
func New(width, height float64, resolution canvas.Resolution, colorSpace canvas.ColorSpace) *Rasterizer {
	img := image.NewRGBA(image.Rect(0, 0, int(width*resolution.DPMM()+0.5), int(height*resolution.DPMM()+0.5)))
	return FromImage(img, resolution, colorSpace)
}

// FromImage returns a renderer that draws to an existing image.
func FromImage(img draw.Image, resolution canvas.Resolution, colorSpace canvas.ColorSpace) *Rasterizer {
	if colorSpace == nil {
		colorSpace = canvas.DefaultColorSpace
	}
	return &Rasterizer{
		Image:      img,
		resolution: resolution,
		ColorSpace: colorSpace,
	}
}

func (r *Rasterizer) Close() {
	if _, ok := r.ColorSpace.(canvas.LinearColorSpace); !ok {
		// gamma compress
		changeColorSpace(r.Image, r.Image, r.ColorSpace.FromLinear)
	}
}

// Size returns the size of the canvas in millimeters.
func (r *Rasterizer) Size() (float64, float64) {
	size := r.Bounds().Size()
	return float64(size.X) / r.resolution.DPMM(), float64(size.Y) / r.resolution.DPMM()
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *Rasterizer) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	// TODO: use fill rule (EvenOdd, NonZero) for rasterizer
	fill := path
	stroke := path
	bounds := canvas.Rect{}
	if style.HasFill() {
		fill = path.Transform(m)
		if !style.HasStroke() {
			bounds = fill.Bounds()
		}
	}
	if style.HasStroke() {
		if 0 < len(style.Dashes) {
			stroke = stroke.Dash(style.DashOffset, style.Dashes...)
		}
		stroke = stroke.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)
		stroke = stroke.Transform(m)
		bounds = stroke.Bounds()
	}

	size := r.Bounds().Size()
	dx, dy := 0, 0
	dpmm := r.resolution.DPMM()
	x := int(bounds.X * dpmm)
	y := int(bounds.Y * dpmm)
	w := int(bounds.W*dpmm) + 1
	h := int(bounds.H*dpmm) + 1
	if (x+w <= 0 || size.X <= x) && (y+h <= 0 || size.Y <= y) {
		return // outside canvas
	}

	if x < 0 {
		dx = -x
		x = 0
	}
	if y < 0 {
		dy = -y
		y = 0
	}
	if size.X <= x+w {
		w = size.X - x
	}
	if size.Y <= y+h {
		h = size.Y - y
	}
	if w <= 0 || h <= 0 {
		return // has no size
	}

	if style.HasFill() {
		ras := vector.NewRasterizer(w, h)
		fill = fill.Translate(-float64(x)/dpmm, -float64(y)/dpmm)
		fill.ToRasterizer(ras, r.resolution)
		col := r.ColorSpace.ToLinear(style.FillColor)
		ras.Draw(r, image.Rect(x, size.Y-y, x+w, size.Y-y-h), image.NewUniform(col), image.Point{dx, dy})
	}
	if style.HasStroke() {
		ras := vector.NewRasterizer(w, h)
		stroke = stroke.Translate(-float64(x)/dpmm, -float64(y)/dpmm)
		stroke.ToRasterizer(ras, r.resolution)
		col := r.ColorSpace.ToLinear(style.StrokeColor)
		ras.Draw(r, image.Rect(x, size.Y-y, x+w, size.Y-y-h), image.NewUniform(col), image.Point{dx, dy})
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
	margin := 4
	size := img.Bounds().Size()
	sp := img.Bounds().Min // starting point
	img2 := image.NewRGBA(image.Rect(0, 0, size.X+margin*2, size.Y+margin*2))
	draw.Draw(img2, image.Rect(margin, margin, size.X+margin, size.Y+margin), img, sp, draw.Over)

	// draw to destination image
	// note that we need to correct for the added margin in origin and m
	dpmm := r.resolution.DPMM()
	origin := m.Dot(canvas.Point{-float64(margin), float64(img2.Bounds().Size().Y - margin)}).Mul(dpmm)
	m = m.Scale(dpmm, dpmm)

	if _, ok := r.ColorSpace.(canvas.LinearColorSpace); !ok {
		// gamma decompress
		changeColorSpace(img2, img2, r.ColorSpace.ToLinear)
	}

	h := float64(r.Bounds().Size().Y)
	aff3 := f64.Aff3{m[0][0], -m[0][1], origin.X, -m[1][0], m[1][1], h - origin.Y}
	draw.CatmullRom.Transform(r, aff3, img2, img2.Bounds(), draw.Over, nil)
}

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
