// +build harfbuzz

package text

//#cgo CPPFLAGS: -I/usr/include/harfbuzz
//#cgo LDFLAGS: -L/usr/lib -lharfbuzz
/*
#include <stdlib.h>
#include <hb.h>

hb_glyph_info_t *get_glyph_info(hb_glyph_info_t *, unsigned int);
hb_glyph_position_t *get_glyph_position(hb_glyph_position_t *, unsigned int);
*/
import "C"
import (
	"unsafe"

	"github.com/tdewolff/canvas/font"
)

// Design inspired by https://github.com/npillmayer/tyse/blob/main/engine/text/textshaping/

type Font struct {
	cb    *C.char
	blob  *C.struct_hb_blob_t
	face  *C.struct_hb_face_t
	fonts map[float64]*C.struct_hb_font_t
}

func NewFont(b []byte, index int) (Font, error) {
	cb := (*C.char)(C.CBytes(b))
	blob := C.hb_blob_create(cb, C.uint(len(b)), C.HB_MEMORY_MODE_WRITABLE, nil, nil)
	face := C.hb_face_create(blob, C.uint(index))
	return Font{
		cb:    cb,
		blob:  blob,
		face:  face,
		fonts: map[float64]*C.struct_hb_font_t{},
	}, nil
}

func NewSFNTFont(sfnt *font.SFNT) (Font, error) {
	return NewFont(sfnt.Data, 0)
}

func (f Font) Destroy() {
	for _, font := range f.fonts {
		C.hb_font_destroy(font)
	}
	C.hb_face_destroy(f.face)
	C.hb_blob_destroy(f.blob)
	C.free(unsafe.Pointer(f.cb))
}

func (f Font) Shape(text string, ppem float64, direction Direction, script Script) []Glyph {
	font, ok := f.fonts[ppem]
	if !ok {
		font = C.hb_font_create(f.face)
		C.hb_font_set_ptem(font, C.float(ppem)) // set font size in points
		f.fonts[ppem] = font
	}

	ctext := C.CString(text)
	buf := C.hb_buffer_create()
	C.hb_buffer_add_utf8(buf, ctext, -1, 0, -1)
	C.hb_buffer_set_direction(buf, C.hb_direction_t(direction))
	C.hb_buffer_set_script(buf, C.hb_script_t(script))
	// TODO: set language
	C.hb_buffer_set_cluster_level(buf, C.HB_BUFFER_CLUSTER_LEVEL_MONOTONE_CHARACTERS)
	C.hb_shape(font, buf, nil, 0) // TODO: set features (liga,clig,sups,subs,unic,titl,smcp,pcap,c2sc,c2pc,swsh,cswh,salt,ornm,nalt)

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
	}

	C.hb_buffer_destroy(buf)
	C.free(unsafe.Pointer(ctext))
	return glyphs
}

type Direction int

const (
	DirectionInvalid Direction = C.HB_DIRECTION_INVALID
	LeftToRight                = C.HB_DIRECTION_LTR
	RightToLeft                = C.HB_DIRECTION_RTL
	TopToBottom                = C.HB_DIRECTION_TTB
	BottomToTop                = C.HB_DIRECTION_BTT
)

type Script uint32

