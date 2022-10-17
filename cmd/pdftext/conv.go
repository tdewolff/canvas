//go:build exclude
package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
)

func main() {
	r, err := os.Open("agl.txt")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	w, err := os.Create("data.go")
	if err != nil {
		panic(err)
	}
	defer w.Close()

	w.WriteString("var charsetName = map[string]rune{\n")
	scanner := bufio.NewScanner(r)
	i := 0
	for scanner.Scan() {
		i++
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		cols := strings.Split(line, ";")
		if len(cols) != 2 || (len(cols[1])+1)%5 != 0 {
			panic(fmt.Sprintf("bad format at line %d", i))
		}

		comment := false
		name := cols[0]

		codes := []uint16{}
		for j := 0; j < len(cols[1]); j += 5 {
			b, err := hex.DecodeString(cols[1][j : j+4])
			if err != nil {
				panic(err)
			}
			codes = append(codes, binary.BigEndian.Uint16(b))
		}
		rs := utf16.Decode(codes)
		if len(rs) != 1 {
			fmt.Printf("bad rune at line %d\n", i)
			comment = true
		}
		if comment {
			w.WriteString("//")
		}
		s := string(rs)
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `'`, `\'`)
		w.WriteString(fmt.Sprintf("    \"%v\": '\\u%04x', // %v\n", name, rs[0], s))
	}
	w.WriteString("}\n")
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
