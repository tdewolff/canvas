package rasterizer

import (
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/tdewolff/canvas"
)

// PNGWriter writes the canvas as a PNG file
func PNGWriter(resolution canvas.DPMM) canvas.Writer {
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution)
		// TODO: optimization: cache img until canvas changes
		return png.Encode(w, img)
	}
}

// JPGWriter writes the canvas as a JPG file
func JPGWriter(resolution canvas.DPMM, opts *jpeg.Options) canvas.Writer {
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution)
		// TODO: optimization: cache img until canvas changes
		return jpeg.Encode(w, img, opts)
	}
}

// GIFWriter writes the canvas as a GIF file
func GIFWriter(resolution canvas.DPMM, opts *gif.Options) canvas.Writer {
	return func(w io.Writer, c *canvas.Canvas) error {
		img := Draw(c, resolution)
		// TODO: optimization: cache img until canvas changes
		return gif.Encode(w, img, opts)
	}
}
