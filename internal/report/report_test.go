// Copyright 2025 Christophe Roeder. All rights reserved.

package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ccda2omop/internal/omop"
)

func TestNewConversionReport(t *testing.T) {
	r := NewConversionReport()
	if r == nil {
		t.Fatal("NewConversionReport returned nil")
	}
	if r.EntriesBySection == nil {
		t.Error("EntriesBySection map is nil")
	}
	if r.RecordsByTable == nil {
		t.Error("RecordsByTable map is nil")
	}
	if r.FieldPopulation == nil {
		t.Error("FieldPopulation map is nil")
	}
	if r.ConceptMappings == nil {
		t.Error("ConceptMappings map is nil")
	}
	if r.SkippedEntries == nil {
		t.Error("SkippedEntries map is nil")
	}
}

func TestAddDocument(t *testing.T) {
	r := NewConversionReport()

	r.AddDocument(false)
	if r.DocumentsProcessed != 1 {
		t.Errorf("DocumentsProcessed = %d, want 1", r.DocumentsProcessed)
	}
	if r.DocumentsWithErrors != 0 {
		t.Errorf("DocumentsWithErrors = %d, want 0", r.DocumentsWithErrors)
	}

	r.AddDocument(true)
	if r.DocumentsProcessed != 2 {
		t.Errorf("DocumentsProcessed = %d, want 2", r.DocumentsProcessed)
	}
	if r.DocumentsWithErrors != 1 {
		t.Errorf("DocumentsWithErrors = %d, want 1", r.DocumentsWithErrors)
	}

	r.AddDocument(false)
	r.AddDocument(true)
	if r.DocumentsProcessed != 4 {
		t.Errorf("DocumentsProcessed = %d, want 4", r.DocumentsProcessed)
	}
	if r.DocumentsWithErrors != 2 {
		t.Errorf("DocumentsWithErrors = %d, want 2", r.DocumentsWithErrors)
	}
}

func TestAddSectionEntry(t *testing.T) {
	r := NewConversionReport()

	r.AddSectionEntry("problems")
	r.AddSectionEntry("problems")
	r.AddSectionEntry("medications")

	if r.EntriesBySection["problems"] == nil {
		t.Fatal("problems section is nil")
	}
	if r.EntriesBySection["problems"].EntriesFound != 2 {
		t.Errorf("problems EntriesFound = %d, want 2", r.EntriesBySection["problems"].EntriesFound)
	}
	if r.EntriesBySection["medications"].EntriesFound != 1 {
		t.Errorf("medications EntriesFound = %d, want 1", r.EntriesBySection["medications"].EntriesFound)
	}
}

func TestAddSectionRecord(t *testing.T) {
	r := NewConversionReport()

	r.AddSectionRecord("problems", "condition_occurrence")
	r.AddSectionRecord("problems", "condition_occurrence")
	r.AddSectionRecord("problems", "observation")
	r.AddSectionRecord("medications", "drug_exposure")

	if r.EntriesBySection["problems"].RecordsCreated != 3 {
		t.Errorf("problems RecordsCreated = %d, want 3", r.EntriesBySection["problems"].RecordsCreated)
	}
	if r.EntriesBySection["problems"].TargetTables["condition_occurrence"] != 2 {
		t.Errorf("problems condition_occurrence = %d, want 2", r.EntriesBySection["problems"].TargetTables["condition_occurrence"])
	}
	if r.EntriesBySection["problems"].TargetTables["observation"] != 1 {
		t.Errorf("problems observation = %d, want 1", r.EntriesBySection["problems"].TargetTables["observation"])
	}
	if r.EntriesBySection["medications"].TargetTables["drug_exposure"] != 1 {
		t.Errorf("medications drug_exposure = %d, want 1", r.EntriesBySection["medications"].TargetTables["drug_exposure"])
	}
}

func TestAddSkipped(t *testing.T) {
	r := NewConversionReport()

	r.AddSkipped("problems", "moodCode != EVN")
	r.AddSkipped("problems", "moodCode != EVN")
	r.AddSkipped("medications", "missing code")

	if r.EntriesBySection["problems"].Skipped != 2 {
		t.Errorf("problems Skipped = %d, want 2", r.EntriesBySection["problems"].Skipped)
	}
	if r.SkippedEntries["moodCode != EVN"] != 2 {
		t.Errorf("moodCode != EVN count = %d, want 2", r.SkippedEntries["moodCode != EVN"])
	}
	if r.SkippedEntries["missing code"] != 1 {
		t.Errorf("missing code count = %d, want 1", r.SkippedEntries["missing code"])
	}
}

