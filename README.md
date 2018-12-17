# Canvas <a name="canvas"></a> [![GoDoc](http://godoc.org/github.com/tdewolff/canvas?status.svg)](http://godoc.org/github.com/tdewolff/canvas)

Canvas is a vector drawing target that exposes a common interface for multiple drawing back-ends. It outputs SVG, PDF or raster images (which can be saved as PNG, JPG, ...).

## Interface
``` go
type C interface {
	Open(width float64, height float64)

	SetColor(color color.Color)
	SetFont(fontName string, fontSize float64) (canvas.FontFace, error)

	DrawPath(x, y float64, path *canvas.Path)
	DrawText(x, y float64, text string)
}
```

The common interface allows to draw either paths or text. All positions and sizes are given in millimeters.

* Get rid of FontFace and pass font size for all function calls?
* Support WOFF and WOFF2 font formats
* Add path IsCW / IsCCW
* Add ArcTo in endpoint format (take begin/end angle and center point)
* Add path length calculation
* Add path splitting at lengths -> support converting path into dashes and spacings
* Add offsetting of path (expand / contract), tricky with overlap
* Add support for easier usage of projections / viewboxes?
* Support partial fonts with only used characters to optimize SVG/PDF file size
* Optimize/minify paths from and to SVG
* Optimize paths by replacing Quad/Cube/Arc to line if they are linear (eg. p0=p1=p2 for cubic Bezier)
* Optimize paths by removing the last Line if followed by Close

## Example
See https://github.com/tdewolff/canvas/tree/master/example for a working example, including fonts. Note that for PDFs you need to pre-compile fonts using `makefont` installed by `go install github.com/jung-kurt/gofpdf/makefont` and then compile them by running `makefont --embed --enc=cp1252.map DejaVuSerif.ttf`.

## License
Released under the [MIT license](LICENSE.md).
