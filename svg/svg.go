package svg

import (
	"image"
	"image/color"
	"io"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

func init() {
	image.RegisterFormat("svg", "<svg", func(r io.Reader) (image.Image, error) {
		c, err := canvas.ParseSVG(r)
		if err != nil {
			return nil, err
		}
		return rasterizer.Draw(c, 96.0, canvas.DefaultColorSpace), nil
	}, func(r io.Reader) (image.Config, error) {
		c, err := canvas.ParseSVG(r)
		if err != nil {
			return image.Config{}, err
		}
		return image.Config{
			ColorModel: color.RGBAModel,
			Width:      int(c.W),
			Height:     int(c.H),
		}, nil
	})
}
