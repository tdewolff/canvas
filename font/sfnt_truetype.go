package font

import (
	"encoding/binary"
	"fmt"
	"strings"
)

////////////////////////////////////////////////////////////////

type glyfContour struct {
	GlyphID                uint16
	XMin, YMin, XMax, YMax int16
	EndPoints              []uint16
	Instructions           []byte
	OnCurve                []bool
	XCoordinates           []int16
	YCoordinates           []int16
}

func (contour *glyfContour) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Glyph %v:\n", contour.GlyphID)
	fmt.Fprintf(&b, "  Contours: %v\n", len(contour.EndPoints))
	fmt.Fprintf(&b, "  XMin: %v\n", contour.XMin)
	fmt.Fprintf(&b, "  YMin: %v\n", contour.YMin)
	fmt.Fprintf(&b, "  XMax: %v\n", contour.XMax)
	fmt.Fprintf(&b, "  YMax: %v\n", contour.YMax)
	fmt.Fprintf(&b, "  EndPoints: %v\n", contour.EndPoints)
	fmt.Fprintf(&b, "  Instruction length: %v\n", len(contour.Instructions))
	if len(contour.EndPoints) == 0 {
		fmt.Fprintf(&b, "  Empty glyph\n")
	} else {
		fmt.Fprintf(&b, "  Coordinates:\n")
		for i := 0; i <= int(contour.EndPoints[len(contour.EndPoints)-1]); i++ {
			fmt.Fprintf(&b, "    ")
			if i < len(contour.XCoordinates) {
				fmt.Fprintf(&b, "%8v", contour.XCoordinates[i])
			} else {
				fmt.Fprintf(&b, "  ----  ")
			}
			if i < len(contour.YCoordinates) {
				fmt.Fprintf(&b, " %8v", contour.YCoordinates[i])
			} else {
				fmt.Fprintf(&b, "   ----  ")
			}
			if i < len(contour.OnCurve) {
				onCurve := "Off"
				if contour.OnCurve[i] {
					onCurve = "On"
				}
				fmt.Fprintf(&b, " %3v\n", onCurve)
			} else {
				fmt.Fprintf(&b, " ---\n")
			}
		}
	}
	return b.String()
}

type glyfTable struct {
	data []byte
	head *headTable
	loca *locaTable
}

func (glyf *glyfTable) Get(glyphID uint16) []byte {
	start, ok1 := glyf.loca.Get(glyphID)
	end, ok2 := glyf.loca.Get(glyphID + 1)
	if !ok1 || !ok2 {
		return nil
	}
	return glyf.data[start:end]
}

func (glyf *glyfTable) Dependencies(glyphID uint16, level int) ([]uint16, error) {
	deps := []uint16{glyphID}
	b := glyf.Get(glyphID)
	if b == nil {
		return nil, fmt.Errorf("glyf: bad glyphID %v", glyphID)
	} else if len(b) == 0 {
		return deps, nil
	}
	r := newBinaryReader(b)
	if r.Len() < 10 {
		return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
	}
	numberOfContours := r.ReadInt16()
	_ = r.ReadBytes(8)
	if numberOfContours < 0 {
		if 7 < level {
			return nil, fmt.Errorf("glyf: compound glyphs too deeply nested")
		}

		// composite glyph
		for {
			if r.Len() < 4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags := r.ReadUint16()
			subGlyphID := r.ReadUint16()
			subDeps, err := glyf.Dependencies(subGlyphID, level+1)
			if err != nil {
				return nil, err
			}
			deps = append(deps, subDeps...)

			length, more := glyfCompositeLength(flags)
			if r.Len() < length-4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}
			_ = r.ReadBytes(length - 4)
			if !more {
				break
			}
		}
	}
	return deps, nil
}

func glyfCompositeLength(flags uint16) (length uint32, more bool) {
	length = 4 + 2
	if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
		length += 2
	}
	if flags&0x0008 != 0 { // WE_HAVE_A_SCALE
		length += 2
	} else if flags&0x0040 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
		length += 4
	} else if flags&0x0080 != 0 { // WE_HAVE_A_TWO_BY_TWO
		length += 8
	}
	more = flags&0x0020 != 0 // MORE_COMPONENTS
	return
}

