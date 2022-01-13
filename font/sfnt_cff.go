package font

import (
	"fmt"
	"math"
	"strconv"
)

// TODO: use FDSelect for Font DICTs
// TODO: CFF has winding rule even-odd? CFF2 has winding rule nonzero

type cffTable struct {
	version     int
	top         *cffTopDICT
	charStrings *cffINDEX
	globalSubrs *cffINDEX
	fonts       *cffFontINDEX
}

func (sfnt *SFNT) parseCFF() error {
	b, ok := sfnt.Tables["CFF "]
	if !ok {
		return fmt.Errorf("CFF: missing table")
	}

	r := NewBinaryReader(b)
	major := r.ReadUint8()
	minor := r.ReadUint8()
	if major != 1 || minor != 0 {
		return fmt.Errorf("CFF: bad version")
	}
	headerSize := r.ReadUint8()
	if headerSize != 4 {
		return fmt.Errorf("CFF: bad headerSize")
	}
	_ = r.ReadUint8() // offSize

	nameINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: Name INDEX: %w", err)
	}
	if len(nameINDEX.offset) != 2 {
		return fmt.Errorf("CFF: Name INDEX: bad count")
	}

	topINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: Top INDEX: %w", err)
	}
	if len(topINDEX.offset) != len(nameINDEX.offset) {
		return fmt.Errorf("CFF: Top INDEX: bad count")
	}

	stringINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: String INDEX: %w", err)
	}

	globalSubrsINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: Global Subrs INDEX: %w", err)
	}

	topDICT, err := parseTopDICT(topINDEX.Get(0), stringINDEX)
	if err != nil {
		return fmt.Errorf("CFF: Top DICT: %w", err)
	} else if topDICT.CharstringType != 2 {
		return fmt.Errorf("CFF: Type %d Charstring format not supported", topDICT.CharstringType)
	}

	r.Seek(uint32(topDICT.CharStrings))
	charStringsINDEX, err := parseINDEX(r, false)
	if err != nil {
		return fmt.Errorf("CFF: CharStrings INDEX: %w", err)
	}

	if !topDICT.IsCID {
		if len(b) < topDICT.PrivateOffset || len(b)-topDICT.PrivateOffset < topDICT.PrivateLength {
			return fmt.Errorf("CFF: bad Private DICT offset")
		}
		privateDICT, err := parsePrivateDICT(b[topDICT.PrivateOffset:topDICT.PrivateOffset+topDICT.PrivateLength], false)
		if err != nil {
			return fmt.Errorf("CFF: Private DICT: %w", err)
		}

		localSubrsINDEX := &cffINDEX{}
		if privateDICT.Subrs != 0 {
			if len(b)-topDICT.PrivateOffset < privateDICT.Subrs {
				return fmt.Errorf("CFF: bad Local Subrs INDEX offset")
			}
			r.Seek(uint32(topDICT.PrivateOffset + privateDICT.Subrs))
			localSubrsINDEX, err = parseINDEX(r, false)
			if err != nil {
				return fmt.Errorf("CFF: Local Subrs INDEX: %w", err)
			}
		}

		sfnt.CFF = &cffTable{
			version:     1,
			charStrings: charStringsINDEX,
			globalSubrs: globalSubrsINDEX,
			fonts: &cffFontINDEX{
				privateDICT:     []*cffPrivateDICT{privateDICT},
				localSubrsINDEX: []*cffINDEX{localSubrsINDEX},
				first:           []uint32{0, uint32(charStringsINDEX.Len())},
				fd:              []uint16{0},
			},
		}
	} else {
		// CID font
		fonts, err := parseFontINDEX(b, topDICT.FDArray, topDICT.FDSelect, charStringsINDEX.Len(), false)
		if err != nil {
			return fmt.Errorf("CFF: %w", err)
		}

		sfnt.CFF = &cffTable{
			version:     1,
			charStrings: charStringsINDEX,
			globalSubrs: globalSubrsINDEX,
			fonts:       fonts,
		}
	}
	return nil
}

