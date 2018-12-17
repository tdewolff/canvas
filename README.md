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

### TODO

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

## Paths
A large deal of this library implements functionality for building paths. Any path can be constructed from a few basic operations, see below. The successive commands start from the current pen position (from the previous command's end point) and are draw towards a new end point. A path can consist of multiple path segments (multiple MoveTos), but be aware that overlapping paths will cancel each other.

``` go
p := &Path{}
p.MoveTo(x, y) // new path segment starting at (x,y)
p.LineTo(x, y) // straight line to (x,y)
p.QuadTo(cpx, cpy, x, y) // a quadratic Bézier with control point (cpx,cpy) and end point (x,y)
p.CubeTo(cp1x, cp1y, cp2x, cp2y, x, y) // a cubic Bézier with control points (cp1x,cp1y), (cp2x,cp2y) and end point (x,y)
p.ArcTo(rx, ry, rot, largeArc, sweep, x, y) // an arc of an ellipse with radii (rx,ry), rotated by rot (in degrees), with flags largeArc and sweep (booleans, see https://www.w3.org/TR/SVG/paths.html#PathDataEllipticalArcCommands)
p.Close() // close the path, essentially a LineTo to the last MoveTo location
```

We can extract information from these paths using:

``` go
p.Empty() bool // returns boolean
p.Pos() (x float64, y float64) // current pen position
p.StartPos() (x0 float64, y0 float64) // position of last MoveTo
p.String() string // to SVG path
p.Bounds() Rect // bounding box of path
p.Length() float64 // length of path in millimeters
```

These paths can be manipulated and transformed with the following commands. Each will return a pointer to the path.

``` go
p.Copy()
p.Append(q) // append path q to p
p.Split() // split the path segments, ie. at Close/MoveTo
p.Reverse() // reverse the direction of the path
p.Flatten(tolerance) // flatten Bézier and arc commands to straight lines, with a maximum deviation of tolarance
p.FlattenArcs(tolerance)
p.FlattenBeziers(tolerance)

p.Translate(x, y)
p.Scale(x, y)
p.Rotate(rot, x, y) // with the rotation rot in degrees, around point (x,y)

p.Stroke(width, capper, joiner, tolerance) // create a stroke from a path of certain width, using capper and joiner for caps and joins
```

## Example
See https://github.com/tdewolff/canvas/tree/master/example for a working example, including fonts. Note that for PDFs you need to pre-compile fonts using `makefont` installed by `go install github.com/jung-kurt/gofpdf/makefont` and then compile them by running `makefont --embed --enc=cp1252.map DejaVuSerif.ttf`.

![Example][https://raw.githubusercontent.com/tdewolff/canvas/master/example/example.png]

## License
Released under the [MIT license](LICENSE.md).
