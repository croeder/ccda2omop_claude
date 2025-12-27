// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import "testing"

func TestOIDToVocabularyID(t *testing.T) {
	tests := []struct {
		name     string
		oid      string
		expected string
	}{
		// Standard medical vocabularies
		{"SNOMED-CT", "2.16.840.1.113883.6.96", "SNOMED"},
		{"RxNorm", "2.16.840.1.113883.6.88", "RxNorm"},
		{"LOINC", "2.16.840.1.113883.6.1", "LOINC"},
		{"ICD-10-CM", "2.16.840.1.113883.6.90", "ICD10CM"},
		{"ICD-9-CM", "2.16.840.1.113883.6.103", "ICD9CM"},
		{"CPT-4", "2.16.840.1.113883.6.12", "CPT4"},
		{"HCPCS", "2.16.840.1.113883.6.14", "HCPCS"},
		{"HCPCS via CDT OID", "2.16.840.1.113883.6.13", "HCPCS"},
		{"CVX", "2.16.840.1.113883.12.292", "CVX"},
		{"CVX alternate", "2.16.840.1.113883.6.59", "CVX"},
		{"NDC", "2.16.840.1.113883.6.69", "NDC"},
		{"UNII", "2.16.840.1.113883.4.9", "UNII"},
		{"NDFRT", "2.16.840.1.113883.3.26.1.5", "NDFRT"},

		// Newly added OIDs
		{"NCI Thesaurus", "2.16.840.1.113883.3.26.1.1", "NCI"},
		{"HL7 ActCode", "2.16.840.1.113883.5.4", "ActCode"},
		{"HL7 RouteOfAdministration", "2.16.840.1.113883.5.112", "RouteOfAdministration"},

		// Direct vocabulary names
		{"SNOMED name", "SNOMED", "SNOMED"},
		{"SNOMED CT name", "SNOMED CT", "SNOMED"},
		{"SNOMEDCT name", "SNOMEDCT", "SNOMED"},
		{"RxNorm name", "RxNorm", "RxNorm"},
		{"LOINC name", "LOINC", "LOINC"},
		{"ICD10CM name", "ICD10CM", "ICD10CM"},
		{"ICD-10-CM name", "ICD-10-CM", "ICD10CM"},
		{"ICD10 name", "ICD10", "ICD10CM"},
		{"ICD9CM name", "ICD9CM", "ICD9CM"},
		{"ICD-9-CM name", "ICD-9-CM", "ICD9CM"},
		{"ICD9 name", "ICD9", "ICD9CM"},
		{"CPT4 name", "CPT4", "CPT4"},
		{"CPT name", "CPT", "CPT4"},
		{"CPT-4 name", "CPT-4", "CPT4"},
		{"HCPCS name", "HCPCS", "HCPCS"},
		{"CVX name", "CVX", "CVX"},
		{"NDC name", "NDC", "NDC"},
		{"UNII name", "UNII", "UNII"},
		{"NDFRT name", "NDFRT", "NDFRT"},
		{"NDF-RT name", "NDF-RT", "NDFRT"},
		{"NCI name", "NCI", "NCI"},
		{"NCIt name", "NCIt", "NCI"},
		{"ActCode name", "ActCode", "ActCode"},
		{"ASSERTION name", "ASSERTION", "ActCode"},
		{"RouteOfAdministration name", "RouteOfAdministration", "RouteOfAdministration"},

		// Unknown OIDs return empty string
		{"unknown OID", "1.2.3.4.5", ""},
		{"empty string", "", ""},
		{"random string", "foobar", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := OIDToVocabularyID(tt.oid)
			if result != tt.expected {
				t.Errorf("OIDToVocabularyID(%q) = %q, want %q", tt.oid, result, tt.expected)
			}
		})
	}
}

func TestNewVocabLoader(t *testing.T) {
	vl := NewVocabLoader()
	if vl == nil {
		t.Fatal("NewVocabLoader returned nil")
	}
	if vl.conceptIndex == nil {
		t.Error("conceptIndex map is nil")
	}
	if vl.conceptByID == nil {
		t.Error("conceptByID map is nil")
	}
	if vl.mapsTo == nil {
		t.Error("mapsTo map is nil")
	}
	if vl.relevantVocabs == nil {
		t.Error("relevantVocabs map is nil")
	}
}

func TestRelevantVocabs(t *testing.T) {
	vl := NewVocabLoader()

	// Check that all expected vocabularies are in the relevant vocabs map
	expectedVocabs := []string{
		"SNOMED", "RxNorm", "LOINC", "ICD10CM", "ICD9CM",
		"CPT4", "HCPCS", "CVX", "NDC", "UNII", "NDFRT",
		"NCI", "ActCode", "RouteOfAdministration",
		"Gender", "Race", "Ethnicity", "UCUM", "Visit",
	}

	for _, vocab := range expectedVocabs {
		if !vl.relevantVocabs[vocab] {
			t.Errorf("relevantVocabs missing %q", vocab)
		}
	}
}

func TestConceptKey(t *testing.T) {
	tests := []struct {
		vocabID  string
		code     string
		expected string
	}{
		{"SNOMED", "44054006", "SNOMED|44054006"},
		{"RxNorm", "860975", "RxNorm|860975"},
		{"LOINC", "8480-6", "LOINC|8480-6"},
		{"", "code", "|code"},
		{"vocab", "", "vocab|"},
	}

	for _, tt := range tests {
		t.Run(tt.vocabID+"|"+tt.code, func(t *testing.T) {
			result := conceptKey(tt.vocabID, tt.code)
			if result != tt.expected {
				t.Errorf("conceptKey(%q, %q) = %q, want %q", tt.vocabID, tt.code, result, tt.expected)
			}
		})
	}
}

func TestLookupConceptEmpty(t *testing.T) {
	vl := NewVocabLoader()

	// Without loading any concepts, lookups should return nil
	result := vl.LookupConcept("SNOMED", "44054006")
	if result != nil {
		t.Errorf("LookupConcept on empty loader returned %v, want nil", result)
	}

	resultByID := vl.LookupConceptByID(44054006)
	if resultByID != nil {
		t.Errorf("LookupConceptByID on empty loader returned %v, want nil", resultByID)
	}
}

func TestGetStandardConceptIDEmpty(t *testing.T) {
	vl := NewVocabLoader()

	// Without loading any concepts, should return 0
	result := vl.GetStandardConceptID("SNOMED", "44054006")
	if result != 0 {
		t.Errorf("GetStandardConceptID on empty loader returned %d, want 0", result)
	}
}

func TestGetConceptDomainEmpty(t *testing.T) {
	vl := NewVocabLoader()

	// Without loading any concepts, should return empty string
	result := vl.GetConceptDomain(44054006)
	if result != "" {
		t.Errorf("GetConceptDomain on empty loader returned %q, want empty string", result)
	}
}