func (sfnt *SFNT) parseCFF2() error {
	return fmt.Errorf("CFF2: not supported")

	b, ok := sfnt.Tables["CFF2"]
	if !ok {
		return fmt.Errorf("CFF2: missing table")
	}

	r := NewBinaryReader(b)
	major := r.ReadUint8()
	minor := r.ReadUint8()
	if major != 2 || minor != 0 {
		return fmt.Errorf("CFF2: bad version")
	}
	headerSize := r.ReadUint8()
	if headerSize != 5 {
		return fmt.Errorf("CFF2: bad headerSize")
	}
	topDictLength := r.ReadUint16()

	topDICT, err := parseTopDICT2(r.ReadBytes(uint32(topDictLength)))
	if err != nil {
		return fmt.Errorf("CFF2: Top DICT: %w", err)
	}

	globalSubrsINDEX, err := parseINDEX(r, true)
	if err != nil {
		return fmt.Errorf("CFF2: Global Subrs INDEX: %w", err)
	}

	r.Seek(uint32(topDICT.CharStrings))
	charStringsINDEX, err := parseINDEX(r, true)
	if err != nil {
		return fmt.Errorf("CFF2: CharStrings INDEX: %w", err)
	}

	fonts, err := parseFontINDEX(b, topDICT.FDArray, topDICT.FDSelect, charStringsINDEX.Len(), true)
	if err != nil {
		return fmt.Errorf("CFF2: %w", err)
	}

	sfnt.CFF = &cffTable{
		version:     2,
		charStrings: charStringsINDEX,
		globalSubrs: globalSubrsINDEX,
		fonts:       fonts,
	}
	return nil
}

func (cff *cffTable) Version() int {
	return cff.version
}

func (cff *cffTable) TopDICT() *cffTopDICT {
	return cff.top
}

func (cff *cffTable) PrivateDICT(glyphID uint16) (*cffPrivateDICT, error) {
	return cff.fonts.GetPrivate(uint32(glyphID))
}

