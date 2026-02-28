package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tdewolff/argp"
)

type Extract struct {
	Password string `default:"" desc:"PDF password"`
	Page     int    `short:"p" default:"0" desc:"Page"`
	Info     bool   `desc:"Get document information"`
	Input    string `index:"0" desc:"Input file"`
}

type Replace struct {
	Password  string  `default:"" desc:"PDF password"`
	Page      int     `short:"p" default:"0" desc:"Select page"`
	XObj      string  `desc:"Select XObject if necessary"`
	Info      string  `desc:"Update info instead of string, either Producer or CreationDate"`
	Index     int     `short:"i" default:"-1" desc:"String index to replace"`
	String    string  `short:"s" desc:"Text replacement"`
	X         float64 `short:"x" long:"" desc:"Horizontal position offset in PDF units"`
	Y         float64 `short:"y" long:"" desc:"Vertical position offset in PDF units"`
	Offset    int     `desc:"Text X-offset in font units"`
	Alignment string  `short:"a" default:"L" desc:"Text alignment: L, C, R"`
	Spacing   string  `default:"none" desc:"Character spacing type, 'none' for regular spacing, a number for character spacing"`
	Copy      bool    `desc:"Copy text element"`
	Output    string  `short:"o" desc:"Output file"`
	Input     string  `index:"0" desc:"Input file"`
}

func main() {
	root := argp.NewCmd(&Extract{}, "PDF text extraction and replacement toolkit by Taco de Wolff")
	root.AddCmd(&Replace{}, "replace", "Replace text")
	root.Parse()
	root.PrintHelp()
}

