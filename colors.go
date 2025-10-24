package canvas

import (
	"encoding/binary"
	"hash/maphash"
	"image/color"
	"math"
)

// RGB returns a color given by red, green, and blue ∈ [0,1].
func RGB(r, g, b float64) color.RGBA {
	return color.RGBA{
		uint8(r * 255.0),
		uint8(g * 255.0),
		uint8(b * 255.0),
		uint8(255.0),
	}
}

// RGBA returns a color given by red, green, blue, and alpha ∈ [0,1] (non alpha premultiplied).
func RGBA(r, g, b, a float64) color.RGBA {
	return color.RGBA{
		uint8(a * r * 255.0),
		uint8(a * g * 255.0),
		uint8(a * b * 255.0),
		uint8(a * 255.0),
	}
}

// Hex parses a CSS hexadecimal color such as e.g. #ff0000 or F00.
func Hex(s string) color.RGBA {
	if 0 < len(s) && s[0] == '#' {
		s = s[1:]
	}
	h := make([]uint8, len(s))
	for i, c := range s {
		if '0' <= c && c <= '9' {
			h[i] = uint8(c - '0')
		} else if 'a' <= c && c <= 'f' {
			h[i] = 10 + uint8(c-'a')
		} else if 'A' <= c && c <= 'F' {
			h[i] = 10 + uint8(c-'A')
		}
	}
	if len(s) == 3 {
		return color.RGBA{h[0]*16 + h[0], h[1]*16 + h[1], h[2]*16 + h[2], 0xff}
	} else if len(s) == 4 {
		a := float64(h[3]*16+h[0]) / 255.0
		return color.RGBA{
			uint8(a * float64(h[0]*16+h[0])),
			uint8(a * float64(h[1]*16+h[1])),
			uint8(a * float64(h[2]*16+h[2])),
			h[3]*16 + h[3],
		}
	} else if len(s) == 6 {
		return color.RGBA{h[0]*16 + h[1], h[2]*16 + h[3], h[4]*16 + h[5], 0xff}
	} else if len(s) == 8 {
		a := float64(h[6]*16+h[7]) / 255.0
		return color.RGBA{
			uint8(a * float64(h[0]*16+h[1])),
			uint8(a * float64(h[2]*16+h[3])),
			uint8(a * float64(h[4]*16+h[5])),
			h[6]*16 + h[7],
		}
	}
	return Black
}

// Gradient is a gradient pattern for filling.
type Gradient interface {
	Transform(Matrix) Gradient
	SetColorSpace(ColorSpace) Gradient
	Hash() uint64
	At(float64, float64) color.RGBA
}

// Stop is a color and offset for gradient patterns.
type Stop struct {
	Offset float64
	Color  color.RGBA
}

// Grad are the colors and offsets for gradient patterns, sorted by offset.
type Grad []Stop

func NewGradient() Grad {
	return Grad{}
}

// Add adds a new color stop to a gradient.
func (g *Grad) Add(t float64, color color.RGBA) {
	stop := Stop{math.Min(math.Max(t, 0.0), 1.0), color}
	// insert or replace stop and keep sort order
	for i := range *g {
		if Equal((*g)[i].Offset, stop.Offset) {
			(*g)[i] = stop
			return
		} else if stop.Offset < (*g)[i].Offset {
			*g = append((*g)[:i], append(Grad{stop}, (*g)[i:]...)...)
			return
		}
	}
	*g = append(*g, stop)
}

// At returns the color at position t ∈ [0,1].
func (g Grad) At(t float64) color.RGBA {
	if len(g) == 0 {
		return Transparent
	} else if len(g) == 1 || t <= g[0].Offset {
		return g[0].Color
	} else if g[len(g)-1].Offset <= t {
		return g[len(g)-1].Color
	}
	for i, after := range g[1:] {
		if t < after.Offset {
			before := g[i]
			t = (t - before.Offset) / (after.Offset - before.Offset)
			return colorLerp(before.Color, after.Color, t)
		}
	}
	return g[len(g)-1].Color
}

func (g Grad) ToLinear(start, end Point) *LinearGradient {
	grad := NewLinearGradient(start, end)
	grad.Grad = g
	return grad
}

func (g Grad) ToRadial(c0 Point, r0 float64, c1 Point, r1 float64) *RadialGradient {
	grad := NewRadialGradient(c0, r0, c1, r1)
	grad.Grad = g
	return grad
}

