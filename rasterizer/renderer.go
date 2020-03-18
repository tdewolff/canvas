package rasterizer

import (
	"image"

	"github.com/tdewolff/canvas"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/vector"
)

// Draw draws the canvas on a new image with given resolution (in dots-per-millimeter).
// Higher resolution will result in bigger images.
func Draw(c *canvas.Canvas, resolution canvas.DPMM) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(c.W*float64(resolution)+0.5), int(c.H*float64(resolution)+0.5)))
	ras := New(img, resolution)
	c.Render(ras)
	return img
}

type Renderer struct {
	img        draw.Image
	resolution canvas.DPMM
}

// New creates a renderer that draws to a rasterized image.
func New(img draw.Image, resolution canvas.DPMM) *Renderer {
	return &Renderer{
		img:        img,
		resolution: resolution,
	}
}

// Size returns the width and height in millimeters
func (r *Renderer) Size() (float64, float64) {
	size := r.img.Bounds().Size()
	return float64(size.X) / float64(r.resolution), float64(size.Y) / float64(r.resolution)
}

func (r *Renderer) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	// TODO: use fill rule (EvenOdd, NonZero) for rasterizer
	path = path.Transform(m)

	strokeWidth := 0.0
	if style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth {
		strokeWidth = style.StrokeWidth
	}

	size := r.img.Bounds().Size()
	bounds := path.Bounds()
	dx, dy := 0, 0
	resolution := float64(r.resolution)
	x := int((bounds.X - strokeWidth) * resolution)
	y := int((bounds.Y - strokeWidth) * resolution)
	w := int((bounds.W+2*strokeWidth)*resolution) + 1
	h := int((bounds.H+2*strokeWidth)*resolution) + 1
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

	path = path.Translate(-float64(x)/resolution, -float64(y)/resolution)
	if style.FillColor.A != 0 {
		ras := vector.NewRasterizer(w, h)
		path.ToRasterizer(ras, resolution)
		ras.Draw(r.img, image.Rect(x, size.Y-y, x+w, size.Y-y-h), image.NewUniform(style.FillColor), image.Point{dx, dy})
	}
	if style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth {
		if 0 < len(style.Dashes) {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)

		ras := vector.NewRasterizer(w, h)
		path.ToRasterizer(ras, resolution)
		ras.Draw(r.img, image.Rect(x, size.Y-y, x+w, size.Y-y-h), image.NewUniform(style.StrokeColor), image.Point{dx, dy})
	}
}

func (r *Renderer) RenderText(text *canvas.Text, m canvas.Matrix) {
	paths, colors := text.ToPaths()
	for i, path := range paths {
		style := canvas.DefaultStyle
		style.FillColor = colors[i]
		r.RenderPath(path, style, m)
	}
}

func (r *Renderer) RenderImage(img image.Image, m canvas.Matrix) {
	// add transparent margin to image for smooth borders when rotating
	margin := 4
	size := img.Bounds().Size()
	img2 := image.NewRGBA(image.Rect(0, 0, size.X+margin*2, size.Y+margin*2))
	draw.Draw(img2, image.Rect(margin, margin, size.X, size.Y), img, image.Point{}, draw.Over)

	// draw to destination image
	// note that we need to correct for the added margin in origin and m
	origin := m.Dot(canvas.Point{-float64(margin), float64(img2.Bounds().Size().Y - margin)}).Mul(float64(r.resolution))
	m = m.Scale(float64(r.resolution)*(float64(size.X+margin)/float64(size.X)), float64(r.resolution)*(float64(size.Y+margin)/float64(size.Y)))

	h := float64(r.img.Bounds().Size().Y)
	aff3 := f64.Aff3{m[0][0], -m[0][1], origin.X, -m[1][0], m[1][1], h - origin.Y}
	draw.CatmullRom.Transform(r.img, aff3, img2, img2.Bounds(), draw.Over, nil)
}
