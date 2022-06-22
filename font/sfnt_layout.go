package font

import "fmt"

type langSys struct {
	requiredFeatureIndex uint16
	featureIndices       []uint16
}

type scriptList map[ScriptTag]map[LanguageTag]langSys

func (scriptList scriptList) getLangSys(scriptTag ScriptTag, languageTag LanguageTag) (langSys, bool) {
	script, ok := scriptList[scriptTag]
	if !ok || scriptTag == UnknownScript {
		script, ok = scriptList[DefaultScript]
		if !ok {
			return langSys{}, false
		}
	}

	langSys, ok := script[languageTag]
	if !ok || languageTag == UnknownLanguage {
		langSys = script[DefaultLanguage] // always present
	}
	return langSys, true
}

func (sfnt *SFNT) parseScriptList(b []byte) (scriptList, error) {
	r := NewBinaryReader(b)
	r2 := NewBinaryReader(b)
	r3 := NewBinaryReader(b)
	scriptCount := r.ReadUint16()
	scripts := make(scriptList, scriptCount)
	for i := 0; i < int(scriptCount); i++ {
		scriptTag := ScriptTag(r.ReadString(4))
		scriptOffset := r.ReadUint16()

		r2.Seek(uint32(scriptOffset))
		defaultLangSysOffset := r2.ReadUint16()
		langSysCount := r2.ReadUint16()
		langSyss := make(map[LanguageTag]langSys, langSysCount)
		for j := -1; j < int(langSysCount); j++ {
			var langSysTag LanguageTag
			var langSysOffset uint16
			if j == -1 {
				langSysTag = DefaultLanguage // permanently reserved and cannot be used in font
				langSysOffset = defaultLangSysOffset
				if langSysOffset == 0 {
					continue
				}
			} else {
				langSysTag = LanguageTag(r2.ReadString(4))
				langSysOffset = r2.ReadUint16()
				if langSysTag == DefaultLanguage {
					return scripts, fmt.Errorf("bad language tag")
				}
			}

			r3.Seek(uint32(scriptOffset) + uint32(langSysOffset))
			lookupOrderOffset := r3.ReadUint16()
			if lookupOrderOffset != 0 {
				return scripts, fmt.Errorf("lookupOrderOffset must be NULL")
			}
			requiredFeatureIndex := r3.ReadUint16()
			featureIndexCount := r3.ReadUint16()
			featureIndices := make([]uint16, featureIndexCount)
			for k := 0; k < int(featureIndexCount); k++ {
				featureIndices[k] = r3.ReadUint16()
			}
			langSyss[langSysTag] = langSys{
				requiredFeatureIndex: requiredFeatureIndex,
				featureIndices:       featureIndices,
			}
		}
		scripts[scriptTag] = langSyss
	}
	return scripts, nil
}

////////////////////////////////////////////////////////////////

type featureList struct {
	tag     []FeatureTag
	feature [][]uint16
}

func (featureList featureList) get(i uint16) (FeatureTag, []uint16, error) {
	if int(i) < len(featureList.tag) {
		return featureList.tag[i], featureList.feature[i], nil
	}
	return UnknownFeature, nil, fmt.Errorf("invalid feature index")
}

func (sfnt *SFNT) parseFeatureList(b []byte) featureList {
	r := NewBinaryReader(b)
	r2 := NewBinaryReader(b)
	featureCount := r.ReadUint16()
	tags := make([]FeatureTag, featureCount)
	features := make([][]uint16, featureCount)
	for i := uint16(0); i < featureCount; i++ {
		featureTag := FeatureTag(r.ReadString(4))
		featureOffset := r.ReadUint16()

		r2.Seek(uint32(featureOffset))
		_ = r2.ReadUint16() // featureParamsOffset
		lookupIndexCount := r2.ReadUint16()
		lookupListIndices := make([]uint16, lookupIndexCount)
		for j := 0; j < int(lookupIndexCount); j++ {
			lookupListIndices[j] = r2.ReadUint16()
		}

		tags[i] = featureTag
		features[i] = lookupListIndices
	}
	return featureList{
		tag:     tags,
		feature: features,
	}
}

////////////////////////////////////////////////////////////////

type lookup struct {
	lookupType       uint16
	lookupFlag       uint16
	subtable         [][]byte
	markFilteringSet uint16
}