func (glyf *glyfTable) Contour(glyphID uint16, level int) (*glyfContour, error) {
	b := glyf.Get(glyphID)
	if b == nil {
		return nil, fmt.Errorf("glyf: bad glyphID %v", glyphID)
	} else if len(b) == 0 {
		return &glyfContour{GlyphID: glyphID}, nil
	}
	r := newBinaryReader(b)
	if r.Len() < 10 {
		return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
	}

	contour := &glyfContour{}
	contour.GlyphID = glyphID
	numberOfContours := r.ReadInt16()
	contour.XMin = r.ReadInt16()
	contour.YMin = r.ReadInt16()
	contour.XMax = r.ReadInt16()
	contour.YMax = r.ReadInt16()
	if 0 <= numberOfContours {
		// simple glyph
		if r.Len() < 2*uint32(numberOfContours)+2 {
			return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
		}
		contour.EndPoints = make([]uint16, numberOfContours)
		for i := 0; i < int(numberOfContours); i++ {
			contour.EndPoints[i] = r.ReadUint16()
		}

		instructionLength := r.ReadUint16()
		if r.Len() < uint32(instructionLength) {
			return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
		}
		contour.Instructions = r.ReadBytes(uint32(instructionLength))

		numPoints := int(contour.EndPoints[numberOfContours-1]) + 1
		flags := make([]byte, numPoints)
		contour.OnCurve = make([]bool, numPoints)
		for i := 0; i < numPoints; i++ {
			if r.Len() < 1 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags[i] = r.ReadByte()
			contour.OnCurve[i] = flags[i]&0x01 != 0
			if flags[i]&0x08 != 0 { // REPEAT_FLAG
				repeat := r.ReadByte()
				for j := 1; j <= int(repeat); j++ {
					flags[i+j] = flags[i]
					contour.OnCurve[i+j] = contour.OnCurve[i]
				}
				i += int(repeat)
			}
		}

		var x int16
		contour.XCoordinates = make([]int16, numPoints)
		for i := 0; i < numPoints; i++ {
			xShortVector := flags[i]&0x02 != 0
			xIsSameOrPositiveXShortVector := flags[i]&0x10 != 0
			if xShortVector {
				if r.Len() < 1 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				if xIsSameOrPositiveXShortVector {
					x += int16(r.ReadUint8())
				} else {
					x -= int16(r.ReadUint8())
				}
			} else if !xIsSameOrPositiveXShortVector {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				x += r.ReadInt16()
			}
			contour.XCoordinates[i] = x
		}

		var y int16
		contour.YCoordinates = make([]int16, numPoints)
		for i := 0; i < numPoints; i++ {
			yShortVector := flags[i]&0x04 != 0
			yIsSameOrPositiveYShortVector := flags[i]&0x20 != 0
			if yShortVector {
				if r.Len() < 1 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				if yIsSameOrPositiveYShortVector {
					y += int16(r.ReadUint8())
				} else {
					y -= int16(r.ReadUint8())
				}
			} else if !yIsSameOrPositiveYShortVector {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table or flags for glyphID %v", glyphID)
				}
				y += r.ReadInt16()
			}
			contour.YCoordinates[i] = y
		}
	} else {
		if 7 < level {
			return nil, fmt.Errorf("glyf: compound glyphs too deeply nested")
		}

		// composite glyph
		hasInstructions := false
		for {
			if r.Len() < 4 {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}

			flags := r.ReadUint16()
			subGlyphID := r.ReadUint16()
			if flags&0x0002 == 0 { // ARGS_ARE_XY_VALUES
				return nil, fmt.Errorf("glyf: composite glyph not supported")
			}
			var dx, dy int16
			if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
				if r.Len() < 4 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				dx = r.ReadInt16()
				dy = r.ReadInt16()
			} else {
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				dx = int16(r.ReadInt8())
				dy = int16(r.ReadInt8())
			}
			var txx, txy, tyx, tyy int16
			if flags&0x0008 != 0 { // WE_HAVE_A_SCALE
				if r.Len() < 2 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				tyy = txx
			} else if flags&0x0040 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
				if r.Len() < 4 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				tyy = r.ReadInt16()
			} else if flags&0x0080 != 0 { // WE_HAVE_A_TWO_BY_TWO
				if r.Len() < 8 {
					return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
				}
				txx = r.ReadInt16()
				txy = r.ReadInt16()
				tyx = r.ReadInt16()
				tyy = r.ReadInt16()
			}
			if flags&0x0100 != 0 {
				hasInstructions = true
			}

			subContour, err := glyf.Contour(subGlyphID, level+1)
			if err != nil {
				return nil, err
			}

			var numPoints uint16
			if 0 < len(contour.EndPoints) {
				numPoints = contour.EndPoints[len(contour.EndPoints)-1] + 1
			}
			for i := 0; i < len(subContour.EndPoints); i++ {
				contour.EndPoints = append(contour.EndPoints, numPoints+subContour.EndPoints[i])
			}
			contour.OnCurve = append(contour.OnCurve, subContour.OnCurve...)
			for i := 0; i < len(subContour.XCoordinates); i++ {
				x := subContour.XCoordinates[i]
				y := subContour.YCoordinates[i]
				if flags&0x00C8 != 0 { // has transformation
					const half = 1 << 13
					xt := int16((int64(x)*int64(txx)+half)>>14) + int16((int64(y)*int64(tyx)+half)>>14)
					yt := int16((int64(x)*int64(txy)+half)>>14) + int16((int64(y)*int64(tyy)+half)>>14)
					x, y = xt, yt
				}
				contour.XCoordinates = append(contour.XCoordinates, dx+x)
				contour.YCoordinates = append(contour.YCoordinates, dy+y)
			}
			if flags&0x0020 == 0 { // MORE_COMPONENTS
				break
			}
		}
		if hasInstructions {
			instructionLength := r.ReadUint16()
			if r.Len() < uint32(instructionLength) {
				return nil, fmt.Errorf("glyf: bad table for glyphID %v", glyphID)
			}
			contour.Instructions = r.ReadBytes(uint32(instructionLength))
		}
	}
	return contour, nil
}

