package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/tdewolff/argp"
)

type Extract struct {
	Info     bool   `desc:"Get document information"`
	Page     int    `short:"p" default:"0" desc:"Page"`
	Password string `default:"" desc:"Password"`
	Input    string `index:"0" desc:"Input file"`
}

type Replace struct {
	Page     int    `short:"p" default:"0" desc:"Page"`
	XObj     string `desc:"XObject"`
	Index    int    `short:"i" default:"-1" desc:"String index to replace"`
	String   string `short:"s" desc:"Text replacement"`
	XOffset  int    `desc:"Text X-offset in font units"`
	Spacing  string `default:"none" desc:"Character spacing type, 'none' for regular spacing, a number for character spacing"`
	Password string `default:"" desc:"Password"`
	Output   string `short:"o" desc:"Output file"`
	Input    string `index:"0" desc:"Input file"`
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
	for i, object := range objects {
		if i == 0 {
			fmt.Printf("Page %d:\n", cmd.Page)
		} else {
			fmt.Printf("\nXObject %s:\n", names[i])
		}
		err = walkStrings(pdf, object, func(_, _, index int, state textState, op string, vals []interface{}) (int, error) {
			var s string
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
			fmt.Printf("i=%4d font=%v: %s\n", index, state.fontName, s)
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
	} else if cmd.Output == "" {
		fmt.Println("ERROR: must specify output filename")
		return argp.ShowUsage
	} else if cmd.Index == -1 {
		fmt.Println("ERROR: must specify string index to replace")
		return argp.ShowUsage
	}

	fr, err := os.Open(cmd.Input)
	if err != nil {
		return err
	}

	fw, err := os.Create(cmd.Output)
	if err != nil {
		return err
	}

	pdf, err := NewPDFReader(fr, cmd.Password)
	if err != nil {
		return err
	}

	var object interface{}
	names, objects := getObjects(pdf, cmd.Page)
	for i, name := range names {
		if name == cmd.XObj {
			object = objects[i]
			break
		}
	}
	if object == nil {
		return fmt.Errorf("ERROR: unknown object: %s", cmd.XObj)
	}

	var ref pdfRef
	stream, err := pdf.GetStream(object)
	if err == nil {
		ref = object.(pdfRef)
	} else {
		page, _ := pdf.GetDict(object)
		fmt.Println(page)
		ref = page["Contents"].(pdfRef)
		stream, _ = pdf.GetStream(page["Contents"])
	}

	err = walkStrings(pdf, object, func(start, end, index int, state textState, op string, vals []interface{}) (int, error) {
		if index == cmd.Index {
			s := state.fonts[state.fontName].FromUnicode(cmd.String)

			b := bytes.Buffer{}
			if op == "'" {
				b.WriteString("T*")
			} else if op == "\"" && len(vals) == 3 {
				pdfWriteVal(&b, nil, pdfRef{}, vals[0])
				b.WriteString(" Tw ")
				pdfWriteVal(&b, nil, pdfRef{}, vals[1])
				b.WriteString(" Tc T*")
			}

			array := pdfArray{}
			if cmd.XOffset != 0 {
				array = append(array, -cmd.XOffset)
			}
			if space, err := strconv.ParseInt(cmd.Spacing, 10, 64); err == nil {
				di := state.fonts[state.fontName].Bytes()
				for i := 0; i < len(s)-di; i += di {
					array = append(array, s[i:i+di], -space)
				}
				array = append(array, s[len(s)-di:])
			} else {
				array = append(array, string(s))
			}
			pdfWriteVal(&b, nil, pdfRef{}, array)
			b.WriteString(" TJ")

			fmt.Println("Old:", string(stream.data[start:end]))
			fmt.Println("New:", b.String())
			n := b.Len() - (end - start)
			stream.data = append(stream.data[:start], append(b.Bytes(), stream.data[end:]...)...)
			return n, io.EOF
		}
		return 0, nil
	})
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
}

func getObjects(pdf *pdfReader, page int) ([]string, []interface{}) {
	dict, _, err := pdf.GetPage(page)
	if err != nil {
		return []string{}, []interface{}{}
	}

	names := []string{""}
	objects := []interface{}{
		dict,
	}
	var addDict func(string, pdfDict)
	addDict = func(prefix string, dict pdfDict) {
		resources, _ := pdf.GetDict(dict["Resources"])
		xobjects, _ := pdf.GetDict(resources["XObject"])
		xnames := []string{}
		for name, _ := range xobjects {
			xnames = append(xnames, name)
		}
		sort.Strings(xnames)
		if prefix != "" {
			prefix += "/"
		}
		for i, xname := range xnames {
			name := fmt.Sprintf("%s%d", prefix, i+1)
			xobject, err := pdf.GetStream(xobjects[xname])
			if _, ok := xobjects[xname].(pdfRef); ok && err == nil {
				if subtype, ok := xobject.dict["Subtype"].(pdfName); ok && (subtype == pdfName("Form") || subtype == pdfName("PS")) {

					names = append(names, name)
					objects = append(objects, xobjects[xname])
				}
			}
			addDict(name, xobject.dict)
		}
	}
	addDict("", dict)
	return names, objects
}

func walkStrings(pdf *pdfReader, obj interface{}, cb func(int, int, int, textState, string, []interface{}) (int, error)) error {
	state := textState{
		fonts: map[pdfName]pdfFont{},
	}

	var dict pdfDict
	var data []byte
	if stream, err := pdf.GetStream(obj); err == nil {
		dict = stream.dict
		data = stream.data
	} else if page, err := pdf.GetDict(obj); err == nil {
		stream, err := pdf.GetStream(page["Contents"])
		if err != nil {
			return err
		}
		dict = page
		data = stream.data
	} else {
		return fmt.Errorf("object is not a stream or page dictionary")
	}

	i := 0
	stream := newPDFStreamReader(data)
	for {
		start := moveWhiteSpace(data, stream.Pos())
		op, vals, err := stream.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
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
		} else if op == "Tj" || op == "TJ" || op == "'" || op == "\"" {
			d, err := cb(start, stream.Pos(), i, state, op, vals)
			if err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
			i += d + 1
		} else {
			//fmt.Println("unknown operator:", op)
		}
	}
	return nil
}