func colorLerp(c0, c1 color.RGBA, t float64) color.RGBA {
	r0, g0, b0, a0 := c0.RGBA()
	r1, g1, b1, a1 := c1.RGBA()
	T := uint32(t*65535.0 + 0.5)
	return color.RGBA{
		lerp(r0, r1, T),
		lerp(g0, g1, T),
		lerp(b0, b1, T),
		lerp(a0, a1, T),
	}
}

func lerp(a, b, t uint32) uint8 {
	return uint8(((0xffff-t)*a + t*b) >> 24)
}

// LinearGradient is a linear gradient pattern between the given start and end points. The color at offset 0 corresponds to the start position, and offset 1 to the end position. Start and end points are in the canvas's coordinate system.
type LinearGradient struct {
	Grad
	Start, End Point
	d          Point
	d2         float64
}

// NewLinearGradient returns a new linear gradient pattern.
func NewLinearGradient(start, end Point) *LinearGradient {
	d := end.Sub(start)
	return &LinearGradient{
		Start: start,
		End:   end,
		d:     d,
		d2:    d.Dot(d),
	}
}

// Transform sets the view. Automatically called by Canvas for coordinate system transformations.
func (g *LinearGradient) Transform(m Matrix) Gradient {
	if m == Identity {
		return g
	}

	gradient := *g
	gradient.Start = m.Dot(gradient.Start)
	gradient.End = m.Dot(gradient.End)
	gradient.d = gradient.End.Sub(gradient.Start)
	gradient.d2 = gradient.d.Dot(gradient.d)
	return &gradient
}

// SetColorSpace sets the color space. Automatically called by the rasterizer.
func (g *LinearGradient) SetColorSpace(colorSpace ColorSpace) Gradient {
	if _, ok := colorSpace.(LinearColorSpace); ok {
		return g
	}
	g.Grad = append(Grad{}, g.Grad...)
	for i := range g.Grad {
		g.Grad[i].Color = colorSpace.ToLinear(g.Grad[i].Color)
	}
	return g
}

func (g *LinearGradient) Hash() uint64 {
	var h maphash.Hash
	binary.Write(&h, binary.LittleEndian, float32(g.Start.X))
	binary.Write(&h, binary.LittleEndian, float32(g.Start.Y))
	binary.Write(&h, binary.LittleEndian, float32(g.End.X))
	binary.Write(&h, binary.LittleEndian, float32(g.End.Y))
	for _, stop := range g.Grad {
		binary.Write(&h, binary.LittleEndian, float32(stop.Offset))
		binary.Write(&h, binary.LittleEndian, stop.Color.R)
		binary.Write(&h, binary.LittleEndian, stop.Color.G)
		binary.Write(&h, binary.LittleEndian, stop.Color.B)
		binary.Write(&h, binary.LittleEndian, stop.Color.A)
	}
	return h.Sum64()
}

// At returns the color at position (x,y).
func (g *LinearGradient) At(x, y float64) color.RGBA {
	if len(g.Grad) == 0 {
		return Transparent
	}

	p := Point{x, y}.Sub(g.Start)
	if Equal(g.d.Y, 0.0) && !Equal(g.d.X, 0.0) {
		return g.Grad.At(p.X / g.d.X) // horizontal
	} else if !Equal(g.d.Y, 0.0) && Equal(g.d.X, 0.0) {
		return g.Grad.At(p.Y / g.d.Y) // vertical
	}
	t := p.Dot(g.d) / g.d2
	return g.Grad.At(t)
}

// RadialGradient is a radial gradient pattern between two circles defined by their center points and radii. Color stop at offset 0 corresponds to the first circle and offset 1 to the second circle.
type RadialGradient struct {
	Grad
	C0, C1 Point
	R0, R1 float64
	cd     Point
	dr, a  float64
}

// NewRadialGradient returns a new radial gradient pattern.
func NewRadialGradient(c0 Point, r0 float64, c1 Point, r1 float64) *RadialGradient {
	cd := c1.Sub(c0)
	dr := r1 - r0
	return &RadialGradient{
		C0: c0,
		R0: r0,
		C1: c1,
		R1: r1,
		cd: cd,
		dr: dr,
		a:  cd.Dot(cd) - dr*dr,
	}
}