func (glyf *glyfTable) ToPath(p Pather, glyphID, ppem uint16, xOffset, yOffset int32, f float64, hinting Hinting) error {
	contour, err := glyf.Contour(glyphID, 0)
	if err != nil {
		return err
	}

	x, y := f*float64(xOffset), f*float64(yOffset)

	var i uint16
	for _, endPoint := range contour.EndPoints {
		j := i
		first := true
		firstOff := false
		prevOff := false
		startX, startY := 0.0, 0.0
		for ; i <= endPoint; i++ {
			if first {
				if contour.OnCurve[i] {
					startX = float64(contour.XCoordinates[i])
					startY = float64(contour.YCoordinates[i])
					p.MoveTo(x+f*startX, y+f*startY)
					first = false
				} else if !prevOff {
					// first point is off
					firstOff = true
					prevOff = true
				} else {
					// first and second point are off
					startX = float64(contour.XCoordinates[i-1]+contour.XCoordinates[i]) / 2.0
					startY = float64(contour.YCoordinates[i-1]+contour.YCoordinates[i]) / 2.0
					p.MoveTo(x+f*startX, y+f*startY)
					first = false
				}
			} else if !prevOff {
				if contour.OnCurve[i] {
					p.LineTo(x+f*float64(contour.XCoordinates[i]), y+f*float64(contour.YCoordinates[i]))
				} else {
					prevOff = true
				}
			} else {
				if contour.OnCurve[i] {
					p.QuadTo(x+f*float64(contour.XCoordinates[i-1]), y+f*float64(contour.YCoordinates[i-1]), x+f*float64(contour.XCoordinates[i]), y+f*float64(contour.YCoordinates[i]))
					prevOff = false
				} else {
					midX := float64(contour.XCoordinates[i-1]+contour.XCoordinates[i]) / 2.0
					midY := float64(contour.YCoordinates[i-1]+contour.YCoordinates[i]) / 2.0
					p.QuadTo(x+f*float64(contour.XCoordinates[i-1]), y+f*float64(contour.YCoordinates[i-1]), x+f*midX, y+f*midY)
				}
			}
		}
		if firstOff {
			if prevOff {
				midX := float64(contour.XCoordinates[i-1]+contour.XCoordinates[j]) / 2.0
				midY := float64(contour.YCoordinates[i-1]+contour.YCoordinates[j]) / 2.0
				p.QuadTo(x+f*midX, y+f*midY, x+f*startX, y+f*startY)
			} else {
				p.QuadTo(x+f*float64(contour.XCoordinates[i-1]), y+f*float64(contour.YCoordinates[i-1]), x+f*startX, y+f*startY)
			}
		} else if prevOff {
			p.QuadTo(x+f*float64(contour.XCoordinates[i-1]), y+f*float64(contour.YCoordinates[i-1]), x+f*startX, y+f*startY)
		}
		p.Close()
	}
	return nil
}

