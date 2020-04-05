package eps

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/minify/v2"
)

var psEllipseDef = `/ellipse {
/rot exch def
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
rot rotate
xrad yrad scale
0 0 1 startangle endangle arc
savematrix setmatrix
} def /ellipsen {
/rot exch def
/endangle exch def
/startangle exch def
/yrad exch def
/xrad exch def
/y exch def
/x exch def
/savematrix matrix currentmatrix def
x y translate
rot rotate
xrad yrad scale
0 0 1 startangle endangle arcn
savematrix setmatrix
} def`

type Renderer struct {
	w             io.Writer
	width, height float64
	color         color.RGBA
}

// New creates an encapsulated PostScript renderer.
func New(w io.Writer, width, height float64) *Renderer {
	fmt.Fprintf(w, "%%!PS-Adobe-3.0 EPSF-3.0\n%%%%BoundingBox: 0 0 %v %v\n", dec(width), dec(height))
	fmt.Fprintf(w, psEllipseDef)
	// TODO: (EPS) generate and add preview

	return &Renderer{
		w:      w,
		width:  width,
		height: height,
		color:  canvas.Black,
	}
}

func (r *Renderer) setColor(color color.RGBA) {
	if color != r.color {
		fmt.Fprintf(r.w, " %v %v %v setrgbcolor", dec(float64(color.R)/255.0), dec(float64(color.G)/255.0), dec(float64(color.B)/255.0))
		r.color = color
	}
}

func (r *Renderer) Size() (float64, float64) {
	return r.width, r.height
}

func (r *Renderer) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	// TODO: (EPS) test ellipse, rotations etc
	// TODO: (EPS) add drawState support
	// TODO: (EPS) use dither to fake transparency
	r.setColor(style.FillColor)
	r.w.Write([]byte(" "))
	r.w.Write([]byte(path.Transform(m).ToPS()))
	r.w.Write([]byte(" fill"))
}

func (r *Renderer) RenderText(text *canvas.Text, m canvas.Matrix) {
	// TODO: (EPS) write text natively
	paths, colors := text.ToPaths()
	for i, path := range paths {
		style := canvas.DefaultStyle
		style.FillColor = colors[i]
		r.RenderPath(path, style, m)
	}
}

func (r *Renderer) RenderImage(img image.Image, m canvas.Matrix) {
	// TODO: (EPS) write image
}

type dec float64

func (f dec) String() string {
	s := fmt.Sprintf("%.*f", canvas.Precision, f)
	s = string(minify.Decimal([]byte(s), canvas.Precision))
	if dec(math.MaxInt32) < f || f < dec(math.MinInt32) {
		if i := strings.IndexByte(s, '.'); i == -1 {
			s += ".0"
		}
	}
	return s
}
