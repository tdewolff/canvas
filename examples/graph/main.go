package main

import (
	"encoding/csv"
	"fmt"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/tdewolff/canvas"
)

var dejaVuSerif *canvas.FontFamily

func main() {
	dejaVuSerif = canvas.NewFontFamily("dejavu-serif")
	dejaVuSerif.Use(canvas.CommonLigatures)
	if err := dejaVuSerif.LoadFontFile("../DejaVuSerif.ttf", canvas.FontRegular); err != nil {
		panic(err)
	}

	c := canvas.New(140, 110)
	draw(c)

	pngFile, err := os.Create("out.png")
	if err != nil {
		panic(err)
	}
	defer pngFile.Close()

	img := c.WriteImage(5.0)
	err = png.Encode(pngFile, img)
	if err != nil {
		panic(err)
	}
}

func draw(c *canvas.Canvas) {
	tickFace := dejaVuSerif.Face(8.0, canvas.Black, canvas.FontRegular, canvas.FontNormal)

	datafile, err := os.Open("co2-mm-mlo.csv")
	if err != nil {
		panic(err)
	}
	r := csv.NewReader(datafile)
	if _, err = r.Read(); err != nil { // skip header
		panic(err)
	}

	date := []float64{}
	co2 := []float64{}
	trend := []float64{}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		fdate, _ := strconv.ParseFloat(row[1], 64)
		fco2, _ := strconv.ParseFloat(row[3], 64)
		ftrend, _ := strconv.ParseFloat(row[4], 64)
		date = append(date, fdate)
		co2 = append(co2, fco2)
		trend = append(trend, ftrend)
	}

	n := len(date)
	xmin, xmax := date[0], date[n-1]
	ymin, ymax := co2[0], co2[0]
	for _, y := range co2[1:] {
		ymin = math.Min(ymin, y)
		ymax = math.Max(ymax, y)
	}
	ymargin := (ymax - ymin) * 0.05
	ymin -= ymargin
	ymax += ymargin

	xscale := 120.0 / (xmax - xmin)
	yscale := 80.0 / (ymax - ymin)

	c.PushState()
	c.SetView(canvas.Identity.Translate(15.0, 15.0))
	viewport := canvas.Identity.Scale(xscale, yscale).Translate(-xmin, -ymin)

	// Draw the function
	co2Line := &canvas.Polyline{}
	trendLine := &canvas.Polyline{}
	for i := range date {
		co2Line.Add(date[i], co2[i])
		trendLine.Add(date[i], trend[i])
	}

	c.SetFillColor(canvas.Seagreen)
	c.DrawPath(0, 0, trendLine.ToPath().Transform(viewport).Stroke(0.4, canvas.RoundCapper, canvas.RoundJoiner))

	c.SetFillColor(color.RGBA{192, 0, 64, 255})
	c.DrawPath(0, 0, co2Line.ToPath().Transform(viewport).Stroke(0.1, canvas.RoundCapper, canvas.RoundJoiner))
	marker := canvas.Ellipse(0.3, 0.3)
	for _, m := range co2Line.ToPath().Transform(viewport).Markers(marker, marker, marker, false) {
		c.DrawPath(0, 0, m)
	}

	// Draw plot frame
	c.SetFillColor(canvas.Transparent)
	c.SetStrokeColor(canvas.Black)
	c.SetStrokeWidth(0.3)
	c.SetStrokeCapper(canvas.RoundCapper)
	c.SetStrokeJoiner(canvas.RoundJoiner)

	frame := canvas.Rectangle(xmax-xmin, ymax-ymin).Translate(xmin, ymin)
	for x := 10.0 * float64(int(xmin/10.0)+1); x < xmax; x += 10.0 {
		frame.MoveTo(x, ymin).LineTo(x, ymin+2.0/yscale)
		c.DrawText(x, ymin-tickFace.Metrics().LineHeight/yscale, canvas.NewTextLine(tickFace, fmt.Sprintf("%g", x), canvas.Center))
	}
	for y := 10.0 * float64(int(ymin/10.0)+1); y < ymax; y += 10.0 {
		frame.MoveTo(xmin, y).LineTo(xmin+2.0/xscale, y)
		c.DrawText(xmin, y-(tickFace.Metrics().CapHeight/2.0)/yscale, canvas.NewTextLine(tickFace, fmt.Sprintf("%g ", y), canvas.Right))
	}
	c.DrawPath(0.0, 0.0, frame.Transform(viewport))

	// Draw the labels
	c.SetFillColor(canvas.Black)

	labelFace := dejaVuSerif.Face(14.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	labelSubFace := dejaVuSerif.Face(14.0, color.Black, canvas.FontRegular, canvas.FontSubscript)
	rt := canvas.NewRichText()
	rt.Add(labelFace, "CO")
	rt.Add(labelSubFace, "2")
	rt.Add(labelFace, " (ppm)")
	c.PushState()
	c.ComposeView(canvas.Identity.Rotate(90))
	text := rt.ToText(0.0, 0.0, canvas.Center, canvas.Top, 0.0, 0.0)
	c.DrawText(-10.0, 40.0, text)
	c.PopState()
	c.DrawText(55.0, -10.0, canvas.NewTextLine(labelFace, "Year", canvas.Center))

	titleFace := dejaVuSerif.Face(16.0, color.Black, canvas.FontRegular, canvas.FontNormal)
	titleSubFace := dejaVuSerif.Face(16.0, color.Black, canvas.FontRegular, canvas.FontSubscript)
	rt = canvas.NewRichText()
	rt.Add(titleFace, "Atmospheric CO")
	rt.Add(titleSubFace, "2")
	rt.Add(titleFace, " at Mauna Loa Observatory")
	c.DrawText(55.0, 91.0, rt.ToText(0.0, 0.0, canvas.Center, canvas.Top, 0.0, 0.0))
}
