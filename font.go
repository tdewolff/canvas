package canvas

import (
	canvasFont "github.com/tdewolff/canvas/font"
)

// Font defines a font of type TTF or OTF which which a FontFace can be generated for use in text drawing operations.
type Font struct {
	name    string
	SFNT    *canvasFont.SFNT
	usedIDs map[uint16]bool
}

func parseFont(name string, b []byte, index int) (*Font, error) {
	SFNT, err := canvasFont.ParseFont(b, index)
	if err != nil {
		return nil, err
	}

	font := &Font{
		name:    name,
		SFNT:    SFNT,
		usedIDs: map[uint16]bool{},
	}
	return font, nil
}

// Name returns the name of the font.
func (font *Font) Name() string {
	return font.name
}

func (font *Font) Use(glyphID uint16) {
	font.usedIDs[glyphID] = true
}

func (font *Font) UsedIndices() []uint16 {
	glyphIDs := make([]uint16, len(font.usedIDs))
	for glyphID, _ := range font.usedIDs {
		glyphIDs = append(glyphIDs, glyphID)
	}
	return glyphIDs
}