func (cff *cffTable) ToPath(p Pather, glyphID, ppem uint16, x, y int32, f float64, hinting Hinting) error {
	table := "CFF"
	if cff.version == 2 {
		table = "CFF2"
	}
	errBadNumOperands := fmt.Errorf("%v: bad number of operands for operator", table)

	charString := cff.charStrings.Get(glyphID)
	if charString == nil {
		return fmt.Errorf("%v: bad glyphID %v", table, glyphID)
	} else if 65525 < len(charString) {
		return fmt.Errorf("%v: charstring too long", table)
	}
	localSubrs, err := cff.fonts.GetLocalSubrs(uint32(glyphID))
	if err != nil {
		return fmt.Errorf("%v: %w", table, err)
	}

	// raise to most-significant 16 bits and treat less-significant bits as fraction
	x <<= 16
	y <<= 16
	f /= float64(1 << 16) // correct back

	hints := 0
	stack := []int32{} // TODO: may overflow?t
	firstOperator := true
	callStack := []*BinaryReader{}
	r := NewBinaryReader(charString)
	for {
		if cff.version == 2 && r.Len() == 0 && 0 < len(callStack) {
			// end of subroutine
			r = callStack[len(callStack)-1]
			callStack = callStack[:len(callStack)-1]
			continue
		} else if r.Len() == 0 {
			break
		}

		b0 := int32(r.ReadUint8())
		if 32 <= b0 || b0 == 28 {
			var v int32
			if b0 == 28 {
				v = int32(r.ReadInt16()) << 16
			} else if b0 < 32 {
			} else if b0 < 247 {
				v = (b0 - 139) << 16
			} else if b0 < 251 {
				b1 := int32(r.ReadUint8())
				v = ((b0-247)*256 + b1 + 108) << 16
			} else if b0 < 255 {
				b1 := int32(r.ReadUint8())
				v = (-(b0-251)*256 - b1 - 108) << 16
			} else {
				v = r.ReadInt32() // less-siginificant bits is fraction
			}
			if cff.version == 1 && 48 <= len(stack) || cff.version == 2 && 513 <= len(stack) {
				return fmt.Errorf("%v: too many operands for operator", table)
			}
			stack = append(stack, v)
		} else {
			if firstOperator && cff.version == 1 && (b0 == 1 || b0 == 3 || b0 == 4 || b0 == 14 || 18 <= b0 && b0 <= 23) {
				// optionally parse width
				hasWidth := len(stack)%2 == 1
				if b0 == 22 || b0 == 4 {
					hasWidth = !hasWidth
				}
				if hasWidth {
					stack = stack[1:]
				}
			}
			if b0 != 29 && b0 != 10 && b0 != 11 {
				// callgsubr, callsubr, and return don't influece the width operator
				firstOperator = false
			}

			if b0 == 12 {
				b0 = 256 + int32(r.ReadUint8())
			}
			switch b0 {
			case 21:
				// rmoveto
				if len(stack) != 2 {
					return errBadNumOperands
				}
				x += stack[0]
				y += stack[1]
				p.Close()
				p.MoveTo(f*float64(x), f*float64(y))
				stack = stack[:0]
			case 22:
				// hmoveto
				if len(stack) != 1 {
					return errBadNumOperands
				}
				x += stack[0]
				p.Close()
				p.MoveTo(f*float64(x), f*float64(y))
				stack = stack[:0]
			case 4:
				// vmoveto
				if len(stack) != 1 {
					return errBadNumOperands
				}
				y += stack[0]
				p.Close()
				p.MoveTo(f*float64(x), f*float64(y))
				stack = stack[:0]
			case 5:
				// rlineto
				if len(stack) == 0 || len(stack)%2 != 0 {
					return errBadNumOperands
				}
				for i := 0; i < len(stack); i += 2 {
					x += stack[i+0]
					y += stack[i+1]
					p.LineTo(f*float64(x), f*float64(y))
				}
				stack = stack[:0]
			case 6, 7:
				// hlineto and vlineto
				if len(stack) == 0 {
					return errBadNumOperands
				}
				vertical := b0 == 7
				for i := 0; i < len(stack); i++ {
					if !vertical {
						x += stack[i]
					} else {
						y += stack[i]
					}
					p.LineTo(f*float64(x), f*float64(y))
					vertical = !vertical
				}
				stack = stack[:0]
			case 8:
				// rrcurveto
				if len(stack) == 0 || len(stack)%6 != 0 {
					return errBadNumOperands
				}
				for i := 0; i < len(stack); i += 6 {
					x += stack[i+0]
					y += stack[i+1]
					cpx1, cpy1 := x, y
					x += stack[i+2]
					y += stack[i+3]
					cpx2, cpy2 := x, y
					x += stack[i+4]
					y += stack[i+5]
					p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
				}
				stack = stack[:0]
			case 27, 26:
				// hhcurvetp and vvcurveto
				if len(stack) < 4 || len(stack)%4 != 0 && (len(stack)-1)%4 != 0 {
					return errBadNumOperands
				}
				vertical := b0 == 26
				i := 0
				if len(stack)%4 == 1 {
					if !vertical {
						y += stack[0]
					} else {
						x += stack[0]
					}
					i++
				}
				for ; i < len(stack); i += 4 {
					if !vertical {
						x += stack[i+0]
					} else {
						y += stack[i+0]
					}
					cpx1, cpy1 := x, y
					x += stack[i+1]
					y += stack[i+2]
					cpx2, cpy2 := x, y
					if !vertical {
						x += stack[i+3]
					} else {
						y += stack[i+3]
					}
					p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
				}
				stack = stack[:0]
			case 31, 30:
				// hvcurvetp and vhcurveto
				if len(stack) < 4 || len(stack)%4 != 0 && (len(stack)-1)%4 != 0 {
					return errBadNumOperands
				}
				vertical := b0 == 30
				for i := 0; i < len(stack); i += 4 {
					if !vertical {
						x += stack[i+0]
					} else {
						y += stack[i+0]
					}
					cpx1, cpy1 := x, y
					x += stack[i+1]
					y += stack[i+2]
					cpx2, cpy2 := x, y
					if !vertical {
						y += stack[i+3]
					} else {
						x += stack[i+3]
					}
					if i+5 == len(stack) {
						if !vertical {
							x += stack[i+4]
						} else {
							y += stack[i+4]
						}
						i++
					}
					p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
					vertical = !vertical
				}
				stack = stack[:0]
			case 24:
				// rcurveline
				if len(stack) < 2 || (len(stack)-2)%6 != 0 {
					return errBadNumOperands
				}
				i := 0
				for ; i < len(stack)-2; i += 6 {
					x += stack[i+0]
					y += stack[i+1]
					cpx1, cpy1 := x, y
					x += stack[i+2]
					y += stack[i+3]
					cpx2, cpy2 := x, y
					x += stack[i+4]
					y += stack[i+5]
					p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
				}
				x += stack[i+0]
				y += stack[i+1]
				p.LineTo(f*float64(x), f*float64(y))
				stack = stack[:0]
			case 25:
				// rlinecurve
				if len(stack) < 6 || (len(stack)-6)%2 != 0 {
					return errBadNumOperands
				}
				i := 0
				for ; i < len(stack)-6; i += 2 {
					x += stack[i+0]
					y += stack[i+1]
					p.LineTo(f*float64(x), f*float64(y))
				}
				x += stack[i+0]
				y += stack[i+1]
				cpx1, cpy1 := x, y
				x += stack[i+2]
				y += stack[i+3]
				cpx2, cpy2 := x, y
				x += stack[i+4]
				y += stack[i+5]
				p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
				stack = stack[:0]
			case 256 + 35:
				// flex
				if len(stack) != 13 {
					return errBadNumOperands
				}
				// always use cubic Béziers
				for i := 0; i < 12; i += 6 {
					x += stack[i+0]
					y += stack[i+1]
					cpx1, cpy1 := x, y
					x += stack[i+2]
					y += stack[i+3]
					cpx2, cpy2 := x, y
					x += stack[i+4]
					y += stack[i+5]
					p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
				}
				stack = stack[:0]
			case 256 + 34:
				// hflex
				if len(stack) != 7 {
					return errBadNumOperands
				}
				// always use cubic Béziers
				y0 := y
				x += stack[0]
				cpx1, cpy1 := x, y
				x += stack[1]
				y += stack[2]
				cpx2, cpy2 := x, y
				x += stack[3]
				p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))

				x += stack[4]
				cpx1, cpy1 = x, y
				x += stack[5]
				y = y0
				cpx2, cpy2 = x, y
				x += stack[6]
				p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
				stack = stack[:0]
			case 256 + 36:
				// hflex1
				if len(stack) != 9 {
					return errBadNumOperands
				}
				// always use cubic Béziers
				y0 := y
				x += stack[0]
				y += stack[1]
				cpx1, cpy1 := x, y
				x += stack[2]
				y += stack[3]
				cpx2, cpy2 := x, y
				x += stack[4]
				p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))

				x += stack[5]
				cpx1, cpy1 = x, y
				x += stack[6]
				y += stack[7]
				cpx2, cpy2 = x, y
				x += stack[8]
				y = y0
				p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
				stack = stack[:0]
			case 256 + 37:
				// flex1
				if len(stack) != 11 {
					return errBadNumOperands
				}
				// always use cubic Béziers
				x0, y0 := x, y
				x += stack[0]
				y += stack[1]
				cpx1, cpy1 := x, y
				x += stack[2]
				y += stack[3]
				cpx2, cpy2 := x, y
				x += stack[4]
				y += stack[5]
				p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))

				x += stack[6]
				y += stack[7]
				cpx1, cpy1 = x, y
				x += stack[8]
				y += stack[9]
				cpx2, cpy2 = x, y
				dx, dy := x-x0, y-y0
				if dx < 0 {
					dx = -dx
				}
				if dy < 0 {
					dy = -dy
				}
				if dy < dx {
					x += stack[10]
					y = y0
				} else {
					x = x0
					y += stack[10]
				}
				p.CubeTo(f*float64(cpx1), f*float64(cpy1), f*float64(cpx2), f*float64(cpy2), f*float64(x), f*float64(y))
				stack = stack[:0]
			case 14:
				// endchar
				if cff.version == 2 {
					return fmt.Errorf("CFF2: unsupported operator %d", b0)
				} else if len(stack) == 4 {
					return fmt.Errorf("CFF: unsupported endchar operands")
				} else if len(stack) != 0 {
					return errBadNumOperands
				}
				p.Close()
				return nil
			case 1, 3, 18, 23:
				// hstem, vstem, hstemhm, vstemhm
				if len(stack) < 2 || len(stack)%2 != 0 {
					return errBadNumOperands
				}
				// hints are not used
				hints += len(stack) / 2
				if 96 < hints {
					return fmt.Errorf("%v: too many stem hints", table)
				}
				stack = stack[:0]
			case 19, 20:
				// hintmask, cntrmask
				if len(stack)%2 != 0 {
					return errBadNumOperands
				}
				if 0 < len(stack) {
					// vstem
					hints += len(stack) / 2
					if 96 < hints {
						return fmt.Errorf("%v: too many stem hints", table)
					}
					stack = stack[:0]
				}
				r.ReadBytes(uint32((hints + 7) / 8))
			// TODO: arithmetic, storage, and conditional operators for CFF version 1?
			case 10, 29:
				// callsubr and callgsubr
				if 10 < len(callStack) {
					return fmt.Errorf("%v: too many nested subroutines", table)
				} else if len(stack) == 0 {
					return errBadNumOperands
				}

				n := 0
				if b0 == 10 {
					n = len(localSubrs.offset) - 1
				} else {
					n = len(cff.globalSubrs.offset) - 1
				}
				i := stack[len(stack)-1] >> 16
				if n < 1240 {
					i += 107
				} else if n < 33900 {
					i += 1131
				} else {
					i += 32768
				}
				stack = stack[:len(stack)-1]
				if i < 0 || math.MaxUint16 < i {
					return fmt.Errorf("%v: bad subroutine", table)
				}

				var subr []byte
				if b0 == 10 {
					subr = localSubrs.Get(uint16(i))
				} else {
					subr = cff.globalSubrs.Get(uint16(i))
				}
				if subr == nil {
					return fmt.Errorf("%v: bad subroutine", table)
				} else if 65525 < len(charString) {
					return fmt.Errorf("%v: subroutine too long", table)
				}
				callStack = append(callStack, r)
				r = NewBinaryReader(subr)
				firstOperator = true
			case 11:
				// return
				if cff.version == 2 {
					return fmt.Errorf("%v: unsupported operator %d", table, b0)
				} else if len(callStack) == 0 {
					return fmt.Errorf("%v: bad return", table)
				}
				r = callStack[len(callStack)-1]
				callStack = callStack[:len(callStack)-1]
			case 16:
				// blend
				if cff.version == 1 {
					return fmt.Errorf("CFF: unsupported operator %d", b0)
				}
				// TODO: blend
			case 15:
				// vsindex
				if cff.version == 1 {
					return fmt.Errorf("CFF: unsupported operator %d", b0)
				}
				// TODO: vsindex
			default:
				if 256 <= b0 {
					return fmt.Errorf("%v: unsupported operator 12 %d", table, b0-256)
				}
				return fmt.Errorf("%v: unsupported operator %d", table, b0)
			}
		}
	}
	if cff.version == 1 {
		return fmt.Errorf("CFF: charstring must end with endchar operator")
	}
	return nil
}

