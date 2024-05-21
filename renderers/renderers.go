package renderers

import (
	"compress/flate"
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/pdf"
	"github.com/tdewolff/canvas/renderers/ps"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	"github.com/tdewolff/canvas/renderers/svg"
	"github.com/tdewolff/canvas/renderers/tex"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

const mmPerPt = 25.4 / 72.0
const ptPerMm = 72.0 / 25.4
const mmPerPx = 25.4 / 96.0

// Write renders the canvas and writes to a file. A renderer is chosen based on the filename extension. The options will be passed to the respective renderer. Supported extensions: .(png|jpe?g|gif|tiff?|bmp|webp|avif|svgz?|pdf|tex|pgf|ps|eps).
func Write(filename string, c *canvas.Canvas, opts ...interface{}) error {
	switch ext := strings.ToLower(filepath.Ext(filename)); ext {
	case ".png":
		return c.WriteFile(filename, PNG(opts...))
	case ".jpg", ".jpeg":
		return c.WriteFile(filename, JPEG(opts...))
	case ".gif":
		return c.WriteFile(filename, GIF(opts...))
	case ".tif", ".tiff":
		return c.WriteFile(filename, TIFF(opts...))
	case ".bmp":
		return c.WriteFile(filename, BMP(opts...))
	case ".webp":
		return c.WriteFile(filename, WebP(opts...))
	case ".avif":
		return c.WriteFile(filename, AVIF(opts...))
	case ".svg":
		return c.WriteFile(filename, SVG(opts...))
	case ".svgz":
		return c.WriteFile(filename, SVGZ(opts...))
	case ".pdf":
		return c.WriteFile(filename, PDF(opts...))
	case ".tex", ".pgf":
		return c.WriteFile(filename, TeX(opts...))
	case ".ps":
		return c.WriteFile(filename, PS(opts...))
	case ".eps":
		return c.WriteFile(filename, EPS(opts...))
	default:
		return fmt.Errorf("unknown file extension: %v", ext)
	}
	return nil
}

func errorWriter(err error) canvas.Writer {
	return func(w io.Writer, c *canvas.Canvas) error {
		return err
	}
}

// PNG returns a PNG writer and accepts the following options: canvas.Resolution, canvas.Colorspace, image/png.Encoder
func PNG(opts ...interface{}) canvas.Writer {
	resolution := canvas.DPMM(1.0)
	colorSpace := canvas.DefaultColorSpace
	encoder := png.Encoder{}
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			resolution = o
		case canvas.ColorSpace:
			colorSpace = o
		case png.Encoder:
			encoder = o
		default:
			return errorWriter(fmt.Errorf("unknown PNG option: %T(%v)", opt, opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return encoder.Encode(w, img)
	}
}

// JPEG returns a JPEG writer and accepts the following options: canvas.Resolution, canvas.Colorspace, image/jpeg.*Options
func JPEG(opts ...interface{}) canvas.Writer {
	resolution := canvas.DPMM(1.0)
	colorSpace := canvas.DefaultColorSpace
	var options *jpeg.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			resolution = o
		case canvas.ColorSpace:
			colorSpace = o
		case *jpeg.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown JPEG option: %T(%v)", opt, opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return jpeg.Encode(w, img, options)
	}
}

// GIF returns a GIF writer and accepts the following options: canvas.Resolution, canvas.Colorspace, image/gif.*Options
func GIF(opts ...interface{}) canvas.Writer {
	resolution := canvas.DPMM(1.0)
	colorSpace := canvas.DefaultColorSpace
	var options *gif.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			resolution = o
		case canvas.ColorSpace:
			colorSpace = o
		case *gif.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown GIF option: %T(%v)", opt, opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return gif.Encode(w, img, options)
	}
}

