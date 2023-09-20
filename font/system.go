package font

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func DefaultFontDirs() []string {
	var dirs []string
	switch runtime.GOOS {
	case "aix", "dragonfly", "freebsd", "illumos", "js", "linux", "nacl", "netbsd", "openbsd", "solaris":
		dirs = []string{
			"/usr/share/fonts",
			"/usr/local/share/fonts",
		}
		if home := os.Getenv("HOME"); home != "" {
			dirs = append(dirs, filepath.Join(home, ".fonts"))
			dirs = append(dirs, filepath.Join(home, ".local/share/fonts"))
		}
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			dirs = append(dirs, filepath.Join(xdgDataHome, "fonts"))
		}
	case "android":
		dirs = []string{
			"/system/fonts",
			"/system/font",
			"/data/fonts",
		}
	case "darwin":
		dirs = []string{
			"/Library/Fonts",
			"/System/Library/Fonts",
			"/Network/Library/Fonts",
			"/System/Library/Assets/com_apple_MobileAsset_Font3",
			"/System/Library/Assets/com_apple_MobileAsset_Font4",
			"/System/Library/Assets/com_apple_MobileAsset_Font5",
		}
		if home := os.Getenv("HOME"); home != "" {
			dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
		}
	case "ios":
		dirs = []string{
			"/System/Library/Fonts",
			"/System/Library/Fonts/Cache",
		}
	case "plan9":
		dirs = []string{
			"/lib/font",
		}
		if home := os.Getenv("HOME"); home != "" {
			dirs = append(dirs, filepath.Join(home, "lib", "font"))
		}
	case "windows":
		sysRoot := os.Getenv("SYSTEMROOT")
		if sysRoot == "" {
			sysRoot = os.Getenv("SYSTEMDRIVE")
		}
		if sysRoot == "" { // try with the common C:
			sysRoot = "C:"
		}
		dirs = []string{
			filepath.Join(filepath.VolumeName(sysRoot), `Windows`, "Fonts"),
		}
	}
	return dirs
}

func DefaultSystemFonts() map[string][]string {
	// TODO: use OS and OS version or maybe even parse fontconfig files for Unix
	fonts := map[string][]string{}
	//switch runtime.GOOS {
	//case "darwin", "ios":
	//	if font == "-apple-system" {
	//	}
	//}

	// these defaults are from ArchLinux
	fonts["serif"] = []string{
		"Noto Serif",
		"DejaVu Serif",
		"Times New Roman",
		"Thorndale AMT",
		"Luxi Serif",
		"Nimbus Roman No9 L",
		"Nimbus Roman",
		"Times",
	}
	fonts["sans-serif"] = []string{
		"Noto Sans",
		"DejaVu Sans",
		"Verdana",
		"Arial",
		"Albany AMT",
		"Luxi Sans",
		"Nimbus Sans L",
		"Nimbus Sans",
		"Helvetica",
		"Lucida Sans Unicode",
		"BPG Glaho International",
		"Tahoma",
	}
	fonts["monospace"] = []string{
		"Noto Sans Mono",
		"DejaVu Sans Mono",
		"Inconsolata",
		"Andale Mono",
		"Courier New",
		"Cumberland AMT",
		"Luxi Mono",
		"Nimbus Mono L",
		"Nimbus Mono",
		"Nimbus Mono PS",
		"Courier",
	}
	fonts["fantasy"] = []string{
		"Impact",
		"Copperplate Gothic Std",
		"Cooper Std",
		"Bauhaus Std",
	}
	fonts["cursive"] = []string{
		"ITC Zapf Chancery Std",
		"Zapfino",
		"Comic Sans MS",
	}
	fonts["system-ui"] = []string{
		"Cantarell",
		"Noto Sans UI",
		"Segoe UI",
		"Segoe UI Historic",
		"Segoe UI Symbol",
	}
	return fonts
}

// Style defines the font style to be used for the font. It specifies a boldness with optionally italic, e.g. Black | Italic will specify a black boldness (a font-weight of 800 in CSS) and italic.
type Style int