func TestAddConceptMapping(t *testing.T) {
	r := NewConversionReport()

	r.AddConceptMapping("SNOMED", true)
	r.AddConceptMapping("SNOMED", true)
	r.AddConceptMapping("SNOMED", false)
	r.AddConceptMapping("RxNorm", true)
	r.AddConceptMapping("RxNorm", false)

	snomed := r.ConceptMappings["SNOMED"]
	if snomed.CodesSeen != 3 {
		t.Errorf("SNOMED CodesSeen = %d, want 3", snomed.CodesSeen)
	}
	if snomed.MappedStandard != 2 {
		t.Errorf("SNOMED MappedStandard = %d, want 2", snomed.MappedStandard)
	}
	if snomed.SourceOnly != 1 {
		t.Errorf("SNOMED SourceOnly = %d, want 1", snomed.SourceOnly)
	}

	rxnorm := r.ConceptMappings["RxNorm"]
	if rxnorm.CodesSeen != 2 {
		t.Errorf("RxNorm CodesSeen = %d, want 2", rxnorm.CodesSeen)
	}
}

func TestAddDomainRoute(t *testing.T) {
	r := NewConversionReport()

	r.AddDomainRoute("problems", "condition_occurrence", "observation", "Domain=Observation")
	r.AddDomainRoute("problems", "condition_occurrence", "observation", "Domain=Observation")
	r.AddDomainRoute("labs", "measurement", "observation", "Domain=Observation")

	if len(r.DomainRouting) != 2 {
		t.Errorf("DomainRouting length = %d, want 2", len(r.DomainRouting))
	}

	// Find the problems route
	var problemsRoute *DomainRoute
	for i := range r.DomainRouting {
		if r.DomainRouting[i].SourceSection == "problems" {
			problemsRoute = &r.DomainRouting[i]
			break
		}
	}
	if problemsRoute == nil {
		t.Fatal("problems route not found")
	}
	if problemsRoute.Count != 2 {
		t.Errorf("problems route Count = %d, want 2", problemsRoute.Count)
	}
}

func TestCalculateFromOMOPData(t *testing.T) {
	r := NewConversionReport()

	now := time.Now()
	data := &omop.OMOPData{
		Persons: []omop.Person{
			{PersonID: 1, GenderConceptID: 8507},
		},
		VisitOccurrences: []omop.VisitOccurrence{
			{VisitOccurrenceID: 1, PersonID: 1, VisitStartDate: now, VisitEndDate: now},
		},
		ConditionOccurrences: []omop.ConditionOccurrence{
			{ConditionOccurrenceID: 1, PersonID: 1, ConditionConceptID: 12345, ConditionStartDate: now, ConditionSourceValue: "Test", MappingRule: "RuleMapper:problems_to_conditions"},
			{ConditionOccurrenceID: 2, PersonID: 1, ConditionConceptID: 0, ConditionStartDate: now, ConditionSourceValue: "Test2", MappingRule: "RuleMapper:problems_to_conditions"},
		},
		DrugExposures: []omop.DrugExposure{
			{DrugExposureID: 1, PersonID: 1, DrugConceptID: 100, DrugExposureStartDate: now, DrugExposureEndDate: now, DrugSourceValue: "Drug1", MappingRule: "RuleMapper:medications_to_drugs"},
		},
		Measurements: []omop.Measurement{
			{MeasurementID: 1, PersonID: 1, MeasurementConceptID: 200, MeasurementDate: now, ValueAsNumber: floatPtr(120), MeasurementSourceValue: "BP", MappingRule: "RuleMapper:vitals_to_measurements"},
			{MeasurementID: 2, PersonID: 1, MeasurementConceptID: 201, MeasurementDate: now, ValueAsNumber: floatPtr(80), UnitConceptID: int64Ptr(100), MeasurementSourceValue: "BP2", MappingRule: "RuleMapper:vitals_to_measurements"},
		},
		Observations: []omop.Observation{
			{ObservationID: 1, PersonID: 1, ObservationConceptID: 300, ObservationDate: now, ValueAsString: "Former smoker", ObservationSourceValue: "Smoking", MappingRule: "RuleMapper:social_to_observations"},
		},
	}

	r.CalculateFromOMOPData(data)

	// Check record counts
	if r.RecordsByTable["person"] != 1 {
		t.Errorf("person count = %d, want 1", r.RecordsByTable["person"])
	}
	if r.RecordsByTable["condition_occurrence"] != 2 {
		t.Errorf("condition_occurrence count = %d, want 2", r.RecordsByTable["condition_occurrence"])
	}
	if r.RecordsByTable["drug_exposure"] != 1 {
		t.Errorf("drug_exposure count = %d, want 1", r.RecordsByTable["drug_exposure"])
	}
	if r.RecordsByTable["measurement"] != 2 {
		t.Errorf("measurement count = %d, want 2", r.RecordsByTable["measurement"])
	}
	if r.RecordsByTable["observation"] != 1 {
		t.Errorf("observation count = %d, want 1", r.RecordsByTable["observation"])
	}

	// Check field population
	condFields := r.FieldPopulation["condition_occurrence"]
	if condFields == nil {
		t.Fatal("condition_occurrence field population is nil")
	}
	if condFields["condition_concept_id (>0)"].Populated != 1 {
		t.Errorf("condition_concept_id populated = %d, want 1", condFields["condition_concept_id (>0)"].Populated)
	}
	if condFields["condition_source_value"].Populated != 2 {
		t.Errorf("condition_source_value populated = %d, want 2", condFields["condition_source_value"].Populated)
	}

	// Check section mappings
	if r.EntriesBySection["problems"] == nil {
		t.Fatal("problems section is nil")
	}
	if r.EntriesBySection["problems"].RecordsCreated != 2 {
		t.Errorf("problems RecordsCreated = %d, want 2", r.EntriesBySection["problems"].RecordsCreated)
	}
}