type lookupList []lookup

func (sfnt *SFNT) parseLookupList(b []byte) lookupList {
	r := NewBinaryReader(b)
	r2 := NewBinaryReader(b)
	lookupCount := r.ReadUint16()
	lookups := make(lookupList, lookupCount)
	for i := 0; i < int(lookupCount); i++ {
		lookupOffset := r.ReadUint16()

		r2.Seek(uint32(lookupOffset))
		lookups[i].lookupType = r2.ReadUint16()
		lookups[i].lookupFlag = r2.ReadUint16()
		subtableCount := r2.ReadUint16()
		lookups[i].subtable = make([][]byte, subtableCount)
		for j := 0; j < int(subtableCount); j++ {
			subtableOffset := r2.ReadUint16()
			lookups[i].subtable[j] = b[lookupOffset+subtableOffset:]
		}
		lookups[i].markFilteringSet = r2.ReadUint16()
	}
	return lookups
}

////////////////////////////////////////////////////////////////

type featureVariationsList struct {
	Data []byte
}

////////////////////////////////////////////////////////////////

type coverageTable interface {
	Index(uint16) (uint16, bool)
}

type coverageFormat1 struct {
	glyphArray []uint16
}

func (table *coverageFormat1) Index(glyphID uint16) (uint16, bool) {
	for i, coverageGlyphID := range table.glyphArray {
		if coverageGlyphID < glyphID {
			break
		} else if coverageGlyphID == glyphID {
			return uint16(i), true
		}
	}
	return 0, false
}

type coverageFormat2 struct {
	startGlyphID       []uint16
	endGlyphID         []uint16
	startCoverageIndex []uint16
}

func (table *coverageFormat2) Index(glyphID uint16) (uint16, bool) {
	for i := 0; i < len(table.startGlyphID); i++ {
		if table.endGlyphID[i] < glyphID {
			break
		} else if table.startGlyphID[i] <= glyphID {
			return table.startCoverageIndex[i] + glyphID - table.startGlyphID[i], true
		}
	}
	return 0, false
}

func (sfnt *SFNT) parseCoverageTable(b []byte) (coverageTable, error) {
	r := NewBinaryReader(b)
	coverageFormat := r.ReadUint16()
	if coverageFormat == 1 {
		glyphCount := r.ReadUint16()
		glyphArray := make([]uint16, glyphCount)
		for i := 0; i < int(glyphCount); i++ {
			glyphArray[i] = r.ReadUint16()
		}
		return &coverageFormat1{
			glyphArray: glyphArray,
		}, nil
	} else if coverageFormat == 2 {
		rangeCount := r.ReadUint16()
		startGlyphIDs := make([]uint16, rangeCount)
		endGlyphIDs := make([]uint16, rangeCount)
		startCoverageIndices := make([]uint16, rangeCount)
		for i := 0; i < int(rangeCount); i++ {
			startGlyphIDs[i] = r.ReadUint16()
			endGlyphIDs[i] = r.ReadUint16()
			startCoverageIndices[i] = r.ReadUint16()
		}
		return &coverageFormat2{
			startGlyphID:       startGlyphIDs,
			endGlyphID:         endGlyphIDs,
			startCoverageIndex: startCoverageIndices,
		}, nil
	}
	return nil, fmt.Errorf("bad coverage table format")
}

////////////////////////////////////////////////////////////////

type classDefTable interface {
	Get(uint16) uint16
}

type classDefFormat1 struct {
	startGlyphID    uint16
	classValueArray []uint16
}

func (table *classDefFormat1) Get(glyphID uint16) uint16 {
	if table.startGlyphID <= glyphID && glyphID-table.startGlyphID < uint16(len(table.classValueArray)) {
		return table.classValueArray[glyphID-table.startGlyphID]
	}
	return 0
}

type classRangeRecord struct {
	startGlyphID uint16
	endGlyphID   uint16
	class        uint16
}

type classDefFormat2 []classRangeRecord

func (table classDefFormat2) Get(glyphID uint16) uint16 {
	for _, classRange := range table {
		if glyphID < classRange.startGlyphID {
			break
		} else if classRange.startGlyphID <= glyphID && glyphID <= classRange.endGlyphID {
			return classRange.class
		}
	}
	return 0
}

