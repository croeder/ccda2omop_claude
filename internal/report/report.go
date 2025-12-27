// Copyright 2025 Christophe Roeder. All rights reserved.

package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ccda2omop/internal/omop"
)

// ConversionReport holds comprehensive metrics about a CCDA-to-OMOP conversion
type ConversionReport struct {
	// Document-level metrics
	DocumentsProcessed int `json:"documents_processed"`
	DocumentsWithErrors int `json:"documents_with_errors"`

	// Section-level metrics (CCDA sections -> entries found)
	EntriesBySection map[string]*SectionMetrics `json:"entries_by_section"`

	// Output metrics (OMOP tables -> records created)
	RecordsByTable map[string]int `json:"records_by_table"`

	// Field population rates per table
	FieldPopulation map[string]map[string]*FieldStats `json:"field_population"`

	// Concept mapping quality
	ConceptMappings map[string]*VocabStats `json:"concept_mappings"`

	// Domain routing - records moved between tables
	DomainRouting []DomainRoute `json:"domain_routing,omitempty"`

	// Skip reasons
	SkippedEntries map[string]int `json:"skipped_entries,omitempty"`
}

// SectionMetrics tracks metrics for a CCDA section
type SectionMetrics struct {
	EntriesFound    int            `json:"entries_found"`
	RecordsCreated  int            `json:"records_created"`
	Skipped         int            `json:"skipped"`
	TargetTables    map[string]int `json:"target_tables"` // table -> count
}

// FieldStats tracks population statistics for a field
type FieldStats struct {
	Populated int `json:"populated"`
	Total     int `json:"total"`
}

// VocabStats tracks vocabulary mapping statistics
type VocabStats struct {
	CodesSeen      int `json:"codes_seen"`
	MappedStandard int `json:"mapped_standard"`
	SourceOnly     int `json:"source_only"`
}

// DomainRoute records when a record is routed to a different table based on domain
type DomainRoute struct {
	SourceSection  string `json:"source_section"`
	OriginalTarget string `json:"original_target"`
	ActualTarget   string `json:"actual_target"`
	Count          int    `json:"count"`
	Reason         string `json:"reason"`
}

// NewConversionReport creates a new empty report
func NewConversionReport() *ConversionReport {
	return &ConversionReport{
		EntriesBySection: make(map[string]*SectionMetrics),
		RecordsByTable:   make(map[string]int),
		FieldPopulation:  make(map[string]map[string]*FieldStats),
		ConceptMappings:  make(map[string]*VocabStats),
		SkippedEntries:   make(map[string]int),
	}
}

// AddDocument increments the document counter
func (r *ConversionReport) AddDocument(hasError bool) {
	r.DocumentsProcessed++
	if hasError {
		r.DocumentsWithErrors++
	}
}

// AddSectionEntry records an entry found in a CCDA section
func (r *ConversionReport) AddSectionEntry(section string) {
	if r.EntriesBySection[section] == nil {
		r.EntriesBySection[section] = &SectionMetrics{
			TargetTables: make(map[string]int),
		}
	}
	r.EntriesBySection[section].EntriesFound++
}

// AddSectionRecord records a record created from a section entry
func (r *ConversionReport) AddSectionRecord(section, targetTable string) {
	if r.EntriesBySection[section] == nil {
		r.EntriesBySection[section] = &SectionMetrics{
			TargetTables: make(map[string]int),
		}
	}
	r.EntriesBySection[section].RecordsCreated++
	r.EntriesBySection[section].TargetTables[targetTable]++
}

// AddSkipped records a skipped entry with reason
func (r *ConversionReport) AddSkipped(section, reason string) {
	if r.EntriesBySection[section] == nil {
		r.EntriesBySection[section] = &SectionMetrics{
			TargetTables: make(map[string]int),
		}
	}
	r.EntriesBySection[section].Skipped++
	r.SkippedEntries[reason]++
}

// AddConceptMapping records a vocabulary mapping attempt
func (r *ConversionReport) AddConceptMapping(vocab string, mappedToStandard bool) {
	if r.ConceptMappings[vocab] == nil {
		r.ConceptMappings[vocab] = &VocabStats{}
	}
	r.ConceptMappings[vocab].CodesSeen++
	if mappedToStandard {
		r.ConceptMappings[vocab].MappedStandard++
	} else {
		r.ConceptMappings[vocab].SourceOnly++
	}
}

