package mapper

// VocabularyMapper provides concept mappings for OMOP using loaded vocabulary tables.
type VocabularyMapper struct {
	// Placeholder concept IDs for common codes (used as fallback)
	genderConcepts    map[string]int64
	raceConcepts      map[string]int64
	ethnicityConcepts map[string]int64
	visitTypeConcepts map[string]int64

	// Vocabulary loader for OMOP concept lookups
	vocabLoader *VocabLoader
}

// Code system OIDs
const (
	OIDSnomedCT     = "2.16.840.1.113883.6.96"
	OIDRxNorm       = "2.16.840.1.113883.6.88"
	OIDLOINC        = "2.16.840.1.113883.6.1"
	OIDICD10CM      = "2.16.840.1.113883.6.90"
	OIDICD9CM       = "2.16.840.1.113883.6.103"
	OIDCPT          = "2.16.840.1.113883.6.12"
	OIDCVX          = "2.16.840.1.113883.12.292"
	OIDAdminGender  = "2.16.840.1.113883.5.1"
	OIDRaceEthnicity = "2.16.840.1.113883.6.238"
)

// OMOP Standard concept IDs (placeholders - real values from vocabulary)
const (
	// Gender concepts
	ConceptMale    int64 = 8507
	ConceptFemale  int64 = 8532
	ConceptUnknown int64 = 0

	// Race concepts (placeholder values)
	ConceptWhite                   int64 = 8527
	ConceptBlackOrAfricanAmerican  int64 = 8516
	ConceptAsian                   int64 = 8515
	ConceptAmericanIndianOrAlaska  int64 = 8657
	ConceptNativeHawaiianOrPacific int64 = 8557
	ConceptOtherRace               int64 = 8522
	ConceptUnknownRace             int64 = 0

	// Ethnicity concepts
	ConceptHispanic    int64 = 38003563
	ConceptNotHispanic int64 = 38003564

	// Visit type concepts
	ConceptInpatient  int64 = 9201
	ConceptOutpatient int64 = 9202
	ConceptEmergency  int64 = 9203
	ConceptOffice     int64 = 581477

	// Type concepts (how data was recorded)
	ConceptEHREncounter    int64 = 32817 // EHR encounter record
	ConceptEHRProblemList  int64 = 32817
	ConceptEHRPrescription int64 = 32817
	ConceptEHRProcedure    int64 = 32817
	ConceptEHRObservation  int64 = 32817

	// Placeholder for unmapped concepts
	ConceptNoMapping int64 = 0
)

// NewVocabularyMapper creates a new vocabulary mapper with placeholder mappings
func NewVocabularyMapper() *VocabularyMapper {
	return NewVocabularyMapperWithLoader(nil)
}

// NewVocabularyMapperWithLoader creates a vocabulary mapper with an optional vocab loader
func NewVocabularyMapperWithLoader(loader *VocabLoader) *VocabularyMapper {
	v := &VocabularyMapper{
		genderConcepts:    make(map[string]int64),
		raceConcepts:      make(map[string]int64),
		ethnicityConcepts: make(map[string]int64),
		visitTypeConcepts: make(map[string]int64),
		vocabLoader:       loader,
	}

	// Gender mappings (HL7 AdministrativeGender) - fallback values
	v.genderConcepts["M"] = ConceptMale
	v.genderConcepts["F"] = ConceptFemale
	v.genderConcepts["UN"] = ConceptUnknown

	// Race mappings (CDC Race and Ethnicity) - fallback values
	v.raceConcepts["2106-3"] = ConceptWhite                   // White
	v.raceConcepts["2054-5"] = ConceptBlackOrAfricanAmerican  // Black or African American
	v.raceConcepts["2028-9"] = ConceptAsian                   // Asian
	v.raceConcepts["1002-5"] = ConceptAmericanIndianOrAlaska  // American Indian or Alaska Native
	v.raceConcepts["2076-8"] = ConceptNativeHawaiianOrPacific // Native Hawaiian or Pacific Islander
	v.raceConcepts["2131-1"] = ConceptOtherRace               // Other Race

	// Ethnicity mappings - fallback values
	v.ethnicityConcepts["2135-2"] = ConceptHispanic    // Hispanic or Latino
	v.ethnicityConcepts["2186-5"] = ConceptNotHispanic // Not Hispanic or Latino

	// Visit type mappings (encounter class codes) - fallback values
	v.visitTypeConcepts["IMP"] = ConceptInpatient  // Inpatient
	v.visitTypeConcepts["AMB"] = ConceptOutpatient // Ambulatory
	v.visitTypeConcepts["EMER"] = ConceptEmergency // Emergency
	v.visitTypeConcepts["VR"] = ConceptOffice      // Virtual (map to office)

	return v
}