func (sfnt *SFNT) parseGlyf() error {
	// requires data from loca
	b, ok := sfnt.Tables["glyf"]
	if !ok {
		return fmt.Errorf("glyf: missing table")
	} else if length, _ := sfnt.Loca.Get(sfnt.Maxp.NumGlyphs); uint32(len(b)) < length {
		return fmt.Errorf("glyf: bad table")
	}

	sfnt.Glyf = &glyfTable{
		data: b,
		loca: sfnt.Loca,
	}
	return nil
}

////////////////////////////////////////////////////////////////

type locaTable struct {
	format int16
	data   []byte
}

func (loca *locaTable) Get(glyphID uint16) (uint32, bool) {
	if loca.format == 0 && int(glyphID)*2 < len(loca.data) {
		return uint32(binary.BigEndian.Uint16(loca.data[int(glyphID)*2:])), true
	} else if loca.format == 1 && int(glyphID)*4 < len(loca.data) {
		return binary.BigEndian.Uint32(loca.data[int(glyphID)*4:]), true
	}
	return 0, false
}

func (sfnt *SFNT) parseLoca() error {
	b, ok := sfnt.Tables["loca"]
	if !ok {
		return fmt.Errorf("loca: missing table")
	}

	sfnt.Loca = &locaTable{
		format: sfnt.Head.IndexToLocFormat,
		data:   b,
	}
	//sfnt.Loca.Offsets = make([]uint32, sfnt.Maxp.NumGlyphs+1)
	//r := newBinaryReader(b)
	//if sfnt.Head.IndexToLocFormat == 0 {
	//	if uint32(len(b)) != 2*(uint32(sfnt.Maxp.NumGlyphs)+1) {
	//		return fmt.Errorf("loca: bad table")
	//	}
	//	for i := 0; i < int(sfnt.Maxp.NumGlyphs+1); i++ {
	//		sfnt.Loca.Offsets[i] = uint32(r.ReadUint16())
	//		if 0 < i && sfnt.Loca.Offsets[i] < sfnt.Loca.Offsets[i-1] {
	//			return fmt.Errorf("loca: bad offsets")
	//		}
	//	}
	//} else if sfnt.Head.IndexToLocFormat == 1 {
	//	if uint32(len(b)) != 4*(uint32(sfnt.Maxp.NumGlyphs)+1) {
	//		return fmt.Errorf("loca: bad table")
	//	}
	//	for i := 0; i < int(sfnt.Maxp.NumGlyphs+1); i++ {
	//		sfnt.Loca.Offsets[i] = r.ReadUint32()
	//		if 0 < i && sfnt.Loca.Offsets[i] < sfnt.Loca.Offsets[i-1] {
	//			return fmt.Errorf("loca: bad offsets")
	//		}
	//	}
	//}
	return nil
}