func (cmd *Extract) Run() error {
	if cmd.Input == "" {
		return argp.ShowUsage
	}

	f, err := os.Open(cmd.Input)
	if err != nil {
		return err
	}

	pdf, err := NewPDFReader(f, cmd.Password)
	if err != nil {
		return err
	}

	if cmd.Info {
		fmt.Println("File name:", filepath.Base(cmd.Input))
		fmt.Println("Pages:", len(pdf.kids))
		if _, ok := pdf.trailer["Encrypt"]; ok {
			fmt.Println("Encrypted: yes")
		} else {
			fmt.Println("Encrypted: no")
		}
		fmt.Println(pdf.GetInfo())
		return nil
	}

	names, objects := getObjects(pdf, cmd.Page)
	for i, obj := range objects {
		if i == 0 {
			fmt.Printf("Page %d:\n", cmd.Page)
		} else {
			fmt.Printf("\nXObject %s:\n", names[i])
		}
		err = walkStrings(pdf, obj, func(index int, ops []textOperator, state textState) (int, error) {
			var s string
			op, vals := ops[0].Op, ops[0].Vals
			if ops[0].Op == "Td" {
				op, vals = ops[1].Op, ops[1].Vals
			}
			if op == "TJ" && len(vals) == 1 {
				if array, ok := vals[0].(pdfArray); ok {
					for _, item := range array {
						if val, ok := item.([]byte); ok {
							s += state.fonts[state.fontName].ToUnicode(val)
						}
					}
				}
			} else if (op == "Tj" || op == "'") && len(vals) == 1 {
				if str, ok := vals[0].([]byte); ok {
					s = state.fonts[state.fontName].ToUnicode(str)
				}
			} else if op == "\"" && len(vals) == 3 {
				if str, ok := vals[2].([]byte); ok {
					s = state.fonts[state.fontName].ToUnicode(str)
				}
			}
			//if names[i] != "" {
			//	fmt.Printf("xobj=%s ", names[i])
			//}
			fmt.Printf("i=%4d  %v/%vpt: %s\n", index, state.fontName, state.fontSize, s)
			return 0, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (cmd *Replace) Run() error {
	if cmd.Input == "" {
		return argp.ShowUsage
	} else if cmd.Index == -1 && cmd.Info == "" {
		fmt.Println("ERROR: must specify string index to replace")
		return argp.ShowUsage
	} else if cmd.Output == "" {
		cmd.Output = cmd.Input
	}

	alignment := 'L'
	switch strings.ToLower(cmd.Alignment) {
	case "", "l", "left":
		alignment = 'L'
	case "c", "center", "centre":
		alignment = 'C'
	case "r", "right":
		alignment = 'R'
	default:
		fmt.Println("ERROR: alignment must be L, C, or R")
		return argp.ShowUsage
	}

	fr, err := os.Open(cmd.Input)
	if err != nil {
		return err
	}

	pdf, err := NewPDFReader(fr, cmd.Password)
	if err != nil {
		fr.Close()
		return err
	}

	var obj any
	names, objects := getObjects(pdf, cmd.Page)
	for i, name := range names {
		if name == cmd.XObj {
			obj = objects[i]
			break
		}
	}
	if obj == nil {
		fr.Close()
		return fmt.Errorf("ERROR: unknown object: %s", cmd.XObj)
	}

	ref, _, stream, err := getContents(pdf, obj)
	if err != nil {
		fr.Close()
		return fmt.Errorf("ERROR: %s", err)
	}

	err = walkStrings(pdf, obj, func(index int, ops []textOperator, state textState) (int, error) {
		if index == cmd.Index {
			start, end := ops[0].Start, ops[len(ops)-1].End
			s := state.fonts[state.fontName].FromUnicode(cmd.String)

			var x0, y0 float64
			if ops[0].Op == "Td" && len(ops[0].Vals) == 2 {
				x0 = parseFloat(ops[0].Vals[0])
				y0 = parseFloat(ops[0].Vals[1])
				ops = ops[1:]
			}

			var x1, y1 float64
			if 1 < len(ops) && ops[1].Op == "Td" && len(ops[1].Vals) == 2 {
				x1 = parseFloat(ops[1].Vals[0])
				y1 = parseFloat(ops[1].Vals[1])
				ops = ops[:1]
			}

			op, vals := ops[0].Op, ops[0].Vals

			b := bytes.Buffer{}
			if cmd.Copy {
				start = end
				b.WriteString(" ")
			}
			if op == "'" {
				b.WriteString("T*")
			} else if op == "\"" && len(vals) == 3 {
				pdfWriteVal(&b, nil, pdfRef{}, vals[0])
				b.WriteString(" Tw ")
				pdfWriteVal(&b, nil, pdfRef{}, vals[1])
				b.WriteString(" Tc T*")
			}

			x0 += cmd.X
			y0 += cmd.Y
			x1 -= cmd.X
			y1 -= cmd.Y
			if x0 != 0.0 || y0 != 0.0 {
				pdfWriteVal(&b, nil, pdfRef{}, x0)
				b.WriteString(" ")
				pdfWriteVal(&b, nil, pdfRef{}, y0)
				b.WriteString(" Td ")
			}

			offset := cmd.Offset
			if alignment == 'C' || alignment == 'R' {
				//var width int
				fmt.Println(vals)
			}

			array := pdfArray{}
			if offset != 0 {
				array = append(array, -offset)
			}
			if space, err := strconv.ParseInt(cmd.Spacing, 10, 64); err == nil && space != 0 {
				di := state.fonts[state.fontName].Bytes()
				for i := 0; i < len(s)-di; i += di {
					array = append(array, s[i:i+di], -space)
				}
				array = append(array, s[len(s)-di:])
			} else {
				array = append(array, string(s))
			}
			pdfWriteVal(&b, nil, pdfRef{}, array)
			b.WriteString("TJ")

			if x1 != 0.0 || y1 != 0.0 {
				b.WriteString(" ")
				pdfWriteVal(&b, nil, pdfRef{}, x1)
				b.WriteString(" ")
				pdfWriteVal(&b, nil, pdfRef{}, y1)
				b.WriteString(" Td")
			}

			fmt.Println("Old:", printable(string(stream.data[start:end])))
			fmt.Println("New:", printable(b.String()))

			n := b.Len() - (end - start)
			stream.data = append(stream.data[:start], append(b.Bytes(), stream.data[end:]...)...)
			return n, io.EOF
		}
		return 0, nil
	})
	if err != nil {
		fr.Close()
		return err
	}
	fr.Close()

	fw, err := os.Create(cmd.Output)
	if err != nil {
		return err
	}
	pdfWriter := NewPDFWriter(fw, pdf)
	pdfWriter.SetObject(ref, stream)
	return pdfWriter.Close()
}

type textState struct {
	fonts    map[pdfName]pdfFont
	fontName pdfName
	fontSize float64
}

func getObjects(pdf *pdfReader, page int) ([]string, []any) {
	dict, _, err := pdf.GetPage(page)
	if err != nil {
		return []string{}, []any{}
	}

	names := []string{""}
	objects := []any{
		dict,
	}
	//var addDict func(string, pdfDict)
	//addDict = func(prefix string, dict pdfDict) {
	//	resources, _ := pdf.GetDict(dict["Resources"])
	//	xobjects, _ := pdf.GetDict(resources["XObject"])
	//	xnames := []string{}
	//	for name := range xobjects {
	//		xnames = append(xnames, name)
	//	}
	//	sort.Strings(xnames)
	//	if prefix != "" {
	//		prefix += "/"
	//	}
	//	for i, xname := range xnames {
	//		name := fmt.Sprintf("%s%d", prefix, i+1)
	//		xobject, err := pdf.GetStream(xobjects[xname])
	//		if _, ok := xobjects[xname].(pdfRef); ok && err == nil {
	//			if subtype, ok := xobject.dict["Subtype"].(pdfName); ok && (subtype == pdfName("Form") || subtype == pdfName("PS")) {

	//				names = append(names, name)
	//				objects = append(objects, xobjects[xname])
	//			}
	//		}
	//		addDict(name, xobject.dict)
	//	}
	//}
	//addDict("", dict)
	return names, objects
}

func getContents(pdf *pdfReader, obj any) (pdfRef, pdfDict, pdfStream, error) {
	if _, ok := obj.(pdfRef); !ok {
		if page, err := pdf.GetDict(obj); err == nil {
			if contents, ok := page["Contents"].(pdfArray); ok {
				if len(contents) != 1 {
					return pdfRef{}, pdfDict{}, pdfStream{}, fmt.Errorf("Contents must be a reference or an array of one element")
				}
				obj = contents[0]
			} else {
				obj = page["Contents"]
			}
			stream, err := pdf.GetStream(obj)
			return obj.(pdfRef), page, stream, err
		} else {
			return pdfRef{}, pdfDict{}, pdfStream{}, fmt.Errorf("object is not a stream or page dictionary")
		}
		if _, ok := obj.(pdfRef); !ok {
			return pdfRef{}, pdfDict{}, pdfStream{}, fmt.Errorf("object is not a stream or page dictionary with a reference")
		}
	}
	stream, err := pdf.GetStream(obj)
	return obj.(pdfRef), stream.dict, stream, err
}

type textOperator struct {
	Start, End int // position in stream
	Op         string
	Vals       []any
}

func parseFloat(v any) float64 {
	if val, ok := v.(float64); ok {
		return val
	} else if val, ok := v.(int); ok {
		return float64(val)
	}
	return math.NaN()
}

func walkStrings(pdf *pdfReader, obj any, cb func(int, []textOperator, textState) (int, error)) error {
	state := textState{
		fonts: map[pdfName]pdfFont{},
	}

	var dict pdfDict
	var data []byte
	if _, page, stream, err := getContents(pdf, obj); err == nil {
		dict = page
		data = stream.data
	} else {
		return err
	}

	i := 0
	hasText := false
	ops := []textOperator{}
	stream := newPDFStreamReader(data)
	for {
		start := moveWhiteSpace(data, stream.Pos())
		op, vals, err := stream.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if op == "Td" {
			ops = append(ops, textOperator{
				Start: start,
				End:   stream.Pos(),
				Op:    op,
				Vals:  vals,
			})
		}

		if hasText {
			d, err := cb(i, ops, state)
			if err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
			i += d + 1

			ops = ops[:0]
			hasText = false
		}

		if op == "Tf" && len(vals) == 2 {
			if name, ok := vals[0].(pdfName); ok {
				if _, ok := state.fonts[name]; !ok {
					state.fonts[name], err = pdf.GetFont(dict, name)
					if err != nil {
						return err
					}
				}
				state.fontName = name
			}
			state.fontSize = parseFloat(vals[1])
		} else if op == "Tj" || op == "TJ" || op == "'" || op == "\"" {
			ops = append(ops, textOperator{
				Start: start,
				End:   stream.Pos(),
				Op:    op,
				Vals:  vals,
			})
			hasText = true
		} else {
			//fmt.Println("unknown operator:", op)
		}
	}
	if hasText {
		d, err := cb(i, ops, state)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		i += d + 1
	}
	return nil
}
