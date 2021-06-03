package renderers

import (
	"fmt"
	"image/gif"
	"image/jpeg"
	"io"
	"path/filepath"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/pdf"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	"github.com/tdewolff/canvas/renderers/svg"
	"github.com/tdewolff/canvas/renderers/tex"
	"golang.org/x/image/tiff"
)

const mmPerPt = 25.4 / 72.0
const ptPerMm = 72.0 / 25.4
const mmPerPx = 25.4 / 96.0

type Options struct {
	canvas.Resolution
	JPG  *jpeg.Options
	GIF  *gif.Options
	TIFF *tiff.Options
	SVG  *svg.Options
	PDF  *pdf.Options
}

func Write(filename string, c *canvas.Canvas, opts ...interface{}) error {
	options := Options{
		Resolution: 1.0,
	}
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			options.Resolution = o
		case *jpeg.Options:
			options.JPG = o
		case *gif.Options:
			options.GIF = o
		case *tiff.Options:
			options.TIFF = o
		case *svg.Options:
			options.SVG = o
		case *pdf.Options:
			options.PDF = o
		default:
			return fmt.Errorf("unknown option: %v", opt)
		}
	}

	switch ext := strings.ToLower(filepath.Ext(filename)); ext {
	case ".png":
		return c.WriteFile(filename, rasterizer.PNGWriter(options.Resolution))
	case ".jpg", ".jpeg":
		return c.WriteFile(filename, rasterizer.JPGWriter(options.Resolution, options.JPG))
	case ".gif":
		return c.WriteFile(filename, rasterizer.GIFWriter(options.Resolution, options.GIF))
	case ".tif", ".tiff":
		return c.WriteFile(filename, rasterizer.TIFFWriter(options.Resolution, options.TIFF))
	case ".svg", ".svgz":
		if ext == ".svgz" && options.SVG.Compression == 0 {
			options.SVG.Compression = -1
		}
		return c.WriteFile(filename, func(w io.Writer, c *canvas.Canvas) error {
			svg := svg.New(w, c.W, c.H, options.SVG)
			c.Render(svg)
			return svg.Close()
		})
	case ".pdf":
		return c.WriteFile(filename, func(w io.Writer, c *canvas.Canvas) error {
			pdf := pdf.New(w, c.W, c.H, options.PDF)
			c.Render(pdf)
			return pdf.Close()
		})
	case ".tex", ".pgf":
		return c.WriteFile(filename, tex.Writer)
	default:
		return fmt.Errorf("unknown file extension: %v", ext)
	}
	return nil
}
