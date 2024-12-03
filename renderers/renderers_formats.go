//go:build formats

package renderers

import (
	"fmt"
	"io"

	"github.com/Kagami/go-avif"
	webp "github.com/kolesa-team/go-webp/encoder"
	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers/rasterizer"
)

func WebP(opts ...interface{}) canvas.Writer {
	resolution := canvas.DPMM(1.0)
	colorSpace := canvas.DefaultColorSpace
	var options *webp.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			resolution = o
		case canvas.ColorSpace:
			colorSpace = o
		case *webp.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown WebP option: %T(%v)", opt, opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		enc, err := webp.NewEncoder(img, options)
		if err != nil {
			return err
		}
		return enc.Encode(w)
	}
}

func AVIF(opts ...interface{}) canvas.Writer {
	resolution := canvas.DPMM(1.0)
	colorSpace := canvas.DefaultColorSpace
	var options *avif.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			resolution = o
		case canvas.ColorSpace:
			colorSpace = o
		case *avif.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown AVIF option: %T(%v)", opt, opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return avif.Encode(w, img, options)
	}
}
