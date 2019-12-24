package canvas

import (
	"image"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/vector"
)

func (l pathLayer) DrawImage(img draw.Image, dpm float64) {
	// TODO: use fill rule (EvenOdd, NonZero) for rasterizer
	w, h := img.Bounds().Size().X, img.Bounds().Size().Y
	if l.fillColor.A != 0 {
		ras := vector.NewRasterizer(w, h)
		l.path.ToRasterizer(ras, dpm)
		size := ras.Size()
		ras.Draw(img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(l.fillColor), image.Point{})
	}
	if l.strokeColor.A != 0 && 0.0 < l.strokeWidth {
		strokePath := l.path
		if 0 < len(l.dashes) {
			strokePath = strokePath.Dash(l.dashOffset, l.dashes...)
		}
		strokePath = strokePath.Stroke(l.strokeWidth, l.strokeCapper, l.strokeJoiner)

		ras := vector.NewRasterizer(w, h)
		strokePath.ToRasterizer(ras, dpm)
		size := ras.Size()
		ras.Draw(img, image.Rect(0, 0, size.X, size.Y), image.NewUniform(l.strokeColor), image.Point{})
	}
}

func (l textLayer) DrawImage(img draw.Image, dpm float64) {
	paths, colors := l.text.ToPaths()
	for i, path := range paths {
		style := defaultStyle
		style.fillColor = colors[i]
		pathLayer{path.Transform(l.m), style, false}.DrawImage(img, dpm)
	}
}

func (l imageLayer) DrawImage(img draw.Image, dpm float64) {
	m := l.m.Scale(dpm, dpm)
	h := float64(img.Bounds().Size().Y)
	origin := l.m.Dot(Point{0, float64(l.img.Bounds().Size().Y)})
	aff3 := f64.Aff3{m[0][0], -m[0][1], origin.X * dpm, -m[1][0], m[1][1], h - origin.Y*dpm}
	draw.CatmullRom.Transform(img, aff3, l.img, l.img.Bounds(), draw.Over, nil)
}