type cffINDEX struct {
	offset []uint32
	data   []byte
}

func (t *cffINDEX) Len() int {
	return len(t.offset) - 1
}

func (t *cffINDEX) Get(i uint16) []byte {
	if int(i) < t.Len() {
		return t.data[t.offset[i]:t.offset[i+1]]
	}
	return nil
}

func (t *cffINDEX) GetSID(sid int) string {
	// only for String INDEX
	if sid < len(cffStandardStrings) {
		return cffStandardStrings[sid]
	}
	sid -= len(cffStandardStrings)
	if math.MaxUint16 < sid {
		return ""
	}
	if b := t.Get(uint16(sid)); b != nil {
		return string(b)
	}
	return ""
}

func parseINDEX(r *BinaryReader, isCFF2 bool) (*cffINDEX, error) {
	t := &cffINDEX{}
	var count uint32
	if !isCFF2 {
		count = uint32(r.ReadUint16())
	} else {
		count = r.ReadUint32()
	}
	if count == 0 {
		// empty
		return t, nil
	} else if 1e6 < count {
		return nil, fmt.Errorf("too big")
	}

	offSize := r.ReadUint8()
	if offSize == 0 || 4 < offSize {
		return nil, fmt.Errorf("bad offSize")
	}
	if r.Len() < uint32(offSize)*(uint32(count)+1) {
		return nil, fmt.Errorf("bad data")
	}

	t.offset = make([]uint32, count+1)
	if offSize == 1 {
		for i := uint32(0); i < count+1; i++ {
			t.offset[i] = uint32(r.ReadUint8()) - 1
		}
	} else if offSize == 2 {
		for i := uint32(0); i < count+1; i++ {
			t.offset[i] = uint32(r.ReadUint16()) - 1
		}
	} else if offSize == 3 {
		for i := uint32(0); i < count+1; i++ {
			t.offset[i] = uint32(r.ReadUint16())<<8 + uint32(r.ReadUint8()) - 1
		}
	} else {
		for i := uint32(0); i < count+1; i++ {
			t.offset[i] = r.ReadUint32() - 1
		}
	}
	if r.Len() < t.offset[count] {
		return nil, fmt.Errorf("bad data")
	}
	t.data = r.ReadBytes(t.offset[count])
	return t, nil
}

