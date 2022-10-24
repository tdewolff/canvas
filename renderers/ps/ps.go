package ps

import (
	"compress/zlib"
	"encoding/ascii85"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"strings"
	"time"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/minify/v2"
)

var psEllipseDef = `/ellipse{/rot exch def /a1 exch def /a0 exch def /ry exch def /rx exch def /y exch def /x exch def /m matrix currentmatrix def x y translate rot rotate rx ry scale 0 0 1 a0 a1 arc m setmatrix}def
/ellipsen{/rot exch def /a1 exch def /a0 exch def /ry exch def /rx exch def /y exch def /x exch def /m matrix currentmatrix def x y translate rot rotate rx ry scale 0 0 1 a0 a1 arcn m setmatrix}def`

type Format int

const (
	PostScript Format = iota
	EncapsulatedPostScript
)

type Options struct {
	Format
	canvas.ImageEncoding
}

var DefaultOptions = Options{
	ImageEncoding: canvas.Lossless,
}

// PS is an PostScript renderer. Be aware that PostScript does not support transparency of colors.
type PS struct {
	w             io.Writer
	width, height float64
	opts          *Options

	color      color.NRGBA
	lineWidth  float64
	miterLimit float64
	lineCap    canvas.Capper
	lineJoin   canvas.Joiner
	dashOffset float64
	dashes     []float64
}

// New returns an PostScript renderer.
func New(w io.Writer, width, height float64, opts *Options) *PS {
	if opts == nil {
		defaultOptions := DefaultOptions
		opts = &defaultOptions
	}

	if opts.Format == PostScript {
		fmt.Fprintf(w, "%%!PS-Adobe-3.0\n")
	} else if opts.Format == EncapsulatedPostScript {
		fmt.Fprintf(w, "%%!PS-Adobe-3.0 EPSF-3.0\n")
	}
	fmt.Fprintf(w, "%%%%Creator: tdewolff/canvas\n")
	fmt.Fprintf(w, "%%%%CreationDate: %v\n", time.Now().Format(time.ANSIC))
	fmt.Fprintf(w, "%%%%BoundingBox: 0 0 %v %v\n", dec(width), dec(height))

	if opts.Format == EncapsulatedPostScript {
		fmt.Fprintf(w, "%%%%EndComments\n")
		// TODO: (EPS) generate and add preview
	}

	fmt.Fprint(w, psEllipseDef)

	return &PS{
		w:          w,
		width:      width,
		height:     height,
		opts:       opts,
		miterLimit: 10.0,
	}
}

func (r *PS) Close() error {
	if r.opts.Format == EncapsulatedPostScript {
		fmt.Fprintf(r.w, "%%%%EOF")
	}
	return nil
}

func (r *PS) setColor(col color.RGBA) {
	color := toNRGBA(col)
	if color.R != r.color.R || color.G != r.color.G || color.B != r.color.B {
		if color.R == color.G && color.R == color.B {
			fmt.Fprintf(r.w, " %v setgray", dec(float64(color.R)/255.0))
		} else {
			fmt.Fprintf(r.w, " %v %v %v setrgbcolor", dec(float64(color.R)/255.0), dec(float64(color.G)/255.0), dec(float64(color.B)/255.0))
		}
		r.color = color
	}
}

func (r *PS) setLineWidth(width float64) {
	if width != r.lineWidth {
		fmt.Fprintf(r.w, " %v setlinewidth", dec(width))
		r.lineWidth = width
	}
}

func (r *PS) setMiterLimit(limit float64) {
	if limit != r.miterLimit {
		fmt.Fprintf(r.w, " %v setmiterlimit", dec(limit))
		r.miterLimit = limit
	}
}

func (r *PS) setLineCap(capper canvas.Capper) {
	if capper != r.lineCap {
		if _, ok := capper.(canvas.RoundCapper); ok {
			fmt.Fprintf(r.w, " 1 setlinecap")
		} else if _, ok := capper.(canvas.SquareCapper); ok {
			fmt.Fprintf(r.w, " 2 setlinecap")
		} else if _, ok := capper.(canvas.ButtCapper); ok {
			fmt.Fprintf(r.w, " 0 setlinecap")
		} else {
			panic("PS: line cap not support")
		}
		r.lineCap = capper
	}
}

func (r *PS) setLineJoin(joiner canvas.Joiner) {
	if joiner != r.lineJoin {
		if _, ok := joiner.(canvas.BevelJoiner); ok {
			fmt.Fprintf(r.w, " 2 setlinejoin")
		} else if _, ok := joiner.(canvas.RoundJoiner); ok {
			fmt.Fprintf(r.w, " 1 setlinejoin")
		} else if miter, ok := joiner.(canvas.MiterJoiner); ok && !math.IsNaN(miter.Limit) && miter.GapJoiner == canvas.BevelJoin {
			fmt.Fprintf(r.w, " 0 setlinejoin")
			r.setMiterLimit(miter.Limit)
		} else {
			panic("PS: line join not support")
		}
		r.lineJoin = joiner
	}
}

func (r *PS) setDashes(offset float64, dashes []float64) {
	if !float64sEqual(dashes, r.dashes) || offset != r.dashOffset {
		if len(dashes) == 0 {
			fmt.Fprintf(r.w, "[")
		} else {
			fmt.Fprintf(r.w, "[%v", dec(dashes[0]))
			for _, dash := range dashes[1:] {
				fmt.Fprintf(r.w, " %v", dec(dash))
			}
		}
		fmt.Fprintf(r.w, "]%v setdash", dec(offset))
		r.dashOffset = offset
		r.dashes = dashes
	}
}

