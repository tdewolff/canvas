package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"image/color"
	"os"

	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmapi"
	"github.com/paulmach/osm/osmgeojson"
	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers"
)

func main() {
	c := canvas.New(100, 100)
	ctx := canvas.NewContext(c)
	draw(ctx)
	renderers.Write("out.png", c, canvas.DPMM(8.0))
}

func fetch(filename string, bounds *osm.Bounds) (*osm.OSM, error) {
	if _, err := os.Stat(filename); err == nil {
		m := &osm.OSM{}
		if f, err := os.Open(filename); err != nil {
			return nil, err
		} else if err := gob.NewDecoder(f).Decode(m); err != nil {
			return nil, err
		}
		return m, nil
	}

	fmt.Printf("Fetching %s from OSM API...", filename)
	m, err := osmapi.Map(context.Background(), bounds)
	if err != nil {
		return nil, err
	}
	fmt.Println("done")
	if f, err := os.Create(filename); err != nil {
		return m, err
	} else if err := gob.NewEncoder(f).Encode(m); err != nil {
		return m, err
	}
	return m, nil
}

func draw(c *canvas.Context) {
	xmin, xmax := 4.8884, 4.9090
	ymin, ymax := 52.3659, 52.3779

	xmid := xmin + (xmax-xmin)/2.0
	ams0, err := fetch("ams0.osm", &osm.Bounds{ymin, ymax, xmin, xmid})
	if err != nil {
		panic(err)
	}
	ams1, err := fetch("ams1.osm", &osm.Bounds{ymin, ymax, xmid, xmax})
	if err != nil {
		panic(err)
	}

	categories := map[string]color.RGBA{
		"route_primary":     {248, 201, 103, 255},
		"route_secondary":   {253, 252, 248, 255},
		"route_residential": {245, 241, 230, 255},
		"route_pedestrian":  {245, 241, 230, 255},
		"route_transit":     {223, 210, 174, 255},
		"water":             {185, 211, 194, 255},
		"park":              {165, 176, 118, 255},
		"building":          {201, 178, 166, 255},
	}

	c.SetFillColor(color.RGBA{235, 227, 205, 255})
	c.DrawPath(0.0, 0.0, canvas.Rectangle(100.0, 100.0))

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
				} else if manMade, ok := tags["man_made"]; ok && manMade == "bridge" {
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
	view := canvas.Identity.Translate(0.0, 0.0).Scale(xscale, yscale).Translate(-xmin, -ymin)

	c.SetStrokeWidth(0.1)
	catOrder := []string{"water", "route_pedestrian", "route_residential", "route_secondary", "route_primary", "route_transit", "park", "building"}
	for _, cat := range catOrder {
		c.SetFillColor(categories[cat])
		if cat == "building" || cat == "park" {
			c.SetStrokeColor(color.RGBA{64, 64, 64, 128})
		} else {
			c.SetStrokeColor(canvas.Transparent)
		}
		if lines[cat] != nil {
			width := 0.5
			if cat == "route_residential" {
				width /= 1.5
			} else if cat == "route_primary" {
				width *= 1.5
			} else if cat == "route_pedestrian" {
				width /= 2.5
			} else if cat == "route_transit" {
				width /= 8.0
			}
			p := lines[cat].Transform(view)
			p = p.Stroke(width, canvas.RoundCap, canvas.RoundJoin, 0.01)
			c.DrawPath(0.0, 0.0, p)
		}
		if rings[cat] != nil {
			p := rings[cat].Transform(view)
			c.DrawPath(0.0, 0.0, p)
		}
	}

	c.ResetView()
	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Darkgray)
	c.SetStrokeWidth(0.5)
	c.DrawPath(0.0, 0.0, canvas.Rectangle(c.Width(), c.Height()))
}
