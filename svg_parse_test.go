package canvas

import (
	"strings"
	"testing"

	"github.com/tdewolff/test"
)

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