// Size returns the size of the canvas in millimeters.
func (r *PS) Size() (float64, float64) {
	return r.width, r.height
}

// RenderPath renders a path to the canvas using a style and a transformation matrix.
func (r *PS) RenderPath(path *canvas.Path, style canvas.Style, m canvas.Matrix) {
	// TODO: (EPS) use dither to fake transparency

	strokeUnsupported := false
	if _, ok := style.StrokeJoiner.(canvas.ArcsJoiner); ok {
		strokeUnsupported = true
	} else if miter, ok := style.StrokeJoiner.(canvas.MiterJoiner); ok {
		if math.IsNaN(miter.Limit) {
			strokeUnsupported = true
		} else if _, ok := miter.GapJoiner.(canvas.BevelJoiner); !ok {
			strokeUnsupported = true
		}
	}
	if !strokeUnsupported {
		if m.IsSimilarity() {
			scale := math.Sqrt(math.Abs(m.Det()))
			style.StrokeWidth *= scale
			style.DashOffset *= scale
			dashes := make([]float64, len(style.Dashes))
			for i := range style.Dashes {
				dashes[i] = style.Dashes[i] * scale
			}
			style.Dashes = dashes
		} else {
			strokeUnsupported = true
		}
	}

	if style.HasFill() || style.HasStroke() && !strokeUnsupported {
		r.w.Write([]byte("\n"))
		r.w.Write([]byte(path.Transform(m).ToPS()))
	}

	if style.HasFill() {
		r.setColor(style.FillColor)
		if style.HasStroke() && !strokeUnsupported {
			r.w.Write([]byte(" gsave"))
		}
		if style.FillRule == canvas.EvenOdd {
			r.w.Write([]byte(" eofill"))
		} else {
			r.w.Write([]byte(" fill"))
		}
		if style.HasStroke() && !strokeUnsupported {
			r.w.Write([]byte(" grestore"))
		}
	}
	if style.HasStroke() {
		if !strokeUnsupported {
			r.setColor(style.StrokeColor)
			r.setLineWidth(style.StrokeWidth)
			r.setLineCap(style.StrokeCapper)
			r.setLineJoin(style.StrokeJoiner)
			r.setDashes(style.DashOffset, style.Dashes)
			r.w.Write([]byte(" stroke"))
		} else {
			// stroke settings unsupported by PDF, draw stroke explicitly
			if style.IsDashed() {
				path = path.Dash(style.DashOffset, style.Dashes...)
			}
			path = path.Stroke(style.StrokeWidth, style.StrokeCapper, style.StrokeJoiner)

			r.w.Write([]byte("\n"))
			r.w.Write([]byte(path.Transform(m).ToPS()))
			r.setColor(style.StrokeColor)
			r.w.Write([]byte(" fill"))
		}
	}
}

// RenderText renders a text object to the canvas using a transformation matrix.
func (r *PS) RenderText(text *canvas.Text, m canvas.Matrix) {
	// TODO: (EPS) write text natively
	text.RenderAsPath(r, m, canvas.DefaultResolution)
}

// RenderImage renders an image to the canvas using a transformation matrix.
func (r *PS) RenderImage(img image.Image, m canvas.Matrix) {
	size := img.Bounds().Size()
	sp := img.Bounds().Min // starting point
	b := make([]byte, size.X*size.Y*3)
	bMask := make([]bool, size.X*size.Y)
	hasMask := false
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			i := (y*size.X + x) * 3
			R, G, B, A := img.At(sp.X+x, sp.Y+y).RGBA()
			if A != 0 {
				b[i+0] = byte((R * 65535 / A) >> 8)
				b[i+1] = byte((G * 65535 / A) >> 8)
				b[i+2] = byte((B * 65535 / A) >> 8)
				bMask[y*size.X+x] = 128 <= (A >> 8)
			}
			if A>>8 != 255 {
				hasMask = true
			}
		}
	}
	_ = hasMask // TODO: PS image mask
	_ = bMask   // TODO: PS image mask

	m = m.Scale(float64(size.X), float64(size.Y))
	fmt.Fprintf(r.w, " gsave")
	fmt.Fprintf(r.w, " /DeviceRGB setcolorspace")
	fmt.Fprintf(r.w, " [%v %v %v %v %v %v] concat", dec(m[0][0]), dec(m[1][0]), dec(m[0][1]), dec(m[1][1]), dec(m[0][2]), dec(m[1][2]))
	fmt.Fprintf(r.w, "<</ImageType 1 /BitsPerComponent 8 /Decode [0 1 0 1 0 1] /Interpolate true")
	fmt.Fprintf(r.w, " /Width %d /Height %d", size.X, size.Y)
	fmt.Fprintf(r.w, " /ImageMatrix [%d %d %d %d %d %d]", size.X, 0, 0, -size.Y, 0, size.Y)
	fmt.Fprintf(r.w, " /DataSource currentfile /ASCII85Decode filter /FlateDecode filter>>image\n")

	wAscii := ascii85.NewEncoder(r.w)
	wZlib := zlib.NewWriter(wAscii)
	wZlib.Write(b)
	wZlib.Close()
	wAscii.Close()
	fmt.Fprintf(r.w, "~>\n")
	fmt.Fprintf(r.w, " grestore")
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