// see Style
const (
	UnknownStyle Style = -1
	Thin         Style = iota
	ExtraLight
	Light
	Regular
	Medium
	SemiBold
	Bold
	ExtraBold
	Black
	Italic Style = 1 << 8
)

func ParseStyleCSS(weight int, italic bool) Style {
	weight = int(float64(weight)/100.0+0.5) * 100
	if weight < 100 {
		weight = 100
	} else if 900 < weight {
		weight = 900
	}

	var style Style
	if italic {
		style = Italic
	}
	switch weight {
	case 100:
		return style | Thin
	case 200:
		return style | ExtraLight
	case 300:
		return style | Light
	case 500:
		return style | Medium
	case 600:
		return style | SemiBold
	case 700:
		return style | Bold
	case 800:
		return style | ExtraBold
	case 900:
		return style | Black
	}
	return style | Regular
}

func ParseStyle(s string) Style {
	var style Style
	s = strings.TrimSpace(s)
	italics := []string{"Italic", "Oblique", "Slanted", "Italique", "Cursiva", "Slant", "Ita", "Obl"}
	for _, italic := range italics {
		if strings.HasSuffix(s, italic) {
			s = s[:len(s)-len(italic)]
			if 0 < len(s) && (s[len(s)-1] == ' ' || s[len(s)-1] == '-') {
				s = s[:len(s)-1]
			}
			style = Italic
			break
		}
	}

	switch s {
	case "", "Regular", "Normal", "Normál", "Reg", "Roman", "Text", "Book":
		style |= Regular
	case "Thin", "Hairline":
		style |= Thin
	case "ExtraLight", "UltraLight", "Extra Light", "ExtraLt":
		style |= ExtraLight
	case "Light", "Lt", "SemiLight":
		style |= Light
	case "Medium", "Med":
		style |= Medium
	case "SemiBold", "Semibold", "Demi", "Semi Bold", "Semibd":
		style |= SemiBold
	case "Bold", "Negrita", "Gras", "Bol":
		style |= Bold
	case "ExtraBold", "Extra Bold":
		style |= ExtraBold
	case "Black", "Heavy":
		style |= Black
	default:
		style = UnknownStyle
	}
	return style
}

// Weight returns the font weight (Regular, Bold, ...)
func (style Style) Weight() Style {
	return style & 0xFF
}

// Italic returns true if italic.
func (style Style) Italic() bool {
	return style&Italic != 0
}

func (style Style) String() string {
	var s string
	switch style.Weight() {
	case Thin:
		s = "Thin"
	case ExtraLight:
		s = "ExtraLight"
	case Light:
		s = "Light"
	case Regular:
		s = "Regular"
	case Medium:
		s = "Medium"
	case SemiBold:
		s = "SemiBold"
	case Bold:
		s = "Bold"
	case ExtraBold:
		s = "ExtraBold"
	case Black:
		s = "Black"
	default:
		return "UnknownStyle"
	}
	if style.Italic() {
		s += " Italic"
	}
	return s
}

type FontMetadata struct {
	Filename string
	Family   string
	Style
}

func (metadata FontMetadata) String() string {
	return fmt.Sprintf("%s (%v): %s", metadata.Family, metadata.Style, metadata.Filename)
}

type SystemFonts struct {
	Defaults map[string][]string
	Fonts    map[string]map[Style]FontMetadata
}

func LoadSystemFonts(filename string) (*SystemFonts, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	fonts := &SystemFonts{}
	if err = gob.NewDecoder(f).Decode(fonts); err != nil {
		return nil, err
	}
	return fonts, nil
}

func (s *SystemFonts) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	if err = gob.NewEncoder(f).Encode(s); err != nil {
		return err
	}
	return nil
}

func (s *SystemFonts) Add(metadata FontMetadata) {
	if _, ok := s.Fonts[metadata.Family]; !ok {
		s.Fonts[metadata.Family] = map[Style]FontMetadata{}
	}
	s.Fonts[metadata.Family][metadata.Style] = metadata
}