// TIFF returns a TIFF writer and accepts the following options: canvas.Resolution, canvas.Colorspace, golang.org/x/image/tiff.*Options
func TIFF(opts ...interface{}) canvas.Writer {
	resolution := canvas.DPMM(1.0)
	colorSpace := canvas.DefaultColorSpace
	var options *tiff.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			resolution = o
		case canvas.ColorSpace:
			colorSpace = o
		case *tiff.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown TIFF option: %T(%v)", opt, opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return tiff.Encode(w, img, options)
	}
}

// BMP returns a BMP writer and accepts the following options: canvas.Resolution, canvas.Colorspace
func BMP(opts ...interface{}) canvas.Writer {
	resolution := canvas.DPMM(1.0)
	colorSpace := canvas.DefaultColorSpace
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			resolution = o
		case canvas.ColorSpace:
			colorSpace = o
		default:
			return errorWriter(fmt.Errorf("unknown BMP option: %T(%v)", opt, opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return bmp.Encode(w, img)
	}
}

// SVG returns an SVG writer and accepts the following options: canvas/renderers/svg.*Options
func SVG(opts ...interface{}) canvas.Writer {
	var options *svg.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *svg.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown SVG option: %T(%v)", opt, opt))
		}
	}
	if options != nil && options.Compression != 0 {
		options.Compression = 0
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		svg := svg.New(w, c.W, c.H, options)
		c.RenderTo(svg)
		return svg.Close()
	}
}

// SVGZ returns a GZIP compressed SVG writer and accepts the following options: canvas/renderers/svgsvg.*Options
func SVGZ(opts ...interface{}) canvas.Writer {
	var options *svg.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *svg.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown SVGZ option: %T(%v)", opt, opt))
		}
	}
	if options == nil {
		options := svg.DefaultOptions
		options.Compression = flate.DefaultCompression
		opts = append(opts, &options)
	} else if options.Compression < -2 || options.Compression == 0 || 9 < options.Compression {
		options.Compression = flate.DefaultCompression
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		svg := svg.New(w, c.W, c.H, options)
		c.RenderTo(svg)
		return svg.Close()
	}
}

// PDF returns a PDF writer and accepts the following options: canvas/renderers/pdf.*Options
func PDF(opts ...interface{}) canvas.Writer {
	var options *pdf.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *pdf.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown PDF option: %T(%v)", opt, opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		pdf := pdf.New(w, c.W, c.H, options)
		c.RenderTo(pdf)
		return pdf.Close()
	}
}

// TeX returns a TeX writer.
func TeX(opts ...interface{}) canvas.Writer {
	for _, opt := range opts {
		return errorWriter(fmt.Errorf("unknown TeX option: %T(%v)", opt, opt))
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		tex := tex.New(w, c.W, c.H)
		c.RenderTo(tex)
		return tex.Close()
	}
}

// PS returns a PostScript writer and accepts the following options: canvas/renderers/ps.*Options
func PS(opts ...interface{}) canvas.Writer {
	var options *ps.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *ps.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown PS option: %T(%v)", opt, opt))
		}
	}
	if options == nil {
		defaultOptions := ps.DefaultOptions
		options = &defaultOptions
	}
	options.Format = ps.PostScript
	return func(w io.Writer, c *canvas.Canvas) error {
		ps := ps.New(w, c.W, c.H, options)
		c.RenderTo(ps)
		return ps.Close()
	}
}

// EPS returns a Encapsulated PostScript writer and accepts the following options: canvas/renderers/ps.*Options
func EPS(opts ...interface{}) canvas.Writer {
	var options *ps.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *ps.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown EPS option: %T(%v)", opt, opt))
		}
	}
	if options == nil {
		defaultOptions := ps.DefaultOptions
		options = &defaultOptions
	}
	options.Format = ps.EncapsulatedPostScript
	return func(w io.Writer, c *canvas.Canvas) error {
		ps := ps.New(w, c.W, c.H, options)
		c.RenderTo(ps)
		return ps.Close()
	}
}