func (sfnt *SFNT) parseClassDefTable(b []byte, classCount uint16) (classDefTable, error) {
	r := NewBinaryReader(b)
	classFormat := r.ReadUint16()
	if classFormat == 1 {
		startGlyphID := r.ReadUint16()
		glyphCount := r.ReadUint16()
		classValueArray := make([]uint16, glyphCount)
		for i := 0; i < int(glyphCount); i++ {
			classValueArray[i] = r.ReadUint16()
			if classCount <= classValueArray[i] {
				return nil, fmt.Errorf("bad class value")
			}
		}
		return &classDefFormat1{
			startGlyphID:    startGlyphID,
			classValueArray: classValueArray,
		}, nil
	} else if classFormat == 2 {
		classRangeCount := r.ReadUint16()
		classRangeRecords := make(classDefFormat2, classRangeCount)
		for i := 0; i < int(classRangeCount); i++ {
			classRangeRecords[i].startGlyphID = r.ReadUint16()
			classRangeRecords[i].endGlyphID = r.ReadUint16()
			classRangeRecords[i].class = r.ReadUint16()
			if classCount <= classRangeRecords[i].class {
				return nil, fmt.Errorf("bad class value")
			}
		}
		return classRangeRecords, nil
	}
	return nil, fmt.Errorf("bad class definition table format")
}

////////////////////////////////////////////////////////////////

type ValueRecord struct {
	XPlacement       int16
	YPlacement       int16
	XAdvance         int16
	YAdvance         int16
	XPlaDeviceOffset uint16
	YPlaDeviceOffset uint16
	XAdvDeviceOffset uint16
	YAdvDeviceOffset uint16
}

func (sfnt *SFNT) parseValueRecord(r *BinaryReader, valueFormat uint16) (valueRecord ValueRecord) {
	if valueFormat == 0 {
		return
	} else if valueFormat&0x0001 != 0 { // X_PLACEMENT
		valueRecord.XPlacement = r.ReadInt16()
	} else if valueFormat&0x0002 != 0 { // Y_PLACEMENT
		valueRecord.YPlacement = r.ReadInt16()
	} else if valueFormat&0x0004 != 0 { // X_ADVANCE
		valueRecord.XAdvance = r.ReadInt16()
	} else if valueFormat&0x0008 != 0 { // Y_ADVANCE
		valueRecord.YAdvance = r.ReadInt16()
	} else if valueFormat&0x0010 != 0 { // X_PLACEMENT_DEVICE
		valueRecord.XPlaDeviceOffset = r.ReadUint16()
	} else if valueFormat&0x0020 != 0 { // Y_PLACEMENT_DEVICE
		valueRecord.YPlaDeviceOffset = r.ReadUint16()
	} else if valueFormat&0x0040 != 0 { // X_ADVANCE_DEVICE
		valueRecord.XAdvDeviceOffset = r.ReadUint16()
	} else if valueFormat&0x0080 != 0 { // Y_ADVANCE_DEVICE
		valueRecord.YAdvDeviceOffset = r.ReadUint16()
	}
	return
}

////////////////////////////////////////////////////////////////

type singlePosTables []singlePosTable

func (tables singlePosTables) Get(glyphID uint16) (ValueRecord, bool) {
	for _, table := range tables {
		if valueRecord, ok := table.Get(glyphID); ok {
			return valueRecord, true
		}
	}
	return ValueRecord{}, false
}

type singlePosTable interface {
	Get(uint16) (ValueRecord, bool)
}

type singlePosFormat1 struct {
	coverageTable
	valueRecord ValueRecord
}

func (table *singlePosFormat1) Get(glyphID uint16) (ValueRecord, bool) {
	if _, ok := table.Index(glyphID); ok {
		return table.valueRecord, true
	}
	return ValueRecord{}, false
}

type singlePosFormat2 struct {
	coverageTable
	valueRecord []ValueRecord
}

func (table *singlePosFormat2) Get(glyphID uint16) (ValueRecord, bool) {
	if i, ok := table.Index(glyphID); ok {
		return table.valueRecord[i], true
	}
	return ValueRecord{}, false
}

