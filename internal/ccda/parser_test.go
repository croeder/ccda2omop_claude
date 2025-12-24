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

func TestParseCodedValue(t *testing.T) {
	input := xmlCode{
		Code:           "12345",
		CodeSystem:     "2.16.840.1.113883.6.96",
		CodeSystemName: "SNOMED CT",
		DisplayName:    "Test Condition",
	}

	result := parseCodedValue(input)

	if result.Code != "12345" {
		t.Errorf("Code = %q, want %q", result.Code, "12345")
	}
	if result.CodeSystem != "2.16.840.1.113883.6.96" {
		t.Errorf("CodeSystem = %q, want %q", result.CodeSystem, "2.16.840.1.113883.6.96")
	}
	if result.CodeSystemName != "SNOMED CT" {
		t.Errorf("CodeSystemName = %q, want %q", result.CodeSystemName, "SNOMED CT")
	}
	if result.DisplayName != "Test Condition" {
		t.Errorf("DisplayName = %q, want %q", result.DisplayName, "Test Condition")
	}
}

func TestParseQuantity(t *testing.T) {
	tests := []struct {
		name          string
		input         xmlQuantity
		expectedValue float64
		expectedUnit  string
	}{
		{
			name:          "integer value",
			input:         xmlQuantity{Value: "500", Unit: "mg"},
			expectedValue: 500,
			expectedUnit:  "mg",
		},
		{
			name:          "decimal value",
			input:         xmlQuantity{Value: "0.5", Unit: "mL"},
			expectedValue: 0.5,
			expectedUnit:  "mL",
		},
		{
			name:          "empty value",
			input:         xmlQuantity{Value: "", Unit: "mg"},
			expectedValue: 0,
			expectedUnit:  "mg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseQuantity(tt.input)
			if result.Value != tt.expectedValue {
				t.Errorf("Value = %v, want %v", result.Value, tt.expectedValue)
			}
			if result.Unit != tt.expectedUnit {
				t.Errorf("Unit = %q, want %q", result.Unit, tt.expectedUnit)
			}
		})
	}
}

func TestGetIDString(t *testing.T) {
	tests := []struct {
		name     string
		input    []xmlID
		expected string
	}{
		{
			name:     "with extension",
			input:    []xmlID{{Root: "2.16.840.1.113883.19", Extension: "12345"}},
			expected: "12345",
		},
		{
			name:     "root only",
			input:    []xmlID{{Root: "2.16.840.1.113883.19", Extension: ""}},
			expected: "2.16.840.1.113883.19",
		},
		{
			name:     "empty slice",
			input:    []xmlID{},
			expected: "",
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIDString(tt.input)
			if result != tt.expected {
				t.Errorf("getIDString() = %q, want %q", result, tt.expected)
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

func TestParseAddress(t *testing.T) {
	input := xmlAddr{
		StreetAddressLine: []string{"123 Main St", "Apt 4"},
		City:              "Anytown",
		State:             "CA",
		PostalCode:        "90210",
		Country:           "US",
	}

	result := parseAddress(input)

	if len(result.StreetAddress) != 2 {
		t.Errorf("len(StreetAddress) = %d, want 2", len(result.StreetAddress))
	}
	if result.City != "Anytown" {
		t.Errorf("City = %q, want %q", result.City, "Anytown")
	}
	if result.State != "CA" {
		t.Errorf("State = %q, want %q", result.State, "CA")
	}
	if result.PostalCode != "90210" {
		t.Errorf("PostalCode = %q, want %q", result.PostalCode, "90210")
	}
	if result.Country != "US" {
		t.Errorf("Country = %q, want %q", result.Country, "US")
	}
}

func TestParseEffectiveTime(t *testing.T) {
	tests := []struct {
		name        string
		input       xmlEffectiveTime
		wantLow     time.Time
		wantHigh    time.Time
		wantValue   time.Time
	}{
		{
			name: "low and high",
			input: xmlEffectiveTime{
				Low:  xmlValue{Value: "20230601"},
				High: xmlValue{Value: "20230630"},
			},
			wantLow:  time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			wantHigh: time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "value only",
			input: xmlEffectiveTime{
				Value: "20231201100000",
			},
			wantValue: time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			name: "low only",
			input: xmlEffectiveTime{
				Low: xmlValue{Value: "20230101"},
			},
			wantLow: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEffectiveTime(tt.input)
			if !result.Low.Equal(tt.wantLow) {
				t.Errorf("Low = %v, want %v", result.Low, tt.wantLow)
			}
			if !result.High.Equal(tt.wantHigh) {
				t.Errorf("High = %v, want %v", result.High, tt.wantHigh)
			}
			if !result.Value.Equal(tt.wantValue) {
				t.Errorf("Value = %v, want %v", result.Value, tt.wantValue)
			}
		})
	}
}

func TestGetSectionTemplateOID(t *testing.T) {
	tests := []struct {
		name     string
		input    []xmlTemplateID
		expected string
	}{
		{
			name: "encounters section",
			input: []xmlTemplateID{
				{Root: "2.16.840.1.113883.10.20.22.2.22.1"},
			},
			expected: OIDEncountersEntriesReq,
		},
		{
			name: "problems section",
			input: []xmlTemplateID{
				{Root: "2.16.840.1.113883.10.20.22.2.5.1"},
			},
			expected: OIDProblemsEntriesReq,
		},
		{
			name: "unknown section",
			input: []xmlTemplateID{
				{Root: "1.2.3.4.5"},
			},
			expected: "",
		},
		{
			name: "multiple templates finds first match",
			input: []xmlTemplateID{
				{Root: "1.2.3.4.5"},
				{Root: "2.16.840.1.113883.10.20.22.2.1.1"},
			},
			expected: OIDMedicationsEntriesReq,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSectionTemplateOID(tt.input)
			if result != tt.expected {
				t.Errorf("getSectionTemplateOID() = %q, want %q", result, tt.expected)
			}
		})
	}
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
