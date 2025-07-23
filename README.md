![Canvas](https://raw.githubusercontent.com/tdewolff/canvas/master/resources/title/title.png)

[![API reference](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/tdewolff/canvas?tab=doc) [![User guide](https://img.shields.io/badge/user-guide-5272B4)](https://github.com/tdewolff/canvas/wiki) [![Go Report Card](https://goreportcard.com/badge/github.com/tdewolff/canvas)](https://goreportcard.com/report/github.com/tdewolff/canvas) [![Coverage Status](https://coveralls.io/repos/github/tdewolff/canvas/badge.svg?branch=master)](https://coveralls.io/github/tdewolff/canvas?branch=master)

**[API documentation](https://pkg.go.dev/github.com/tdewolff/canvas?tab=doc)**

**[User guide](https://github.com/tdewolff/canvas/wiki)**

**[Live HTMLCanvas demo](https://tdewolff.github.io/canvas/examples/html-canvas/index.html)**

Canvas is a common vector drawing target that can output SVG, PDF, EPS, raster images (PNG, JPG, GIF, ...), HTML Canvas through WASM, OpenGL, and Gio. It has a wide range of path manipulation functionality such as flattening, stroking and dashing implemented. Additionally, it has a text formatter and embeds and subsets fonts (TTF, OTF, WOFF, WOFF2, or EOT) or converts them to outlines. It can be considered a Cairo or node-canvas alternative in Go. See the example below in Figure 1 for an overview of the functionality.

![Preview](https://raw.githubusercontent.com/tdewolff/canvas/master/resources/preview/preview.png)

**Figure 1**: top-left you can see text being fitted into a box, justified using Donald Knuth's linea breaking algorithm to stretch the spaces between words to fill the whole width. You can observe a variety of styles and text decorations applied, as well as support for LTR/RTL mixing and complex scripts. In the bottom-right the word "stroke" is being stroked and drawn as a path. Top-right we see a LaTeX formula that has been converted to a path. Left of that we see an ellipse showcasing precise dashing, notably the length of e.g. the short dash is equal wherever it is on the curve. Note that the dashes themselves are elliptical arcs as well (thus exactly precise even if magnified greatly). To the right we see a closed polygon of four points being smoothed by cubic Béziers that are smooth along the whole path, and the blue line on the left shows a smoothed open path. On the bottom you can see a rotated rasterized image. The bottom-left shows path boolean operations. The result is equivalent for all renderers (PNG, PDF, SVG, etc.).

### Recent changes
- `RichText.ToText` replaces some parameters by `*TextOptions`, use `rt.ToText(..., &canvas.TextOptions{Indent: indent, LineStretch: lineStretch})` or if `indent` and `lineStretch` are zero use `rt.ToText(..., nil)` for the original result.
- `NewTextBox` replaces some parameters by `*TextOptions`, use `rt.ToText(..., &canvas.TextOptions{Indent: indent, LineStretch: lineStretch})` or if `indent` and `lineStretch` are zero use `rt.ToText(..., nil)` for the original result.

### Sponsors
I'm actively looking for support in the form of donations or sponsorships to keep developing this library and highly appreciate any gesture. Please see the Sponsors button in GitHub for ways to contribute, or contact me directly.

## State
Whether this library is ready for production environments is up to your own judgment. In general, this library is written thoughtfully and complete, but the scope of this work is so big and the implementation can be quite complex that inevitably it must have a great amount of bugs. Effort was put in writing unit and fuzz tests so that I suspect only special use-cases will stumble into bugs, but coverage is still lacking. As time permits, work is done to flesh-out functionality, find bugs, and optimize code. Optimization could be in execution time / reducing code complexity, reducing memory footprint, or reducing the length of paths from operation.

Execution performance is actually really good, especially the rasterizer is highly optimized with ASM. See for example a comparison of an extreme case in https://github.com/tdewolff/canvas/issues/280#issuecomment-1995990038, where this library is at least twice as fast as existing solutions, and can handle bigger images than the likes of Inkscape and Cairo.

The path intersection code and path boolean operation code is quite complete and fast, and more importantly has a time complexity of O(n log n). It is numerically stable and does not suffer from floating-point precision errors.

Please issue bug reports or feature requests to help this library mature! All help is appreciated. Also see [Wiki - Planning](https://github.com/tdewolff/canvas/wiki/Planning) for an inexhaustive list of ideas and TODOs.

## Features
### General
- Path segment types: MoveTo, LineTo, QuadTo, CubeTo, ArcTo, Close (see [Paths](https://github.com/tdewolff/canvas/wiki/Paths))
- Precise path flattening, stroking, and dashing for all segment types (see papers below)
- Smooth spline generation through points for open and closed paths
- LaTeX to path conversion (native Go and CGO implementations available)
- sRGB compliance (use `SRGBColorSpace`, only available for rasterizer)

### Rendering targets
Paths can be exported as or rendered to:
- Raster images (PNG, GIF, JPEG, TIFF, BMP, WEBP, AVIF, ...)
- PDF
- SVG and SVGZ
- PS and EPS
- HTMLCanvas
- OpenGL
- [Gio](https://gioui.org/)
- [Fyne](https://fyne.io/)

Additionally, it has bindings to be used as renderer for:
- [go-chart](https://github.com/wcharczuk/go-chart)
- [gonum/plot](https://github.com/gonum/plot)

See [Renderers](https://github.com/tdewolff/canvas/wiki/Renderers) for more information.

### Stable path boolean operations
Numerically stable (!) path boolean operations, supporting AND, OR, XOR, NOT, and DIV operations in `O((n+k) log n)`, with `n` the number of segments and `k` the number of intersections. This is very fast and allows handling huge paths. It uses 64-bit floating-point precision for highly accurate computation and employs an additional strategy to ensure numerical stability. In particular:
- Allows paths, subject or clipping, with any number of (overlapping) contours.
- Allows contours with any orientation, clockwise or anticlockwise.
- Contours may be concave or of any shape.
- Contours may self-intersect any number of times.
- Segments may overlap any number of times by any contour.
- Points may be crossed any number of times.
- Segments may be vertical.
- Clipping path is implicitly closed (it makes no sense if it's an open path).
- Subject path may be either open or closed.
- Paths are currently flattened, but supporting Bézier or elliptical arcs is a WIP (not anytime soon).

Numerical stability refers to cases where two segments are extremely close where floating-point precision can alter the computation whether they intersect or not. This is a very difficult problem to solve, and many libraries cannot handle this properly (nor can they handle 'degenerate' paths in general, see the list of properties above). Note that fixed-point precision suffers from the same problem. This library builds on papers from Bentley & Ottmann, de Berg, Martínez, Hobby, and Hershberger (see bibliography below).

Correctness and performance has been tested by drawing all land masses and islands from OpenStreetMap at various scales, which is a huge input (1 GB of compressed Shape files) with extremely degenerate data (many overlapping segments, overlapping points, vertical segments, self-intersections, extremely close intersections, different contour orientations, and so on).

TODO: add benchmark with other libraries

See [Boolean operations](https://github.com/tdewolff/canvas/wiki/Boolean-operations) for more information.

### Advanced text rendering
High-quality (comparable to TeX) text rendering and line breaking. It uses HarfBuzz for text shaping (native Go and CGO implementations available) and FriBidi for text bidirectionality (native Go and CGO implementations available), and uses Donald Knuth's line breaking algorithm for text layout. This enables the following features:
- Align text left, center, right and justified, including indentation.
- Align text top, middle, bottom in a text box, including setting line spacing.
- Handle any script, eg. latin, cyrillic, devanagari, arabic, hebrew, han, etc.
- Handle left-to-right, right-to-left, or top-to-bottom/bottom-to-top writing systems.
- Mix scripts and fonts in a single line, eg. combine latin and arabic, or bold and regular styles.

Additionally, many font formats are supported (such as TTF, OTF, WOFF, WOFF2, EOT) and rendering can apply a gamma correction of 1.43 for better results. See [Fonts & Text](https://github.com/tdewolff/canvas/wiki/Fonts-&-Text) for more information.

## Examples

**[Amsterdam city centre](https://github.com/tdewolff/canvas/tree/master/examples/amsterdam-centre)**: the centre of Amsterdam is drawn from data loaded from the Open Street Map API.

**[Mauna-Loa CO2 concentration](https://github.com/tdewolff/canvas/tree/master/examples/co2-mauna-loa)**: using data from the Mauna-Loa observatory, carbon dioxide concentrations over time are drawn

**[Text document](https://github.com/tdewolff/canvas/tree/master/examples/text-document)**: an example of a text document using the PDF backend.

**[OpenGL](https://github.com/tdewolff/canvas/tree/master/examples/opengl)**: an example using the OpenGL backend.

**[Gio](https://github.com/tdewolff/canvas/tree/master/examples/gio)**: an example using the Gio backend.

**[Fyne](https://github.com/tdewolff/canvas/tree/master/examples/fyne)**: an example using the Fyne backend.

**[TeX/PGF](https://github.com/tdewolff/canvas/tree/master/examples/tex)**: an example showing the usage of the PGF (TikZ) LaTeX package as renderer in order to generated a PDF using LaTeX.

**[go-chart](https://github.com/tdewolff/canvas/tree/master/examples/go-chart)**: an example using the [go-chart](https://github.com/wcharczuk/go-chart) library, plotting a financial graph.

**[gonum/plot](https://github.com/tdewolff/canvas/tree/master/examples/gonum-plot)**: an example using the [gonum/plot](https://github.com/gonum/plot) library.

**[HTMLCanvas](https://github.com/tdewolff/canvas/tree/master/examples/html-canvas)**: an example using the HTMLCanvas backend, see the [live demo](https://tdewolff.github.io/canvas/examples/html-canvas/index.html).

## Users

This is a non-exhaustive list of library users I've come across. PRs are welcome to extend the list!

- https://github.com/aldernero/gaul (generative art utility library)
- https://github.com/aldernero/sketchy (generative art framework)
- https://github.com/carbocation/genomisc (genomics tools)
- https://github.com/davidhampgonsalves/life-dashboard (show text and emoticons in Kindle)
- https://github.com/davidhampgonsalves/quickdraw (grid of Google Quick Draw Drawings)
- https://github.com/dotaspirit/dotaspirit (draw Dota match data: https://vk.com/rsltdtk)
- https://github.com/engelsjk/go-annular (generative art of annular rings)
- https://github.com/eukarya-inc/reearth-plateauview
- https://github.com/holedaemon/gopster (Topster port)
- https://github.com/html2any/layout (flex layout)
- https://github.com/iand/genster (family trees)
- https://github.com/jansorg/marketplace-stats (reports for JetBrains marketplace)
- https://github.com/kpym/marianne (French republic logo)
- https://github.com/mrmelon54/favicon (Favicon generator)
- https://github.com/namsor/go-qrcode (QR code encoder)
- https://github.com/octohelm/gio-compose (UI component solution for Gio)
- https://github.com/omniskop/vitrum (GUI framework)
- https://github.com/Pavel7004/GraphPlot (plot graphs)
- https://github.com/peteraba/roadmapper (tracking roadmaps: https://rdmp.app/)
- https://github.com/Preston-PLB/choRenderer (render chord charts in propresenter7)
- https://github.com/stv0g/vand (camper/van monitor and control)
- https://github.com/uncopied/chirograph (barcode security for art)
- https://github.com/uncopied/go-qrcode (QR code encoder)
- https://github.com/wisepythagoras/gis-utils (GIS utilities)
- https://supertxt.net/git/st-int.html (SuperTXT integrations)
- https://github.com/kenshaw/fv (Command-line font viewer using terminal graphics)

## Articles

- [Numerically stable quadratic formula](https://math.stackexchange.com/questions/866331/numerically-stable-algorithm-for-solving-the-quadratic-equation-when-a-is-very/2007723#2007723)
- [Quadratic Bézier length](https://malczak.linuxpl.com/blog/quadratic-bezier-curve-length/)
- [Bézier spline through open path](https://www.particleincell.com/2012/bezier-splines/)
- [Bézier spline through closed path](https://www.jacos.nl/jacos_html/spline/circular/index.html)
- [Point inclusion in polygon test](https://wrf.ecse.rpi.edu/Research/Short_Notes/pnpoly.html)

#### My own

- [Arc length parametrization](https://tacodewolff.nl/posts/20190525-arc-length/)

#### Papers

- [M. Walter, A. Fournier, "Approximate Arc Length Parametrization", Anais do IX SIBGRAPHI (1996), p. 143--150](https://www.visgraf.impa.br/sibgrapi96/trabs/pdf/a14.pdf)
- [T.F. Hain, et al., "Fast, precise flattening of cubic Bézier path and offset curves", Computers & Graphics 29 (2005). p. 656--666](https://doi.org/10.1016/j.cag.2005.08.002)
- [M. Goldapp, "Approximation of circular arcs by cubic polynomials", Computer Aided Geometric Design 8 (1991), p. 227--238](https://doi.org/10.1016/0167-8396%2891%2990007-X)
- [L. Maisonobe, "Drawing and elliptical arc using polylines, quadratic or cubic Bézier curves" (2003)](https://spaceroots.org/documents/ellipse/elliptical-arc.pdf)
- [S.H. Kim and Y.J. Ahn, "An approximation of circular arcs by quartic Bezier curves", Computer-Aided Design 39 (2007, p. 490--493)](https://doi.org/10.1016/j.cad.2007.01.004)
- D.E. Knuth and M.F. Plass, Breaking Paragraphs into Lines, Software: Practice and Experience 11 (1981), p. 1119--1184
- L. Subramaniam, Partition of a non-simple polygon into simple polygons, 2003
- [F. Martínez, et al., "A simple algorithm for Boolean operations on polygons"](https://doi.org/10.1016/j.advengsoft.2013.04.004)
- [J.D. Hobby, "Practical segment intersection with ﬁnite precision output", Computational Geometry (1997)](https://doi.org/10.1016/S0925-7721%2899%2900021-8)
- [J. Hershberger, "Stable snap rounding", Computational Geometry: Theory and Applications, 2013](https://doi.org/10.1016/j.comgeo.2012.02.011)

## License

Released under the [MIT license](LICENSE.md).

Be aware that Fribidi uses the LGPL license.