func (sfnt *SFNT) parseSinglePosTable(b []byte) (interface{}, error) {
	r := NewBinaryReader(b)
	posFormat := r.ReadUint16()
	coverageOffset := r.ReadUint16()
	coverageTable, err := sfnt.parseCoverageTable(b[coverageOffset:])
	if err != nil {
		return nil, err
	}

	valueFormat := r.ReadUint16()
	if posFormat == 1 {
		valueRecord := sfnt.parseValueRecord(r, valueFormat)
		return &singlePosFormat1{
			coverageTable: coverageTable,
			valueRecord:   valueRecord,
		}, nil
	} else if posFormat == 2 {
		valueCount := r.ReadUint16()
		valueRecord := make([]ValueRecord, valueCount)
		for i := 0; i < int(valueCount); i++ {
			valueRecord[i] = sfnt.parseValueRecord(r, valueFormat)
		}
		return &singlePosFormat2{
			coverageTable: coverageTable,
			valueRecord:   valueRecord,
		}, nil
	}
	return nil, fmt.Errorf("bad single adjustment positioning table format")
}

////////////////////////////////////////////////////////////////

type pairPosTables []pairPosTable

func (tables pairPosTables) Get(glyphID1, glyphID2 uint16) (ValueRecord, ValueRecord, bool) {
	for _, table := range tables {
		if valueRecord1, valueRecord2, ok := table.Get(glyphID1, glyphID2); ok {
			return valueRecord1, valueRecord2, true
		}
	}
	return ValueRecord{}, ValueRecord{}, false
}

type pairPosTable interface {
	Get(uint16, uint16) (ValueRecord, ValueRecord, bool)
}

type pairValueRecord struct {
	secondGlyph  uint16
	valueRecord1 ValueRecord
	valueRecord2 ValueRecord
}

type pairPosFormat1 struct {
	coverageTable
	pairSet [][]pairValueRecord
}

func (table *pairPosFormat1) Get(glyphID1, glyphID2 uint16) (ValueRecord, ValueRecord, bool) {
	if i, ok := table.Index(glyphID1); ok {
		for j := 0; j < len(table.pairSet[i]); j++ {
			if table.pairSet[i][j].secondGlyph == glyphID2 {
				return table.pairSet[i][j].valueRecord1, table.pairSet[i][j].valueRecord2, true
			} else if glyphID2 < table.pairSet[i][j].secondGlyph {
				break
			}
		}
	}
	return ValueRecord{}, ValueRecord{}, false
}

type class2Record struct {
	valueRecord1 ValueRecord
	valueRecord2 ValueRecord
}

type pairPosFormat2 struct {
	coverageTable
	classDef1     classDefTable
	classDef2     classDefTable
	class1Records [][]class2Record
}

func (table *pairPosFormat2) Get(glyphID1, glyphID2 uint16) (ValueRecord, ValueRecord, bool) {
	if _, ok := table.Index(glyphID1); ok {
		class1 := table.classDef1.Get(glyphID1)
		class2 := table.classDef1.Get(glyphID2)
		return table.class1Records[class1][class2].valueRecord1, table.class1Records[class1][class2].valueRecord2, true
	}
	return ValueRecord{}, ValueRecord{}, false
}