type cffTopDICT struct {
	IsSynthetic bool
	IsCID       bool

	Version            string
	Notice             string
	Copyright          string
	FullName           string
	FamilyName         string
	Weight             string
	IsFixedPitch       bool
	ItalicAngle        float64
	UnderlinePosition  float64
	UnderlineThickness float64
	PaintType          int
	CharstringType     int
	FontMatrix         [6]float64
	UniqueID           int
	FontBBox           [4]float64
	StrokeWidth        float64
	XUID               []int
	Charset            int
	Encoding           int
	CharStrings        int
	PrivateOffset      int
	PrivateLength      int
	SyntheticBase      int
	PostScript         string
	BaseFontName       string
	BaseFontBlend      []int
	ROS1               string
	ROS2               string
	ROS3               int
	CIDFontVersion     int
	CIDFontRevision    int
	CIDFontType        int
	CIDCount           int
	UIDBase            int
	FDArray            int
	FDSelect           int
	FontName           string
	Vstore             int // CFF2
}

type cffFontDICT struct {
	PrivateOffset int
	PrivateLength int
}

type cffPrivateDICT struct {
	BlueValues        []float64
	OtherBlues        []float64
	FamilyBlues       []float64
	FamilyOtherBlues  []float64
	BlueScale         float64
	BlueShift         float64
	BlueFuzz          float64
	StdHW             float64
	StdVW             float64
	StemSnapH         []float64
	StemSnapV         []float64
	ForceBold         bool
	LanguageGroup     int
	ExpansionFactor   float64
	InitialRandomSeed int
	Subrs             int
	DefaultWidthX     float64
	NominalWidthX     float64

	// CFF2
	Vsindex int
	Blend   []float64
}

