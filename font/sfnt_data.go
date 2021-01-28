package font

type PlatformID uint16

const (
	PlatformUnicode   = PlatformID(0)
	PlatformMacintosh = PlatformID(1)
	PlatformWindows   = PlatformID(3)
	PlatformCustom    = PlatformID(4)
)

type EncodingID uint16

const (
	EncodingUnicode2BMP                 = EncodingID(3)
	EncodingUnicode2FullRepertoir       = EncodingID(4)
	EncodingUnicodeVariationSequences   = EncodingID(5)
	EncodingUnicodeFullRepertoire       = EncodingID(6)
	EncodingMacintoshRoman              = EncodingID(0)
	EncodingMacintoshJapanese           = EncodingID(1)
	EncodingMacintoshChineseTraditional = EncodingID(2)
	EncodingMacintoshKorean             = EncodingID(3)
	EncodingMacintoshArabic             = EncodingID(4)
	EncodingMacintoshHebrew             = EncodingID(5)
	EncodingMacintoshGreek              = EncodingID(6)
	EncodingMacintoshRussian            = EncodingID(7)
	EncodingMacintoshRSymbol            = EncodingID(8)
	EncodingMacintoshDevanagari         = EncodingID(9)
	EncodingMacintoshGurmukhi           = EncodingID(10)
	EncodingMacintoshGujarati           = EncodingID(11)
	EncodingMacintoshOriya              = EncodingID(12)
	EncodingMacintoshBengali            = EncodingID(13)
	EncodingMacintoshTamil              = EncodingID(14)
	EncodingMacintoshTelugu             = EncodingID(15)
	EncodingMacintoshKannada            = EncodingID(16)
	EncodingMacintoshMalayalam          = EncodingID(17)
	EncodingMacintoshSinhalese          = EncodingID(18)
	EncodingMacintoshBurmese            = EncodingID(19)
	EncodingMacintoshKhmer              = EncodingID(20)
	EncodingMacintoshThai               = EncodingID(21)
	EncodingMacintoshLaotian            = EncodingID(22)
	EncodingMacintoshGeorgian           = EncodingID(23)
	EncodingMacintoshArmenian           = EncodingID(24)
	EncodingMacintoshChineseSimplified  = EncodingID(25)
	EncodingMacintoshTibetan            = EncodingID(26)
	EncodingMacintoshMongolian          = EncodingID(27)
	EncodingMacintoshGeez               = EncodingID(28)
	EncodingMacintoshSlavic             = EncodingID(29)
	EncodingMacintoshVietnamese         = EncodingID(30)
	EncodingMacintoshSindhi             = EncodingID(31)
	EncodingMacintoshUninterpreted      = EncodingID(32)
	EncodingWindowsSymbol               = EncodingID(0)
	EncodingWindowsUnicodeBMP           = EncodingID(1)
	EncodingWindowsShiftJIS             = EncodingID(2)
	EncodingWindowsPRC                  = EncodingID(3)
	EncodingWindowsBig5                 = EncodingID(4)
	EncodingWindowsWansung              = EncodingID(5)
	EncodingWindowsJohab                = EncodingID(6)
	EncodingWindowsUnicodeFullRepertoir = EncodingID(10)
)

type NameID uint16

const (
	NameCopyrightNotice            = NameID(0)
	NameFontFamily                 = NameID(1)
	NameFontSubfamily              = NameID(2)
	NameUniqueIdentifier           = NameID(3)
	NameFull                       = NameID(4)
	NameVersion                    = NameID(5)
	NamePostScript                 = NameID(6)
	NameTrademark                  = NameID(7)
	NameManufacturer               = NameID(8)
	NameDesigner                   = NameID(9)
	NameDescription                = NameID(10)
	NameVendorURL                  = NameID(11)
	NameDesignerURL                = NameID(12)
	NameLicense                    = NameID(13)
	NameLicenseURL                 = NameID(14)
	NamePreferredFamily            = NameID(16)
	NamePreferredSubfamily         = NameID(17)
	NameCompatibleFull             = NameID(18)
	NameSampleText                 = NameID(19)
	NamePostScriptCID              = NameID(20)
	NameWWSFamily                  = NameID(21)
	NameWWSSubfamily               = NameID(22)
	NameLightBackgroundPalette     = NameID(23)
	NameDarkBackgroundPalette      = NameID(24)
	NameVariationsPostScriptPrefix = NameID(25)
)

