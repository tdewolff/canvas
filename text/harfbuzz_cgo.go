// +build harfbuzz,!js

package text

//#cgo CPPFLAGS: -I/usr/include/harfbuzz
//#cgo LDFLAGS: -L/usr/lib -lharfbuzz
/*
#include <stdlib.h>
#include <hb.h>

hb_glyph_info_t *get_glyph_info(hb_glyph_info_t *info, unsigned int i) {
	return &info[i];
}

hb_glyph_position_t *get_glyph_position(hb_glyph_position_t *pos, unsigned int i) {
	return &pos[i];
}
*/
import "C"
import (
	"strings"
	"unicode/utf8"
	"unsafe"

	"github.com/tdewolff/canvas/font"
)

// Design inspired by https://github.com/npillmayer/tyse/blob/main/engine/text/textshaping/

// Shaper is a text shaper formatting a string in properly positioned glyphs.
type Shaper struct {
	cb    *C.char
	blob  *C.struct_hb_blob_t
	face  *C.struct_hb_face_t
	fonts map[uint16]*C.struct_hb_font_t
}

// NewShaper returns a new text shaper.
func NewShaper(b []byte, index int) (Shaper, error) {
	cb := (*C.char)(C.CBytes(b))
	blob := C.hb_blob_create(cb, C.uint(len(b)), C.HB_MEMORY_MODE_WRITABLE, nil, nil)
	face := C.hb_face_create(blob, C.uint(index))
	return Shaper{
		cb:    cb,
		blob:  blob,
		face:  face,
		fonts: map[uint16]*C.struct_hb_font_t{},
	}, nil
}

// NewShaperSFNT returns a new text shaper using a SFNT structure.
func NewShaperSFNT(sfnt *font.SFNT) (Shaper, error) {
	return NewShaper(sfnt.Data, 0)
}

// Destroy destroys the allocated C memory.
func (s Shaper) Destroy() {
	for _, font := range s.fonts {
		C.hb_font_destroy(font)
	}
	C.hb_face_destroy(s.face)
	C.hb_blob_destroy(s.blob)
	C.free(unsafe.Pointer(s.cb))
}

// Shape shapes the string for a given direction, script, and language.
func (s Shaper) Shape(text string, ppem uint16, direction Direction, script Script, language string, features string, variations string) []Glyph {
	font, ok := s.fonts[ppem]
	if !ok {
		font = C.hb_font_create(s.face)
		C.hb_font_set_ppem(font, C.uint(ppem), C.uint(ppem)) // set font size in points
		s.fonts[ppem] = font
	}

	if variations != "" {
		var cvariations []C.hb_variation_t
		for _, variation := range strings.Split(variations, ",") {
			cvariation := C.CString(variation)
			cvariations = append(cvariations, C.hb_variation_t{})
			ok := C.hb_variation_from_string(cvariation, -1, &cvariations[len(cvariations)-1])
			if ok == 0 {
				cvariations = cvariations[:len(cvariations)-1]
			}
			C.free(unsafe.Pointer(cvariation))
		}
		C.hb_font_set_variations(font, &cvariations[0], C.uint(len(cvariations)))
	}

	ctext := C.CString(text)
	buf := C.hb_buffer_create()
	C.hb_buffer_add_utf8(buf, ctext, -1, 0, -1)
	C.hb_buffer_set_cluster_level(buf, C.HB_BUFFER_CLUSTER_LEVEL_MONOTONE_CHARACTERS)

	C.hb_buffer_set_direction(buf, C.hb_direction_t(direction))
	C.hb_buffer_set_script(buf, C.hb_script_t(script))
	var clanguage *C.char
	if language != "" {
		clanguage = C.CString(language)
		C.hb_buffer_set_language(buf, C.hb_language_from_string(clanguage, -1))
	}
	C.hb_buffer_guess_segment_properties(buf)

	if Direction(C.hb_buffer_get_direction(buf)) == RightToLeft {
		// FriBidi already reversed the direction
		C.hb_buffer_set_direction(buf, C.hb_direction_t(LeftToRight))
	}
	reverse := Direction(C.hb_buffer_get_direction(buf)) == RightToLeft || Direction(C.hb_buffer_get_direction(buf)) == BottomToTop

	var cfeatures []C.hb_feature_t
	for _, feature := range strings.Split(features, ",") {
		cfeature := C.CString(feature)
		cfeatures = append(cfeatures, C.hb_feature_t{})
		ok := C.hb_feature_from_string(cfeature, -1, &cfeatures[len(cfeatures)-1])
		if ok == 0 {
			cfeatures = cfeatures[:len(cfeatures)-1]
		}
		C.free(unsafe.Pointer(cfeature))
	}
	if 0 < len(cfeatures) {
		C.hb_shape(font, buf, &cfeatures[0], C.uint(len(cfeatures)))
	} else {
		C.hb_shape(font, buf, nil, 0)
	}

	length := C.hb_buffer_get_length(buf)
	infos := C.hb_buffer_get_glyph_infos(buf, nil)
	positions := C.hb_buffer_get_glyph_positions(buf, nil)

	glyphs := make([]Glyph, length)
	for i := uint(0); i < uint(length); i++ {
		info := C.get_glyph_info(infos, C.uint(i))
		position := C.get_glyph_position(positions, C.uint(i))
		glyphs[i].ID = uint16(info.codepoint)
		glyphs[i].Cluster = uint32(info.cluster)
		glyphs[i].XAdvance = int32(position.x_advance)
		glyphs[i].YAdvance = int32(position.y_advance)
		glyphs[i].XOffset = int32(position.x_offset)
		glyphs[i].YOffset = int32(position.y_offset)
		if reverse {
			if i != 0 {
				glyphs[i].Text = text[glyphs[i].Cluster:glyphs[i-1].Cluster]
			} else {
				glyphs[i].Text = text[glyphs[i].Cluster:]
			}
		} else if i != 0 {
			glyphs[i-1].Text = text[glyphs[i-1].Cluster:glyphs[i].Cluster]
		}
	}
	if !reverse && 0 < len(glyphs) {
		glyphs[len(glyphs)-1].Text = text[glyphs[len(glyphs)-1].Cluster:]
	}

	if language != "" {
		C.free(unsafe.Pointer(clanguage))
	}
	C.hb_buffer_destroy(buf)
	C.free(unsafe.Pointer(ctext))
	return glyphs
}

