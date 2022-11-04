package renderers

import (
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	//webp "github.com/kolesa-team/go-webp/encoder"
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
	//case ".webp":
	//	return c.WriteFile(filename, WEBP(opts...))
	case ".svgz":
		return c.WriteFile(filename, SVGZ(opts...))
	case ".svg":
		return c.WriteFile(filename, SVG(opts...))
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

func PNG(opts ...interface{}) canvas.Writer {
	resolution := canvas.DPMM(1.0)
	colorSpace := canvas.DefaultColorSpace
	for _, opt := range opts {
		switch o := opt.(type) {
		case canvas.Resolution:
			resolution = o
		case canvas.ColorSpace:
			colorSpace = o
		default:
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return png.Encode(w, img)
	}
}

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
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return jpeg.Encode(w, img, options)
	}
}

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
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return gif.Encode(w, img, options)
	}
}

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
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return tiff.Encode(w, img, options)
	}
}

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
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		img := rasterizer.Draw(c, resolution, colorSpace)
		return bmp.Encode(w, img)
	}
}

//func WEBP(opts ...interface{}) canvas.Writer {
//	options := &webp.Options{}
//	resolution := canvas.DPMM(1.0)
//	colorSpace := canvas.DefaultColorSpace
//	for _, opt := range opts {
//		switch o := opt.(type) {
//		case *webp.Options:
//			options = o
//		case canvas.Resolution:
//			resolution = o
//		case canvas.ColorSpace:
//			colorSpace = o
//		default:
//			return errorWriter(fmt.Errorf("unknown option: %v", opt))
//		}
//	}
//	return func(w io.Writer, c *canvas.Canvas) error {
//		img := rasterizer.Draw(c, resolution, colorSpace)
//		enc, err := webp.NewEncoder(img, options)
//		if err != nil {
//			return err
//		}
//		return enc.Encode(w)
//	}
//}

func SVGZ(opts ...interface{}) canvas.Writer {
	var options *svg.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *svg.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
		}
	}
	if options == nil {
		options := svg.DefaultOptions
		options.Compression = -1
		opts = append(opts, &options)
	} else {
		options.Compression = -1
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		svg := svg.New(w, c.W, c.H, options)
		c.RenderTo(svg)
		return svg.Close()
	}
}

func SVG(opts ...interface{}) canvas.Writer {
	var options *svg.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *svg.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		svg := svg.New(w, c.W, c.H, options)
		c.RenderTo(svg)
		return svg.Close()
	}
}

func PDF(opts ...interface{}) canvas.Writer {
	var options *pdf.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *pdf.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
		}
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		pdf := pdf.New(w, c.W, c.H, options)
		c.RenderTo(pdf)
		return pdf.Close()
	}
}

func TeX(opts ...interface{}) canvas.Writer {
	for _, opt := range opts {
		return errorWriter(fmt.Errorf("unknown option: %v", opt))
	}
	return func(w io.Writer, c *canvas.Canvas) error {
		tex := tex.New(w, c.W, c.H)
		c.RenderTo(tex)
		return tex.Close()
	}
}

func PS(opts ...interface{}) canvas.Writer {
	var options *ps.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *ps.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
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

func EPS(opts ...interface{}) canvas.Writer {
	var options *ps.Options
	for _, opt := range opts {
		switch o := opt.(type) {
		case *ps.Options:
			options = o
		default:
			return errorWriter(fmt.Errorf("unknown option: %v", opt))
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