// Transform sets the view. Automatically called by Canvas for coordinate system transformations.
func (g *RadialGradient) Transform(m Matrix) Gradient {
	if m == Identity {
		return g
	}

	gradient := *g
	gradient.C0 = m.Dot(gradient.C0)
	gradient.C1 = m.Dot(gradient.C1)
	gradient.cd = gradient.C1.Sub(gradient.C0)
	gradient.a = gradient.cd.Dot(gradient.cd) - gradient.dr*gradient.dr
	return &gradient
}

// SetColorSpace sets the color space. Automatically called by the rasterizer.
func (g *RadialGradient) SetColorSpace(colorSpace ColorSpace) Gradient {
	if _, ok := colorSpace.(LinearColorSpace); ok {
		return g
	}
	g.Grad = append(Grad{}, g.Grad...)
	for i := range g.Grad {
		g.Grad[i].Color = colorSpace.ToLinear(g.Grad[i].Color)
	}
	return g
}

func (g *RadialGradient) Hash() uint64 {
	var h maphash.Hash
	binary.Write(&h, binary.LittleEndian, float32(g.C0.X))
	binary.Write(&h, binary.LittleEndian, float32(g.C0.Y))
	binary.Write(&h, binary.LittleEndian, float32(g.C1.X))
	binary.Write(&h, binary.LittleEndian, float32(g.C1.Y))
	binary.Write(&h, binary.LittleEndian, float32(g.R0))
	binary.Write(&h, binary.LittleEndian, float32(g.R1))
	for _, stop := range g.Grad {
		binary.Write(&h, binary.LittleEndian, float32(stop.Offset))
		binary.Write(&h, binary.LittleEndian, stop.Color.R)
		binary.Write(&h, binary.LittleEndian, stop.Color.G)
		binary.Write(&h, binary.LittleEndian, stop.Color.B)
		binary.Write(&h, binary.LittleEndian, stop.Color.A)
	}
	return h.Sum64()
}

// At returns the color at position (x,y).
func (g *RadialGradient) At(x, y float64) color.RGBA {
	if len(g.Grad) == 0 {
		return Transparent
	}

	// see reference implementation of pixman-radial-gradient
	// https://github.com/servo/pixman/blob/master/pixman/pixman-radial-gradient.c#L161
	pd := Point{x, y}.Sub(g.C0)
	b := pd.Dot(g.cd) + g.R0*g.dr
	c := pd.Dot(pd) - g.R0*g.R0
	t0, t1 := solveQuadraticFormula(g.a, -2.0*b, c)
	if !math.IsNaN(t1) {
		return g.Grad.At(t1)
	} else if !math.IsNaN(t0) {
		return g.Grad.At(t0)
	}
	return Transparent
}

// ColorSpace defines the color space within the RGB color model. All colors passed to this library are assumed to be in the sRGB color space, which is a ubiquitous assumption in most software. This works great for most applications, but fails when blending semi-transparent layers. See an elaborate explanation at https://blog.johnnovak.net/2016/09/21/what-every-coder-should-know-about-gamma/, which goes into depth of the problems of using sRGB for blending and the need for gamma correction. In short, we need to transform the colors, which are in the sRGB color space, to the linear color space, perform blending, and then transform them back to the sRGB color space.
// Unfortunately, almost all software does blending the wrong way (all PDF renderers and browsers I've tested), so by default this library will do the same by using LinearColorSpace which does no conversion from sRGB to linear and back but blends directly in sRGB. Or in other words, it assumes that colors are given in the linear color space and that the output image is expected to be in the linear color space as well. For technical correctness we should really be using the SRGBColorSpace, which will convert from sRGB to linear space, do blending in linear space, and then go back to sRGB space.
type ColorSpace interface {
	ToLinear(color.Color) color.RGBA
	FromLinear(color.Color) color.RGBA
}

// DefaultColorSpace is set to LinearColorSpace to match other renderers.
var DefaultColorSpace ColorSpace = LinearColorSpace{}

// LinearColorSpace is the default color space that does not do color space conversion for blending purposes. This is only correct if the input colors and output images are assumed to be in the linear color space so that blending is in linear space as well. In general though, we assume that input colors and output images are using the sRGB color space almost ubiquitously, resulting in blending in sRGB space which is wrong! Even though it is technically incorrect, many PDF viewers and browsers do this anyway.
type LinearColorSpace struct{}

// ToLinear encodes color to color space.
func (LinearColorSpace) ToLinear(col color.Color) color.RGBA {
	if rgba, ok := col.(color.RGBA); ok {
		return rgba
	}
	R, G, B, A := col.RGBA()
	return color.RGBA{uint8(R >> 8), uint8(G >> 8), uint8(B >> 8), uint8(A >> 8)}
}

