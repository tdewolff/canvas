package canvas

import (
	"fmt"
	"image/color"
	"testing"
)

func TestGradAdd(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}
	green := color.RGBA{0, 255, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}
	white := color.RGBA{255, 255, 255, 255}

	tests := []struct {
		name     string
		initial  Grad
		addT     float64
		addColor color.RGBA
		want     Grad
	}{
		{
			name:     "add to empty gradient",
			initial:  Grad{},
			addT:     0.5,
			addColor: red,
			want:     Grad{{0.5, red}},
		},
		{
			name:     "add at end",
			initial:  Grad{{0.0, red}},
			addT:     1.0,
			addColor: blue,
			want:     Grad{{0.0, red}, {1.0, blue}},
		},
		{
			name:     "add at beginning",
			initial:  Grad{{0.5, red}},
			addT:     0.0,
			addColor: blue,
			want:     Grad{{0.0, blue}, {0.5, red}},
		},
		{
			name:     "insert in middle maintains sort order",
			initial:  Grad{{0.0, red}, {1.0, blue}},
			addT:     0.5,
			addColor: green,
			want:     Grad{{0.0, red}, {0.5, green}, {1.0, blue}},
		},
		{
			name:     "replace existing offset",
			initial:  Grad{{0.0, red}, {0.5, green}, {1.0, blue}},
			addT:     0.5,
			addColor: white,
			want:     Grad{{0.0, red}, {0.5, white}, {1.0, blue}},
		},
		{
			name:     "clamp t below 0",
			initial:  Grad{},
			addT:     -0.5,
			addColor: red,
			want:     Grad{{0.0, red}},
		},
		{
			name:     "clamp t above 1",
			initial:  Grad{},
			addT:     1.5,
			addColor: red,
			want:     Grad{{1.0, red}},
		},
		{
			name:     "add multiple maintains order",
			initial:  Grad{{0.2, red}, {0.8, blue}},
			addT:     0.4,
			addColor: green,
			want:     Grad{{0.2, red}, {0.4, green}, {0.8, blue}},
		},
		{
			name:     "add semi-transparent color clips to premultiplied",
			initial:  Grad{},
			addT:     0.5,
			addColor: color.RGBA{255, 0, 0, 128},
			want:     Grad{{0.5, color.RGBA{128, 0, 0, 128}}},
		},
		{
			name:     "replace opaque with semi-transparent clips to premultiplied",
			initial:  Grad{{0.0, red}, {0.5, green}, {1.0, blue}},
			addT:     0.5,
			addColor: color.RGBA{0, 255, 0, 64},
			want:     Grad{{0.0, red}, {0.5, color.RGBA{0, 64, 0, 64}}, {1.0, blue}},
		},
		{
			name:     "fully transparent color",
			initial:  Grad{{0.0, red}},
			addT:     1.0,
			addColor: color.RGBA{0, 0, 0, 0},
			want:     Grad{{0.0, red}, {1.0, color.RGBA{0, 0, 0, 0}}},
		},
		{
			name:     "mixed transparent stops maintain order",
			initial:  Grad{{0.0, color.RGBA{255, 0, 0, 200}}, {1.0, color.RGBA{0, 0, 255, 100}}},
			addT:     0.5,
			addColor: color.RGBA{0, 255, 0, 50},
			want:     Grad{{0.0, color.RGBA{255, 0, 0, 200}}, {0.5, color.RGBA{0, 50, 0, 50}}, {1.0, color.RGBA{0, 0, 255, 100}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := make(Grad, len(tt.initial))
			copy(g, tt.initial)
			g.Add(tt.addT, tt.addColor)

			if len(g) != len(tt.want) {
				t.Fatalf("got %d stops, want %d", len(g), len(tt.want))
			}
			for i := range tt.want {
				if !Equal(g[i].Offset, tt.want[i].Offset) {
					t.Errorf("stop[%d].Offset = %v, want %v", i, g[i].Offset, tt.want[i].Offset)
				}
				if g[i].Color != tt.want[i].Color {
					t.Errorf("stop[%d].Color = %v, want %v", i, g[i].Color, tt.want[i].Color)
				}
			}
		})
	}
}

