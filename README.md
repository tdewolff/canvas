# Canvas <a name="canvas"></a> [![GoDoc](http://godoc.org/github.com/tdewolff/canvas?status.svg)](http://godoc.org/github.com/tdewolff/canvas)

Canvas is a vector drawing target that exposes a common interface for multiple drawing back-ends. It outputs SVG, PDF or raster images (which can be saved as PNG, JPG, ...).

## Interface
``` go
type C interface {
	Open(width float64, height float64)

	SetColor(color color.Color)
	SetFont(fontName string, fontSize float64) (canvas.FontFace, error)

	DrawPath(path *canvas.Path)
	DrawText(x float64, y float64, string)
}
```

The common interface allows to draw either paths or text. All positions and sizes are given in millimeters.

* The handling of fonts should be improved in the future.
* More functionality will be added to paths, such as generating strokes (ref. https://github.com/golang/freetype/pull/50).
* Text to path will be implemented.
* Simplify the ArcTo command (take begin/end angle and center point)
* Merge large-arc-flag and sweep-flag into one float64
* Optimize/minify paths from and to SVG

## Example
See https://github.com/tdewolff/canvas/tree/master/example for a working example, including fonts. Note that for PDFs you need to pre-compile fonts using `makefont` installed by `go install github.com/jung-kurt/gofpdf/makefont` and then compile them by running `makefont --embed --enc=cp1252.map DejaVuSerif.ttf`.

## License
Released under the [MIT license](LICENSE.md).
