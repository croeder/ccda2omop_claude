// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import "testing"

func TestNewVocabularyMapper(t *testing.T) {
	v := NewVocabularyMapper()
	if v == nil {
		t.Fatal("NewVocabularyMapper returned nil")
	}
	if v.genderConcepts == nil {
		t.Error("genderConcepts map is nil")
	}
	if v.raceConcepts == nil {
		t.Error("raceConcepts map is nil")
	}
	if v.ethnicityConcepts == nil {
		t.Error("ethnicityConcepts map is nil")
	}
	if v.visitTypeConcepts == nil {
		t.Error("visitTypeConcepts map is nil")
	}
}

func TestMapGender(t *testing.T) {
	v := NewVocabularyMapper()

	tests := []struct {
		name     string
		code     string
		expected int64
	}{
		{"male", "M", ConceptMale},
		{"female", "F", ConceptFemale},
		{"unknown", "UN", ConceptUnknown},
		{"unmapped", "X", ConceptUnknown},
		{"empty", "", ConceptUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.MapGender(tt.code)
			if result != tt.expected {
				t.Errorf("MapGender(%q) = %d, want %d", tt.code, result, tt.expected)
			}
		})
	}
}

func TestMapRace(t *testing.T) {
	v := NewVocabularyMapper()

	tests := []struct {
		name     string
		code     string
		expected int64
	}{
		{"white", "2106-3", ConceptWhite},
		{"black", "2054-5", ConceptBlackOrAfricanAmerican},
		{"asian", "2028-9", ConceptAsian},
		{"american indian", "1002-5", ConceptAmericanIndianOrAlaska},
		{"pacific islander", "2076-8", ConceptNativeHawaiianOrPacific},
		{"other", "2131-1", ConceptOtherRace},
		{"unmapped", "9999-9", ConceptUnknownRace},
		{"empty", "", ConceptUnknownRace},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.MapRace(tt.code)
			if result != tt.expected {
				t.Errorf("MapRace(%q) = %d, want %d", tt.code, result, tt.expected)
			}
		})
	}
}

func TestMapEthnicity(t *testing.T) {
	v := NewVocabularyMapper()

	tests := []struct {
		name     string
		code     string
		expected int64
	}{
		{"hispanic", "2135-2", ConceptHispanic},
		{"not hispanic", "2186-5", ConceptNotHispanic},
		{"unmapped", "9999-9", ConceptNoMapping},
		{"empty", "", ConceptNoMapping},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.MapEthnicity(tt.code)
			if result != tt.expected {
				t.Errorf("MapEthnicity(%q) = %d, want %d", tt.code, result, tt.expected)
			}
		})
	}
}

func TestMapVisitType(t *testing.T) {
	v := NewVocabularyMapper()

	tests := []struct {
		name     string
		code     string
		expected int64
	}{
		{"inpatient", "IMP", ConceptInpatient},
		{"ambulatory", "AMB", ConceptOutpatient},
		{"emergency", "EMER", ConceptEmergency},
		{"virtual", "VR", ConceptOffice},
		{"unmapped defaults to outpatient", "UNKNOWN", ConceptOutpatient},
		{"empty defaults to outpatient", "", ConceptOutpatient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.MapVisitType(tt.code)
			if result != tt.expected {
				t.Errorf("MapVisitType(%q) = %d, want %d", tt.code, result, tt.expected)
			}
		})
	}
}

func TestPlaceholderMappings(t *testing.T) {
	v := NewVocabularyMapper()

	// All these should return ConceptNoMapping (0) as placeholders
	t.Run("condition code", func(t *testing.T) {
		result := v.MapConditionCode("44054006", OIDSnomedCT)
		if result != ConceptNoMapping {
			t.Errorf("MapConditionCode() = %d, want %d", result, ConceptNoMapping)
		}
	})

	t.Run("drug code", func(t *testing.T) {
		result := v.MapDrugCode("860975", OIDRxNorm)
		if result != ConceptNoMapping {
			t.Errorf("MapDrugCode() = %d, want %d", result, ConceptNoMapping)
		}
	})

	t.Run("procedure code", func(t *testing.T) {
		result := v.MapProcedureCode("73761001", OIDSnomedCT)
		if result != ConceptNoMapping {
			t.Errorf("MapProcedureCode() = %d, want %d", result, ConceptNoMapping)
		}
	})

	t.Run("measurement code", func(t *testing.T) {
		result := v.MapMeasurementCode("8480-6", OIDLOINC)
		if result != ConceptNoMapping {
			t.Errorf("MapMeasurementCode() = %d, want %d", result, ConceptNoMapping)
		}
	})

	t.Run("observation code", func(t *testing.T) {
		result := v.MapObservationCode("72166-2", OIDLOINC)
		if result != ConceptNoMapping {
			t.Errorf("MapObservationCode() = %d, want %d", result, ConceptNoMapping)
		}
	})

	t.Run("device code", func(t *testing.T) {
		result := v.MapDeviceCode("706689003", OIDSnomedCT)
		if result != ConceptNoMapping {
			t.Errorf("MapDeviceCode() = %d, want %d", result, ConceptNoMapping)
		}
	})

	t.Run("unit code", func(t *testing.T) {
		result := v.MapUnitCode("mg")
		if result != ConceptNoMapping {
			t.Errorf("MapUnitCode() = %d, want %d", result, ConceptNoMapping)
		}
	})

	t.Run("route code", func(t *testing.T) {
		result := v.MapRouteCode("PO", "2.16.840.1.113883.5.112")
		if result != ConceptNoMapping {
			t.Errorf("MapRouteCode() = %d, want %d", result, ConceptNoMapping)
		}
	})
}

func TestGetCodeSystemName(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		expected string
	}{
		{"SNOMED-CT", OIDSnomedCT, "SNOMED-CT"},
		{"RxNorm", OIDRxNorm, "RxNorm"},
		{"LOINC", OIDLOINC, "LOINC"},
		{"ICD-10-CM", OIDICD10CM, "ICD-10-CM"},
		{"ICD-9-CM", OIDICD9CM, "ICD-9-CM"},
		{"CPT", OIDCPT, "CPT"},
		{"CVX", OIDCVX, "CVX"},
		{"unknown returns OID", "1.2.3.4.5", "1.2.3.4.5"},
		{"empty returns empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCodeSystemName(tt.oid)
			if result != tt.expected {
				t.Errorf("GetCodeSystemName(%q) = %q, want %q", tt.oid, result, tt.expected)
			}
		})
	}
}

func TestConceptConstants(t *testing.T) {
	// Verify key OMOP concept IDs are set correctly
	// These are real OMOP concept IDs
	if ConceptMale != 8507 {
		t.Errorf("ConceptMale = %d, want 8507", ConceptMale)
	}
	if ConceptFemale != 8532 {
		t.Errorf("ConceptFemale = %d, want 8532", ConceptFemale)
	}
	if ConceptInpatient != 9201 {
		t.Errorf("ConceptInpatient = %d, want 9201", ConceptInpatient)
	}
	if ConceptOutpatient != 9202 {
		t.Errorf("ConceptOutpatient = %d, want 9202", ConceptOutpatient)
	}
	if ConceptEmergency != 9203 {
		t.Errorf("ConceptEmergency = %d, want 9203", ConceptEmergency)
	}
}
