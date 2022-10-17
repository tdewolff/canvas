package main

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

type pdfRef [2]uint32
type pdfName string
type pdfArray []interface{}
type pdfDict map[string]interface{}
type pdfStream struct {
	dict    pdfDict
	filters []pdfName
	params  []pdfDict
	data    []byte
}

func (v pdfRef) String() string {
	return fmt.Sprintf("%d %d R", v[0], v[1])
}

const (
	pdfFilterASCII85 pdfName = "ASCII85Decode"
	pdfFilterFlate   pdfName = "FlateDecode"
)

func (v pdfStream) SetFilters(filters []pdfName) {
	v.filters = filters
}

func (v pdfStream) Decompress() (pdfStream, error) {
	if len(v.filters) == 0 {
		delete(v.dict, "Filter")
		return v, nil
	}

	b := v.data
	for i := len(v.filters) - 1; 0 <= i; i-- {
		filter := v.filters[i]
		switch filter {
		case pdfFilterASCII85:
			var err error
			b = b[:len(b)-2] // remove ~> characters
			r := ascii85.NewDecoder(bytes.NewReader(b))
			b, err = io.ReadAll(r)
			if err != nil {
				return pdfStream{}, err
			}
		case pdfFilterFlate:
			r, err := zlib.NewReader(bytes.NewReader(b))
			if err != nil {
				return pdfStream{}, err
			}
			b, err = io.ReadAll(r)
			if err != nil {
				return pdfStream{}, err
			}
			r.Close()

			predictor, _ := v.params[i]["Predictor"].(int)
			if predictor == 0 || predictor == 1 {
				// no-op
			} else if predictor == 2 {
				// TIFF predictor
				return pdfStream{}, fmt.Errorf("unsupported flate predictor: %v", predictor)
			} else if 10 <= predictor && predictor <= 15 {
				// PNG prediction
				columns, _ := v.params[i]["Columns"].(int)
				if columns == 0 {
					columns = 1
				} else if len(b)%(columns+1) != 0 {
					return pdfStream{}, fmt.Errorf("bad flate predictor columns")
				}

				colors, _ := v.params[i]["Colors"].(int)
				if colors == 0 {
					colors = 1
				} else if colors < 1 {
					return pdfStream{}, fmt.Errorf("bad flate predictor colors")
				}

				bpc, _ := v.params[i]["BitsPerComponent"].(int)
				if bpc == 0 {
					bpc = 8
				} else if bpc != 1 && bpc != 2 && bpc != 4 && bpc != 8 && bpc != 16 {
					return pdfStream{}, fmt.Errorf("bad flate predictor bits per component")
				}

				bpp := int((colors*bpc + 7) / 8) // round up to whole bytes
				if columns < bpp {
					return pdfStream{}, fmt.Errorf("bad flate predictor bits per pixel")
				}

				n := len(b) / (columns + 1)
				for j := 0; j < n; j++ {
					pos := j * (columns + 1)                   // src
					start, end := j*columns, j*columns+columns // dst

					filter := b[pos]
					copy(b[start:], b[pos+1:pos+1+columns])
					if filter == 0 {
						// None
					} else if filter == 1 {
						// Sub
						for k := start + bpp; k < end; k++ {
							b[k] += b[k-bpp]
						}
					} else if filter == 2 {
						// Up
						if j != 0 {
							for k := start; k < end; k++ {
								b[k] += b[k-columns]
							}
						}
					} else if filter == 3 {
						// Average
						for k := start; k < end; k++ {
							A, B := 0, 0
							if start+bpp <= k {
								A = int(b[k-bpp])
							}
							if j != 0 {
								B = int(b[k-columns])
							}
							b[k] += byte(math.Floor(float64(A+B) / 2.0))
						}
					} else if filter == 4 {
						// Paeth
						for k := start; k < end; k++ {
							A, B, C := 0, 0, 0 // left, above, above-left
							if start+bpp <= k {
								A = int(b[k-bpp])
							}
							if j != 0 {
								B = int(b[k-columns])
								if start-columns+bpp <= k {
									C = int(b[k-columns-bpp])
								}
							}
							// Paeth predictor function
							p := A + B - C
							pa, pb, pc := p-A, p-B, p-C
							if pa < 0 {
								pa = -pa
							}
							if pb < 0 {
								pb = -pb
							}
							if pc < 0 {
								pc = -pc
							}
							if pa <= pb && pa <= pc {
								p = A
							} else if pb <= pc {
								p = B
							} else {
								p = C
							}
							b[k] += byte(p)
						}
					} else {
						return pdfStream{}, fmt.Errorf("bad flate PNG predictor filter: %v", filter)
					}
				}
				b = b[:n*columns]
			} else {
				return pdfStream{}, fmt.Errorf("unsupported flate predictor: %v", predictor)
			}
		default:
			return pdfStream{}, fmt.Errorf("unsupported filter: %v", filter)
		}
	}
	delete(v.dict, "Filter")
	delete(v.dict, "DecodeParms")
	v.dict["Length"] = len(b)
	v.data = b
	return v, nil
}

func (v pdfStream) Compress() (pdfStream, error) {
	if len(v.filters) == 0 {
		return v, nil
	}

	b := v.data
	filterArray := make(pdfArray, len(v.filters))
	for i, filter := range v.filters {
		var b2 bytes.Buffer
		switch filter {
		case pdfFilterASCII85:
			w := ascii85.NewEncoder(&b2)
			w.Write(b)
			w.Write([]byte("~>"))
			w.Close()
		case pdfFilterFlate:
			w := zlib.NewWriter(&b2)
			w.Write(b)
			w.Close()
		default:
			return pdfStream{}, fmt.Errorf("unsupported filter: %v", filter)
		}
		b = b2.Bytes()
		filterArray[len(filterArray)-i-1] = filter
	}
	v.dict["Filter"] = filterArray
	v.dict["Length"] = len(b)
	v.data = b
	return v, nil
}

type pdfInfo struct {
	Title        string
	Author       string
	Subject      string
	Keywords     string
	Creator      string
	Producer     string
	CreationDate time.Time
	ModDate      time.Time
}

var dateFormat = "2006-01-02 15:04:05 UTC-0700"

func (d pdfInfo) String() string {
	sb := strings.Builder{}
	if d.Title != "" {
		sb.WriteString("Title: " + d.Title + "\n")
	}
	if d.Author != "" {
		sb.WriteString("Author: " + d.Author + "\n")
	}
	if d.Subject != "" {
		sb.WriteString("Subject: " + d.Subject + "\n")
	}
	if d.Keywords != "" {
		sb.WriteString("Keywords: " + d.Keywords + "\n")
	}
	if d.Creator != "" {
		sb.WriteString("Creator: " + d.Creator + "\n")
	}
	if d.Producer != "" {
		sb.WriteString("Producer: " + d.Producer + "\n")
	}
	if !d.CreationDate.IsZero() {
		sb.WriteString("CreationDate: " + d.CreationDate.Format(dateFormat) + "\n")
	}
	if !d.ModDate.IsZero() {
		sb.WriteString("ModDate: " + d.ModDate.Format(dateFormat) + "\n")
	}
	s := sb.String()
	if 0 < len(s) {
		s = s[:len(s)-1]
	}
	return s
}
