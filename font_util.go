package canvas

import "encoding/binary"

func toTag(s string) uint32 {
	if len(s) != 4 {
		panic("tag must be four bytes")
	}
	return binary.BigEndian.Uint32([]byte(s))
}