func (s *SystemFonts) Match(name string, style Style) (FontMetadata, bool) {
	var metadatas map[Style]FontMetadata
	if names, ok := s.Defaults[name]; ok {
		// get font names for serif, sans-serif, system-ui, ...
		for _, name := range names {
			if metadatas, ok = s.Fonts[name]; ok {
				break
			}
		}
	} else if metadatas, ok = s.Fonts[name]; !ok {
		return FontMetadata{}, false
	}

	if metadata, ok := metadatas[style]; ok {
		return metadata, true
	}

	styles := []Style{}
	weight := style.Weight()
	if weight == Regular {
		styles = append(styles, Medium)
	} else if weight == Medium {
		styles = append(styles, Regular)
	}
	if weight == SemiBold || weight == Bold || weight == ExtraBold || weight == Black {
		for s := weight + 1; s <= Black; s++ {
			styles = append(styles, s)
		}
		for s := weight - 1; Thin <= s; s-- {
			styles = append(styles, s)
		}
	} else {
		for s := weight - 1; Thin <= s; s-- {
			styles = append(styles, s)
		}
		for s := weight + 1; s <= Black; s++ {
			styles = append(styles, s)
		}
	}

	for _, s := range styles {
		if metadata, ok := metadatas[style&Italic|s]; ok {
			return metadata, true
		}
	}
	return FontMetadata{}, false
}

func (s *SystemFonts) String() string {
	sb := &strings.Builder{}

	fmt.Fprintf(sb, "Default font families:")
	defaults := make([]string, 0, len(s.Defaults))
	for def := range s.Defaults {
		defaults = append(defaults, def)
	}
	sort.Strings(defaults)
	for _, def := range defaults {
		fmt.Fprintf(sb, "\n  %s:", def)
		for i, family := range s.Defaults[def] {
			if i != 0 {
				fmt.Fprintf(sb, ",")
			}
			fmt.Fprintf(sb, " %s", family)
		}
	}

	fmt.Fprintf(sb, "\n\nFont family styles:")
	families := make([]string, 0, len(s.Fonts))
	for family := range s.Fonts {
		families = append(families, family)
	}
	sort.Strings(families)
	for _, family := range families {
		fmt.Fprintf(sb, "\n  %s:", family)
		styles := make([]Style, 0, len(s.Fonts[family]))
		for style := range s.Fonts[family] {
			styles = append(styles, style)
		}
		sort.SliceStable(styles, func(i, j int) bool {
			return styles[i] < styles[j]
		})
		for i, style := range styles {
			if i != 0 {
				fmt.Fprintf(sb, ",")
			}
			fmt.Fprintf(sb, " %v", style)
		}
	}
	fmt.Fprintf(sb, "\n")
	return sb.String()
}

func FindSystemFonts(dirs []string) (*SystemFonts, error) {
	// TODO: use concurrency
	fonts := &SystemFonts{
		Fonts: map[string]map[Style]FontMetadata{},
	}
	walkedDirs := map[string]bool{}
	walkDir := func(dir string) error {
		return fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
			path = filepath.Join(dir, path)
			if err != nil {
				return err
			} else if d.IsDir() {
				if walkedDirs[path] {
					return filepath.SkipDir
				}
				walkedDirs[path] = true
				return nil
			} else if !d.Type().IsRegular() {
				return nil
			}

			var getMetadata func(io.ReadSeeker) (FontMetadata, error)
			switch filepath.Ext(path) {
			case ".ttf", ".otf":
				getMetadata = getSFNTMetadata
				// TODO: handle .ttc, .woff, .woff2, .eot
			}

			if getMetadata != nil {
				f, err := os.Open(path)
				if err != nil {
					return nil
				}
				defer f.Close()

				metadata, err := getMetadata(f)
				if err != nil {
					return nil
				}
				metadata.Filename = path
				fonts.Add(metadata)
			}
			return nil
		})
	}

	var Err error
	for _, dir := range dirs {
		if info, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		} else if !info.IsDir() {
			continue
		}

		if err := walkDir(dir); err != nil && Err == nil {
			Err = err
		}
	}
	if Err != nil {
		return nil, Err
	}
	fonts.Defaults = DefaultSystemFonts()
	return fonts, nil
}