var macintoshGlyphNames []string = []string{
	"notdef",
	".null",
	"nonmarkingreturn",
	"space",
	"exclam",
	"quotedbl",
	"numbersign",
	"dollar",
	"percent",
	"ampersand",
	"quotesingle",
	"parenleft",
	"parenright",
	"asterisk",
	"plus",
	"comma",
	"hyphen",
	"period",
	"slash",
	"zero",
	"one",
	"two",
	"three",
	"four",
	"five",
	"six",
	"seven",
	"eight",
	"nine",
	"colon",
	"semicolon",
	"less",
	"equal",
	"greater",
	"question",
	"at",
	"A",
	"B",
	"C",
	"D",
	"E",
	"F",
	"G",
	"H",
	"I",
	"J",
	"K",
	"L",
	"M",
	"N",
	"O",
	"P",
	"Q",
	"R",
	"S",
	"T",
	"U",
	"V",
	"W",
	"X",
	"Y",
	"Z",
	"bracketleft",
	"backslash",
	"bracketright",
	"asciicircum",
	"underscore",
	"grave",
	"a",
	"b",
	"c",
	"d",
	"e",
	"f",
	"g",
	"h",
	"i",
	"j",
	"k",
	"l",
	"m",
	"n",
	"o",
	"p",
	"q",
	"r",
	"s",
	"t",
	"u",
	"v",
	"w",
	"x",
	"y",
	"z",
	"braceleft",
	"bar",
	"braceright",
	"asciitilde",
	"Adieresis",
	"Aring",
	"Ccedilla",
	"Eacute",
	"Ntilde",
	"Odieresis",
	"Udieresis",
	"aacute",
	"agrave",
	"acircumflex",
	"adieresis",
	"atilde",
	"aring",
	"ccedilla",
	"eacute",
	"egrave",
	"ecircumflex",
	"edieresis",
	"iacute",
	"igrave",
	"icircumflex",
	"idieresis",
	"ntilde",
	"oacute",
	"ograve",
	"ocircumflex",
	"odieresis",
	"otilde",
	"uacute",
	"ugrave",
	"ucircumflex",
	"udieresis",
	"dagger",
	"degree",
	"cent",
	"sterling",
	"section",
	"bullet",
	"paragraph",
	"germandbls",
	"registered",
	"copyright",
	"trademark",
	"acute",
	"dieresis",
	"notequal",
	"AE",
	"Oslash",
	"infinity",
	"plusminus",
	"lessequal",
	"greaterequal",
	"yen",
	"mu",
	"partialdiff",
	"summation",
	"product",
	"pi",
	"integral",
	"ordfeminine",
	"ordmasculine",
	"Omega",
	"ae",
	"oslash",
	"questiondown",
	"exclamdown",
	"logicalnot",
	"radical",
	"florin",
	"approxequal",
	"Delta",
	"guillemotleft",
	"guillemotright",
	"ellipsis",
	"nonbreakingspace",
	"Agrave",
	"Atilde",
	"Otilde",
	"OE",
	"oe",
	"endash",
	"emdash",
	"quotedblleft",
	"quotedblright",
	"quoteleft",
	"quoteright",
	"divide",
	"lozenge",
	"ydieresis",
	"Ydieresis",
	"fraction",
	"currency",
	"guilsinglleft",
	"guilsinglright",
	"fi",
	"fl",
	"daggerdbl",
	"periodcentered",
	"quotesinglbase",
	"quotedblbase",
	"perthousand",
	"Acircumflex",
	"Ecircumflex",
	"Aacute",
	"Edieresis",
	"Egrave",
	"Iacute",
	"Icircumflex",
	"Idieresis",
	"Igrave",
	"Oacute",
	"Ocircumflex",
	"apple",
	"Ograve",
	"Uacute",
	"Ucircumflex",
	"Ugrave",
	"dotlessi",
	"circumflex",
	"tilde",
	"macron",
	"breve",
	"dotaccent",
	"ring",
	"cedilla",
	"hungarumlaut",
	"ogonek",
	"caron",
	"Lslash",
	"lslash",
	"Scaron",
	"scaron",
	"Zcaron",
	"zcaron",
	"brokenbar",
	"Eth",
	"eth",
	"Yacute",
	"yacute",
	"Thorn",
	"thorn",
	"minus",
	"multiply",
	"onesuperior",
	"twosuperior",
	"threesuperior",
	"onehalf",
	"onequarter",
	"threequarters",
	"franc",
	"Gbreve",
	"gbreve",
	"Idotaccent",
	"Scedilla",
	"scedilla",
	"Cacute",
	"cacute",
	"Ccaron",
	"ccaron",
	"dcroat",
}
