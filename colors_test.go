package canvas

import (
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