func read(r io.Reader, length int) ([]byte, error) {
	b := make([]byte, length) // TODO: reuse
	if n, err := r.Read(b); err != nil {
		return nil, err
	} else if n != length {
		return nil, fmt.Errorf("invalid length")
	}
	return b, nil
}

func u16(b []byte) uint16 {
	return (uint16(b[0]) << 8) + uint16(b[1])
}

func u32(b []byte) uint32 {
	return (uint32(b[0]) << 24) + (uint32(b[1]) << 16) + (uint32(b[2]) << 8) + uint32(b[3])
}

func getSFNTMetadata(r io.ReadSeeker) (FontMetadata, error) {
	header, err := read(r, 12)
	if err != nil {
		return FontMetadata{}, err
	}
	numTables := u16(header[4:])

	// read tables list
	var offset uint32
	tables, err := read(r, 16*int(numTables))
	if err != nil {
		return FontMetadata{}, err
	}
	for i := 0; i < 16*int(numTables); i += 16 {
		if bytes.Equal(tables[i:i+4], []byte("name")) {
			offset = u32(tables[i+8:])
			break
		}
	}
	if offset == 0 {
		return FontMetadata{}, fmt.Errorf("name table not found")
	}

	// read name table
	if _, err = r.Seek(int64(offset), io.SeekStart); err != nil {
		return FontMetadata{}, err
	}
	nameTable, err := read(r, 6)
	if err != nil {
		return FontMetadata{}, err
	}
	version := u16(nameTable)
	count := u16(nameTable[2:])
	storageOffset := int64(offset) + int64(u16(nameTable[4:]))

	metadata := FontMetadata{}
	decodeUTF16 := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
	if version == 0 {
		records, err := read(r, 12*int(count))
		if err != nil {
			return FontMetadata{}, err
		}

		found := 0
		var family, subfamily string
		for i := 0; i < 12*int(count); i += 12 {
			// TODO: check platform and encoding?
			platform := PlatformID(u16(records[i:]))
			//encoding := EncodingID(u16(records[i+2:]))
			language := u16(records[i+4:])
			if platform != PlatformWindows && (language&0x00FF) != 0x0009 {
				continue // not English or not Windows
			}

			name := NameID(u16(records[i+6:]))
			if name == NameFontFamily || name == NameFontSubfamily || name == NamePreferredFamily || name == NamePreferredSubfamily {
				length := u16(records[i+8:])
				offset := u16(records[i+10:])
				if _, err = r.Seek(storageOffset+int64(offset), io.SeekStart); err != nil {
					return FontMetadata{}, err
				}
				val, err := read(r, int(length))
				if err != nil {
					return FontMetadata{}, err
				}
				val, _, err = transform.Bytes(decodeUTF16, val)
				if err != nil {
					return FontMetadata{}, err
				}
				if name == NameFontFamily || name == NamePreferredFamily {
					family = string(val)
				} else if name == NameFontSubfamily || name == NamePreferredSubfamily {
					subfamily = string(val)
				}
				if name == NamePreferredFamily || name == NamePreferredSubfamily {
					found++
					//if found == 2 {
					//	break // break early
					//}
				}
			}
		}
		if family == "" {
			return FontMetadata{}, fmt.Errorf("font family not found")
		}

		style := ParseStyle(subfamily)
		if style == UnknownStyle {
			return FontMetadata{}, fmt.Errorf("unknown subfamily style: %s", subfamily)
		}

		metadata.Family = family
		metadata.Style = style
	} else if version == 1 {
		// TODO
	}
	return metadata, nil
}