// MapGender maps a gender code to an OMOP concept ID
func (v *VocabularyMapper) MapGender(code string) int64 {
	if id, ok := v.genderConcepts[code]; ok {
		return id
	}
	return ConceptUnknown
}

// MapRace maps a race code to an OMOP concept ID
func (v *VocabularyMapper) MapRace(code string) int64 {
	if id, ok := v.raceConcepts[code]; ok {
		return id
	}
	return ConceptUnknownRace
}

// MapEthnicity maps an ethnicity code to an OMOP concept ID
func (v *VocabularyMapper) MapEthnicity(code string) int64 {
	if id, ok := v.ethnicityConcepts[code]; ok {
		return id
	}
	return ConceptNoMapping
}

// MapVisitType maps an encounter class code to an OMOP visit concept ID
func (v *VocabularyMapper) MapVisitType(classCode string) int64 {
	if id, ok := v.visitTypeConcepts[classCode]; ok {
		return id
	}
	return ConceptOutpatient // Default to outpatient
}

// MapConditionCode maps a condition code to an OMOP concept ID (first match)
func (v *VocabularyMapper) MapConditionCode(code, codeSystem string) int64 {
	ids := v.MapConditionCodes(code, codeSystem)
	if len(ids) > 0 {
		return ids[0]
	}
	return ConceptNoMapping
}

// MapConditionCodes maps a condition code to all matching OMOP concept IDs
func (v *VocabularyMapper) MapConditionCodes(code, codeSystem string) []int64 {
	if v.vocabLoader == nil || code == "" {
		return nil
	}

	vocabID := OIDToVocabularyID(codeSystem)
	if vocabID == "" {
		return nil
	}

	return v.vocabLoader.GetStandardConceptIDs(vocabID, code)
}

// MapDrugCode maps a drug code (RxNorm, NDC, CVX) to an OMOP concept ID (first match)
func (v *VocabularyMapper) MapDrugCode(code, codeSystem string) int64 {
	ids := v.MapDrugCodes(code, codeSystem)
	if len(ids) > 0 {
		return ids[0]
	}
	return ConceptNoMapping
}

// MapDrugCodes maps a drug code to all matching OMOP concept IDs
func (v *VocabularyMapper) MapDrugCodes(code, codeSystem string) []int64 {
	if v.vocabLoader == nil || code == "" {
		return nil
	}

	vocabID := OIDToVocabularyID(codeSystem)
	if vocabID == "" {
		return nil
	}

	return v.vocabLoader.GetStandardConceptIDs(vocabID, code)
}

// MapProcedureCode maps a procedure code (CPT, HCPCS, SNOMED) to an OMOP concept ID (first match)
func (v *VocabularyMapper) MapProcedureCode(code, codeSystem string) int64 {
	ids := v.MapProcedureCodes(code, codeSystem)
	if len(ids) > 0 {
		return ids[0]
	}
	return ConceptNoMapping
}