func (sfnt *SFNT) parsePairPosTable(b []byte) (interface{}, error) {
	r := NewBinaryReader(b)
	r2 := NewBinaryReader(b)
	posFormat := r.ReadUint16()
	coverageOffset := r.ReadUint16()
	coverageTable, err := sfnt.parseCoverageTable(b[coverageOffset:])
	if err != nil {
		return nil, err
	}

	valueFormat1 := r.ReadUint16()
	valueFormat2 := r.ReadUint16()
	if posFormat == 1 {
		pairSetCount := r.ReadUint16()
		pairSet := make([][]pairValueRecord, pairSetCount)
		for i := 0; i < int(pairSetCount); i++ {
			pairSetOffset := r.ReadUint16()

			r2.Seek(uint32(pairSetOffset))
			pairValueCount := r2.ReadUint16()
			pairValueRecords := make([]pairValueRecord, pairValueCount)
			for j := 0; j < int(pairValueCount); j++ {
				pairValueRecords[j].secondGlyph = r2.ReadUint16()
				pairValueRecords[j].valueRecord1 = sfnt.parseValueRecord(r2, valueFormat1)
				pairValueRecords[j].valueRecord2 = sfnt.parseValueRecord(r2, valueFormat2)
			}
			pairSet[i] = pairValueRecords
		}
		return &pairPosFormat1{
			coverageTable: coverageTable,
			pairSet:       pairSet,
		}, nil
	} else if posFormat == 2 {
		classDef1Offset := r.ReadUint16()
		classDef2Offset := r.ReadUint16()
		class1Count := r.ReadUint16()
		class2Count := r.ReadUint16()
		classDef1, err := sfnt.parseClassDefTable(b[classDef1Offset:], class1Count)
		if err != nil {
			return nil, err
		}
		classDef2 := classDef1
		if classDef1Offset != classDef2Offset {
			if classDef2, err = sfnt.parseClassDefTable(b[classDef2Offset:], class2Count); err != nil {
				return nil, err
			}
		}

		class1Records := make([][]class2Record, class1Count)
		for j := 0; j < int(class1Count); j++ {
			class1Records[j] = make([]class2Record, class2Count)
			for i := 0; i < int(class2Count); i++ {
				class1Records[j][i].valueRecord1 = sfnt.parseValueRecord(r, valueFormat1)
				class1Records[j][i].valueRecord2 = sfnt.parseValueRecord(r, valueFormat2)
			}
		}
		return &pairPosFormat2{
			coverageTable: coverageTable,
			classDef1:     classDef1,
			classDef2:     classDef2,
			class1Records: class1Records,
		}, nil
	}
	return nil, fmt.Errorf("bad single adjustment positioning table format")
}

////////////////////////////////////////////////////////////////

type singleSubstFormat1 struct {
	coverageTable
	deltaGlyphID int16
}

func (table *singleSubstFormat1) Get(glyphID uint16) (uint16, bool) {
	if _, ok := table.Index(glyphID); ok {
		// uint16 does modulo%65536
		return uint16(int(glyphID) + int(table.deltaGlyphID)), true
	}
	return 0, false
}

type singleSubstFormat2 struct {
	coverageTable
	substituteGlyphIDs []uint16
}

func (table *singleSubstFormat2) Get(glyphID uint16) (uint16, bool) {
	if i, ok := table.Index(glyphID); ok {
		return table.substituteGlyphIDs[i], true
	}
	return 0, false
}

func (sfnt *SFNT) parseSingleSubstTable(b []byte) (interface{}, error) {
	r := NewBinaryReader(b)
	substFormat := r.ReadUint16()
	coverageOffset := r.ReadUint16()
	coverageTable, err := sfnt.parseCoverageTable(b[coverageOffset:])
	if err != nil {
		return nil, err
	}

	if substFormat == 1 {
		deltaGlyphID := r.ReadInt16()
		return &singleSubstFormat1{
			coverageTable: coverageTable,
			deltaGlyphID:  deltaGlyphID,
		}, nil
	} else if substFormat == 2 {
		glyphCount := r.ReadUint16()
		substituteGlyphIDs := make([]uint16, glyphCount)
		for i := 0; i < int(glyphCount); i++ {
			substituteGlyphIDs[i] = r.ReadUint16()
		}
		return &singleSubstFormat2{
			coverageTable:      coverageTable,
			substituteGlyphIDs: substituteGlyphIDs,
		}, nil
	}
	return nil, fmt.Errorf("bad single substitution table format")
}

////////////////////////////////////////////////////////////////

type multipleSubstFormat1 struct {
	coverageTable
	sequences [][]uint16
}

func (table *multipleSubstFormat1) Get(glyphID uint16) ([]uint16, bool) {
	if i, ok := table.Index(glyphID); ok {
		return table.sequences[i], true
	}
	return nil, false
}

func (sfnt *SFNT) parseMultipleSubstTable(b []byte) (interface{}, error) {
	r := NewBinaryReader(b)
	r2 := NewBinaryReader(b)
	substFormat := r.ReadUint16()
	if substFormat != 1 {
		return nil, fmt.Errorf("bad multiple substitution table format")
	}

	coverageOffset := r.ReadUint16()
	coverageTable, err := sfnt.parseCoverageTable(b[coverageOffset:])
	if err != nil {
		return nil, err
	}

	sequenceCount := r.ReadUint16()
	sequences := make([][]uint16, sequenceCount)
	for i := 0; i < int(sequenceCount); i++ {
		sequenceOffset := r.ReadUint16()

		r2.Seek(uint32(sequenceOffset))
		glyphCount := r2.ReadUint16()
		substituteGlyphIDs := make([]uint16, glyphCount)
		for i := 0; i < int(glyphCount); i++ {
			substituteGlyphIDs[i] = r2.ReadUint16()
		}
		sequences[i] = substituteGlyphIDs
	}
	return &multipleSubstFormat1{
		coverageTable: coverageTable,
		sequences:     sequences,
	}, nil
}

