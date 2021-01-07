package font

import "fmt"

type cffTable struct {
}

func (sfnt *SFNT) parseCFF() error {
	_, ok := sfnt.Tables["CFF "]
	if !ok {
		return fmt.Errorf("CFF: missing table")
	}

	sfnt.CFF = &cffTable{}
	// TODO
	return nil
}
