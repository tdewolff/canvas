package canvas

import (
	"image/color"
	"math"
	"sync"
)

type ColorFunc func(float64) color.RGBA

var sRGBToLinearLUT [65536]uint32
var sRGBFromLinearLUT [65536]uint32

var genSRGBLUTs = sync.OnceFunc(func() {
	for i := 0; i < 65536; i++ {
		c := float64(i) / 65535.0

		// Formula from EXT_sRGB.
		var v float64
		if c <= 0.04045 {
			v = c / 12.92
		} else {
			v = math.Pow((c+0.055)/1.055, 2.4)
		}
		sRGBToLinearLUT[i] = uint32(v*65535.0 + 0.5)

		// Formula from EXT_sRGB.
		if c <= 0.0031308 {
			v = 12.92 * c
		} else {
			v = 1.055*math.Pow(c, 1.0/2.4) - 0.055
		}
		sRGBFromLinearLUT[i] = uint32(v*65535.0 + 0.5)
	}
})

func lab_finv(t float64) float64 {
	if t <= 6.0/29.0 {
		return 3.0 * 36.0 / 29.0 / 29.0 * (t - 4.0/29.0)
	}
	return t * t * t
}

func lab2rgb(L, a, b float64) color.RGBA {
	R := 0.950489 * lab_finv((L+0.16)/1.16+a/5.0)
	G := lab_finv((L + 0.16) / 1.16)
	B := 1.088840 * lab_finv((L+0.16)/1.16-b/2.0)
	if R < 0.0 || 1.0 < R || G < 0.0 || 1.0 < G || B < 0.0 || 1.0 < B {
		return color.RGBA{}
	}
	return color.RGBA{
		uint8(sRGBFromLinearLUT[uint32(R*65535.0+0.5)] >> 8),
		uint8(sRGBFromLinearLUT[uint32(G*65535.0+0.5)] >> 8),
		uint8(sRGBFromLinearLUT[uint32(B*65535.0+0.5)] >> 8),
		255,
	}
}

func oklab2rgb(L, a, b float64) color.RGBA {
	l_ := L + 0.3963377774*a + 0.2158037573*b
	m_ := L - 0.1055613458*a - 0.0638541728*b
	s_ := L - 0.0894841775*a - 1.2914855480*b
	l := l_ * l_ * l_
	m := m_ * m_ * m_
	s := s_ * s_ * s_
	R := 4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	G := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	B := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s
	if R < 0.0 || 1.0 < R || G < 0.0 || 1.0 < G || B < 0.0 || 1.0 < B {
		return color.RGBA{}
	}
	return color.RGBA{
		uint8(sRGBFromLinearLUT[uint32(R*65535.0+0.5)] >> 8),
		uint8(sRGBFromLinearLUT[uint32(G*65535.0+0.5)] >> 8),
		uint8(sRGBFromLinearLUT[uint32(B*65535.0+0.5)] >> 8),
		255,
	}
}

func interpolate(a, b, t float64) float64 {
	return a*(1.0-t) + b*t
}

func interpolateGradient(y float64, points [][4]float64) [4]float64 {
	index := 0
	for ; index < len(points) && points[index][0] < y; index++ {
	}
	if index == 0 {
		return [4]float64{
			0.0,
			points[0][1],
			points[0][2],
			points[0][3],
		}
	} else if index == len(points) {
		return [4]float64{
			1.0,
			points[index-1][1],
			points[index-1][2],
			points[index-1][3],
		}
	}
	prev, next := points[index-1], points[index]
	t := (y - prev[0]) / (next[0] - prev[0])
	return [4]float64{
		t,
		interpolate(prev[1], next[1], t),
		interpolate(prev[2], next[2], t),
		interpolate(prev[3], next[3], t),
	}
}

// RGBCradient creates a gradient from (t,R,G,B) values with t∈[0,1] the position, R∈[0,1] the redness, G∈[0,1] the greenness, and B∈[0,1] the blueness.
func RGBGradient(points [][4]float64) ColorFunc {
	return func(y float64) color.RGBA {
		p := interpolateGradient(y, points)
		return color.RGBA{
			uint8(p[1]*255.0 + 0.5),
			uint8(p[2]*255.0 + 0.5),
			uint8(p[3]*255.0 + 0.5),
			255,
		}
	}
}

// LCHGradient creates a gradient from (t,L,C,H) values with t∈[0,1] the position, L∈[0,1] the lightness, C∈[0,1] the chroma, and H∈[0,360] the hue.
func LCHGradient(points [][4]float64) ColorFunc {
	for i, pt := range points {
		b, a := math.Sincos(pt[3] * math.Pi / 180.0)
		a *= pt[2]
		b *= pt[2]
		points[i] = [4]float64{pt[0], pt[1], a, b}
	}
	return LabGradient(points)
}

// LabGradient creates a gradient from (t,L,a,b) values with t∈[0,1] the position, L∈[0,1] the lightness, a∈[-1,1], and b∈[-1,1].
func LabGradient(points [][4]float64) ColorFunc {
	genSRGBLUTs()
	if len(points) == 0 {
		return func(y float64) color.RGBA { return color.RGBA{} }
	}
	return func(y float64) color.RGBA {
		p := interpolateGradient(y, points)
		return lab2rgb(p[1], p[2], p[3])
	}
}

// OKLCHGradient creates a gradient from (t,L,C,H) values with t∈[0,1] the position, L∈[0,1] the lightness, C∈[0,1] the chroma, and H∈[0,360] the hue.
func OKLCHGradient(points [][4]float64) ColorFunc {
	for i, pt := range points {
		b, a := math.Sincos(pt[3] * math.Pi / 180.0)
		a *= pt[2]
		b *= pt[2]
		points[i] = [4]float64{pt[0], pt[1], a, b}
	}
	return OKLabGradient(points)
}

// OKLabGradient creates a gradient from (t,L,a,b) values with t∈[0,1] the position, L∈[0,1] the lightness, a∈[-1,1], and b∈[-1,1].
func OKLabGradient(points [][4]float64) ColorFunc {
	genSRGBLUTs()
	if len(points) == 0 {
		return func(y float64) color.RGBA { return color.RGBA{} }
	}
	return func(y float64) color.RGBA {
		p := interpolateGradient(y, points)
		return oklab2rgb(p[1], p[2], p[3])
	}
}