func parseTopDICT(b []byte, stringINDEX *cffINDEX) (*cffTopDICT, error) {
	dict := &cffTopDICT{
		UnderlinePosition:  -100,
		UnderlineThickness: 50,
		CharstringType:     2,
		FontMatrix:         [6]float64{0.001, 0.0, 0.0, 0.001, 0.0, 0.0},
		CIDCount:           8720,
	}
	return dict, parseDICT(b, false, func(b0 int, is []int, fs []float64) bool {
		switch b0 {
		case 0:
			dict.Version = stringINDEX.GetSID(is[0])
		case 1:
			dict.Notice = stringINDEX.GetSID(is[0])
		case 256 + 0:
			dict.Copyright = stringINDEX.GetSID(is[0])
		case 2:
			dict.FullName = stringINDEX.GetSID(is[0])
		case 3:
			dict.FamilyName = stringINDEX.GetSID(is[0])
		case 4:
			dict.Weight = stringINDEX.GetSID(is[0])
		case 256 + 1:
			dict.IsFixedPitch = is[0] != 0
		case 256 + 2:
			dict.ItalicAngle = fs[0]
		case 256 + 3:
			dict.UnderlinePosition = fs[0]
		case 256 + 4:
			dict.UnderlineThickness = fs[0]
		case 256 + 5:
			dict.PaintType = is[0]
		case 256 + 6:
			dict.CharstringType = is[0]
		case 256 + 7:
			copy(dict.FontMatrix[:], fs)
		case 13:
			dict.UniqueID = is[0]
		case 5:
			copy(dict.FontBBox[:], fs)
		case 256 + 8:
			dict.StrokeWidth = fs[0]
		case 14:
			dict.XUID = is
		case 15:
			dict.Charset = is[0]
		case 16:
			dict.Encoding = is[0]
		case 17:
			dict.CharStrings = is[0]
		case 18:
			dict.PrivateOffset = is[1]
			dict.PrivateLength = is[0]
		case 256 + 20:
			dict.IsSynthetic = true
			dict.SyntheticBase = is[0]
		case 256 + 21:
			dict.PostScript = stringINDEX.GetSID(is[0])
		case 256 + 22:
			dict.BaseFontName = stringINDEX.GetSID(is[0])
		case 256 + 23:
			dict.BaseFontBlend = is
		case 256 + 30:
			// TODO: it is unclear how the ROS operator influences the GIDs/CIDs
			dict.IsCID = true
			dict.ROS1 = stringINDEX.GetSID(is[0])
			dict.ROS2 = stringINDEX.GetSID(is[1])
			dict.ROS3 = is[2]
		case 256 + 31:
			dict.CIDFontVersion = is[0]
		case 256 + 32:
			dict.CIDFontRevision = is[0]
		case 256 + 33:
			dict.CIDFontType = is[0]
		case 256 + 34:
			dict.CIDCount = is[0]
		case 256 + 35:
			dict.UIDBase = is[0]
		case 256 + 36:
			dict.FDArray = is[0]
		case 256 + 37:
			dict.FDSelect = is[0]
		case 256 + 38:
			dict.FontName = stringINDEX.GetSID(is[0])
		default:
			return false
		}
		return true
	})
}

func parseFontDICT(b []byte, isCFF2 bool) (*cffFontDICT, error) {
	dict := &cffFontDICT{}
	return dict, parseDICT(b, isCFF2, func(b0 int, is []int, fs []float64) bool {
		switch b0 {
		case 18:
			dict.PrivateOffset = is[1]
			dict.PrivateLength = is[0]
		case 256 + 7:
			// FontMatrix
		case 256 + 38:
			// FontName
		default:
			return false
		}
		return true
	})
}