// MapProcedureCodes maps a procedure code to all matching OMOP concept IDs
func (v *VocabularyMapper) MapProcedureCodes(code, codeSystem string) []int64 {
	if v.vocabLoader == nil || code == "" {
		return nil
	}

	vocabID := OIDToVocabularyID(codeSystem)
	if vocabID == "" {
		return nil
	}

	return v.vocabLoader.GetStandardConceptIDs(vocabID, code)
}

// MapMeasurementCode maps a measurement code (LOINC) to an OMOP concept ID (first match)
func (v *VocabularyMapper) MapMeasurementCode(code, codeSystem string) int64 {
	ids := v.MapMeasurementCodes(code, codeSystem)
	if len(ids) > 0 {
		return ids[0]
	}
	return ConceptNoMapping
}

// MapMeasurementCodes maps a measurement code to all matching OMOP concept IDs
func (v *VocabularyMapper) MapMeasurementCodes(code, codeSystem string) []int64 {
	if v.vocabLoader == nil || code == "" {
		return nil
	}

	vocabID := OIDToVocabularyID(codeSystem)
	if vocabID == "" {
		return nil
	}

	return v.vocabLoader.GetStandardConceptIDs(vocabID, code)
}

// MapObservationCode maps an observation code to an OMOP concept ID (first match)
func (v *VocabularyMapper) MapObservationCode(code, codeSystem string) int64 {
	ids := v.MapObservationCodes(code, codeSystem)
	if len(ids) > 0 {
		return ids[0]
	}
	return ConceptNoMapping
}

// MapObservationCodes maps an observation code to all matching OMOP concept IDs
func (v *VocabularyMapper) MapObservationCodes(code, codeSystem string) []int64 {
	if v.vocabLoader == nil || code == "" {
		return nil
	}

	vocabID := OIDToVocabularyID(codeSystem)
	if vocabID == "" {
		return nil
	}

	return v.vocabLoader.GetStandardConceptIDs(vocabID, code)
}

// MapDeviceCode maps a device code to an OMOP concept ID (first match)
func (v *VocabularyMapper) MapDeviceCode(code, codeSystem string) int64 {
	ids := v.MapDeviceCodes(code, codeSystem)
	if len(ids) > 0 {
		return ids[0]
	}
	return ConceptNoMapping
}

// MapDeviceCodes maps a device code to all matching OMOP concept IDs
func (v *VocabularyMapper) MapDeviceCodes(code, codeSystem string) []int64 {
	if v.vocabLoader == nil || code == "" {
		return nil
	}

	vocabID := OIDToVocabularyID(codeSystem)
	if vocabID == "" {
		return nil
	}

	return v.vocabLoader.GetStandardConceptIDs(vocabID, code)
}

// MapUnitCode maps a unit code (UCUM) to an OMOP concept ID
func (v *VocabularyMapper) MapUnitCode(unit string) int64 {
	if v.vocabLoader == nil || unit == "" {
		return ConceptNoMapping
	}

	// Units are typically UCUM codes
	return v.vocabLoader.GetStandardConceptID("UCUM", unit)
}

// MapRouteCode maps a route code to an OMOP concept ID
func (v *VocabularyMapper) MapRouteCode(code, codeSystem string) int64 {
	if v.vocabLoader == nil || code == "" {
		return ConceptNoMapping
	}

	vocabID := OIDToVocabularyID(codeSystem)
	if vocabID == "" {
		// Try SNOMED as default for route codes
		vocabID = "SNOMED"
	}

	return v.vocabLoader.GetStandardConceptID(vocabID, code)
}

// GetCodeSystemName returns a human-readable name for a code system OID
func GetCodeSystemName(oid string) string {
	switch oid {
	case OIDSnomedCT:
		return "SNOMED-CT"
	case OIDRxNorm:
		return "RxNorm"
	case OIDLOINC:
		return "LOINC"
	case OIDICD10CM:
		return "ICD-10-CM"
	case OIDICD9CM:
		return "ICD-9-CM"
	case OIDCPT:
		return "CPT"
	case OIDCVX:
		return "CVX"
	default:
		return oid
	}
}