func TestExtractSectionFromRule(t *testing.T) {
	tests := []struct {
		rule     string
		expected string
	}{
		{"RuleMapper:problems_to_conditions", "problems"},
		{"RuleMapper:medications_to_drugs", "medications"},
		{"RuleMapper:vitals_to_measurements", "vitals"},
		{"RuleMapper:labs_to_observations", "labs"},
		{"RuleMapper:social_to_observations", "social"},
		{"SomeOtherMapper:test", ""},
		{"", ""},
		{"RuleMapper:", ""},
	}

	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			result := extractSectionFromRule(tt.rule)
			if result != tt.expected {
				t.Errorf("extractSectionFromRule(%q) = %q, want %q", tt.rule, result, tt.expected)
			}
		})
	}
}

func TestWriteText(t *testing.T) {
	r := NewConversionReport()
	r.DocumentsProcessed = 10
	r.DocumentsWithErrors = 1
	r.RecordsByTable["person"] = 10
	r.RecordsByTable["condition_occurrence"] = 25
	r.RecordsByTable["measurement"] = 100

	r.AddSectionRecord("problems", "condition_occurrence")
	r.AddConceptMapping("SNOMED", true)
	r.AddConceptMapping("SNOMED", false)

	var buf bytes.Buffer
	err := r.WriteText(&buf)
	if err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()

	// Check for expected content
	expectedStrings := []string{
		"# CCDA-to-OMOP Conversion Report",
		"## Document Summary",
		"Documents Processed | 10",
		"Documents with Errors | 1",
		"Success Rate | 90.0%",
		"## Records Created by OMOP Table",
		"condition_occurrence | 25",
		"measurement | 100",
		"## CCDA Section to OMOP Table Mapping",
		"problems",
		"## Concept Mapping Quality",
		"SNOMED",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("WriteText output missing %q", expected)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	r := NewConversionReport()
	r.DocumentsProcessed = 5
	r.DocumentsWithErrors = 0
	r.RecordsByTable["person"] = 5
	r.RecordsByTable["condition_occurrence"] = 15

	r.AddSectionRecord("problems", "condition_occurrence")
	r.AddConceptMapping("LOINC", true)

	var buf bytes.Buffer
	err := r.WriteJSON(&buf)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed ConversionReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("WriteJSON output is not valid JSON: %v", err)
	}

	// Verify values
	if parsed.DocumentsProcessed != 5 {
		t.Errorf("parsed DocumentsProcessed = %d, want 5", parsed.DocumentsProcessed)
	}
	if parsed.RecordsByTable["person"] != 5 {
		t.Errorf("parsed person count = %d, want 5", parsed.RecordsByTable["person"])
	}
	if parsed.RecordsByTable["condition_occurrence"] != 15 {
		t.Errorf("parsed condition_occurrence count = %d, want 15", parsed.RecordsByTable["condition_occurrence"])
	}
}