// ScriptItemizer divides the string in parts for each different script.
func ScriptItemizer(text string) []string {
	i := 0
	items := []string{}
	curScript := ScriptInvalid
	funcs := C.hb_unicode_funcs_get_default()
	for j := 0; j < len(text); {
		r, n := utf8.DecodeRuneInString(text[j:])
		script := Script(C.hb_unicode_script(funcs, C.uint(r)))
		if j == 0 || curScript == ScriptInherited || curScript == ScriptCommon {
			curScript = script
		} else if script != curScript && script != ScriptInherited && script != ScriptCommon {
			items = append(items, text[i:j])
			curScript = script
			i = j
		}
		j += n
	}
	items = append(items, text[i:])
	return items
}

// Direction is the text direction.
type Direction int

// see Direction
const (
	DirectionInvalid Direction = C.HB_DIRECTION_INVALID
	LeftToRight                = C.HB_DIRECTION_LTR
	RightToLeft                = C.HB_DIRECTION_RTL
	TopToBottom                = C.HB_DIRECTION_TTB
	BottomToTop                = C.HB_DIRECTION_BTT
)

// Script is the script.
type Script uint32

// see Script
const (
	ScriptCommon    Script = C.HB_SCRIPT_COMMON
	ScriptInherited Script = C.HB_SCRIPT_INHERITED
	ScriptUnknown   Script = C.HB_SCRIPT_UNKNOWN

	Arabic     Script = C.HB_SCRIPT_ARABIC
	Armenian   Script = C.HB_SCRIPT_ARMENIAN
	Bengali    Script = C.HB_SCRIPT_BENGALI
	Cyrillic   Script = C.HB_SCRIPT_CYRILLIC
	Devanagari Script = C.HB_SCRIPT_DEVANAGARI
	Georgian   Script = C.HB_SCRIPT_GEORGIAN
	Greek      Script = C.HB_SCRIPT_GREEK
	Gujarati   Script = C.HB_SCRIPT_GUJARATI
	Gurmukhi   Script = C.HB_SCRIPT_GURMUKHI
	Hangul     Script = C.HB_SCRIPT_HANGUL
	Han        Script = C.HB_SCRIPT_HAN
	Hebrew     Script = C.HB_SCRIPT_HEBREW
	Hiragana   Script = C.HB_SCRIPT_HIRAGANA
	Kannada    Script = C.HB_SCRIPT_KANNADA
	Katakana   Script = C.HB_SCRIPT_KATAKANA
	Lao        Script = C.HB_SCRIPT_LAO
	Latin      Script = C.HB_SCRIPT_LATIN
	Malayalam  Script = C.HB_SCRIPT_MALAYALAM
	Oriya      Script = C.HB_SCRIPT_ORIYA
	Tamil      Script = C.HB_SCRIPT_TAMIL
	Telugu     Script = C.HB_SCRIPT_TELUGU
	Thai       Script = C.HB_SCRIPT_THAI

	Tibetan Script = C.HB_SCRIPT_TIBETAN

	Bopomofo          Script = C.HB_SCRIPT_BOPOMOFO
	Braille           Script = C.HB_SCRIPT_BRAILLE
	CanadianSyllabics Script = C.HB_SCRIPT_CANADIAN_SYLLABICS
	Cherokee          Script = C.HB_SCRIPT_CHEROKEE
	Ethiopic          Script = C.HB_SCRIPT_ETHIOPIC
	Khmer             Script = C.HB_SCRIPT_KHMER
	Mongolian         Script = C.HB_SCRIPT_MYANMAR
	Myanmar           Script = C.HB_SCRIPT_OGHAM
	Ogham             Script = C.HB_SCRIPT_OGHAM
	Runic             Script = C.HB_SCRIPT_RUNIC
	Sinhala           Script = C.HB_SCRIPT_SINHALA
	Syriac            Script = C.HB_SCRIPT_SYRIAC
	Thaana            Script = C.HB_SCRIPT_THAANA
	Yi                Script = C.HB_SCRIPT_YI

	Deseret   Script = C.HB_SCRIPT_DESERET
	Gothic    Script = C.HB_SCRIPT_GOTHIC
	OldItalic Script = C.HB_SCRIPT_OLD_ITALIC

	Buhid    Script = C.HB_SCRIPT_BUHID
	Hanunoo  Script = C.HB_SCRIPT_HANUNOO
	Tagalog  Script = C.HB_SCRIPT_TAGALOG
	Tagbanwa Script = C.HB_SCRIPT_TAGBANWA

	Cypriot  Script = C.HB_SCRIPT_CYPRIOT
	Limbu    Script = C.HB_SCRIPT_LIMBU
	LinearB  Script = C.HB_SCRIPT_LINEAR_B
	Osmanya  Script = C.HB_SCRIPT_OSMANYA
	Shavian  Script = C.HB_SCRIPT_SHAVIAN
	TaiLe    Script = C.HB_SCRIPT_TAI_LE
	Ugaritic Script = C.HB_SCRIPT_UGARITIC

	Buginese    Script = C.HB_SCRIPT_BUGINESE
	Coptic      Script = C.HB_SCRIPT_COPTIC
	Glagolitic  Script = C.HB_SCRIPT_GLAGOLITIC
	Kharoshthi  Script = C.HB_SCRIPT_KHAROSHTHI
	NewTaiLue   Script = C.HB_SCRIPT_NEW_TAI_LUE
	OldPersian  Script = C.HB_SCRIPT_OLD_PERSIAN
	SylotiNagri Script = C.HB_SCRIPT_SYLOTI_NAGRI
	Tifinagh    Script = C.HB_SCRIPT_TIFINAGH

	Balinese   Script = C.HB_SCRIPT_BALINESE
	Cuneiform  Script = C.HB_SCRIPT_CUNEIFORM
	Nko        Script = C.HB_SCRIPT_NKO
	PhagsPa    Script = C.HB_SCRIPT_PHAGS_PA
	Phoenician Script = C.HB_SCRIPT_PHOENICIAN

	Carian     Script = C.HB_SCRIPT_CARIAN
	Cham       Script = C.HB_SCRIPT_CHAM
	KayahLi    Script = C.HB_SCRIPT_KAYAH_LI
	Lepcha     Script = C.HB_SCRIPT_LEPCHA
	Lycian     Script = C.HB_SCRIPT_LYCIAN
	Lydian     Script = C.HB_SCRIPT_LYDIAN
	OlChiki    Script = C.HB_SCRIPT_OL_CHIKI
	Rejang     Script = C.HB_SCRIPT_REJANG
	Saurashtra Script = C.HB_SCRIPT_SAURASHTRA
	Sundanese  Script = C.HB_SCRIPT_SUNDANESE
	Vai        Script = C.HB_SCRIPT_VAI

	Avestan               Script = C.HB_SCRIPT_AVESTAN
	Bamum                 Script = C.HB_SCRIPT_BAMUM
	EgyptianHieroglyphs   Script = C.HB_SCRIPT_EGYPTIAN_HIEROGLYPHS
	ImperialAramaic       Script = C.HB_SCRIPT_IMPERIAL_ARAMAIC
	InscriptionalPahlavi  Script = C.HB_SCRIPT_INSCRIPTIONAL_PAHLAVI
	InscriptionalParthian Script = C.HB_SCRIPT_INSCRIPTIONAL_PARTHIAN
	Javanese              Script = C.HB_SCRIPT_JAVANESE
	Kaithi                Script = C.HB_SCRIPT_KAITHI
	Lisu                  Script = C.HB_SCRIPT_LISU
	MeeteiMayek           Script = C.HB_SCRIPT_MEETEI_MAYEK
	OldSouthArabian       Script = C.HB_SCRIPT_OLD_SOUTH_ARABIAN
	OldTurkic             Script = C.HB_SCRIPT_OLD_TURKIC
	Samaritan             Script = C.HB_SCRIPT_SAMARITAN
	TaiTham               Script = C.HB_SCRIPT_TAI_THAM
	TaiViet               Script = C.HB_SCRIPT_TAI_VIET

	Batak   Script = C.HB_SCRIPT_BATAK
	Brahmi  Script = C.HB_SCRIPT_BRAHMI
	Mandaic Script = C.HB_SCRIPT_MANDAIC

	Chakma              Script = C.HB_SCRIPT_CHAKMA
	MeroiticCursive     Script = C.HB_SCRIPT_MEROITIC_CURSIVE
	MeroiticHieroglyphs Script = C.HB_SCRIPT_MEROITIC_HIEROGLYPHS
	Miao                Script = C.HB_SCRIPT_MIAO
	Sharada             Script = C.HB_SCRIPT_SHARADA
	SoraSompeng         Script = C.HB_SCRIPT_SORA_SOMPENG
	Takri               Script = C.HB_SCRIPT_TAKRI

	BassaVah          Script = C.HB_SCRIPT_BASSA_VAH
	CaucasianAlbanian Script = C.HB_SCRIPT_CAUCASIAN_ALBANIAN
	Duployan          Script = C.HB_SCRIPT_DUPLOYAN
	Elbasan           Script = C.HB_SCRIPT_ELBASAN
	Grantha           Script = C.HB_SCRIPT_GRANTHA
	Khojki            Script = C.HB_SCRIPT_KHOJKI
	Khudawadi         Script = C.HB_SCRIPT_KHUDAWADI
	LinearA           Script = C.HB_SCRIPT_LINEAR_A
	Mahajani          Script = C.HB_SCRIPT_MAHAJANI
	Manichaean        Script = C.HB_SCRIPT_MANICHAEAN
	MendeKikakui      Script = C.HB_SCRIPT_MENDE_KIKAKUI
	Modi              Script = C.HB_SCRIPT_MODI
	Mro               Script = C.HB_SCRIPT_MRO
	Nabataean         Script = C.HB_SCRIPT_NABATAEAN
	OldNorthArabian   Script = C.HB_SCRIPT_OLD_NORTH_ARABIAN
	OldPermic         Script = C.HB_SCRIPT_OLD_PERMIC
	PahawhHmong       Script = C.HB_SCRIPT_PAHAWH_HMONG
	Palmyrene         Script = C.HB_SCRIPT_PALMYRENE
	PauCinHau         Script = C.HB_SCRIPT_PAU_CIN_HAU
	PsalterPahlavi    Script = C.HB_SCRIPT_PSALTER_PAHLAVI
	Siddham           Script = C.HB_SCRIPT_SIDDHAM
	Tirhuta           Script = C.HB_SCRIPT_TIRHUTA
	WarangCiti        Script = C.HB_SCRIPT_WARANG_CITI

	Adlam     Script = C.HB_SCRIPT_ADLAM
	Bhaiksuki Script = C.HB_SCRIPT_BHAIKSUKI
	Marchen   Script = C.HB_SCRIPT_MARCHEN
	Osage     Script = C.HB_SCRIPT_OSAGE
	Tangut    Script = C.HB_SCRIPT_TANGUT
	Newa      Script = C.HB_SCRIPT_NEWA

	MasaramGondi    Script = C.HB_SCRIPT_MASARAM_GONDI
	Nushu           Script = C.HB_SCRIPT_NUSHU
	Soyombo         Script = C.HB_SCRIPT_SOYOMBO
	ZanabazarSquare Script = C.HB_SCRIPT_ZANABAZAR_SQUARE

	Dogra          Script = C.HB_SCRIPT_DOGRA
	GunjalaGondi   Script = C.HB_SCRIPT_GUNJALA_GONDI
	HanifiRohingya Script = C.HB_SCRIPT_HANIFI_ROHINGYA
	Makasar        Script = C.HB_SCRIPT_MAKASAR
	Medefaidrin    Script = C.HB_SCRIPT_MEDEFAIDRIN
	OldSogdian     Script = C.HB_SCRIPT_OLD_SOGDIAN
	Sogdian        Script = C.HB_SCRIPT_SOGDIAN

	Elymaic              Script = C.HB_SCRIPT_ELYMAIC
	Nandinagari          Script = C.HB_SCRIPT_NANDINAGARI
	NyiakengPuachueHmong Script = C.HB_SCRIPT_NYIAKENG_PUACHUE_HMONG
	Wancho               Script = C.HB_SCRIPT_WANCHO

	//Chorasmian        Script = C.HB_SCRIPT_CHORASMIAN
	//DivesAkuru        Script = C.HB_SCRIPT_DIVES_AKURU
	//KhitanSmallScript Script = C.HB_SCRIPT_KHITAN_SMALL_SCRIPT
	//Yezidi            Script = C.HB_SCRIPT_YEZIDI

	ScriptInvalid Script = C.HB_TAG_NONE
)