// AddDomainRoute records domain-based routing
func (r *ConversionReport) AddDomainRoute(section, originalTarget, actualTarget, reason string) {
	// Find existing route or create new one
	for i := range r.DomainRouting {
		if r.DomainRouting[i].SourceSection == section &&
			r.DomainRouting[i].OriginalTarget == originalTarget &&
			r.DomainRouting[i].ActualTarget == actualTarget {
			r.DomainRouting[i].Count++
			return
		}
	}
	r.DomainRouting = append(r.DomainRouting, DomainRoute{
		SourceSection:  section,
		OriginalTarget: originalTarget,
		ActualTarget:   actualTarget,
		Count:          1,
		Reason:         reason,
	})
}

// CalculateFromOMOPData populates the report from OMOP output data
func (r *ConversionReport) CalculateFromOMOPData(data *omop.OMOPData) {
	// Record counts by table
	r.RecordsByTable["person"] = len(data.Persons)
	r.RecordsByTable["visit_occurrence"] = len(data.VisitOccurrences)
	r.RecordsByTable["condition_occurrence"] = len(data.ConditionOccurrences)
	r.RecordsByTable["drug_exposure"] = len(data.DrugExposures)
	r.RecordsByTable["procedure_occurrence"] = len(data.ProcedureOccurrences)
	r.RecordsByTable["measurement"] = len(data.Measurements)
	r.RecordsByTable["observation"] = len(data.Observations)
	r.RecordsByTable["device_exposure"] = len(data.DeviceExposures)

	// Calculate field population rates
	r.calculateConditionFields(data.ConditionOccurrences)
	r.calculateDrugFields(data.DrugExposures)
	r.calculateProcedureFields(data.ProcedureOccurrences)
	r.calculateMeasurementFields(data.Measurements)
	r.calculateObservationFields(data.Observations)
	r.calculateDeviceFields(data.DeviceExposures)

	// Track section -> table mappings from MappingRule field
	r.trackSectionMappings(data)
}

func (r *ConversionReport) trackSectionMappings(data *omop.OMOPData) {
	// Parse MappingRule to extract section info
	// Format is "RuleMapper:section_to_table" e.g. "RuleMapper:problems_to_conditions"

	for _, c := range data.ConditionOccurrences {
		section := extractSectionFromRule(c.MappingRule)
		if section != "" {
			r.AddSectionRecord(section, "condition_occurrence")
		}
	}
	for _, d := range data.DrugExposures {
		section := extractSectionFromRule(d.MappingRule)
		if section != "" {
			r.AddSectionRecord(section, "drug_exposure")
		}
	}
	for _, p := range data.ProcedureOccurrences {
		section := extractSectionFromRule(p.MappingRule)
		if section != "" {
			r.AddSectionRecord(section, "procedure_occurrence")
		}
	}
	for _, m := range data.Measurements {
		section := extractSectionFromRule(m.MappingRule)
		if section != "" {
			r.AddSectionRecord(section, "measurement")
		}
	}
	for _, o := range data.Observations {
		section := extractSectionFromRule(o.MappingRule)
		if section != "" {
			r.AddSectionRecord(section, "observation")
		}
	}
	for _, d := range data.DeviceExposures {
		section := extractSectionFromRule(d.MappingRule)
		if section != "" {
			r.AddSectionRecord(section, "device_exposure")
		}
	}
}

func extractSectionFromRule(rule string) string {
	// Format: "RuleMapper:section_to_table"
	if !strings.HasPrefix(rule, "RuleMapper:") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(rule, "RuleMapper:"), "_to_")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func (r *ConversionReport) calculateConditionFields(records []omop.ConditionOccurrence) {
	if len(records) == 0 {
		return
	}
	r.FieldPopulation["condition_occurrence"] = make(map[string]*FieldStats)
	total := len(records)

	conceptIDCount := 0
	endDateCount := 0
	sourceValueCount := 0
	visitIDCount := 0

	for _, rec := range records {
		if rec.ConditionConceptID > 0 {
			conceptIDCount++
		}
		if rec.ConditionEndDate != nil {
			endDateCount++
		}
		if rec.ConditionSourceValue != "" {
			sourceValueCount++
		}
		if rec.VisitOccurrenceID != nil {
			visitIDCount++
		}
	}

	r.FieldPopulation["condition_occurrence"]["condition_concept_id (>0)"] = &FieldStats{conceptIDCount, total}
	r.FieldPopulation["condition_occurrence"]["condition_end_date"] = &FieldStats{endDateCount, total}
	r.FieldPopulation["condition_occurrence"]["condition_source_value"] = &FieldStats{sourceValueCount, total}
	r.FieldPopulation["condition_occurrence"]["visit_occurrence_id"] = &FieldStats{visitIDCount, total}
}