// FromLinear decodes color from color space.
func (LinearColorSpace) FromLinear(col color.Color) color.RGBA {
	if rgba, ok := col.(color.RGBA); ok {
		return rgba
	}
	R, G, B, A := col.RGBA()
	return color.RGBA{uint8(R >> 8), uint8(G >> 8), uint8(B >> 8), uint8(A >> 8)}
}

// GammaColorSpace assumes that input colors and output images are gamma-corrected with the given gamma value. The sRGB space uses a gamma=2.4 for most of the curve, but will on average have a gamma=2.2 best approximating the sRGB curve. See https://en.wikipedia.org/wiki/SRGB#The_sRGB_transfer_function_(%22gamma%22). According to https://www.puredevsoftware.com/blog/2019/01/22/sub-pixel-gamma-correct-font-rendering/, a gamma=1.43 is recommended for fonts.
type GammaColorSpace struct {
	Gamma float64
}

// ToLinear encodes color to color space.
func (cs GammaColorSpace) ToLinear(col color.Color) color.RGBA {
	R, G, B, A := col.RGBA()
	r := math.Pow(float64(R)/float64(A), cs.Gamma)
	g := math.Pow(float64(G)/float64(A), cs.Gamma)
	b := math.Pow(float64(B)/float64(A), cs.Gamma)
	a := float64(A) / 0xffff
	return color.RGBA{
		uint8(r*a*255.0 + 0.5),
		uint8(g*a*255.0 + 0.5),
		uint8(b*a*255.0 + 0.5),
		uint8(a*255.0 + 0.5),
	}
}

// FromLinear decodes color from color space.
func (cs GammaColorSpace) FromLinear(col color.Color) color.RGBA {
	R, G, B, A := col.RGBA()
	r := math.Pow(float64(R)/float64(A), 1.0/cs.Gamma)
	g := math.Pow(float64(G)/float64(A), 1.0/cs.Gamma)
	b := math.Pow(float64(B)/float64(A), 1.0/cs.Gamma)
	a := float64(A) / 0xffff
	return color.RGBA{
		uint8(r*a*255.0 + 0.5),
		uint8(g*a*255.0 + 0.5),
		uint8(b*a*255.0 + 0.5),
		uint8(a*255.0 + 0.5),
	}
}

// SRGBColorSpace assumes that input colors and output images are in the sRGB color space (ubiquitous in almost all applications), which implies that for blending we need to convert to the linear color space, do blending, and then convert back to the sRGB color space. This will give technically correct blending, but may differ from common PDF viewer and browsers (which are wrong).
type SRGBColorSpace struct{}

// ToLinear encodes color to color space.
func (SRGBColorSpace) ToLinear(col color.Color) color.RGBA {
	sRGBToLinear := func(c float64) float64 {
		// Formula from EXT_sRGB.
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}

	R, G, B, A := col.RGBA()
	r := sRGBToLinear(float64(R) / float64(A))
	g := sRGBToLinear(float64(G) / float64(A))
	b := sRGBToLinear(float64(B) / float64(A))
	a := float64(A) / 0xffff
	return color.RGBA{
		uint8(r*a*255.0 + 0.5),
		uint8(g*a*255.0 + 0.5),
		uint8(b*a*255.0 + 0.5),
		uint8(a*255.0 + 0.5),
	}
}

// FromLinear decodes color from color space.
func (SRGBColorSpace) FromLinear(col color.Color) color.RGBA {
	linearTosRGB := func(c float64) float64 {
		// Formula from EXT_sRGB.
		switch {
		case c <= 0.0:
			return 0.0
		case 0 < c && c < 0.0031308:
			return 12.92 * c
		case 0.0031308 <= c && c < 1:
			return 1.055*math.Pow(c, 0.41666) - 0.055
		}
		return 1.0
	}

	R, G, B, A := col.RGBA()
	r := linearTosRGB(float64(R) / float64(A))
	g := linearTosRGB(float64(G) / float64(A))
	b := linearTosRGB(float64(B) / float64(A))
	a := float64(A) / 0xffff
	return color.RGBA{
		uint8(r*a*255.0 + 0.5),
		uint8(g*a*255.0 + 0.5),
		uint8(b*a*255.0 + 0.5),
		uint8(a*255.0 + 0.5),
	}
}
