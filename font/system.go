package font

import (
	"encoding/gob"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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
			filepath.Join(filepath.VolumeName(sysRoot), `\Windows`, "Fonts"),
		}
	}
	return dirs
}

func DefaultGenericFonts() map[string][]string {
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
	fonts["cursive"] = []string{
		"ITC Zapf Chancery Std",
		"Zapfino",
		"Comic Sans MS",
	}
	fonts["fantasy"] = []string{
		"Impact",
		"Copperplate Gothic Std",
		"Cooper Std",
		"Bauhaus Std",
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
	Families []string
	Style
}

func (metadata FontMetadata) String() string {
	return fmt.Sprintf("%s (%v): %s", strings.Join(metadata.Families, ","), metadata.Style, metadata.Filename)
}

type SystemFonts struct {
	Generics map[string][]string
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
	for _, family := range metadata.Families {
		if _, ok := s.Fonts[family]; !ok {
			s.Fonts[family] = map[Style]FontMetadata{}
		}
		s.Fonts[family][metadata.Style] = metadata
	}
}

func (s *SystemFonts) Match(name string, style Style) (FontMetadata, bool) {
	// expand generic font names
	families := strings.Split(name, ",")
	for i := 0; i < len(families); i++ {
		families[i] = strings.TrimSpace(families[i])
		if names, ok := s.Generics[families[i]]; ok {
			families = append(families[:i], append(names, families[i+1:]...)...)
			i += len(names) - 1
		}
	}

	// find the first font name that exists
	var metadatas map[Style]FontMetadata
	for _, family := range families {
		metadatas, _ = s.Fonts[family]
		if metadatas != nil {
			break
		}
	}
	if metadatas == nil {
		return FontMetadata{}, false
	}

	// exact style match
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

	fmt.Fprintf(sb, "Generic font families:")
	generics := make([]string, 0, len(s.Generics))
	for generic := range s.Generics {
		generics = append(generics, generic)
	}
	sort.Strings(generics)
	for _, generic := range generics {
		fmt.Fprintf(sb, "\n  %s:", generic)
		for i, family := range s.Generics[generic] {
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

			fontData, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			metadata, err := ParseMetadata(fontData, 0)
			if err != nil {
				return nil
			}
			metadata.Filename = path
			fonts.Add(*metadata)

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
	fonts.Generics = DefaultGenericFonts()
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