func TestFormatTargetTables(t *testing.T) {
	tests := []struct {
		name     string
		tables   map[string]int
		expected string
	}{
		{"empty", map[string]int{}, "-"},
		{"nil", nil, "-"},
		{"single", map[string]int{"condition_occurrence": 5}, "condition_occurrence(5)"},
		{"multiple", map[string]int{"condition_occurrence": 5, "observation": 3}, "condition_occurrence(5), observation(3)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTargetTables(tt.tables)
			if result != tt.expected {
				t.Errorf("formatTargetTables() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFieldStatsCalculation(t *testing.T) {
	r := NewConversionReport()

	now := time.Now()
	endDate := now.Add(24 * time.Hour)

	data := &omop.OMOPData{
		ConditionOccurrences: []omop.ConditionOccurrence{
			{ConditionOccurrenceID: 1, ConditionConceptID: 100, ConditionStartDate: now, ConditionEndDate: &endDate, ConditionSourceValue: "A", VisitOccurrenceID: int64Ptr(1)},
			{ConditionOccurrenceID: 2, ConditionConceptID: 0, ConditionStartDate: now, ConditionSourceValue: "B"},
			{ConditionOccurrenceID: 3, ConditionConceptID: 200, ConditionStartDate: now, ConditionSourceValue: ""},
		},
	}

	r.CalculateFromOMOPData(data)

	fields := r.FieldPopulation["condition_occurrence"]

	// condition_concept_id > 0: 2 out of 3
	if fields["condition_concept_id (>0)"].Populated != 2 {
		t.Errorf("condition_concept_id populated = %d, want 2", fields["condition_concept_id (>0)"].Populated)
	}
	if fields["condition_concept_id (>0)"].Total != 3 {
		t.Errorf("condition_concept_id total = %d, want 3", fields["condition_concept_id (>0)"].Total)
	}

	// condition_end_date: 1 out of 3
	if fields["condition_end_date"].Populated != 1 {
		t.Errorf("condition_end_date populated = %d, want 1", fields["condition_end_date"].Populated)
	}

	// condition_source_value: 2 out of 3
	if fields["condition_source_value"].Populated != 2 {
		t.Errorf("condition_source_value populated = %d, want 2", fields["condition_source_value"].Populated)
	}

	// visit_occurrence_id: 1 out of 3
	if fields["visit_occurrence_id"].Populated != 1 {
		t.Errorf("visit_occurrence_id populated = %d, want 1", fields["visit_occurrence_id"].Populated)
	}
}

func TestMeasurementFieldStats(t *testing.T) {
	r := NewConversionReport()

	now := time.Now()
	data := &omop.OMOPData{
		Measurements: []omop.Measurement{
			{MeasurementID: 1, MeasurementConceptID: 100, MeasurementDate: now, ValueAsNumber: floatPtr(120), UnitConceptID: int64Ptr(50), MeasurementSourceValue: "BP"},
			{MeasurementID: 2, MeasurementConceptID: 0, MeasurementDate: now, ValueAsNumber: floatPtr(80), MeasurementSourceValue: "BP2"},
			{MeasurementID: 3, MeasurementConceptID: 200, MeasurementDate: now, ValueAsConceptID: int64Ptr(999), RangeLow: floatPtr(60), RangeHigh: floatPtr(100), MeasurementSourceValue: "HR"},
		},
	}

	r.CalculateFromOMOPData(data)

	fields := r.FieldPopulation["measurement"]

	if fields["measurement_concept_id (>0)"].Populated != 2 {
		t.Errorf("measurement_concept_id populated = %d, want 2", fields["measurement_concept_id (>0)"].Populated)
	}
	if fields["value_as_number"].Populated != 2 {
		t.Errorf("value_as_number populated = %d, want 2", fields["value_as_number"].Populated)
	}
	if fields["value_as_concept_id (>0)"].Populated != 1 {
		t.Errorf("value_as_concept_id populated = %d, want 1", fields["value_as_concept_id (>0)"].Populated)
	}
	if fields["unit_concept_id (>0)"].Populated != 1 {
		t.Errorf("unit_concept_id populated = %d, want 1", fields["unit_concept_id (>0)"].Populated)
	}
	if fields["range_low/high"].Populated != 1 {
		t.Errorf("range_low/high populated = %d, want 1", fields["range_low/high"].Populated)
	}
}

func TestEmptyDataCalculation(t *testing.T) {
	r := NewConversionReport()
	data := &omop.OMOPData{}

	// Should not panic on empty data
	r.CalculateFromOMOPData(data)

	if r.RecordsByTable["person"] != 0 {
		t.Errorf("person count = %d, want 0", r.RecordsByTable["person"])
	}
	if len(r.FieldPopulation) != 0 {
		t.Errorf("FieldPopulation length = %d, want 0", len(r.FieldPopulation))
	}
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func int64Ptr(i int64) *int64 {
	return &i
}
