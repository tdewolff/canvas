//go:build !harfbuzz || js

package text

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	typesettingFont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/harfbuzz"
	"github.com/go-text/typesetting/language"
	"github.com/tdewolff/font"
)

// Shaper is a text shaper formatting a string in properly positioned glyphs.
type Shaper struct {
	font *harfbuzz.Font
}

// NewShaper returns a new text shaper.
func NewShaper(b []byte, _ int) (Shaper, error) {
	loader, err := opentype.NewLoader(bytes.NewReader(b))
	if err != nil {
		return Shaper{}, err
	}
	font, err := typesettingFont.NewFont(loader)
	if err != nil {
		return Shaper{}, err
	}
	return Shaper{
		font: harfbuzz.NewFont(&typesettingFont.Face{Font: font}),
	}, nil
}

// NewShaperSFNT returns a new text shaper using a SFNT structure.
func NewShaperSFNT(sfnt *font.SFNT) (Shaper, error) {
	// TODO: add interface to SFNT for use in this harfbuzz implementation
	return NewShaper(sfnt.Write(), 0)
}

// Destroy destroys the allocated C memory.
func (s Shaper) Destroy() {
}

// Shape shapes the string for a given direction, script, and language.
func (s Shaper) Shape(text string, ppem uint16, direction Direction, script Script, lang string, features string, variations string) []Glyph {
	buf := harfbuzz.NewBuffer()
	rtext := []rune(text)
	buf.AddRunes(rtext, 0, -1)
	buf.ClusterLevel = harfbuzz.MonotoneCharacters
	buf.Props.Direction = harfbuzz.Direction(direction)
	buf.Props.Script = language.Script(script)
	buf.Props.Language = language.NewLanguage(lang)
	buf.GuessSegmentProperties() // only sets direction, script, and language if unset
	buf.Shape(s.font, parseFeatures(features))

	runeMap := make([]int, len(rtext)+1)
	j := 0
	for i := range text {
		runeMap[j] = i
		j++
	}
	runeMap[len(rtext)] = len(text)

	glyphs := make([]Glyph, len(buf.Info))
	for i := 0; i < len(buf.Info); i++ {
		info := buf.Info[i]
		position := buf.Pos[i]
		glyphs[i].ID = uint16(info.Glyph)
		glyphs[i].Cluster = uint32(runeMap[info.Cluster])
		glyphs[i].XAdvance = int32(position.XAdvance)
		glyphs[i].YAdvance = int32(position.YAdvance)
		glyphs[i].XOffset = int32(position.XOffset)
		glyphs[i].YOffset = int32(position.YOffset)
		glyphs[i].Text = rtext[info.Cluster]
	}
	return glyphs
}

func parseFeatures(input string) []harfbuzz.Feature {
	if input == "" {
		return nil
	}

	features := []harfbuzz.Feature{}
	for _, part := range strings.Split(input, ",") {
		if part == "" {
			continue
		}

		feature, err := harfbuzz.ParseFeature(part)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing feature '%s': %v\n", part, err)
			continue
		}

		features = append(features, feature)
	}

	return features
}

func LookupScript(r rune) Script {
	return Script(language.LookupScript(r))
}

// Direction is the text direction.
type Direction int

// see Direction
const (
	DirectionInvalid Direction = 0
	LeftToRight                = Direction(harfbuzz.LeftToRight)
	RightToLeft                = Direction(harfbuzz.RightToLeft)
	TopToBottom                = Direction(harfbuzz.TopToBottom)
	BottomToTop                = Direction(harfbuzz.BottomToTop)
)

//go:generate stringer -type Script

// Script is the script.
type Script uint32

