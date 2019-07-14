package main

import (
	"context"
	"fmt"
	"image/color"
	"image/png"
	"os"

	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmapi"
	"github.com/paulmach/osm/osmgeojson"
	"github.com/tdewolff/canvas"
)

var dejaVuSerif *canvas.FontFamily

func main() {
	dejaVuSerif = canvas.NewFontFamily("dejavu-serif")
	dejaVuSerif.Use(canvas.CommonLigatures)
	if err := dejaVuSerif.LoadFontFile("DejaVuSerif.ttf", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(100, 100)
	Draw(c)

	pngFile, err := os.Create("map_example.png")
	if err != nil {
		panic(err)
	}
	defer pngFile.Close()

	img := c.WriteImage(216.0)
	err = png.Encode(pngFile, img)
	if err != nil {
		panic(err)
	}
}

func Draw(c *canvas.Canvas) {
	xmin, xmax := 4.8884, 4.9090
	ymin, ymax := 52.3659, 52.3779

	xmid := xmin + (xmax-xmin)/2.0
	ams0, err := osmapi.Map(context.Background(), &osm.Bounds{ymin, ymax, xmin, xmid})
	if err != nil {
		panic(err)
	}
	ams1, err := osmapi.Map(context.Background(), &osm.Bounds{ymin, ymax, xmid, xmax})
	if err != nil {
		panic(err)
	}

	categories := map[string]color.RGBA{
		"route_primary":     color.RGBA{248, 201, 103, 255},
		"route_secondary":   color.RGBA{253, 252, 248, 255},
		"route_residential": color.RGBA{245, 241, 230, 255},
		"route_pedestrian":  color.RGBA{245, 241, 230, 255},
		"route_transit":     color.RGBA{223, 210, 174, 255},
		"water":             color.RGBA{185, 211, 194, 255},
		"park":              color.RGBA{165, 176, 118, 255},
		"building":          color.RGBA{201, 178, 166, 255},
	}

	c.SetFillColor(color.RGBA{235, 227, 205, 255})
	c.DrawPath(0.0, 0.0, canvas.Rectangle(141.0, 111.0))

	lines := map[string]*canvas.Path{}
	rings := map[string]*canvas.Path{}
	for _, ams := range []*osm.OSM{ams0, ams1} {

		fc, err := osmgeojson.Convert(ams,
			osmgeojson.NoID(true),
			osmgeojson.NoMeta(true),
			osmgeojson.NoRelationMembership(true))
		if err != nil {
			panic(err)
		}

		for _, f := range fc.Features {
			if tags, ok := f.Properties["tags"].(map[string]string); ok {

				var category string
				if hw, ok := tags["highway"]; ok {
					if hw != "primary" && hw != "secondary" && hw != "unclassified" && hw != "residential" && hw != "pedestrian" {
						continue
					}
					if hw == "unclassified" {
						hw = "residential"
					}
					category = "route_" + hw
				} else if man_made, ok := tags["man_made"]; ok && man_made == "bridge" {
					category = "route_residential"
				} else if _, ok := tags["natural"]; ok {
					category = "water"
				} else if railway, ok := tags["railway"]; ok && railway == "rail" {
					category = "route_transit"
				} else if leisure, ok := tags["leisure"]; ok {
					if leisure != "park" && leisure != "garden" && leisure != "playground" {
						continue
					}
					category = "park"
				} else if _, ok := tags["amenity"]; ok {
					category = "building"
				} else {
					continue
				}

				if g, ok := f.Geometry.(orb.LineString); ok && 1 < len(g) {
					p := &canvas.Path{}
					p.MoveTo(g[0][0], g[0][1])
					for _, point := range g {
						p.LineTo(point[0], point[1])
					}
					if _, ok := lines[category]; !ok {
						lines[category] = p
					} else {
						lines[category] = lines[category].Append(p)
					}
				} else if g, ok := f.Geometry.(orb.Polygon); ok {
					for _, ring := range g {
						if len(ring) == 0 {
							continue
						}

						p := &canvas.Path{}
						p.MoveTo(ring[0][0], ring[0][1])
						for _, point := range ring {
							p.LineTo(point[0], point[1])
						}
						p.Close()
						if _, ok := rings[category]; !ok {
							rings[category] = p
						} else {
							rings[category] = rings[category].Append(p)
						}
					}
				} else if g, ok := f.Geometry.(orb.MultiPolygon); ok {
					for _, poly := range g {
						for _, ring := range poly {
							if len(ring) == 0 {
								continue
							}

							p := &canvas.Path{}
							p.MoveTo(ring[0][0], ring[0][1])
							for _, point := range ring {
								p.LineTo(point[0], point[1])
							}
							p.Close()
							if _, ok := rings[category]; !ok {
								rings[category] = p
							} else {
								rings[category] = rings[category].Append(p)
							}
						}
					}
				} else if _, ok := f.Geometry.(orb.Point); ok {
				} else {
					fmt.Println("unsupported geometry:", f.Geometry)
				}
			}
		}
	}

	xscale := 100.0 / (xmax - xmin)
	yscale := 100.0 / (ymax - ymin)
	c.SetView(canvas.Identity.Translate(0.0, 0.0).Scale(xscale, yscale).Translate(-xmin, -ymin))

	catOrder := []string{"water", "route_pedestrian", "route_residential", "route_secondary", "route_primary", "route_transit", "park", "building"}
	for _, cat := range catOrder {
		c.SetFillColor(categories[cat])
		if lines[cat] != nil {
			width := 0.00015
			if cat == "route_residential" {
				width /= 1.5
			} else if cat == "route_primary" {
				width *= 1.5
			} else if cat == "route_pedestrian" || cat == "route_transit" {
				width /= 2.5
			}
			c.DrawPath(0.0, 0.0, lines[cat].Stroke(width, canvas.RoundCapper, canvas.RoundJoiner))
		}
		if rings[cat] != nil {
			c.DrawPath(0.0, 0.0, rings[cat])
		}
	}
}
