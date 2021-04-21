package rasterizer

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/tdewolff/canvas"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/tiff"
	"golang.org/x/image/vector"
)

// PNGWriter writes the canvas as a PNG file.
func PNGWriter(resolution canvas.Resolution) canvas.Writer {
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution)
		// TODO: optimization: cache img until canvas changes
		return png.Encode(w, img)
	}
}

// JPGWriter writes the canvas as a JPG file.
func JPGWriter(resolution canvas.Resolution, opts *jpeg.Options) canvas.Writer {
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution)
		// TODO: optimization: cache img until canvas changes
		return jpeg.Encode(w, img, opts)
	}
}

// GIFWriter writes the canvas as a GIF file.
func GIFWriter(resolution canvas.Resolution, opts *gif.Options) canvas.Writer {
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution)
		// TODO: optimization: cache img until canvas changes
		return gif.Encode(w, img, opts)
	}
}

// TIFFWriter writes the canvas as a TIFF file.
func TIFFWriter(resolution canvas.Resolution, opts *tiff.Options) canvas.Writer {
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution)
		// TODO: optimization: cache img until canvas changes
		return tiff.Encode(w, img, opts)
	}
}

// Draw draws the canvas on a new image with given resolution (in dots-per-millimeter). Higher resolution will result in larger images.
func Draw(c *canvas.Canvas, resolution canvas.Resolution) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(c.W*resolution.DPMM()+0.5), int(c.H*resolution.DPMM()+0.5)))
	ras := New(img, resolution)
	c.Render(ras)
	return img
}

// Rasterizer is a rasterizing renderer.
type Rasterizer struct {
	img        draw.Image
	resolution canvas.Resolution
}

// New returns a renderer that draws to a rasterized image.
func New(img draw.Image, resolution canvas.Resolution) *Rasterizer {
	return &Rasterizer{
		img:        img,
		resolution: resolution,
	}
}

// Size returns the size of the canvas in millimeters.
func (r *Rasterizer) Size() (float64, float64) {
	size := r.img.Bounds().Size()
	return float64(size.X) / r.resolution.DPMM(), float64(size.Y) / r.resolution.DPMM()
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *Rasterizer) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	// TODO: use fill rule (EvenOdd, NonZero) for rasterizer
	path = path.Transform(m)

	strokeWidth := 0.0
	if style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth {
		strokeWidth = style.StrokeWidth
	}

	size := r.img.Bounds().Size()
	bounds := path.Bounds()
	dx, dy := 0, 0
	dpmm := r.resolution.DPMM()
	x := int((bounds.X - strokeWidth) * dpmm)
	y := int((bounds.Y - strokeWidth) * dpmm)
	w := int((bounds.W+2*strokeWidth)*dpmm) + 1
	h := int((bounds.H+2*strokeWidth)*dpmm) + 1
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

	path = path.Translate(-float64(x)/dpmm, -float64(y)/dpmm)
	if style.FillColor.A != 0 {
		ras := vector.NewRasterizer(w, h)
		path.ToRasterizer(ras, r.resolution)
		ras.Draw(r.img, image.Rect(x, size.Y-y, x+w, size.Y-y-h), image.NewUniform(style.FillColor), image.Point{dx, dy})
	}
	if style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth {
		if 0 < len(style.Dashes) {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)

		ras := vector.NewRasterizer(w, h)
		path.ToRasterizer(ras, r.resolution)
		ras.Draw(r.img, image.Rect(x, size.Y-y, x+w, size.Y-y-h), image.NewUniform(style.StrokeColor), image.Point{dx, dy})
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *Rasterizer) RenderText(text *canvas.Text, m canvas.Matrix) {
	text.RenderAsPath(r, m, r.resolution)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *Rasterizer) RenderImage(img image.Image, m canvas.Matrix) {
	// add transparent margin to image for smooth borders when rotating
	margin := 4
	size := img.Bounds().Size()
	sp := img.Bounds().Min // starting point
	img2 := image.NewRGBA(image.Rect(0, 0, size.X+margin*2, size.Y+margin*2))
	draw.Draw(img2, image.Rect(margin, margin, size.X+margin, size.Y+margin), img, sp, draw.Over)

	// draw to destination image
	// note that we need to correct for the added margin in origin and m
	// TODO: optimize when transformation is only translation or stretch
	dpmm := r.resolution.DPMM()
	origin := m.Dot(canvas.Point{-float64(margin), float64(img2.Bounds().Size().Y - margin)}).Mul(dpmm)
	m = m.Scale(dpmm*(float64(size.X+margin)/float64(size.X)), dpmm*(float64(size.Y+margin)/float64(size.Y)))

	h := float64(r.img.Bounds().Size().Y)
	aff3 := f64.Aff3{m[0][0], -m[0][1], origin.X, -m[1][0], m[1][1], h - origin.Y}
	draw.CatmullRom.Transform(r.img, aff3, img2, img2.Bounds(), draw.Over, nil)
}
