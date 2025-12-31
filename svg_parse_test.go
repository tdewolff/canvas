package canvas

import (
	"image/color"
	"strings"
	"testing"

	"github.com/tdewolff/test"
)

func TestParseSVGRGBADecimalAlpha(t *testing.T) {
	var tests = []struct {
		name     string
		svg      string
		expected color.RGBA
	}{
		{
			"decimal alpha 0.5",
			`<svg width="100" height="100"><rect x="0" y="0" width="10" height="10" fill="rgba(255,0,0,0.5)"/></svg>`,
			color.RGBA{127, 0, 0, 127}, // premultiplied alpha
		},
		{
			"decimal alpha 1.0",
			`<svg width="100" height="100"><rect x="0" y="0" width="10" height="10" fill="rgba(255,0,0,1.0)"/></svg>`,
			color.RGBA{255, 0, 0, 255},
		},
		{
			"integer alpha 1 (fully opaque)",
			`<svg width="100" height="100"><rect x="0" y="0" width="10" height="10" fill="rgba(255,0,0,1)"/></svg>`,
			color.RGBA{255, 0, 0, 255},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseSVG(strings.NewReader(tt.svg))
			test.Error(t, err)
			if len(c.layers) == 0 || len(c.layers[0]) == 0 {
				t.Fatal("no layers rendered")
			}
			layer := c.layers[0][0]
			test.T(t, layer.style.Fill.Color, tt.expected)
		})
	}
}

func TestParseSVGMarkerWithoutViewBox(t *testing.T) {
	svg := `<svg width="100" height="100">
		<defs>
			<marker id="arrow" markerWidth="6" markerHeight="6" refX="3" refY="3" orient="auto" markerUnits="strokeWidth">
				<path d="M0,0 L0,6 L6,3 z" fill="rgb(255,0,0)"/>
			</marker>
		</defs>
		<path d="M10,50 L90,50" stroke="black" stroke-width="2" marker-end="url(#arrow)"/>
	</svg>`

	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	if c == nil {
		t.Fatal("canvas is nil")
	}

	totalLayers := 0
	for _, layers := range c.layers {
		totalLayers += len(layers)
	}
	if totalLayers < 2 {
		t.Fatalf("expected at least 2 layers (path + marker), got %d", totalLayers)
	}
}

func TestParseSVGMarkerWithViewBox(t *testing.T) {
	svg := `<svg width="100" height="100">
		<defs>
			<marker id="arrow" markerWidth="6" markerHeight="6" refX="3" refY="3" orient="auto" markerUnits="strokeWidth" viewBox="0 0 6 6">
				<path d="M0,0 L0,6 L6,3 z" fill="rgb(255,0,0)"/>
			</marker>
		</defs>
		<path d="M10,50 L90,50" stroke="black" stroke-width="2" marker-end="url(#arrow)"/>
	</svg>`

	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	if c == nil {
		t.Fatal("canvas is nil")
	}

	totalLayers := 0
	for _, layers := range c.layers {
		totalLayers += len(layers)
	}
	if totalLayers < 2 {
		t.Fatalf("expected at least 2 layers (path + marker), got %d", totalLayers)
	}
}

func TestParseSVGPatternLineHatch(t *testing.T) {
	svg := `<svg width="100" height="100">
		<defs>
			<pattern id="hatch" patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(45)">
				<line x1="0" y1="0" x2="0" y2="6" stroke="rgb(255,0,0)" stroke-width="1"/>
			</pattern>
		</defs>
		<circle cx="50" cy="50" r="40" fill="url(#hatch)"/>
	</svg>`

	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	if len(c.layers) == 0 || len(c.layers[0]) == 0 {
		t.Fatal("no layers rendered")
	}

	layer := c.layers[0][0]
	if !layer.style.Fill.IsPattern() {
		t.Fatal("expected fill to be a pattern")
	}

	hatch, ok := layer.style.Fill.Pattern.(*HatchPattern)
	if !ok {
		t.Fatal("expected HatchPattern")
	}

	test.T(t, hatch.Fill.Color, color.RGBA{255, 0, 0, 255})
}

func TestParseSVGPatternCurrentColor(t *testing.T) {
	svg := `<svg width="100" height="100">
		<defs>
			<pattern id="hatch" patternUnits="userSpaceOnUse" width="6" height="6" patternTransform="rotate(45)">
				<line x1="0" y1="0" x2="0" y2="6" stroke="currentColor" stroke-width="1" stroke-opacity="0.7"/>
			</pattern>
		</defs>
		<circle cx="50" cy="50" r="40" fill="url(#hatch)" style="color:rgb(0,255,0)"/>
	</svg>`

	c, err := ParseSVG(strings.NewReader(svg))
	test.Error(t, err)

	if len(c.layers) == 0 || len(c.layers[0]) == 0 {
		t.Fatal("no layers rendered")
	}

	layer := c.layers[0][0]
	if !layer.style.Fill.IsPattern() {
		t.Fatal("expected fill to be a pattern")
	}

	hatch, ok := layer.style.Fill.Pattern.(*HatchPattern)
	if !ok {
		t.Fatal("expected HatchPattern")
	}

	expected := color.RGBA{0, 178, 0, 178}
	test.T(t, hatch.Fill.Color, expected)
}