func parsePrivateDICT(b []byte, isCFF2 bool) (*cffPrivateDICT, error) {
	dict := &cffPrivateDICT{
		BlueScale:       0.039625,
		BlueShift:       7.0,
		BlueFuzz:        1.0,
		ExpansionFactor: 0.06,
	}

	return dict, parseDICT(b, isCFF2, func(b0 int, is []int, fs []float64) bool {
		switch b0 {
		case 6:
			dict.BlueValues = fs
		case 7:
			dict.OtherBlues = fs
		case 8:
			dict.FamilyBlues = fs
		case 9:
			dict.FamilyOtherBlues = fs
		case 256 + 9:
			dict.BlueScale = fs[0]
		case 256 + 10:
			dict.BlueShift = fs[0]
		case 256 + 11:
			dict.BlueFuzz = fs[0]
		case 10:
			dict.StdHW = fs[0]
		case 11:
			dict.StdVW = fs[0]
		case 256 + 12:
			dict.StemSnapH = fs
		case 256 + 13:
			dict.StemSnapV = fs
		case 256 + 14:
			dict.ForceBold = is[0] != 0
		case 256 + 17:
			dict.LanguageGroup = is[0]
		case 256 + 18:
			dict.ExpansionFactor = fs[0]
		case 256 + 19:
			dict.InitialRandomSeed = is[0]
		case 19:
			dict.Subrs = is[0]
		case 20:
			dict.DefaultWidthX = fs[0]
		case 21:
			dict.NominalWidthX = fs[0]
		case 22:
			dict.Vsindex = is[0]
		case 23:
			dict.Blend = fs
		default:
			return false
		}
		return true
	})
}

func parseTopDICT2(b []byte) (*cffTopDICT, error) {
	dict := &cffTopDICT{
		FontMatrix: [6]float64{0.001, 0.0, 0.0, 0.001, 0.0, 0.0},
	}
	return dict, parseDICT(b, true, func(b0 int, is []int, fs []float64) bool {
		switch b0 {
		case 256 + 7:
			copy(dict.FontMatrix[:], fs)
		case 17:
			dict.CharStrings = is[0]
		case 256 + 36:
			dict.FDArray = is[0]
		case 256 + 37:
			dict.FDSelect = is[0]
		case 24:
			dict.Vstore = is[0]
		default:
			return false
		}
		return true
	})
}

func parseDICT(b []byte, isCFF2 bool, callback func(b0 int, is []int, fs []float64) bool) error {
	opSize := map[int]int{
		256 + 7:  6,
		5:        4,
		14:       -1,
		18:       2,
		256 + 23: -1,
		256 + 30: 3,
		6:        -1,
		7:        -1,
		8:        -1,
		9:        -1,
		256 + 12: -1,
		256 + 13: -1,
	}

	r := NewBinaryReader(b)
	ints := []int{}
	reals := []float64{}
	for 0 < r.Len() {
		b0 := int(r.ReadUint8())
		if b0 < 22 {
			// operator
			if b0 == 12 {
				b0 = 256 + int(r.ReadUint8())
			}

			size := 1
			if s, ok := opSize[b0]; ok {
				if s == -1 {
					size = len(ints)
				} else {
					size = s
				}
			}
			if len(ints) < size {
				return fmt.Errorf("too few operands for operator")
			}

			is := ints[len(ints)-size:]
			fs := reals[len(reals)-size:]
			ints = ints[:len(ints)-size]
			reals = reals[:len(reals)-size]

			if ok := callback(b0, is, fs); !ok {
				return fmt.Errorf("bad operator")
			}
		} else if 22 <= b0 && b0 < 28 || b0 == 31 || b0 == 255 {
			// reserved
		} else {
			if !isCFF2 && 48 <= len(ints) || isCFF2 && 513 <= len(ints) {
				return fmt.Errorf("too many operands for operator")
			}
			i, f := parseDICTNumber(b0, r)
			if math.IsNaN(f) {
				f = float64(i)
			} else {
				i = int(f + 0.5)
			}
			ints = append(ints, i)
			reals = append(reals, f)
		}
	}
	return nil
}

func parseDICTNumber(b0 int, r *BinaryReader) (int, float64) {
	if b0 == 28 {
		return int(r.ReadInt16()), math.NaN()
	} else if b0 == 29 {
		return int(r.ReadInt32()), math.NaN()
	} else if b0 == 30 {
		num := []byte{}
		for {
			b := r.ReadUint8()
			for i := 0; i < 2; i++ {
				switch b >> 4 {
				case 0x0A:
					num = append(num, '.')
				case 0x0B:
					num = append(num, 'E')
				case 0x0C:
					num = append(num, 'E', '-')
				case 0x0D:
					// reserved
				case 0x0E:
					num = append(num, '-')
				case 0x0F:
					f, err := strconv.ParseFloat(string(num), 32)
					if err != nil {
						return 0, math.NaN()
					}
					return 0, f
				default:
					num = append(num, '0'+byte(b>>4))
				}
				b = b << 4
			}
		}
	} else if b0 < 247 {
		return b0 - 139, math.NaN()
	} else if b0 < 251 {
		b1 := int(r.ReadUint8())
		return (b0-247)*256 + b1 + 108, math.NaN()
	} else {
		b1 := int(r.ReadUint8())
		return -(b0-251)*256 - b1 - 108, math.NaN()
	}
}