// see Script
const (
	ScriptInvalid   Script = 0
	ScriptCommon           = Script(language.Common)
	ScriptInherited        = Script(language.Inherited)
	ScriptUnknown          = Script(language.Unknown)

	Arabic     = Script(language.Arabic)
	Armenian   = Script(language.Armenian)
	Bengali    = Script(language.Bengali)
	Cyrillic   = Script(language.Cyrillic)
	Devanagari = Script(language.Devanagari)
	Georgian   = Script(language.Georgian)
	Greek      = Script(language.Greek)
	Gujarati   = Script(language.Gujarati)
	Gurmukhi   = Script(language.Gurmukhi)
	Hangul     = Script(language.Hangul)
	Han        = Script(language.Han)
	Hebrew     = Script(language.Hebrew)
	Hiragana   = Script(language.Hiragana)
	Kannada    = Script(language.Kannada)
	Katakana   = Script(language.Katakana)
	Lao        = Script(language.Lao)
	Latin      = Script(language.Latin)
	Malayalam  = Script(language.Malayalam)
	Oriya      = Script(language.Oriya)
	Tamil      = Script(language.Tamil)
	Telugu     = Script(language.Telugu)
	Thai       = Script(language.Thai)

	Tibetan = Script(language.Tibetan)

	Bopomofo          = Script(language.Bopomofo)
	Braille           = Script(language.Braille)
	CanadianSyllabics = Script(language.Canadian_Aboriginal)
	Cherokee          = Script(language.Cherokee)
	Ethiopic          = Script(language.Ethiopic)
	Khmer             = Script(language.Khmer)
	Mongolian         = Script(language.Mongolian)
	Myanmar           = Script(language.Myanmar)
	Ogham             = Script(language.Ogham)
	Runic             = Script(language.Runic)
	Sinhala           = Script(language.Sinhala)
	Syriac            = Script(language.Syriac)
	Thaana            = Script(language.Thaana)
	Yi                = Script(language.Yi)

	Deseret   = Script(language.Deseret)
	Gothic    = Script(language.Gothic)
	OldItalic = Script(language.Old_Italic)

	Buhid    = Script(language.Buhid)
	Hanunoo  = Script(language.Hanunoo)
	Tagalog  = Script(language.Tagalog)
	Tagbanwa = Script(language.Tagbanwa)

	Cypriot  = Script(language.Cypriot)
	Limbu    = Script(language.Limbu)
	LinearB  = Script(language.Linear_B)
	Osmanya  = Script(language.Osmanya)
	Shavian  = Script(language.Shavian)
	TaiLe    = Script(language.Tai_Le)
	Ugaritic = Script(language.Ugaritic)

	Buginese    = Script(language.Buginese)
	Coptic      = Script(language.Coptic)
	Glagolitic  = Script(language.Glagolitic)
	Kharoshthi  = Script(language.Kharoshthi)
	NewTaiLue   = Script(language.New_Tai_Lue)
	OldPersian  = Script(language.Old_Persian)
	SylotiNagri = Script(language.Syloti_Nagri)
	Tifinagh    = Script(language.Tifinagh)

	Balinese   = Script(language.Balinese)
	Cuneiform  = Script(language.Cuneiform)
	Nko        = Script(language.Nko)
	PhagsPa    = Script(language.Phags_Pa)
	Phoenician = Script(language.Phoenician)

	Carian     = Script(language.Carian)
	Cham       = Script(language.Cham)
	KayahLi    = Script(language.Kayah_Li)
	Lepcha     = Script(language.Lepcha)
	Lycian     = Script(language.Lycian)
	Lydian     = Script(language.Lydian)
	OlChiki    = Script(language.Ol_Chiki)
	Rejang     = Script(language.Rejang)
	Saurashtra = Script(language.Saurashtra)
	Sundanese  = Script(language.Sundanese)
	Vai        = Script(language.Vai)

	Avestan               = Script(language.Avestan)
	Bamum                 = Script(language.Bamum)
	EgyptianHieroglyphs   = Script(language.Egyptian_Hieroglyphs)
	ImperialAramaic       = Script(language.Imperial_Aramaic)
	InscriptionalPahlavi  = Script(language.Inscriptional_Pahlavi)
	InscriptionalParthian = Script(language.Inscriptional_Parthian)
	Javanese              = Script(language.Javanese)
	Kaithi                = Script(language.Kaithi)
	Lisu                  = Script(language.Lisu)
	MeeteiMayek           = Script(language.Meetei_Mayek)
	OldSouthArabian       = Script(language.Old_South_Arabian)
	OldTurkic             = Script(language.Old_Turkic)
	Samaritan             = Script(language.Samaritan)
	TaiTham               = Script(language.Tai_Tham)
	TaiViet               = Script(language.Tai_Viet)

	Batak   = Script(language.Batak)
	Brahmi  = Script(language.Brahmi)
	Mandaic = Script(language.Mandaic)

	Chakma              = Script(language.Chakma)
	MeroiticCursive     = Script(language.Meroitic_Cursive)
	MeroiticHieroglyphs = Script(language.Meroitic_Hieroglyphs)
	Miao                = Script(language.Miao)
	Sharada             = Script(language.Sharada)
	SoraSompeng         = Script(language.Sora_Sompeng)
	Takri               = Script(language.Takri)

	BassaVah          = Script(language.Bassa_Vah)
	CaucasianAlbanian = Script(language.Caucasian_Albanian)
	Duployan          = Script(language.Duployan)
	Elbasan           = Script(language.Elbasan)
	Grantha           = Script(language.Grantha)
	Khojki            = Script(language.Khojki)
	Khudawadi         = Script(language.Khudawadi)
	LinearA           = Script(language.Linear_A)
	Mahajani          = Script(language.Mahajani)
	Manichaean        = Script(language.Manichaean)
	MendeKikakui      = Script(language.Mende_Kikakui)
	Modi              = Script(language.Modi)
	Mro               = Script(language.Mro)
	Nabataean         = Script(language.Nabataean)
	OldNorthArabian   = Script(language.Old_North_Arabian)
	OldPermic         = Script(language.Old_Permic)
	PahawhHmong       = Script(language.Pahawh_Hmong)
	Palmyrene         = Script(language.Palmyrene)
	PauCinHau         = Script(language.Pau_Cin_Hau)
	PsalterPahlavi    = Script(language.Psalter_Pahlavi)
	Siddham           = Script(language.Siddham)
	Tirhuta           = Script(language.Tirhuta)
	WarangCiti        = Script(language.Warang_Citi)

	Adlam     = Script(language.Adlam)
	Bhaiksuki = Script(language.Bhaiksuki)
	Marchen   = Script(language.Marchen)
	Osage     = Script(language.Osage)
	Tangut    = Script(language.Tangut)
	Newa      = Script(language.Newa)

	MasaramGondi    = Script(language.Masaram_Gondi)
	Nushu           = Script(language.Nushu)
	Soyombo         = Script(language.Soyombo)
	ZanabazarSquare = Script(language.Zanabazar_Square)

	Dogra          = Script(language.Dogra)
	GunjalaGondi   = Script(language.Gunjala_Gondi)
	HanifiRohingya = Script(language.Hanifi_Rohingya)
	Makasar        = Script(language.Makasar)
	Medefaidrin    = Script(language.Medefaidrin)
	OldSogdian     = Script(language.Old_Sogdian)
	Sogdian        = Script(language.Sogdian)

	Elymaic              = Script(language.Elymaic)
	Nandinagari          = Script(language.Nandinagari)
	NyiakengPuachueHmong = Script(language.Nyiakeng_Puachue_Hmong)
	Wancho               = Script(language.Wancho)

	Chorasmian        = Script(language.Chorasmian)
	DivesAkuru        = Script(language.Dives_Akuru)
	KhitanSmallScript = Script(language.Khitan_Small_Script)
	Yezidi            = Script(language.Yezidi)
)
