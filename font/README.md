# Font
[![GoDoc](http://godoc.org/github.com/tdewolff/canvas/font?status.svg)](http://godoc.org/github.com/tdewolff/canvas/font)

This library contains font parsers for WOFF, WOFF2, and EOT. It takes raw input from said font formats and converts them to TTF or OTF fonts. As font formats for the web, WOFF, WOFF2, and EOT are really just containers for SFNT fonts (such as TTF and OTF).

## Examples
### WOFF
``` go
b, err := ioutil.ReadFile("DejaVuSerif.woff")
if err != nil {
    panic(err)
}

b, _, err = cf.ParseWOFF(b)
if err != nil {
    panic(err)
}

err = ioutil.WriteFile("dejavuserif_out.ttf", b, 0644)
if err != nil {
    panic(err)
}
```

### WOFF2
``` go
b, err := ioutil.ReadFile("DejaVuSerif.woff2")
if err != nil {
    panic(err)
}

b, _, err = cf.ParseWOFF2(b)
if err != nil {
    panic(err)
}

err = ioutil.WriteFile("dejavuserif_out.ttf", b, 0644)
if err != nil {
    panic(err)
}
```

### EOT
``` go
b, err := ioutil.ReadFile("DejaVuSerif.woff2")
if err != nil {
    panic(err)
}

b, _, err = cf.ParseWOFF2(b)
if err != nil {
    panic(err)
}

err = ioutil.WriteFile("dejavuserif_out.ttf", b, 0644)
if err != nil {
    panic(err)
}
```

## License
Released under the [MIT license](LICENSE.md).
