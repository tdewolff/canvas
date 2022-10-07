package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	Index    int    `short:"i" default:"-1" desc:"String index to replace"`
	String   string `short:"s" desc:"Text replacement"`
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

	_, data, err := pdf.GetPage(cmd.Page)
	if err != nil {
		return err
	}

	err = walkStrings(pdf, cmd.Page, data, func(_, _, index int, state textState, op string, vals []interface{}) (int, error) {
		s := []byte{}
		if op == "TJ" && len(vals) == 1 {
			if array, ok := vals[0].(pdfArray); ok {
				for _, item := range array {
					if val, ok := item.([]byte); ok {
						s = append(s, state.fonts[state.fontName].ToUnicode(val)...)
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
		fmt.Printf("%4d %s: %s %s\n", index, state.fontName, op, string(s))
		return 0, nil
	})
	if err != nil {
		return err
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

	dict, data, err := pdf.GetPage(cmd.Page)
	if err != nil {
		return err
	}

	err = walkStrings(pdf, cmd.Page, data, func(start, end, index int, state textState, op string, vals []interface{}) (int, error) {
		if index == cmd.Index {
			s := state.fonts[state.fontName].FromUnicode([]byte(cmd.String))

			b := bytes.Buffer{}
			if op == "Tj" {
				pdfWriteVal(&b, nil, pdfRef{}, s)
				b.WriteString(" Tj")
			} else if op == "'" {
				pdfWriteVal(&b, nil, pdfRef{}, s)
				b.WriteString(" '")
			} else if op == "\"" && len(vals) == 3 {
				pdfWriteVal(&b, nil, pdfRef{}, vals[0])
				b.WriteByte(' ')
				pdfWriteVal(&b, nil, pdfRef{}, vals[1])
				b.WriteByte(' ')
				pdfWriteVal(&b, nil, pdfRef{}, s)
				b.WriteString(" \"")
			} else if op == "TJ" && len(vals) == 1 {
				array := pdfArray{}
				array = append(array, s)
				pdfWriteVal(&b, nil, pdfRef{}, array)
				b.WriteString(" TJ")
			}

			fmt.Println("Old:", string(data[start:end]))
			fmt.Println("New:", b.String())
			n := b.Len() - (end - start)
			data = append(data[:start], append(b.Bytes(), data[end:]...)...)
			return n, io.EOF
		}
		return 0, nil
	})
	if err != nil {
		return err
	}

	ref := dict["Contents"].(pdfRef)
	stream, err := pdf.GetStream(ref)
	if err != nil {
		return err
	}
	stream.data = data

	pdfWriter := NewPDFWriter(fw, pdf)
	pdfWriter.SetObject(ref, stream)
	return pdfWriter.Close()
}

type textState struct {
	fonts    map[pdfName]pdfFont
	fontName pdfName
}

func walkStrings(pdf *pdfReader, page int, data []byte, cb func(int, int, int, textState, string, []interface{}) (int, error)) error {
	state := textState{
		fonts: map[pdfName]pdfFont{},
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
					state.fonts[name], err = pdf.GetFont(page, name)
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