func (r *ConversionReport) calculateDrugFields(records []omop.DrugExposure) {
	if len(records) == 0 {
		return
	}
	r.FieldPopulation["drug_exposure"] = make(map[string]*FieldStats)
	total := len(records)

	conceptIDCount := 0
	quantityCount := 0
	routeCount := 0
	sourceValueCount := 0

	for _, rec := range records {
		if rec.DrugConceptID > 0 {
			conceptIDCount++
		}
		if rec.Quantity != nil {
			quantityCount++
		}
		if rec.RouteConceptID != nil && *rec.RouteConceptID > 0 {
			routeCount++
		}
		if rec.DrugSourceValue != "" {
			sourceValueCount++
		}
	}

	r.FieldPopulation["drug_exposure"]["drug_concept_id (>0)"] = &FieldStats{conceptIDCount, total}
	r.FieldPopulation["drug_exposure"]["quantity"] = &FieldStats{quantityCount, total}
	r.FieldPopulation["drug_exposure"]["route_concept_id (>0)"] = &FieldStats{routeCount, total}
	r.FieldPopulation["drug_exposure"]["drug_source_value"] = &FieldStats{sourceValueCount, total}
}

func (r *ConversionReport) calculateProcedureFields(records []omop.ProcedureOccurrence) {
	if len(records) == 0 {
		return
	}
	r.FieldPopulation["procedure_occurrence"] = make(map[string]*FieldStats)
	total := len(records)

	conceptIDCount := 0
	sourceValueCount := 0
	visitIDCount := 0

	for _, rec := range records {
		if rec.ProcedureConceptID > 0 {
			conceptIDCount++
		}
		if rec.ProcedureSourceValue != "" {
			sourceValueCount++
		}
		if rec.VisitOccurrenceID != nil {
			visitIDCount++
		}
	}

	r.FieldPopulation["procedure_occurrence"]["procedure_concept_id (>0)"] = &FieldStats{conceptIDCount, total}
	r.FieldPopulation["procedure_occurrence"]["procedure_source_value"] = &FieldStats{sourceValueCount, total}
	r.FieldPopulation["procedure_occurrence"]["visit_occurrence_id"] = &FieldStats{visitIDCount, total}
}

func (r *ConversionReport) calculateMeasurementFields(records []omop.Measurement) {
	if len(records) == 0 {
		return
	}
	r.FieldPopulation["measurement"] = make(map[string]*FieldStats)
	total := len(records)

	conceptIDCount := 0
	valueNumCount := 0
	valueConceptCount := 0
	unitConceptCount := 0
	rangeCount := 0
	sourceValueCount := 0

	for _, rec := range records {
		if rec.MeasurementConceptID > 0 {
			conceptIDCount++
		}
		if rec.ValueAsNumber != nil {
			valueNumCount++
		}
		if rec.ValueAsConceptID != nil && *rec.ValueAsConceptID > 0 {
			valueConceptCount++
		}
		if rec.UnitConceptID != nil && *rec.UnitConceptID > 0 {
			unitConceptCount++
		}
		if rec.RangeLow != nil || rec.RangeHigh != nil {
			rangeCount++
		}
		if rec.MeasurementSourceValue != "" {
			sourceValueCount++
		}
	}

	r.FieldPopulation["measurement"]["measurement_concept_id (>0)"] = &FieldStats{conceptIDCount, total}
	r.FieldPopulation["measurement"]["value_as_number"] = &FieldStats{valueNumCount, total}
	r.FieldPopulation["measurement"]["value_as_concept_id (>0)"] = &FieldStats{valueConceptCount, total}
	r.FieldPopulation["measurement"]["unit_concept_id (>0)"] = &FieldStats{unitConceptCount, total}
	r.FieldPopulation["measurement"]["range_low/high"] = &FieldStats{rangeCount, total}
	r.FieldPopulation["measurement"]["measurement_source_value"] = &FieldStats{sourceValueCount, total}
}

