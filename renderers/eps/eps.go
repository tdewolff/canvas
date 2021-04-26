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

// Writer writes the canvas as an EPS file.
func Writer(w io.Writer, c *canvas.Canvas) error {
	eps := New(w, c.W, c.H)
	c.Render(eps)
	return nil
}

// EPS is an encasulated PostScript renderer. Be aware that EPS does not support transparency of colors.
type EPS struct {
	w             io.Writer
	width, height float64
	color         color.RGBA
}

// New returns an encapsulated PostScript (EPS) renderer.
func New(w io.Writer, width, height float64) *EPS {
	fmt.Fprintf(w, "%%!PS-Adobe-3.0 EPSF-3.0\n%%%%BoundingBox: 0 0 %v %v\n", dec(width), dec(height))
	fmt.Fprint(w, psEllipseDef)
	// TODO: (EPS) generate and add preview

	return &EPS{
		w:      w,
		width:  width,
		height: height,
		color:  canvas.Black,
	}
}

func (r *EPS) setColor(color color.RGBA) {
	if color != r.color {
		fmt.Fprintf(r.w, " %v %v %v setrgbcolor", dec(float64(color.R)/255.0), dec(float64(color.G)/255.0), dec(float64(color.B)/255.0))
		r.color = color
	}
}

// Size returns the size of the canvas in millimeters.
func (r *EPS) Size() (float64, float64) {
	return r.width, r.height
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *EPS) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	// TODO: (EPS) test ellipse, rotations etc
	// TODO: (EPS) add drawState support
	// TODO: (EPS) use dither to fake transparency
	r.setColor(style.FillColor)
	r.w.Write([]byte(" "))
	r.w.Write([]byte(path.Transform(m).ToPS()))
	r.w.Write([]byte(" fill"))
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *EPS) RenderText(text *canvas.Text, m canvas.Matrix) {
	// TODO: (EPS) write text natively
	text.RenderAsPath(r, m, canvas.DefaultResolution)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *EPS) RenderImage(img image.Image, m canvas.Matrix) {
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
