// +build harfbuzz

#include <hb.h>

/* Helper: get a glyph info struct from an array.
 */
hb_glyph_info_t *get_glyph_info(hb_glyph_info_t *info, unsigned int i) {
    return &info[i];
}

/* Helper: get a glyph position struct from an array.
 */
hb_glyph_position_t *get_glyph_position(hb_glyph_position_t *pos, unsigned int i) {
    return &pos[i];
}
