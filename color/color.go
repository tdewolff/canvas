package color

import "image/color"

func OpaqueModel(cm color.Model) bool {
	switch cm {
	case color.GrayModel, color.Gray16Model, color.CMYKModel, color.YCbCrModel, RGBModel, RGB48Model:
		return true
	}
	return false
}

type RGB struct {
	R, G, B uint8
}

func (c RGB) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)<<8 | uint32(c.R)
	g = uint32(c.G)<<8 | uint32(c.G)
	b = uint32(c.B)<<8 | uint32(c.B)
	a = 0xffff
	return
}

var RGBModel = color.ModelFunc(rgbModel)

func rgbModel(c color.Color) color.Color {
	if _, ok := c.(RGB); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return RGB{}
	} else if a != 0xffff {
		r = r * 0xffff / a
		g = g * 0xffff / a
		b = b * 0xffff / a
		return RGB{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
	}
	return RGB{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}

type RGB48 struct {
	R, G, B uint16
}

func (c RGB48) RGBA() (r, g, b, a uint32) {
	r, g, b, a = uint32(c.R), uint32(c.G), uint32(c.B), 0xffff
	return
}

var RGB48Model = color.ModelFunc(rgb48Model)

func rgb48Model(c color.Color) color.Color {
	if _, ok := c.(RGB48); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return RGB48{}
	} else if a != 0xffff {
		r = r * 0xffff / a
		g = g * 0xffff / a
		b = b * 0xffff / a
	}
	return RGB48{uint16(r), uint16(g), uint16(b)}
}

type GrayA struct {
	Y, A uint8
}

func (c GrayA) RGBA() (r, g, b, a uint32) {
	y := uint32(c.Y)<<8 | uint32(c.Y)
	r, g, b, a = y, y, y, uint32(c.A)<<8|uint32(c.A)
	return
}

var GrayAModel = color.ModelFunc(grayaModel)

func grayaModel(c color.Color) color.Color {
	if _, ok := c.(GrayA); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	y := uint8((19595*r + 38470*g + 7471*b + 1<<15) >> 24)
	return GrayA{y, uint8(a >> 8)}
}

type GrayA32 struct {
	Y, A uint16
}

func (c GrayA32) RGBA() (r, g, b, a uint32) {
	r, g, b, a = uint32(c.Y), uint32(c.Y), uint32(c.Y), uint32(c.A)
	return
}

var GrayA32Model = color.ModelFunc(graya32Model)

func graya32Model(c color.Color) color.Color {
	if _, ok := c.(GrayA32); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	y := (19595*r + 38470*g + 7471*b + 1<<15) >> 16
	return GrayA32{uint16(y), uint16(a)}
}
