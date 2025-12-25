// Copyright 2025 Christophe Roeder. All rights reserved.

package ccda

import (
	"testing"
	"time"
)

func TestParseHL7Time(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "full datetime",
			input:    "20231215120000",
			expected: time.Date(2023, 12, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "datetime with minutes",
			input:    "202312151230",
			expected: time.Date(2023, 12, 15, 12, 30, 0, 0, time.UTC),
		},
		{
			name:     "date only",
			input:    "20231215",
			expected: time.Date(2023, 12, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "year and month",
			input:    "202312",
			expected: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "year only",
			input:    "2023",
			expected: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "with timezone Z",
			input:    "20231215120000Z",
			expected: time.Date(2023, 12, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "with timezone offset",
			input:    "20231215120000-0500",
			expected: time.Date(2023, 12, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "empty string",
			input:    "",
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHL7Time(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("parseHL7Time(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSampleDocument(t *testing.T) {
	doc, err := ParseFile("../../testdata/sample.xml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test patient parsing
	t.Run("patient demographics", func(t *testing.T) {
		if doc.Patient.ID != "123-45-6789" {
			t.Errorf("Patient.ID = %q, want %q", doc.Patient.ID, "123-45-6789")
		}
		if doc.Patient.Name.Given != "John Q" {
			t.Errorf("Patient.Name.Given = %q, want %q", doc.Patient.Name.Given, "John Q")
		}
		if doc.Patient.Name.Family != "Public" {
			t.Errorf("Patient.Name.Family = %q, want %q", doc.Patient.Name.Family, "Public")
		}
		if doc.Patient.Gender.Code != "M" {
			t.Errorf("Patient.Gender.Code = %q, want %q", doc.Patient.Gender.Code, "M")
		}
		expectedBirth := time.Date(1980, 5, 15, 0, 0, 0, 0, time.UTC)
		if !doc.Patient.BirthTime.Equal(expectedBirth) {
			t.Errorf("Patient.BirthTime = %v, want %v", doc.Patient.BirthTime, expectedBirth)
		}
		if doc.Patient.Race.Code != "2106-3" {
			t.Errorf("Patient.Race.Code = %q, want %q", doc.Patient.Race.Code, "2106-3")
		}
		if doc.Patient.Ethnicity.Code != "2186-5" {
			t.Errorf("Patient.Ethnicity.Code = %q, want %q", doc.Patient.Ethnicity.Code, "2186-5")
		}
	})

	// Test encounters
	t.Run("encounters", func(t *testing.T) {
		if len(doc.Encounters) != 1 {
			t.Fatalf("len(Encounters) = %d, want 1", len(doc.Encounters))
		}
		enc := doc.Encounters[0]
		if enc.ID != "ENC-001" {
			t.Errorf("Encounter.ID = %q, want %q", enc.ID, "ENC-001")
		}
		if enc.Code.Code != "AMB" {
			t.Errorf("Encounter.Code.Code = %q, want %q", enc.Code.Code, "AMB")
		}
		if enc.Performer != "Jane Doctor" {
			t.Errorf("Encounter.Performer = %q, want %q", enc.Performer, "Jane Doctor")
		}
	})

	// Test problems
	t.Run("problems", func(t *testing.T) {
		if len(doc.Problems) != 2 {
			t.Fatalf("len(Problems) = %d, want 2", len(doc.Problems))
		}
		// Check first problem (diabetes)
		prob := doc.Problems[0]
		if prob.Code.Code != "44054006" {
			t.Errorf("Problem[0].Code.Code = %q, want %q", prob.Code.Code, "44054006")
		}
		if prob.Code.DisplayName != "Type 2 Diabetes Mellitus" {
			t.Errorf("Problem[0].Code.DisplayName = %q, want %q", prob.Code.DisplayName, "Type 2 Diabetes Mellitus")
		}
	})

	// Test medications
	t.Run("medications", func(t *testing.T) {
		if len(doc.Medications) != 2 {
			t.Fatalf("len(Medications) = %d, want 2", len(doc.Medications))
		}
		med := doc.Medications[0]
		if med.Code.Code != "860975" {
			t.Errorf("Medication[0].Code.Code = %q, want %q", med.Code.Code, "860975")
		}
		if med.DoseQuantity.Value != 500 {
			t.Errorf("Medication[0].DoseQuantity.Value = %v, want %v", med.DoseQuantity.Value, 500)
		}
		if med.DoseQuantity.Unit != "mg" {
			t.Errorf("Medication[0].DoseQuantity.Unit = %q, want %q", med.DoseQuantity.Unit, "mg")
		}
		if med.RouteCode.Code != "PO" {
			t.Errorf("Medication[0].RouteCode.Code = %q, want %q", med.RouteCode.Code, "PO")
		}
	})

	// Test vital signs
	t.Run("vital signs", func(t *testing.T) {
		if len(doc.VitalSigns) != 4 {
			t.Fatalf("len(VitalSigns) = %d, want 4", len(doc.VitalSigns))
		}
		// Find systolic BP
		var systolic *VitalSign
		for i := range doc.VitalSigns {
			if doc.VitalSigns[i].Code.Code == "8480-6" {
				systolic = &doc.VitalSigns[i]
				break
			}
		}
		if systolic == nil {
			t.Fatal("Systolic BP not found")
		}
		if systolic.Value != 128 {
			t.Errorf("Systolic.Value = %v, want %v", systolic.Value, 128)
		}
		if systolic.Unit != "mm[Hg]" {
			t.Errorf("Systolic.Unit = %q, want %q", systolic.Unit, "mm[Hg]")
		}
	})

	// Test lab results
	t.Run("lab results", func(t *testing.T) {
		if len(doc.LabResults) != 2 {
			t.Fatalf("len(LabResults) = %d, want 2", len(doc.LabResults))
		}
		// Find HbA1c
		var hba1c *LabResult
		for i := range doc.LabResults {
			if doc.LabResults[i].Code.Code == "4548-4" {
				hba1c = &doc.LabResults[i]
				break
			}
		}
		if hba1c == nil {
			t.Fatal("HbA1c not found")
		}
		if hba1c.Value != 7.2 {
			t.Errorf("HbA1c.Value = %v, want %v", hba1c.Value, 7.2)
		}
		if hba1c.Unit != "%" {
			t.Errorf("HbA1c.Unit = %q, want %q", hba1c.Unit, "%")
		}
	})

	// Test allergies
	t.Run("allergies", func(t *testing.T) {
		if len(doc.Allergies) != 1 {
			t.Fatalf("len(Allergies) = %d, want 1", len(doc.Allergies))
		}
		allergy := doc.Allergies[0]
		if allergy.Substance.Code != "7980" {
			t.Errorf("Allergy.Substance.Code = %q, want %q", allergy.Substance.Code, "7980")
		}
		if allergy.Substance.DisplayName != "Penicillin" {
			t.Errorf("Allergy.Substance.DisplayName = %q, want %q", allergy.Substance.DisplayName, "Penicillin")
		}
	})

	// Test immunizations
	t.Run("immunizations", func(t *testing.T) {
		if len(doc.Immunizations) != 1 {
			t.Fatalf("len(Immunizations) = %d, want 1", len(doc.Immunizations))
		}
		imm := doc.Immunizations[0]
		if imm.Code.Code != "141" {
			t.Errorf("Immunization.Code.Code = %q, want %q", imm.Code.Code, "141")
		}
		if imm.LotNumber != "LOT-2023-FLU-456" {
			t.Errorf("Immunization.LotNumber = %q, want %q", imm.LotNumber, "LOT-2023-FLU-456")
		}
		if imm.DoseQuantity.Value != 0.5 {
			t.Errorf("Immunization.DoseQuantity.Value = %v, want %v", imm.DoseQuantity.Value, 0.5)
		}
	})

	// Test procedures
	t.Run("procedures", func(t *testing.T) {
		if len(doc.Procedures) != 1 {
			t.Fatalf("len(Procedures) = %d, want 1", len(doc.Procedures))
		}
		proc := doc.Procedures[0]
		if proc.Code.Code != "73761001" {
			t.Errorf("Procedure.Code.Code = %q, want %q", proc.Code.Code, "73761001")
		}
		if proc.Code.DisplayName != "Colonoscopy" {
			t.Errorf("Procedure.Code.DisplayName = %q, want %q", proc.Code.DisplayName, "Colonoscopy")
		}
	})

	// Test devices
	t.Run("devices", func(t *testing.T) {
		if len(doc.Devices) != 1 {
			t.Fatalf("len(Devices) = %d, want 1", len(doc.Devices))
		}
		dev := doc.Devices[0]
		if dev.Code.Code != "706689003" {
			t.Errorf("Device.Code.Code = %q, want %q", dev.Code.Code, "706689003")
		}
		if dev.UDI != "(01)00884838049032" {
			t.Errorf("Device.UDI = %q, want %q", dev.UDI, "(01)00884838049032")
		}
	})

	// Test social history observations
	t.Run("observations", func(t *testing.T) {
		if len(doc.Observations) != 1 {
			t.Fatalf("len(Observations) = %d, want 1", len(doc.Observations))
		}
		obs := doc.Observations[0]
		if obs.Code.Code != "72166-2" {
			t.Errorf("Observation.Code.Code = %q, want %q", obs.Code.Code, "72166-2")
		}
		if obs.Value.DisplayName != "Former smoker" {
			t.Errorf("Observation.Value.DisplayName = %q, want %q", obs.Value.DisplayName, "Former smoker")
		}
	})
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("nonexistent.xml")
	if err == nil {
		t.Error("ParseFile should return error for nonexistent file")
	}
}

func TestParseInvalidXML(t *testing.T) {
	invalidXML := []byte(`<?xml version="1.0"?><ClinicalDocument><invalid>`)
	_, err := Parse(invalidXML)
	if err == nil {
		t.Error("Parse should return error for invalid XML")
	}
}

func TestParseEmptyDocument(t *testing.T) {
	emptyDoc := []byte(`<?xml version="1.0"?><ClinicalDocument xmlns="urn:hl7-org:v3"></ClinicalDocument>`)
	doc, err := Parse(emptyDoc)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have empty sections
	if len(doc.Encounters) != 0 {
		t.Errorf("len(Encounters) = %d, want 0", len(doc.Encounters))
	}
	if len(doc.Problems) != 0 {
		t.Errorf("len(Problems) = %d, want 0", len(doc.Problems))
	}
}

func TestParsePatientAddress(t *testing.T) {
	doc, err := ParseFile("../../testdata/sample.xml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	addr := doc.Patient.Address
	if len(addr.StreetAddress) == 0 || addr.StreetAddress[0] != "123 Main Street" {
		t.Errorf("StreetAddress = %v, want [123 Main Street]", addr.StreetAddress)
	}
	if addr.City != "Anytown" {
		t.Errorf("City = %q, want %q", addr.City, "Anytown")
	}
	if addr.State != "CA" {
		t.Errorf("State = %q, want %q", addr.State, "CA")
	}
	if addr.PostalCode != "90210" {
		t.Errorf("PostalCode = %q, want %q", addr.PostalCode, "90210")
	}
}

func TestParseAuthor(t *testing.T) {
	doc, err := ParseFile("../../testdata/sample.xml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if doc.Author.Name.Given != "Jane" {
		t.Errorf("Author.Name.Given = %q, want %q", doc.Author.Name.Given, "Jane")
	}
	if doc.Author.Name.Family != "Doctor" {
		t.Errorf("Author.Name.Family = %q, want %q", doc.Author.Name.Family, "Doctor")
	}
	if doc.Author.Organization != "Sample Health System" {
		t.Errorf("Author.Organization = %q, want %q", doc.Author.Organization, "Sample Health System")
	}
}

func TestParseCustodian(t *testing.T) {
	doc, err := ParseFile("../../testdata/sample.xml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if doc.Custodian.Name != "Sample Health System" {
		t.Errorf("Custodian.Name = %q, want %q", doc.Custodian.Name, "Sample Health System")
	}
}

func TestGetSectionTemplateOID(t *testing.T) {
	// Test with sample document sections
	doc, err := ParseFile("../../testdata/sample.xml")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Verify sections were found by checking data
	if len(doc.Encounters) == 0 {
		t.Error("Expected encounters section to be parsed")
	}
	if len(doc.Problems) == 0 {
		t.Error("Expected problems section to be parsed")
	}
	if len(doc.Medications) == 0 {
		t.Error("Expected medications section to be parsed")
	}
}