// Taken from github.com/npillmayer/gotype
const (
	Common    Script = C.HB_SCRIPT_COMMON
	Inherited        = C.HB_SCRIPT_INHERITED
	Unknown          = C.HB_SCRIPT_UNKNOWN

	Arabic     = C.HB_SCRIPT_ARABIC
	Armenian   = C.HB_SCRIPT_ARMENIAN
	Bengali    = C.HB_SCRIPT_BENGALI
	Cyrillic   = C.HB_SCRIPT_CYRILLIC
	Devanagari = C.HB_SCRIPT_DEVANAGARI
	Georgian   = C.HB_SCRIPT_GEORGIAN
	Greek      = C.HB_SCRIPT_GREEK
	Gujarati   = C.HB_SCRIPT_GUJARATI
	Gurmukhi   = C.HB_SCRIPT_GURMUKHI
	Hangul     = C.HB_SCRIPT_HANGUL
	Han        = C.HB_SCRIPT_HAN
	Hebrew     = C.HB_SCRIPT_HEBREW
	Hiragana   = C.HB_SCRIPT_HIRAGANA
	Kannada    = C.HB_SCRIPT_KANNADA
	Katakana   = C.HB_SCRIPT_KATAKANA
	Lao        = C.HB_SCRIPT_LAO
	Latin      = C.HB_SCRIPT_LATIN
	Malayalam  = C.HB_SCRIPT_MALAYALAM
	Oriya      = C.HB_SCRIPT_ORIYA
	Tamil      = C.HB_SCRIPT_TAMIL
	Telugu     = C.HB_SCRIPT_TELUGU
	Thai       = C.HB_SCRIPT_THAI

	Tibetan = C.HB_SCRIPT_TIBETAN

	Bopomofo          = C.HB_SCRIPT_BOPOMOFO
	Braille           = C.HB_SCRIPT_BRAILLE
	CanadianSyllabics = C.HB_SCRIPT_CANADIAN_SYLLABICS
	Cherokee          = C.HB_SCRIPT_CHEROKEE
	Ethiopic          = C.HB_SCRIPT_ETHIOPIC
	Khmer             = C.HB_SCRIPT_KHMER
	Mongolian         = C.HB_SCRIPT_MYANMAR
	Myanmar           = C.HB_SCRIPT_OGHAM
	Ogham             = C.HB_SCRIPT_OGHAM
	Runic             = C.HB_SCRIPT_RUNIC
	Sinhala           = C.HB_SCRIPT_SINHALA
	Syriac            = C.HB_SCRIPT_SYRIAC
	Thaana            = C.HB_SCRIPT_THAANA
	Yi                = C.HB_SCRIPT_YI

	Deseret   = C.HB_SCRIPT_DESERET
	Gothic    = C.HB_SCRIPT_GOTHIC
	OldItalic = C.HB_SCRIPT_OLD_ITALIC

	Buhid    = C.HB_SCRIPT_BUHID
	Hanunoo  = C.HB_SCRIPT_HANUNOO
	Tagalog  = C.HB_SCRIPT_TAGALOG
	Tagbanwa = C.HB_SCRIPT_TAGBANWA

	Cypriot  = C.HB_SCRIPT_CYPRIOT
	Limbu    = C.HB_SCRIPT_LIMBU
	LinearB  = C.HB_SCRIPT_LINEAR_B
	Osmanya  = C.HB_SCRIPT_OSMANYA
	Shavian  = C.HB_SCRIPT_SHAVIAN
	TaiLe    = C.HB_SCRIPT_TAI_LE
	Ugaritic = C.HB_SCRIPT_UGARITIC

	Buginese    = C.HB_SCRIPT_BUGINESE
	Coptic      = C.HB_SCRIPT_COPTIC
	Glagolitic  = C.HB_SCRIPT_GLAGOLITIC
	Kharoshthi  = C.HB_SCRIPT_KHAROSHTHI
	NewTaiLue   = C.HB_SCRIPT_NEW_TAI_LUE
	OldPersian  = C.HB_SCRIPT_OLD_PERSIAN
	SylotiNagri = C.HB_SCRIPT_SYLOTI_NAGRI
	Tifinagh    = C.HB_SCRIPT_TIFINAGH

	Balinese   = C.HB_SCRIPT_BALINESE
	Cuneiform  = C.HB_SCRIPT_CUNEIFORM
	Nko        = C.HB_SCRIPT_NKO
	PhagsPa    = C.HB_SCRIPT_PHAGS_PA
	Phoenician = C.HB_SCRIPT_PHOENICIAN

	Carian     = C.HB_SCRIPT_CARIAN
	Cham       = C.HB_SCRIPT_CHAM
	KayahLi    = C.HB_SCRIPT_KAYAH_LI
	Lepcha     = C.HB_SCRIPT_LEPCHA
	Lycian     = C.HB_SCRIPT_LYCIAN
	Lydian     = C.HB_SCRIPT_LYDIAN
	OlChiki    = C.HB_SCRIPT_OL_CHIKI
	Rejang     = C.HB_SCRIPT_REJANG
	Saurashtra = C.HB_SCRIPT_SAURASHTRA
	Sundanese  = C.HB_SCRIPT_SUNDANESE
	Vai        = C.HB_SCRIPT_VAI

	Avestan               = C.HB_SCRIPT_AVESTAN
	Bamum                 = C.HB_SCRIPT_BAMUM
	EgyptianHieroglyphs   = C.HB_SCRIPT_EGYPTIAN_HIEROGLYPHS
	ImperialAramaic       = C.HB_SCRIPT_IMPERIAL_ARAMAIC
	InscriptionalPahlavi  = C.HB_SCRIPT_INSCRIPTIONAL_PAHLAVI
	InscriptionalParthian = C.HB_SCRIPT_INSCRIPTIONAL_PARTHIAN
	Javanese              = C.HB_SCRIPT_JAVANESE
	Kaithi                = C.HB_SCRIPT_KAITHI
	Lisu                  = C.HB_SCRIPT_LISU
	MeeteiMayek           = C.HB_SCRIPT_MEETEI_MAYEK
	OldSouthArabian       = C.HB_SCRIPT_OLD_SOUTH_ARABIAN
	OldTurkic             = C.HB_SCRIPT_OLD_TURKIC
	Samaritan             = C.HB_SCRIPT_SAMARITAN
	TaiTham               = C.HB_SCRIPT_TAI_THAM
	TaiViet               = C.HB_SCRIPT_TAI_VIET

	Batak   = C.HB_SCRIPT_BATAK
	Brahmi  = C.HB_SCRIPT_BRAHMI
	Mandaic = C.HB_SCRIPT_MANDAIC

	Chakma              = C.HB_SCRIPT_CHAKMA
	MeroiticCursive     = C.HB_SCRIPT_MEROITIC_CURSIVE
	MeroiticHieroglyphs = C.HB_SCRIPT_MEROITIC_HIEROGLYPHS
	Miao                = C.HB_SCRIPT_MIAO
	Sharada             = C.HB_SCRIPT_SHARADA
	SoraSompeng         = C.HB_SCRIPT_SORA_SOMPENG
	Takri               = C.HB_SCRIPT_TAKRI

	BassaVah          = C.HB_SCRIPT_BASSA_VAH
	CaucasianAlbanian = C.HB_SCRIPT_CAUCASIAN_ALBANIAN
	Duployan          = C.HB_SCRIPT_DUPLOYAN
	Elbasan           = C.HB_SCRIPT_ELBASAN
	Grantha           = C.HB_SCRIPT_GRANTHA
	Khojki            = C.HB_SCRIPT_KHOJKI
	Khudawadi         = C.HB_SCRIPT_KHUDAWADI
	LinearA           = C.HB_SCRIPT_LINEAR_A
	Mahajani          = C.HB_SCRIPT_MAHAJANI
	Manichaean        = C.HB_SCRIPT_MANICHAEAN
	MendeKikakui      = C.HB_SCRIPT_MENDE_KIKAKUI
	Modi              = C.HB_SCRIPT_MODI
	Mro               = C.HB_SCRIPT_MRO
	Nabataean         = C.HB_SCRIPT_NABATAEAN
	OldNorthArabian   = C.HB_SCRIPT_OLD_NORTH_ARABIAN
	OldPermic         = C.HB_SCRIPT_OLD_PERMIC
	PahawhHmong       = C.HB_SCRIPT_PAHAWH_HMONG
	Palmyrene         = C.HB_SCRIPT_PALMYRENE
	PauCinHau         = C.HB_SCRIPT_PAU_CIN_HAU
	PsalterPahlavi    = C.HB_SCRIPT_PSALTER_PAHLAVI
	Siddham           = C.HB_SCRIPT_SIDDHAM
	Tirhuta           = C.HB_SCRIPT_TIRHUTA
	WarangCiti        = C.HB_SCRIPT_WARANG_CITI

	Adlam     = C.HB_SCRIPT_ADLAM
	Bhaiksuki = C.HB_SCRIPT_BHAIKSUKI
	Marchen   = C.HB_SCRIPT_MARCHEN
	Osage     = C.HB_SCRIPT_OSAGE
	Tangut    = C.HB_SCRIPT_TANGUT
	Newa      = C.HB_SCRIPT_NEWA

	MasaramGondi    = C.HB_SCRIPT_MASARAM_GONDI
	Nushu           = C.HB_SCRIPT_NUSHU
	Soyombo         = C.HB_SCRIPT_SOYOMBO
	ZanabazarSquare = C.HB_SCRIPT_ZANABAZAR_SQUARE

	Dogra          = C.HB_SCRIPT_DOGRA
	GunjalaGondi   = C.HB_SCRIPT_GUNJALA_GONDI
	HanifiRohingya = C.HB_SCRIPT_HANIFI_ROHINGYA
	Makasar        = C.HB_SCRIPT_MAKASAR
	Medefaidrin    = C.HB_SCRIPT_MEDEFAIDRIN
	OldSogdian     = C.HB_SCRIPT_OLD_SOGDIAN
	Sogdian        = C.HB_SCRIPT_SOGDIAN

	Elymaic              = C.HB_SCRIPT_ELYMAIC
	Nandinagari          = C.HB_SCRIPT_NANDINAGARI
	NyiakengPuachueHmong = C.HB_SCRIPT_NYIAKENG_PUACHUE_HMONG
	Wancho               = C.HB_SCRIPT_WANCHO

	//Chorasmian        = C.HB_SCRIPT_CHORASMIAN
	//DivesAkuru        = C.HB_SCRIPT_DIVES_AKURU
	//KhitanSmallScript = C.HB_SCRIPT_KHITAN_SMALL_SCRIPT
	//Yezidi            = C.HB_SCRIPT_YEZIDI

	ScriptInvalid = C.HB_TAG_NONE
)
