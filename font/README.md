# Font [![API reference](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/tdewolff/canvas/font?tab=doc)

This library contains font parsers for WOFF, WOFF2, and EOT. It takes a byte-slice as input and converts it to SFNT formats (either TTF or OTF). As font formats for the web, WOFF, WOFF2, and EOT are really just containers for SFNT fonts (such as TTF and OTF) that have better compression.

The WOFF and WOFF2 converters have been testing using the validation tests from the W3C. Font collections (such as TTC and OTC) are not yet supported, but will be in the near future. Compression in EOT files are also not yet supported.

## Usage
Import using:

``` go
import "github.com/tdewolff/canvas/font"
```

Then we can parse any byte-slice that is in the WOFF/WOFF2/EOT file format and extract its TTF/OTF content.

``` go
font, err := ioutil.ReadFile("DejaVuSerif.woff")
if err != nil {
    panic(err)
}

sfnt, err := font.ToSFNT(font)
if err != nil {
    panic(err)
}
```

or using an `io.Reader`

``` go
font, err := os.Open("DejaVuSerif.woff")
if err != nil {
    panic(err)
}

font, err = font.NewSFNTReader(font)
if err != nil {
    panic(err)
}
```

### WOFF
``` go
woff, err := ioutil.ReadFile("DejaVuSerif.woff")
if err != nil {
    panic(err)
}

sfnt, err := font.ParseWOFF(woff)
if err != nil {
    panic(err)
}

ext := font.Extension(sfnt)
if err = ioutil.WriteFile("DejaVuSerif"+ext, sfnt, 0644); err != nil {
    panic(err)
}
```

Tested using https://github.com/w3c/woff/tree/master/woff1/tests.

### WOFF2
``` go
woff2, err := ioutil.ReadFile("DejaVuSerif.woff2")
if err != nil {
    panic(err)
}

sfnt, err := font.ParseWOFF2(woff2)
if err != nil {
    panic(err)
}

ext := font.Extension(sfnt)
if err = ioutil.WriteFile("DejaVuSerif"+ext, sfnt, 0644); err != nil {
    panic(err)
}
```

Tested using https://github.com/w3c/woff2-tests.

### EOT
``` go
eof, err := ioutil.ReadFile("DejaVuSerif.eot")
if err != nil {
    panic(err)
}

sfnt, err = font.ParseEOT(eot)
if err != nil {
    panic(err)
}

ext := font.Extension(sfnt)
if err = ioutil.WriteFile("DejaVuSerif"+ext, sfnt, 0644); err != nil {
    panic(err)
}
```

## License
Released under the [MIT license](LICENSE.md).
