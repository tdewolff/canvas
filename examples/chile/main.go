package main

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"github.com/wroge/wgs84/v2"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

func read(filename string) (*canvas.Path, error) {
	var p *canvas.Path
	if r, err := os.Open(filename); err != nil {
		return nil, err
	} else if rGzip, err := gzip.NewReader(r); err != nil {
		r.Close()
		return nil, err
	} else if err := gob.NewDecoder(rGzip).Decode(&p); err != nil {
		rGzip.Close()
		r.Close()
		return nil, err
	}
	return p, nil
}

func main() {
	chile, err := read("chile.path.gz")
	if err != nil {
		panic(err)
	}

	europe, err := read("europe.path.gz")
	if err != nil {
		panic(err)
	}
	fmt.Println(europe.Len(), europe.Bounds())

	// remove islands (coordinates in WGS84)
	chile = chile.Clip(-78, -60, -62, -16)
	europe = europe.Clip(-12, 30, 32, 72)

	// transform Chile to UTM 19 south, this has the least distortion for Chile
	utm19S := wgs84.Transform(wgs84.EPSG(4326), wgs84.EPSG(32719))
	chile = chile.TransformFunc(func(x, y float64) (float64, float64) {
		x, y, _ = utm19S(x, y, 0.0)
		return x / 1e5, y / 1e5
	})

	// transform Europe to UTM 33 north, this has the least distortion for Norway/Italy
	utm33N := wgs84.Transform(wgs84.EPSG(4326), wgs84.EPSG(32633))
	europe = europe.TransformFunc(func(x, y float64) (float64, float64) {
		x, y, _ = utm33N(x, y, 0.0)
		return x / 1e5, y / 1e5
	})

	// simplify using the Visvalingam-Whyatt algorithm this greatly reduces the number of line segments
	chile = chile.SimplifyVisvalingamWhyatt(0.0002)
	europe = europe.SimplifyVisvalingamWhyatt(0.0002)

	bounds := chile.Bounds().Add(europe.Bounds())
	chile.Translate(-bounds.X0+1.0, -bounds.Y0+1.0)
	europe.Translate(-bounds.X0+1.0, -bounds.Y0+1.0)

	c := canvas.New(bounds.W()+2.0, bounds.H()+2.0)
	ctx := canvas.NewContext(c)

	// background
	ctx.SetFillColor(canvas.White)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.W, c.H))

	candyShop := canvas.Stops{}
	candyShop.Add(0.0, canvas.RGB(0.31, 0.14, 0.33))
	candyShop.Add(0.5, canvas.RGB(0.87, 0.85, 0.65))
	candyShop.Add(1.0, canvas.RGB(0.54, 0.99, 0.77))

	ctx.SetStrokeWidth(0.015)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetFillColor(canvas.Hex("#0001"))
	ctx.DrawPath(0.0, 0.0, europe)
	//ctx.SetFillColor(color.RGBA{164, 204, 144, 255})
	ctx.SetFillColor(canvas.Hex("#F002"))
	ctx.DrawPath(0.0, 0.0, chile)
	//ctx.SetFillColor(color.RGBA{44, 48, 113, 128})

	t := time.Now()
	overlap := chile.And(europe)
	fmt.Printf("%v chile=%d europe=%d\n", time.Since(t), chile.Len(), europe.Len())
	bounds = overlap.Bounds()
	gradient := canvas.NewLinearGradient(canvas.Point{bounds.X0, bounds.Y0}, canvas.Point{bounds.X1, bounds.Y1})
	gradient.Stops = candyShop
	ctx.SetFill(gradient)
	ctx.SetStrokeColor(canvas.Black)
	ctx.SetStrokeWidth(0.025)
	ctx.DrawPath(0.0, 0.0, overlap)

	renderers.Write("chile.png", c, canvas.DPMM(30.0))
}