func (sfnt *SFNT) parseAlternateSubstTable(b []byte) (interface{}, error) {
	r := NewBinaryReader(b)
	substFormat := r.ReadUint16()
	if substFormat != 1 {
		return nil, fmt.Errorf("bad alternate substitution table format")
	}
	return sfnt.parseMultipleSubstTable(b)
}

////////////////////////////////////////////////////////////////

type ligature struct {
	ligatureGlyph     uint16
	componentGlyphIDs []uint16
}

type ligatureSubstFormat1 struct {
	coverageTable
	ligatures [][]ligature
}

func (table *ligatureSubstFormat1) Get(glyphIDs []uint16) (uint16, bool) {
	if i, ok := table.Index(glyphIDs[0]); ok {
	LigatureLoop:
		for _, ligature := range table.ligatures[i] {
			for j, componentGlyphID := range ligature.componentGlyphIDs {
				if len(glyphIDs) <= j+1 || componentGlyphID != glyphIDs[j+1] {
					continue LigatureLoop
				}
			}
			return ligature.ligatureGlyph, true
		}
	}
	return 0, false
}

func (sfnt *SFNT) parseLigatureSubstTable(b []byte) (interface{}, error) {
	r := NewBinaryReader(b)
	r2 := NewBinaryReader(b)
	r3 := NewBinaryReader(b)
	substFormat := r.ReadUint16()
	if substFormat != 1 {
		return nil, fmt.Errorf("bad ligature substitution table format")
	}

	coverageOffset := r.ReadUint16()
	coverageTable, err := sfnt.parseCoverageTable(b[coverageOffset:])
	if err != nil {
		return nil, err
	}

	ligatureSetCount := r.ReadUint16()
	ligatures := make([][]ligature, ligatureSetCount)
	for i := 0; i < int(ligatureSetCount); i++ {
		ligatureSetOffset := r.ReadUint16()

		r2.Seek(uint32(ligatureSetOffset))
		ligatureCount := r2.ReadUint16()
		ligatures[i] = make([]ligature, ligatureCount)
		for j := 0; j < int(ligatureCount); j++ {
			ligatureOffset := r2.ReadUint16()

			r3.Seek(uint32(ligatureOffset))
			ligatures[i][j].ligatureGlyph = r3.ReadUint16()
			componentCount := r3.ReadUint16() - 1
			ligatures[i][j].componentGlyphIDs = make([]uint16, componentCount)
			for k := 0; k < int(componentCount); k++ {
				ligatures[i][j].componentGlyphIDs[k] = r3.ReadUint16()
			}
		}
	}
	return &ligatureSubstFormat1{
		coverageTable: coverageTable,
		ligatures:     ligatures,
	}, nil
}

////////////////////////////////////////////////////////////////

type gposgsubTable struct {
	scriptList
	featureList
	lookupList
	featureVariationsList

	tables []interface{}
}

func (table *gposgsubTable) GetLookups(script ScriptTag, language LanguageTag, features []FeatureTag) ([]interface{}, error) {
	var featureIndices []uint16
	if langSys, ok := table.scriptList.getLangSys(script, language); ok {
		featureIndices = append([]uint16{langSys.requiredFeatureIndex}, langSys.featureIndices...)
	}

	var lookupIndices []uint16
	for _, feature := range featureIndices {
		tag, lookups, err := table.featureList.get(feature)
		if err != nil {
			return nil, err
		}
		for _, selectedTag := range features {
			if selectedTag == tag {
				// insert to list and sort
			InsertLoop:
				for _, lookup := range lookups {
					for i, lookupIndex := range lookupIndices {
						if lookupIndex < lookup {
							lookupIndices = append(lookupIndices[:i], append([]uint16{lookup}, lookupIndices[i:]...)...)
						} else if lookupIndex == lookup {
							break InsertLoop
						}
					}
				}
				break
			}
		}
	}

	tables := make([]interface{}, len(lookupIndices))
	for i := 0; i < len(lookupIndices); i++ {
		tables[i] = table.tables[lookupIndices[i]]
	}
	return tables, nil
}