func (r *ConversionReport) calculateObservationFields(records []omop.Observation) {
	if len(records) == 0 {
		return
	}
	r.FieldPopulation["observation"] = make(map[string]*FieldStats)
	total := len(records)

	conceptIDCount := 0
	valueNumCount := 0
	valueStringCount := 0
	valueConceptCount := 0
	sourceValueCount := 0

	for _, rec := range records {
		if rec.ObservationConceptID > 0 {
			conceptIDCount++
		}
		if rec.ValueAsNumber != nil {
			valueNumCount++
		}
		if rec.ValueAsString != "" {
			valueStringCount++
		}
		if rec.ValueAsConceptID != nil && *rec.ValueAsConceptID > 0 {
			valueConceptCount++
		}
		if rec.ObservationSourceValue != "" {
			sourceValueCount++
		}
	}

	r.FieldPopulation["observation"]["observation_concept_id (>0)"] = &FieldStats{conceptIDCount, total}
	r.FieldPopulation["observation"]["value_as_number"] = &FieldStats{valueNumCount, total}
	r.FieldPopulation["observation"]["value_as_string"] = &FieldStats{valueStringCount, total}
	r.FieldPopulation["observation"]["value_as_concept_id (>0)"] = &FieldStats{valueConceptCount, total}
	r.FieldPopulation["observation"]["observation_source_value"] = &FieldStats{sourceValueCount, total}
}

func (r *ConversionReport) calculateDeviceFields(records []omop.DeviceExposure) {
	if len(records) == 0 {
		return
	}
	r.FieldPopulation["device_exposure"] = make(map[string]*FieldStats)
	total := len(records)

	conceptIDCount := 0
	sourceValueCount := 0
	uniqueIDCount := 0

	for _, rec := range records {
		if rec.DeviceConceptID > 0 {
			conceptIDCount++
		}
		if rec.DeviceSourceValue != "" {
			sourceValueCount++
		}
		if rec.UniqueDeviceID != "" {
			uniqueIDCount++
		}
	}

	r.FieldPopulation["device_exposure"]["device_concept_id (>0)"] = &FieldStats{conceptIDCount, total}
	r.FieldPopulation["device_exposure"]["device_source_value"] = &FieldStats{sourceValueCount, total}
	r.FieldPopulation["device_exposure"]["unique_device_id"] = &FieldStats{uniqueIDCount, total}
}

