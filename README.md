# Canvas <a name="canvas"></a> [![GoDoc](http://godoc.org/github.com/tdewolff/canvas?status.svg)](http://godoc.org/github.com/tdewolff/canvas)

Canvas is a common vector drawing target that can output SVG, PDF, EPS and raster images (which can be saved as PNG, JPG, ...).

![Example](https://raw.githubusercontent.com/tdewolff/canvas/master/example/example.png)


## Planning

General

* **Add support for easier usage of projections / viewboxes?**

Fonts

* **Embedding only used characters**
* **Font embedding for PDFs**
* **Add text rotation support**
* Support ligatures and font hinting
* Support WOFF2 font format
* Support Type1 font format?

Paths

* **Fix bounds calculation of ellipses**
* **Easier support for building paths from strings, like AppendSVG for example**
* **Fix edge-cases for length calculation of Beziérs and ellipses**
* **Approximate elliptic arcs by lines given a tolerance for use in `Flatten`**
* **Approximate elliptic arcs by Beziérs given a tolerance for use in `WriteImage`, `ToPDF`, `SplitAt` and `Dash`**
* **Introduce splitting up ellipses into partial arcs for `SplitAt` and `Dash` and remove approximating them by Beziérs**
* Introduce elliptic arc function for PDFs much like for PostScript?
* Add ArcTo in endpoint format (take begin/end angle and center point)
* Add function to convert lines to cubic Beziérs to smooth out a path
* Add offsetting of path (expand / contract), tricky with overlap
* Implement miter and arcs line join types for stroking

Optimization

* Approximate Beziérs by elliptic arcs instead of lines when stroking, if number of path elements is reduced by more than 2 times (check)
* Optimize/minify paths from and to SVG in `Optimize()` (incl. the last Line if followed by Close)
* Avoid overlapping paths when stroking, we need to know the ending and starting angle of the previous and next command respectively
* Store location of last `MoveToCmd` to optimize `direction()` and others?


## Canvas
``` go
c := canvas.New()
c.Open(width, height float64)
c.SetColor(color color.Color)
c.SetFont(fontFace Face)
c.DrawPath(x, y float64, path *Path)
c.DrawText(x, y float64, text string)

c.WriteSVG(w io.Writer)
c.WriteEPS(w io.Writer)
c.WritePDF(w io.Writer)
c.WriteImage(dpi float64) *image.RGBA
```

Canvas allows to draw either paths or text. All positions and sizes are given in millimeters.

## Fonts
``` go
dejaVuSerif, err := canvas.LoadFontFile("DejaVuSerif", canvas.Regular, "DejaVuSerif.ttf")  // TTF, OTF or WOFF

ff := dejaVuSerif.Face(size float64)
ff.Info() (name string, style FontStyle, size float64)
ff.Metrics() Metrics                           // font metrics such as line height
ff.Bounds(text string) (width, height float64) // bounding box of the text in mm, processes new lines
ff.ToPath(text string) *Path                   // convert text to path
```


## Paths
A large deal of this library implements functionality for building paths. Any path can be constructed from a few basic operations, see below. The successive commands start from the current pen position (from the previous command's end point) and are drawn towards a new end point. A path can consist of multiple path segments (multiple MoveTos), but be aware that overlapping paths will cancel each other.

``` go
p := &Path{}
p.MoveTo(x, y float64)                                            // new path segment starting at (x,y)
p.LineTo(x, y float64)                                            // straight line to (x,y)
p.QuadTo(cpx, cpy, x, y float64)                                  // a quadratic Bézier with control point (cpx,cpy) and end point (x,y)
p.CubeTo(cp1x, cp1y, cp2x, cp2y, x, y float64)                    // a cubic Bézier with control points (cp1x,cp1y), (cp2x,cp2y) and end point (x,y)
p.ArcTo(rx, ry, rot float64, largeArc, sweep bool, x, y float64)  // an arc of an ellipse with radii (rx,ry), rotated by rot (in degrees), with flags largeArc and sweep (booleans, see https://www.w3.org/TR/SVG/paths.html#PathDataEllipticalArcCommands)
p.Close()                                                         // close the path, essentially a LineTo to the last MoveTo location
```

We can extract information from these paths using:

``` go
p.Empty() bool
p.Pos() (x, y float64)       // current pen position
p.StartPos() (x, y float64)  // position of last MoveTo
p.CW() bool                  // true if the last path segment has a clockwise direction
p.CCW() bool                 // true if the last path segment has a counter clockwise direction
p.Bounds() Rect              // WIP: bounding box of path
p.Length() float64           // WIP: length of path in millimeters
p.ToSVG() string             // to SVG
p.ToPS() string              // to PostScript
```

These paths can be manipulated and transformed with the following commands. Each will return a pointer to the path.

``` go
p.Copy()
p.Append(q *Path)        // append path q to p
p.Join(q *Path)          // join path q to p
p.Split()                // split the path segments, ie. at Close/MoveTo
p.SplitAt(d ...float64)  // split the path at certain lengths d
p.Reverse()              // reverse the direction of the path

p.Translate(x, y float64)
p.Scale(x, y float64)
p.Rotate(rot, x, y float64)  // with the rotation rot in degrees, around point (x,y)

p.Flatten()                                            // flatten Bézier and arc commands to straight lines
p.Stroke(width float64, capper Capper, joiner Joiner)  // create a stroke from a path of certain width, using capper and joiner for caps and joins
p.Dash(d ...float64)                                   // create dashed path with lengths d which are alternating the dash and the space
```


## LaTeX
To generate outlines generated by LaTeX, you need `latex` and `dvisvgm` installed on your system.

``` go
p, err := ParseLaTeX(`$y=\sin\(\frac{x}{180}\pi\)$`)
if err != nil {
    panic(err)
}
```

Where the provided string gets inserted into the following document template:

``` latex
\documentclass{article}
\begin{document}
\thispagestyle{empty}
{{input}}
\end{document}
```


## Example
See https://github.com/tdewolff/canvas/tree/master/example for a working example, including fonts. Note that for PDFs you need to pre-compile fonts using `makefont` installed by `go install github.com/jung-kurt/gofpdf/makefont` and then compile them by running `makefont --embed --enc=cp1252.map DejaVuSerif.ttf`.


## License
Released under the [MIT license](LICENSE.md).
