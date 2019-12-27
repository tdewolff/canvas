package canvas

import (
	"image"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/vector"
)

type rasterizer struct {
	*Context
	img draw.Image
	dpm float64
}

func Rasterizer(img draw.Image, dpm float64) *rasterizer {
	r := &rasterizer{
		img: img,
		dpm: dpm,
	}
	size := img.Bounds().Size()
	r.Context = newContext(r, float64(size.X)/dpm, float64(size.Y)/dpm)
	return r
}

func (r *rasterizer) renderPath(path *Path, style Style, m Matrix) {
	// TODO: use fill rule (EvenOdd, NonZero) for rasterizer
	path = path.Transform(m)
	size := r.img.Bounds().Size()
	if style.FillColor.A != 0 {
		ras := vector.NewRasterizer(size.X, size.Y)
		path.ToRasterizer(ras, r.dpm)
		ras.Draw(r.img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(style.FillColor), image.Point{})
	}
	if style.StrokeColor.A != 0 && 0.0 < style.StrokeWidth {
		if 0 < len(style.Dashes) {
			path = path.Dash(style.DashOffset, style.Dashes...)
		}
		path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)

		ras := vector.NewRasterizer(size.X, size.Y)
		path.ToRasterizer(ras, r.dpm)
		ras.Draw(r.img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(style.StrokeColor), image.Point{})
	}
}

func (r *rasterizer) renderText(text *Text, m Matrix) {
	paths, colors := text.ToPaths()
	for i, path := range paths {
		style := DefaultStyle
		style.FillColor = colors[i]
		r.renderPath(path, style, m)
	}
}

func (r *rasterizer) renderImage(img image.Image, m Matrix) {
	origin := m.Dot(Point{0, float64(img.Bounds().Size().Y)}).Mul(r.dpm)
	m = m.Scale(r.dpm, r.dpm)

	h := float64(r.img.Bounds().Size().Y)
	aff3 := f64.Aff3{m[0][0], -m[0][1], origin.X, -m[1][0], m[1][1], h - origin.Y}
	draw.CatmullRom.Transform(r.img, aff3, img, img.Bounds(), draw.Over, nil)
}