type subtableMap map[uint16]func([]byte) (interface{}, error)

func (sfnt *SFNT) parseGPOS() error {
	var err error
	subtableMap := subtableMap{
		1: sfnt.parseSinglePosTable,
		2: sfnt.parsePairPosTable,
	}
	sfnt.Gpos, err = sfnt.parseGPOSGSUB("GPOS", subtableMap)
	return err
}

func (sfnt *SFNT) parseGSUB() error {
	var err error
	subtableMap := subtableMap{
		1: sfnt.parseSingleSubstTable,
		2: sfnt.parseMultipleSubstTable,
		3: sfnt.parseAlternateSubstTable,
		4: sfnt.parseLigatureSubstTable,
	}
	sfnt.Gsub, err = sfnt.parseGPOSGSUB("GSUB", subtableMap)
	return err
}

func (sfnt *SFNT) parseGPOSGSUB(name string, subtableMap subtableMap) (*gposgsubTable, error) {
	b, ok := sfnt.Tables[name]
	if !ok {
		return nil, fmt.Errorf("%s: missing table", name)
	} else if len(b) < 10 {
		return nil, fmt.Errorf("%s: bad table", name)
	}

	table := &gposgsubTable{}
	r := NewBinaryReader(b)
	majorVersion := r.ReadUint16()
	minorVersion := r.ReadUint16()
	if majorVersion != 1 && minorVersion != 0 && minorVersion != 1 {
		return nil, fmt.Errorf("%s: bad version", name)
	}

	var err error
	scriptListOffset := r.ReadUint16()
	if len(b)-2 < int(scriptListOffset) {
		return nil, fmt.Errorf("%s: bad scriptList offset", name)
	}
	table.scriptList, err = sfnt.parseScriptList(b[scriptListOffset:])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	featureListOffset := r.ReadUint16()
	if len(b)-2 < int(featureListOffset) {
		return nil, fmt.Errorf("%s: bad featureList offset", name)
	}
	table.featureList = sfnt.parseFeatureList(b[featureListOffset:])

	lookupListOffset := r.ReadUint16()
	if len(b)-2 < int(lookupListOffset) {
		return nil, fmt.Errorf("%s: bad lookupList offset", name)
	}
	if lookupListOffset != 0 {
		table.lookupList = sfnt.parseLookupList(b[lookupListOffset:])
	}
	table.tables = make([]interface{}, len(table.lookupList))
	for j, lookup := range table.lookupList {
		if parseSubtable, ok := subtableMap[lookup.lookupType]; ok {
			tables := make([]interface{}, len(lookup.subtable))
			for i, data := range lookup.subtable {
				var err error
				tables[i], err = parseSubtable(data)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", name, err)
				}
			}
			table.tables[j] = tables
		} else if lookup.lookupType == 0 || 9 < lookup.lookupType {
			return nil, fmt.Errorf("%s: bad lookup table type", name)
		} else {
			//fmt.Printf("%s: lookup table type %d not supported\n", name, lookup.lookupType)
		}
	}

	var featureVariationsOffset uint32
	if minorVersion == 1 {
		featureVariationsOffset = r.ReadUint32()
		if len(b)-8 < int(featureVariationsOffset) {
			return nil, fmt.Errorf("%s: bad featureVariations offset", name)
		}
		table.featureVariationsList = featureVariationsList{b[featureVariationsOffset:]}
	}
	return table, nil
}

////////////////////////////////////////////////////////////////

type jsftTable struct {
}

func (sfnt *SFNT) parseJSFT() error {
	b, ok := sfnt.Tables["JSFT"]
	if !ok {
		return fmt.Errorf("JSFT: missing table")
	} else if len(b) < 32 {
		return fmt.Errorf("JSFT: bad table")
	}

	sfnt.Jsft = &jsftTable{}
	r := NewBinaryReader(b)
	majorVersion := r.ReadUint16()
	minorVersion := r.ReadUint16()
	if majorVersion != 1 && minorVersion != 0 {
		return fmt.Errorf("JSFT: bad version")
	}
	// TODO
	return nil
}