func TestRadialGradientAt(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}

	// Helper to build a radial gradient with red at t=0 and blue at t=1.
	makeGrad := func(c0 Point, r0 float64, c1 Point, r1 float64) *RadialGradient {
		g := NewRadialGradient(c0, r0, c1, r1)
		g.Add(0.0, red)
		g.Add(1.0, blue)
		return g
	}

	// colorApproxEqual allows small rounding differences from interpolation.
	colorApproxEqual := func(a, b color.RGBA) bool {
		diff := func(x, y uint8) int {
			d := int(x) - int(y)
			if d < 0 {
				return -d
			}
			return d
		}
		return diff(a.R, b.R) <= 1 && diff(a.G, b.G) <= 1 && diff(a.B, b.B) <= 1 && diff(a.A, b.A) <= 1
	}

	tests := []struct {
		name string
		grad *RadialGradient
		x, y float64
		want color.RGBA
	}{
		{
			name: "empty gradient returns transparent",
			grad: NewRadialGradient(Point{0, 0}, 0, Point{0, 0}, 10),
			x:    5,
			y:    0,
			want: Transparent,
		},
		{
			name: "concentric at center returns first stop",
			grad: makeGrad(Point{0, 0}, 0, Point{0, 0}, 10),
			x:    0,
			y:    0,
			want: red,
		},
		{
			name: "concentric at outer edge returns last stop",
			grad: makeGrad(Point{0, 0}, 0, Point{0, 0}, 10),
			x:    10,
			y:    0,
			want: blue,
		},
		{
			name: "concentric beyond outer edge clamps to last stop",
			grad: makeGrad(Point{0, 0}, 0, Point{0, 0}, 10),
			x:    20,
			y:    0,
			want: blue,
		},
		{
			name: "concentric at midpoint interpolates",
			grad: makeGrad(Point{0, 0}, 0, Point{0, 0}, 10),
			x:    5,
			y:    0,
			want: color.RGBA{128, 0, 127, 255},
		},
		{
			name: "concentric along y axis",
			grad: makeGrad(Point{0, 0}, 0, Point{0, 0}, 10),
			x:    0,
			y:    10,
			want: blue,
		},
		{
			name: "concentric with offset center",
			grad: makeGrad(Point{5, 5}, 0, Point{5, 5}, 10),
			x:    5,
			y:    5,
			want: red,
		},
		{
			name: "concentric with offset center at edge",
			grad: makeGrad(Point{5, 5}, 0, Point{5, 5}, 10),
			x:    15,
			y:    5,
			want: blue,
		},
		{
			name: "non-concentric gradient at inner center",
			grad: makeGrad(Point{0, 0}, 0, Point{10, 0}, 10),
			x:    0,
			y:    0,
			want: red,
		},
		{
			name: "single stop gradient returns that stop everywhere",
			grad: func() *RadialGradient {
				g := NewRadialGradient(Point{0, 0}, 0, Point{0, 0}, 10)
				g.Add(0.0, red)
				return g
			}(),
			x:    5,
			y:    0,
			want: red,
		},
		{
			name: "offset centers uses valid root not largest root",
			grad: func() *RadialGradient {
				g := NewRadialGradient(Point{30, 30}, 5, Point{170, 170}, 60)
				g.Add(0.0, color.RGBA{255, 192, 203, 255})
				g.Add(0.5, color.RGBA{255, 255, 255, 255})
				g.Add(1.0, color.RGBA{0, 128, 0, 255})
				return g
			}(),
			x:    160,
			y:    111,
			want: color.RGBA{180, 218, 180, 255},
		},
		{
			name: "offset centers near outer boundary not clamped to last stop",
			grad: func() *RadialGradient {
				g := NewRadialGradient(Point{30, 30}, 5, Point{170, 170}, 60)
				g.Add(0.0, color.RGBA{255, 192, 203, 255})
				g.Add(0.5, color.RGBA{255, 255, 255, 255})
				g.Add(1.0, color.RGBA{0, 128, 0, 255})
				return g
			}(),
			x:    165,
			y:    111,
			want: color.RGBA{164, 210, 164, 255},
		},
		{
			name: "concentric with semi-transparent stops",
			grad: func() *RadialGradient {
				g := NewRadialGradient(Point{0, 0}, 0, Point{0, 0}, 10)
				g.Add(0.0, color.RGBA{128, 0, 0, 128})
				g.Add(1.0, color.RGBA{0, 0, 128, 128})
				return g
			}(),
			x:    0,
			y:    0,
			want: color.RGBA{128, 0, 0, 128},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.grad.At(tt.x, tt.y)
			if !colorApproxEqual(got, tt.want) {
				t.Errorf("At(%v, %v) = %v, want %v (within ±1 per channel)",
					tt.x, tt.y, formatRGBA(got), formatRGBA(tt.want))
			}
		})
	}
}

func formatRGBA(c color.RGBA) string {
	return fmt.Sprintf("RGBA{%d, %d, %d, %d}", c.R, c.G, c.B, c.A)
}