// WriteText writes the report in human-readable text format
func (r *ConversionReport) WriteText(w io.Writer) error {
	fmt.Fprintf(w, "# CCDA-to-OMOP Conversion Report\n\n")

	// Document summary
	fmt.Fprintf(w, "## Document Summary\n\n")
	fmt.Fprintf(w, "| Metric | Value |\n")
	fmt.Fprintf(w, "|--------|-------|\n")
	fmt.Fprintf(w, "| Documents Processed | %d |\n", r.DocumentsProcessed)
	fmt.Fprintf(w, "| Documents with Errors | %d |\n", r.DocumentsWithErrors)
	if r.DocumentsProcessed > 0 {
		successRate := float64(r.DocumentsProcessed-r.DocumentsWithErrors) / float64(r.DocumentsProcessed) * 100
		fmt.Fprintf(w, "| Success Rate | %.1f%% |\n", successRate)
	}
	fmt.Fprintf(w, "\n")

	// Records by table
	fmt.Fprintf(w, "## Records Created by OMOP Table\n\n")
	fmt.Fprintf(w, "| Table | Records |\n")
	fmt.Fprintf(w, "|-------|--------:|\n")
	tables := []string{"person", "visit_occurrence", "condition_occurrence", "drug_exposure",
		"procedure_occurrence", "measurement", "observation", "device_exposure"}
	totalRecords := 0
	for _, table := range tables {
		count := r.RecordsByTable[table]
		totalRecords += count
		fmt.Fprintf(w, "| %s | %d |\n", table, count)
	}
	fmt.Fprintf(w, "| **Total** | **%d** |\n", totalRecords)
	fmt.Fprintf(w, "\n")

	// Section to table mapping
	if len(r.EntriesBySection) > 0 {
		fmt.Fprintf(w, "## CCDA Section to OMOP Table Mapping\n\n")
		fmt.Fprintf(w, "| Section | Records | Target Tables |\n")
		fmt.Fprintf(w, "|---------|--------:|---------------|\n")

		sections := make([]string, 0, len(r.EntriesBySection))
		for section := range r.EntriesBySection {
			sections = append(sections, section)
		}
		sort.Strings(sections)

		for _, section := range sections {
			metrics := r.EntriesBySection[section]
			targetStr := formatTargetTables(metrics.TargetTables)
			fmt.Fprintf(w, "| %s | %d | %s |\n", section, metrics.RecordsCreated, targetStr)
		}
		fmt.Fprintf(w, "\n")
	}

	// Field population rates
	if len(r.FieldPopulation) > 0 {
		fmt.Fprintf(w, "## Field Population Rates\n\n")

		for _, table := range tables {
			fields := r.FieldPopulation[table]
			if len(fields) == 0 {
				continue
			}

			fmt.Fprintf(w, "### %s\n\n", table)
			fmt.Fprintf(w, "| Field | Populated | Total | Rate |\n")
			fmt.Fprintf(w, "|-------|----------:|------:|-----:|\n")

			fieldNames := make([]string, 0, len(fields))
			for name := range fields {
				fieldNames = append(fieldNames, name)
			}
			sort.Strings(fieldNames)

			for _, name := range fieldNames {
				stats := fields[name]
				rate := float64(0)
				if stats.Total > 0 {
					rate = float64(stats.Populated) / float64(stats.Total) * 100
				}
				fmt.Fprintf(w, "| %s | %d | %d | %.1f%% |\n", name, stats.Populated, stats.Total, rate)
			}
			fmt.Fprintf(w, "\n")
		}
	}

	// Concept mapping quality
	if len(r.ConceptMappings) > 0 {
		fmt.Fprintf(w, "## Concept Mapping Quality\n\n")
		fmt.Fprintf(w, "| Vocabulary | Codes Seen | Mapped Standard | Source Only | Rate |\n")
		fmt.Fprintf(w, "|------------|----------:|-----------------:|------------:|-----:|\n")

		vocabs := make([]string, 0, len(r.ConceptMappings))
		for vocab := range r.ConceptMappings {
			vocabs = append(vocabs, vocab)
		}
		sort.Strings(vocabs)

		for _, vocab := range vocabs {
			stats := r.ConceptMappings[vocab]
			rate := float64(0)
			if stats.CodesSeen > 0 {
				rate = float64(stats.MappedStandard) / float64(stats.CodesSeen) * 100
			}
			fmt.Fprintf(w, "| %s | %d | %d | %d | %.1f%% |\n",
				vocab, stats.CodesSeen, stats.MappedStandard, stats.SourceOnly, rate)
		}
		fmt.Fprintf(w, "\n")
	}

	// Domain routing
	if len(r.DomainRouting) > 0 {
		fmt.Fprintf(w, "## Domain Routing\n\n")
		fmt.Fprintf(w, "Records moved to different tables based on OMOP concept domain:\n\n")
		fmt.Fprintf(w, "| Source Section | Original Target | Actual Target | Count | Reason |\n")
		fmt.Fprintf(w, "|----------------|-----------------|---------------|------:|--------|\n")

		for _, route := range r.DomainRouting {
			fmt.Fprintf(w, "| %s | %s | %s | %d | %s |\n",
				route.SourceSection, route.OriginalTarget, route.ActualTarget, route.Count, route.Reason)
		}
		fmt.Fprintf(w, "\n")
	}

	// Skip reasons
	if len(r.SkippedEntries) > 0 {
		fmt.Fprintf(w, "## Skipped Entries\n\n")
		fmt.Fprintf(w, "| Reason | Count |\n")
		fmt.Fprintf(w, "|--------|------:|\n")

		reasons := make([]string, 0, len(r.SkippedEntries))
		for reason := range r.SkippedEntries {
			reasons = append(reasons, reason)
		}
		sort.Strings(reasons)

		for _, reason := range reasons {
			fmt.Fprintf(w, "| %s | %d |\n", reason, r.SkippedEntries[reason])
		}
		fmt.Fprintf(w, "\n")
	}

	return nil
}

// WriteJSON writes the report in JSON format
func (r *ConversionReport) WriteJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

func formatTargetTables(tables map[string]int) string {
	if len(tables) == 0 {
		return "-"
	}

	parts := make([]string, 0, len(tables))
	for table, count := range tables {
		parts = append(parts, fmt.Sprintf("%s(%d)", table, count))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
