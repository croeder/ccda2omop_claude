// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import (
	"testing"
	"time"

	"github.com/ccda2omop/internal/ccda"
)

func TestNew(t *testing.T) {
	m := New(false)
	if m == nil {
		t.Fatal("New returned nil")
	}
	if m.vocab == nil {
		t.Error("vocab is nil")
	}
	if m.verbose != false {
		t.Error("verbose should be false")
	}

	m2 := New(true)
	if m2.verbose != true {
		t.Error("verbose should be true")
	}
}

func TestMapDocument(t *testing.T) {
	m := New(false)

	doc := &ccda.Document{
		Patient: ccda.Patient{
			ID: "12345",
			Name: ccda.Name{
				Given:  "John",
				Family: "Doe",
			},
			BirthTime: time.Date(1980, 5, 15, 0, 0, 0, 0, time.UTC),
			Gender: ccda.CodedValue{
				Code:        "M",
				DisplayName: "Male",
			},
			Race: ccda.CodedValue{
				Code:        "2106-3",
				DisplayName: "White",
			},
			Ethnicity: ccda.CodedValue{
				Code:        "2186-5",
				DisplayName: "Not Hispanic or Latino",
			},
		},
		Encounters: []ccda.Encounter{
			{
				ID: "ENC-001",
				Code: ccda.CodedValue{
					Code:        "AMB",
					DisplayName: "Ambulatory",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low:  time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
					High: time.Date(2023, 12, 1, 11, 0, 0, 0, time.UTC),
				},
			},
		},
		Problems: []ccda.Problem{
			{
				ID: "PROB-001",
				Code: ccda.CodedValue{
					Code:        "44054006",
					CodeSystem:  "2.16.840.1.113883.6.96",
					DisplayName: "Type 2 Diabetes",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		Medications: []ccda.Medication{
			{
				ID: "MED-001",
				Code: ccda.CodedValue{
					Code:        "860975",
					DisplayName: "Metformin 500mg",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
				},
				DoseQuantity: ccda.Quantity{
					Value: 500,
					Unit:  "mg",
				},
				RouteCode: ccda.CodedValue{
					Code:        "PO",
					DisplayName: "Oral",
				},
			},
		},
		VitalSigns: []ccda.VitalSign{
			{
				ID: "VS-001",
				Code: ccda.CodedValue{
					Code:        "8480-6",
					DisplayName: "Systolic BP",
				},
				EffectiveTime: time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
				Value:         128,
				Unit:          "mm[Hg]",
			},
		},
		LabResults: []ccda.LabResult{
			{
				ID: "LAB-001",
				Code: ccda.CodedValue{
					Code:        "4548-4",
					DisplayName: "HbA1c",
				},
				EffectiveTime: time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC),
				Value:         7.2,
				Unit:          "%",
				ReferenceRange: ccda.ReferenceRange{
					Low:  4.0,
					High: 5.6,
				},
			},
		},
		Allergies: []ccda.Allergy{
			{
				ID: "ALLERGY-001",
				Substance: ccda.CodedValue{
					Code:        "7980",
					DisplayName: "Penicillin",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low: time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				Reaction: ccda.CodedValue{
					DisplayName: "Hives",
				},
				Severity: ccda.CodedValue{
					DisplayName: "Moderate",
				},
			},
		},
		Immunizations: []ccda.Immunization{
			{
				ID: "IMM-001",
				Code: ccda.CodedValue{
					Code:        "141",
					DisplayName: "Influenza vaccine",
				},
				EffectiveTime: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				LotNumber:     "LOT-123",
				DoseQuantity: ccda.Quantity{
					Value: 0.5,
					Unit:  "mL",
				},
			},
		},
		Procedures: []ccda.Procedure{
			{
				ID: "PROC-001",
				Code: ccda.CodedValue{
					Code:        "73761001",
					DisplayName: "Colonoscopy",
				},
				EffectiveTime: ccda.EffectiveTime{
					Value: time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		Devices: []ccda.Device{
			{
				ID: "DEV-001",
				Code: ccda.CodedValue{
					Code:        "706689003",
					DisplayName: "Glucose monitor",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
				},
				UDI: "UDI-12345",
			},
		},
		Observations: []ccda.SocialObservation{
			{
				ID: "OBS-001",
				Code: ccda.CodedValue{
					Code:        "72166-2",
					DisplayName: "Tobacco smoking status",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				Value: ccda.CodedValue{
					Code:        "8517006",
					DisplayName: "Former smoker",
				},
			},
		},
	}

	data, err := m.MapDocument(doc)
	if err != nil {
		t.Fatalf("MapDocument failed: %v", err)
	}

	// Verify person
	t.Run("person", func(t *testing.T) {
		if len(data.Persons) != 1 {
			t.Fatalf("len(Persons) = %d, want 1", len(data.Persons))
		}
		person := data.Persons[0]
		if person.PersonID == 0 {
			t.Error("PersonID should not be 0")
		}
		if person.GenderConceptID != ConceptMale {
			t.Errorf("GenderConceptID = %d, want %d", person.GenderConceptID, ConceptMale)
		}
		if person.YearOfBirth != 1980 {
			t.Errorf("YearOfBirth = %d, want 1980", person.YearOfBirth)
		}
		if *person.MonthOfBirth != 5 {
			t.Errorf("MonthOfBirth = %d, want 5", *person.MonthOfBirth)
		}
		if *person.DayOfBirth != 15 {
			t.Errorf("DayOfBirth = %d, want 15", *person.DayOfBirth)
		}
		if person.RaceConceptID != ConceptWhite {
			t.Errorf("RaceConceptID = %d, want %d", person.RaceConceptID, ConceptWhite)
		}
		if person.EthnicityConceptID != ConceptNotHispanic {
			t.Errorf("EthnicityConceptID = %d, want %d", person.EthnicityConceptID, ConceptNotHispanic)
		}
		if person.PersonSourceValue != "12345" {
			t.Errorf("PersonSourceValue = %q, want %q", person.PersonSourceValue, "12345")
		}
	})

	// Verify visit occurrence
	t.Run("visit occurrence", func(t *testing.T) {
		if len(data.VisitOccurrences) != 1 {
			t.Fatalf("len(VisitOccurrences) = %d, want 1", len(data.VisitOccurrences))
		}
		visit := data.VisitOccurrences[0]
		if visit.VisitOccurrenceID == 0 {
			t.Error("VisitOccurrenceID should not be 0")
		}
		if visit.VisitConceptID != ConceptOutpatient {
			t.Errorf("VisitConceptID = %d, want %d", visit.VisitConceptID, ConceptOutpatient)
		}
		if visit.VisitSourceValue != "Ambulatory" {
			t.Errorf("VisitSourceValue = %q, want %q", visit.VisitSourceValue, "Ambulatory")
		}
	})

	// Verify condition occurrence
	t.Run("condition occurrence", func(t *testing.T) {
		if len(data.ConditionOccurrences) != 1 {
			t.Fatalf("len(ConditionOccurrences) = %d, want 1", len(data.ConditionOccurrences))
		}
		cond := data.ConditionOccurrences[0]
		if cond.ConditionOccurrenceID == 0 {
			t.Error("ConditionOccurrenceID should not be 0")
		}
		if cond.ConditionSourceValue != "Type 2 Diabetes" {
			t.Errorf("ConditionSourceValue = %q, want %q", cond.ConditionSourceValue, "Type 2 Diabetes")
		}
		if cond.ConditionTypeConceptID != ConceptEHRProblemList {
			t.Errorf("ConditionTypeConceptID = %d, want %d", cond.ConditionTypeConceptID, ConceptEHRProblemList)
		}
	})

	// Verify drug exposure (medications + immunizations)
	t.Run("drug exposure", func(t *testing.T) {
		if len(data.DrugExposures) != 2 {
			t.Fatalf("len(DrugExposures) = %d, want 2", len(data.DrugExposures))
		}
		// Check medication
		med := data.DrugExposures[0]
		if med.DrugSourceValue != "Metformin 500mg" {
			t.Errorf("DrugSourceValue = %q, want %q", med.DrugSourceValue, "Metformin 500mg")
		}
		if med.Quantity == nil || *med.Quantity != 500 {
			t.Error("Quantity should be 500")
		}
		if med.RouteSourceValue != "Oral" {
			t.Errorf("RouteSourceValue = %q, want %q", med.RouteSourceValue, "Oral")
		}
		// Check immunization
		imm := data.DrugExposures[1]
		if imm.LotNumber != "LOT-123" {
			t.Errorf("LotNumber = %q, want %q", imm.LotNumber, "LOT-123")
		}
	})

	// Verify procedure occurrence
	t.Run("procedure occurrence", func(t *testing.T) {
		if len(data.ProcedureOccurrences) != 1 {
			t.Fatalf("len(ProcedureOccurrences) = %d, want 1", len(data.ProcedureOccurrences))
		}
		proc := data.ProcedureOccurrences[0]
		if proc.ProcedureSourceValue != "Colonoscopy" {
			t.Errorf("ProcedureSourceValue = %q, want %q", proc.ProcedureSourceValue, "Colonoscopy")
		}
	})

	// Verify measurements (vitals + labs)
	t.Run("measurements", func(t *testing.T) {
		if len(data.Measurements) != 2 {
			t.Fatalf("len(Measurements) = %d, want 2", len(data.Measurements))
		}
		// Check vital sign
		vital := data.Measurements[0]
		if vital.MeasurementSourceValue != "Systolic BP" {
			t.Errorf("MeasurementSourceValue = %q, want %q", vital.MeasurementSourceValue, "Systolic BP")
		}
		if vital.ValueAsNumber == nil || *vital.ValueAsNumber != 128 {
			t.Error("ValueAsNumber should be 128")
		}
		if vital.UnitSourceValue != "mm[Hg]" {
			t.Errorf("UnitSourceValue = %q, want %q", vital.UnitSourceValue, "mm[Hg]")
		}
		// Check lab
		lab := data.Measurements[1]
		if lab.RangeLow == nil || *lab.RangeLow != 4.0 {
			t.Error("RangeLow should be 4.0")
		}
		if lab.RangeHigh == nil || *lab.RangeHigh != 5.6 {
			t.Error("RangeHigh should be 5.6")
		}
	})

	// Verify observations (allergies + social history)
	t.Run("observations", func(t *testing.T) {
		if len(data.Observations) != 2 {
			t.Fatalf("len(Observations) = %d, want 2", len(data.Observations))
		}
		// Check allergy
		allergy := data.Observations[0]
		if allergy.ObservationSourceValue != "Penicillin" {
			t.Errorf("ObservationSourceValue = %q, want %q", allergy.ObservationSourceValue, "Penicillin")
		}
		if allergy.ValueAsString != "Hives" {
			t.Errorf("ValueAsString = %q, want %q", allergy.ValueAsString, "Hives")
		}
		if allergy.QualifierSourceValue != "Moderate" {
			t.Errorf("QualifierSourceValue = %q, want %q", allergy.QualifierSourceValue, "Moderate")
		}
		// Check social observation
		social := data.Observations[1]
		if social.ValueAsString != "Former smoker" {
			t.Errorf("ValueAsString = %q, want %q", social.ValueAsString, "Former smoker")
		}
	})

	// Verify device exposure
	t.Run("device exposure", func(t *testing.T) {
		if len(data.DeviceExposures) != 1 {
			t.Fatalf("len(DeviceExposures) = %d, want 1", len(data.DeviceExposures))
		}
		device := data.DeviceExposures[0]
		if device.DeviceSourceValue != "Glucose monitor" {
			t.Errorf("DeviceSourceValue = %q, want %q", device.DeviceSourceValue, "Glucose monitor")
		}
		if device.UniqueDeviceID != "UDI-12345" {
			t.Errorf("UniqueDeviceID = %q, want %q", device.UniqueDeviceID, "UDI-12345")
		}
	})
}

func TestMapEmptyDocument(t *testing.T) {
	m := New(false)
	doc := &ccda.Document{
		Patient: ccda.Patient{
			ID:        "empty-patient",
			BirthTime: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	data, err := m.MapDocument(doc)
	if err != nil {
		t.Fatalf("MapDocument failed: %v", err)
	}

	if len(data.Persons) != 1 {
		t.Errorf("len(Persons) = %d, want 1", len(data.Persons))
	}
	if len(data.VisitOccurrences) != 0 {
		t.Errorf("len(VisitOccurrences) = %d, want 0", len(data.VisitOccurrences))
	}
	if len(data.ConditionOccurrences) != 0 {
		t.Errorf("len(ConditionOccurrences) = %d, want 0", len(data.ConditionOccurrences))
	}
}

func TestFormatSourceValue(t *testing.T) {
	tests := []struct {
		name     string
		input    ccda.CodedValue
		expected string
	}{
		{
			name: "display name preferred",
			input: ccda.CodedValue{
				Code:         "12345",
				DisplayName:  "Test Display",
				OriginalText: "Original",
			},
			expected: "Test Display",
		},
		{
			name: "code when no display name",
			input: ccda.CodedValue{
				Code:         "12345",
				OriginalText: "Original",
			},
			expected: "12345",
		},
		{
			name: "original text as fallback",
			input: ccda.CodedValue{
				OriginalText: "Original",
			},
			expected: "Original",
		},
		{
			name:     "empty",
			input:    ccda.CodedValue{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSourceValue(tt.input)
			if result != tt.expected {
				t.Errorf("formatSourceValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"integer", 100, "100"},
		{"decimal", 7.2, "7.2"},
		{"small decimal", 0.5, "0.5"},
		{"zero", 0, "0"},
		{"large integer", 1000000, "1000000"},
		{"negative", -42.5, "-42.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFloat(tt.input)
			if result != tt.expected {
				t.Errorf("formatFloat(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimePtr(t *testing.T) {
	t.Run("non-zero time", func(t *testing.T) {
		input := time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)
		result := timePtr(input)
		if result == nil {
			t.Fatal("timePtr returned nil for non-zero time")
		}
		if !result.Equal(input) {
			t.Errorf("timePtr() = %v, want %v", *result, input)
		}
	})

	t.Run("zero time", func(t *testing.T) {
		var input time.Time
		result := timePtr(input)
		if result != nil {
			t.Errorf("timePtr returned non-nil for zero time: %v", result)
		}
	})
}

func TestDeterministicIDs(t *testing.T) {
	m := New(false)

	doc := &ccda.Document{
		Patient: ccda.Patient{
			ID:        "test-patient",
			BirthTime: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Problems: []ccda.Problem{
			{
				ID: "PROB-001",
				Code: ccda.CodedValue{
					Code: "12345",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	// Map document twice
	data1, _ := m.MapDocument(doc)
	data2, _ := m.MapDocument(doc)

	// IDs should be identical
	if data1.Persons[0].PersonID != data2.Persons[0].PersonID {
		t.Error("PersonID should be deterministic")
	}
	if data1.ConditionOccurrences[0].ConditionOccurrenceID != data2.ConditionOccurrences[0].ConditionOccurrenceID {
		t.Error("ConditionOccurrenceID should be deterministic")
	}
}

func TestConditionWithEndDate(t *testing.T) {
	m := New(false)

	doc := &ccda.Document{
		Patient: ccda.Patient{
			ID:        "test-patient",
			BirthTime: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Problems: []ccda.Problem{
			{
				ID: "PROB-001",
				Code: ccda.CodedValue{
					Code: "12345",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					High: time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, _ := m.MapDocument(doc)
	cond := data.ConditionOccurrences[0]

	if cond.ConditionEndDate == nil {
		t.Fatal("ConditionEndDate should not be nil")
	}
	if cond.ConditionEndDate.Year() != 2023 || cond.ConditionEndDate.Month() != 6 {
		t.Errorf("ConditionEndDate = %v, want 2023-06-30", cond.ConditionEndDate)
	}
}

func TestDeviceWithEndDate(t *testing.T) {
	m := New(false)

	doc := &ccda.Document{
		Patient: ccda.Patient{
			ID:        "test-patient",
			BirthTime: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Devices: []ccda.Device{
			{
				ID: "DEV-001",
				Code: ccda.CodedValue{
					Code: "12345",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					High: time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	data, _ := m.MapDocument(doc)
	device := data.DeviceExposures[0]

	if device.DeviceExposureEndDate == nil {
		t.Fatal("DeviceExposureEndDate should not be nil")
	}
}

func TestSocialObservationWithQuantity(t *testing.T) {
	m := New(false)

	doc := &ccda.Document{
		Patient: ccda.Patient{
			ID:        "test-patient",
			BirthTime: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Observations: []ccda.SocialObservation{
			{
				ID: "OBS-001",
				Code: ccda.CodedValue{
					Code: "test-code",
				},
				EffectiveTime: ccda.EffectiveTime{
					Low: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				ValueQuantity: ccda.Quantity{
					Value: 10,
					Unit:  "pack-years",
				},
			},
		},
	}

	data, _ := m.MapDocument(doc)
	obs := data.Observations[0]

	if obs.ValueAsNumber == nil || *obs.ValueAsNumber != 10 {
		t.Error("ValueAsNumber should be 10")
	}
	if obs.UnitSourceValue != "pack-years" {
		t.Errorf("UnitSourceValue = %q, want %q", obs.UnitSourceValue, "pack-years")
	}
}