type cffFontINDEX struct {
	privateDICT     []*cffPrivateDICT
	localSubrsINDEX []*cffINDEX

	fds   []uint8 // fds or the other two are used
	first []uint32
	fd    []uint16
}

func (t *cffFontINDEX) Index(glyphID uint32) (uint16, bool) {
	if t.fds != nil {
		if len(t.fds) <= int(glyphID) {
			return 0, false
		}
		return uint16(t.fds[glyphID]), true
	} else if t.first[len(t.first)-1] <= glyphID {
		return 0, false
	}

	i := 0
	for t.first[i+1] <= glyphID {
		i++
	}
	return t.fd[i], true
}

func (t *cffFontINDEX) GetPrivate(glyphID uint32) (*cffPrivateDICT, error) {
	i, ok := t.Index(glyphID)
	if !ok {
		return nil, fmt.Errorf("bad glyph ID %v", glyphID)
	}
	return t.privateDICT[i], nil
}

func (t *cffFontINDEX) GetLocalSubrs(glyphID uint32) (*cffINDEX, error) {
	i, ok := t.Index(glyphID)
	if !ok {
		return nil, fmt.Errorf("bad glyph ID %v", glyphID)
	}
	return t.localSubrsINDEX[i], nil
}

func parseFontINDEX(b []byte, fdArray, fdSelect, nGlyphs int, isCFF2 bool) (*cffFontINDEX, error) {
	if len(b) < fdArray {
		return nil, fmt.Errorf("bad Font INDEX offset")
	}

	r := NewBinaryReader(b)
	r.Seek(uint32(fdArray))
	fontINDEX, err := parseINDEX(r, false)
	if err != nil {
		return nil, fmt.Errorf("Font INDEX: %w", err)
	}

	fonts := &cffFontINDEX{}
	fonts.privateDICT = make([]*cffPrivateDICT, fontINDEX.Len())
	fonts.localSubrsINDEX = make([]*cffINDEX, fontINDEX.Len())
	for i := 0; i < fontINDEX.Len(); i++ {
		fontDICT, err := parseFontDICT(fontINDEX.Get(uint16(i)), isCFF2)
		if err != nil {
			return nil, fmt.Errorf("Font DICT: %w", err)
		}
		if len(b) < fontDICT.PrivateOffset || len(b)-fontDICT.PrivateOffset < fontDICT.PrivateLength {
			return nil, fmt.Errorf("Font DICT: bad Private DICT offset")
		}
		privateDICT, err := parsePrivateDICT(b[fontDICT.PrivateOffset:fontDICT.PrivateOffset+fontDICT.PrivateLength], isCFF2)
		if err != nil {
			return nil, fmt.Errorf("Private DICT: %w", err)
		}
		fonts.privateDICT[i] = privateDICT

		if privateDICT.Subrs != 0 {
			if len(b)-fontDICT.PrivateOffset < privateDICT.Subrs {
				return nil, fmt.Errorf("bad Local Subrs INDEX offset")
			}
			r.Seek(uint32(fontDICT.PrivateOffset + privateDICT.Subrs))
			fonts.localSubrsINDEX[i], err = parseINDEX(r, isCFF2)
			if err != nil {
				return nil, fmt.Errorf("Local Subrs INDEX: %w", err)
			}
		} else if isCFF2 {
			return nil, fmt.Errorf("Private DICT must have Local Subrs INDEX offset")
		}
	}

	r.Seek(uint32(fdSelect))
	format := r.ReadUint8()
	if format == 0 {
		fonts.fds = make([]uint8, nGlyphs)
		for i := 0; i < nGlyphs; i++ {
			fonts.fds[i] = r.ReadUint8()
		}
	} else if format == 3 {
		nRanges := r.ReadUint16()
		fonts.first = make([]uint32, nRanges+1)
		fonts.fd = make([]uint16, nRanges)
		for i := 0; i < int(nRanges); i++ {
			fonts.first[i] = uint32(r.ReadUint16())
			fonts.fd[i] = uint16(r.ReadUint8())
		}
		fonts.first[nRanges] = uint32(r.ReadUint16())
	} else if isCFF2 && format == 4 {
		nRanges := r.ReadUint32()
		fonts.first = make([]uint32, nRanges+1)
		fonts.fd = make([]uint16, nRanges)
		for i := 0; i < int(nRanges); i++ {
			fonts.first[i] = r.ReadUint32()
			fonts.fd[i] = r.ReadUint16()
		}
		fonts.first[nRanges] = r.ReadUint32()
	} else {
		return nil, fmt.Errorf("FDSelect: bad format")
	}
	return fonts, nil
}
