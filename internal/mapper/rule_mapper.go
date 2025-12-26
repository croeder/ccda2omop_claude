// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import (
	"log"
	"time"

	"github.com/ccda2omop/internal/ccda"
	"github.com/ccda2omop/internal/omop"
)

// RuleBasedMapper uses declarative rules to transform C-CDA documents to OMOP
type RuleBasedMapper struct {
	engine  *RuleEngine
	rules   []MappingRule
	verbose bool
}

// NewRuleBasedMapper creates a new rule-based mapper with default Go-defined rules
func NewRuleBasedMapper(vocab *VocabularyMapper, verbose bool) *RuleBasedMapper {
	return &RuleBasedMapper{
		engine:  NewRuleEngine(vocab, verbose),
		rules:   AllRules, // Use Go-defined rules as default
		verbose: verbose,
	}
}

// NewRuleBasedMapperWithLoader creates a rule-based mapper with vocabulary loader
func NewRuleBasedMapperWithLoader(loader *VocabLoader, verbose bool) *RuleBasedMapper {
	vocab := NewVocabularyMapperWithLoader(loader)
	return NewRuleBasedMapper(vocab, verbose)
}

// NewRuleBasedMapperFromYAML creates a rule-based mapper with rules loaded from YAML
func NewRuleBasedMapperFromYAML(rulesFile string, vocab *VocabularyMapper, verbose bool) (*RuleBasedMapper, error) {
	rules, err := LoadRulesFromYAML(rulesFile)
	if err != nil {
		return nil, err
	}
	return &RuleBasedMapper{
		engine:  NewRuleEngine(vocab, verbose),
		rules:   rules,
		verbose: verbose,
	}, nil
}

// NewRuleBasedMapperFromYAMLWithLoader creates a rule-based mapper with YAML rules and vocab loader
func NewRuleBasedMapperFromYAMLWithLoader(rulesFile string, loader *VocabLoader, verbose bool) (*RuleBasedMapper, error) {
	vocab := NewVocabularyMapperWithLoader(loader)
	return NewRuleBasedMapperFromYAML(rulesFile, vocab, verbose)
}

// MapDocument transforms a C-CDA document to OMOP data using rules
func (m *RuleBasedMapper) MapDocument(doc *ccda.Document) (*omop.OMOPData, error) {
	data := &omop.OMOPData{}

	// Generate person ID
	personID := omop.GeneratePersonID(doc.Patient.ID, "CCDA")

	// Map patient (still uses direct mapping - person is special)
	person := m.mapPerson(doc.Patient, personID)
	data.Persons = append(data.Persons, person)

	// Map encounters (still direct - visits are special)
	visitMap := make(map[string]int64)
	for _, enc := range doc.Encounters {
		visit := m.mapEncounter(enc, personID)
		visitMap[enc.ID] = visit.VisitOccurrenceID
		data.VisitOccurrences = append(data.VisitOccurrences, visit)
	}
	if m.verbose {
		log.Printf("Mapped %d encounters", len(doc.Encounters))
	}

	// Map problems using rules (respecting section metadata for optionality)
	if rule := m.getRuleBySection("Problems"); rule != nil {
		conditions, err := m.mapWithRuleAndMeta(*rule, doc.Problems, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		for _, c := range conditions {
			data.ConditionOccurrences = append(data.ConditionOccurrences, m.toConditionOccurrence(c))
		}
		if m.verbose {
			log.Printf("Mapped %d problems to %d condition records (rule-based)", len(doc.Problems), len(conditions))
		}
	}

	// Map medications using rules
	if rule := m.getRuleBySection("Medications"); rule != nil {
		drugs, err := m.mapWithRuleAndMeta(*rule, doc.Medications, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		for _, d := range drugs {
			data.DrugExposures = append(data.DrugExposures, m.toDrugExposure(d))
		}
		if m.verbose {
			log.Printf("Mapped %d medications to %d drug records (rule-based)", len(doc.Medications), len(drugs))
		}
	}

	// Map immunizations using rules
	if rule := m.getRuleBySection("Immunizations"); rule != nil {
		imms, err := m.mapWithRuleAndMeta(*rule, doc.Immunizations, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		for _, d := range imms {
			data.DrugExposures = append(data.DrugExposures, m.toDrugExposure(d))
		}
		if m.verbose {
			log.Printf("Mapped %d immunizations to %d drug records (rule-based)", len(doc.Immunizations), len(imms))
		}
	}

	// Map procedures using rules
	if rule := m.getRuleBySection("Procedures"); rule != nil {
		procs, err := m.mapWithRuleAndMeta(*rule, doc.Procedures, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		for _, p := range procs {
			data.ProcedureOccurrences = append(data.ProcedureOccurrences, m.toProcedureOccurrence(p))
		}
		if m.verbose {
			log.Printf("Mapped %d procedures to %d procedure records (rule-based)", len(doc.Procedures), len(procs))
		}
	}

	// Map vital signs using rules
	if rule := m.getRuleBySection("VitalSigns"); rule != nil {
		vitals, err := m.mapWithRuleAndMeta(*rule, doc.VitalSigns, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		for _, v := range vitals {
			data.Measurements = append(data.Measurements, m.toMeasurement(v))
		}
		if m.verbose {
			log.Printf("Mapped %d vital signs to %d measurement records (rule-based)", len(doc.VitalSigns), len(vitals))
		}
	}

	// Map lab results using rules with domain-aware routing
	if rule := m.getRuleBySection("LabResults"); rule != nil {
		labs, err := m.mapWithRuleAndMeta(*rule, doc.LabResults, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		measurementCount := 0
		observationCount := 0
		for _, l := range labs {
			conceptID := getInt64(l, "measurement_concept_id")
			domain := m.engine.vocab.GetConceptDomain(conceptID)
			if domain == "Observation" {
				// Route to observation table
				data.Observations = append(data.Observations, m.labToObservation(l))
				observationCount++
			} else {
				// Default to measurement table
				data.Measurements = append(data.Measurements, m.toMeasurement(l))
				measurementCount++
			}
		}
		if m.verbose {
			log.Printf("Mapped %d lab results: %d to measurement, %d to observation (domain-aware)", len(doc.LabResults), measurementCount, observationCount)
		}
	}

	// Map allergies using rules
	if rule := m.getRuleBySection("Allergies"); rule != nil {
		allergies, err := m.mapWithRuleAndMeta(*rule, doc.Allergies, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		for _, a := range allergies {
			data.Observations = append(data.Observations, m.toObservation(a))
		}
		if m.verbose {
			log.Printf("Mapped %d allergies to %d observation records (rule-based)", len(doc.Allergies), len(allergies))
		}
	}

	// Map social observations using rules
	if rule := m.getRuleBySection("Observations"); rule != nil {
		socialObs, err := m.mapWithRuleAndMeta(*rule, doc.Observations, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		for _, o := range socialObs {
			data.Observations = append(data.Observations, m.toObservation(o))
		}
		if m.verbose {
			log.Printf("Mapped %d social observations to %d observation records (rule-based)", len(doc.Observations), len(socialObs))
		}
	}

	// Map devices using rules
	if rule := m.getRuleBySection("Devices"); rule != nil {
		devices, err := m.mapWithRuleAndMeta(*rule, doc.Devices, personID, visitMap, doc.SectionMeta)
		if err != nil {
			return nil, err
		}
		for _, d := range devices {
			data.DeviceExposures = append(data.DeviceExposures, m.toDeviceExposure(d))
		}
		if m.verbose {
			log.Printf("Mapped %d devices to %d device records (rule-based)", len(doc.Devices), len(devices))
		}
	}

	return data, nil
}

// getRuleBySection returns a rule by section name from the loaded rules
func (m *RuleBasedMapper) getRuleBySection(section string) *MappingRule {
	for i := range m.rules {
		if m.rules[i].Source.Section == section {
			return &m.rules[i]
		}
	}
	return nil
}

// mapWithRule applies a rule to a slice of entries
func (m *RuleBasedMapper) mapWithRule(rule MappingRule, entries interface{}, personID int64, visitMap map[string]int64) ([]map[string]interface{}, error) {
	return m.engine.MapEntries(rule, entries, personID, visitMap)
}

// mapWithRuleAndMeta applies a rule to a slice of entries, using section metadata for optionality
func (m *RuleBasedMapper) mapWithRuleAndMeta(rule MappingRule, entries interface{}, personID int64, visitMap map[string]int64, sectionMeta map[string]ccda.SectionMetadata) ([]map[string]interface{}, error) {
	// Check if entries are required for this section
	entriesRequired := true
	if meta, ok := sectionMeta[rule.Source.Section]; ok {
		entriesRequired = meta.EntriesRequired
	}
	return m.engine.MapEntriesWithOptional(rule, entries, personID, visitMap, entriesRequired)
}

// Conversion functions from map to OMOP structs

func (m *RuleBasedMapper) toConditionOccurrence(record map[string]interface{}) omop.ConditionOccurrence {
	c := omop.ConditionOccurrence{
		ConditionOccurrenceID:  getInt64(record, "condition_occurrence_id"),
		PersonID:               getInt64(record, "person_id"),
		ConditionConceptID:     getInt64(record, "condition_concept_id"),
		ConditionTypeConceptID: getInt64(record, "condition_type_concept_id"),
		ConditionSourceValue:   getString(record, "condition_source_value"),
		MappingRule:            getString(record, "mapping_rule"),
	}

	if v, ok := record["condition_start_date"].(time.Time); ok {
		c.ConditionStartDate = v
	}
	if v := getTimePtr(record, "condition_start_datetime"); v != nil {
		c.ConditionStartDatetime = v
	}
	if v := getTimePtr(record, "condition_end_date"); v != nil {
		c.ConditionEndDate = v
	}
	if v := getTimePtr(record, "condition_end_datetime"); v != nil {
		c.ConditionEndDatetime = v
	}

	return c
}

func (m *RuleBasedMapper) toDrugExposure(record map[string]interface{}) omop.DrugExposure {
	d := omop.DrugExposure{
		DrugExposureID:       getInt64(record, "drug_exposure_id"),
		PersonID:             getInt64(record, "person_id"),
		DrugConceptID:        getInt64(record, "drug_concept_id"),
		DrugTypeConceptID:    getInt64(record, "drug_type_concept_id"),
		DrugSourceValue:      getString(record, "drug_source_value"),
		RouteSourceValue:     getString(record, "route_source_value"),
		LotNumber:            getString(record, "lot_number"),
		Sig:                  getString(record, "sig"),
		DoseUnitSourceValue:  getString(record, "dose_unit_source_value"),
		MappingRule:          getString(record, "mapping_rule"),
	}

	if v, ok := record["drug_exposure_start_date"].(time.Time); ok {
		d.DrugExposureStartDate = v
	}
	if v := getTimePtr(record, "drug_exposure_start_datetime"); v != nil {
		d.DrugExposureStartDatetime = v
	}
	if v, ok := record["drug_exposure_end_date"].(time.Time); ok {
		d.DrugExposureEndDate = v
	}
	if v := getTimePtr(record, "drug_exposure_end_datetime"); v != nil {
		d.DrugExposureEndDatetime = v
	}
	if v := getFloat64Ptr(record, "quantity"); v != nil {
		d.Quantity = v
	}
	if v := getIntPtr(record, "days_supply"); v != nil {
		d.DaysSupply = v
	}
	if v := getIntPtr(record, "refills"); v != nil {
		d.Refills = v
	}
	if v := getInt64Ptr(record, "route_concept_id"); v != nil {
		d.RouteConceptID = v
	}

	return d
}

func (m *RuleBasedMapper) toProcedureOccurrence(record map[string]interface{}) omop.ProcedureOccurrence {
	p := omop.ProcedureOccurrence{
		ProcedureOccurrenceID:  getInt64(record, "procedure_occurrence_id"),
		PersonID:               getInt64(record, "person_id"),
		ProcedureConceptID:     getInt64(record, "procedure_concept_id"),
		ProcedureTypeConceptID: getInt64(record, "procedure_type_concept_id"),
		ProcedureSourceValue:   getString(record, "procedure_source_value"),
		ModifierSourceValue:    getString(record, "modifier_source_value"),
		MappingRule:            getString(record, "mapping_rule"),
	}

	if v, ok := record["procedure_date"].(time.Time); ok {
		p.ProcedureDate = v
	}
	if v := getTimePtr(record, "procedure_datetime"); v != nil {
		p.ProcedureDatetime = v
	}

	return p
}

func (m *RuleBasedMapper) toMeasurement(record map[string]interface{}) omop.Measurement {
	meas := omop.Measurement{
		MeasurementID:            getInt64(record, "measurement_id"),
		PersonID:                 getInt64(record, "person_id"),
		MeasurementConceptID:     getInt64(record, "measurement_concept_id"),
		MeasurementTypeConceptID: getInt64(record, "measurement_type_concept_id"),
		MeasurementSourceValue:   getString(record, "measurement_source_value"),
		UnitSourceValue:          getString(record, "unit_source_value"),
		ValueSourceValue:         getString(record, "value_source_value"),
		MappingRule:              getString(record, "mapping_rule"),
	}

	if v, ok := record["measurement_date"].(time.Time); ok {
		meas.MeasurementDate = v
	}
	if v := getTimePtr(record, "measurement_datetime"); v != nil {
		meas.MeasurementDatetime = v
	}
	if v := getFloat64Ptr(record, "value_as_number"); v != nil {
		meas.ValueAsNumber = v
	}
	if v := getInt64Ptr(record, "value_as_concept_id"); v != nil {
		meas.ValueAsConceptID = v
	}
	if v := getInt64Ptr(record, "unit_concept_id"); v != nil {
		meas.UnitConceptID = v
	}
	if v := getFloat64Ptr(record, "range_low"); v != nil {
		meas.RangeLow = v
	}
	if v := getFloat64Ptr(record, "range_high"); v != nil {
		meas.RangeHigh = v
	}

	return meas
}

// labToObservation converts a lab result (mapped as measurement) to an observation
// Used for domain-aware routing when lab concept's domain is "Observation"
func (m *RuleBasedMapper) labToObservation(record map[string]interface{}) omop.Observation {
	obs := omop.Observation{
		ObservationID:            getInt64(record, "measurement_id"),
		PersonID:                 getInt64(record, "person_id"),
		ObservationConceptID:     getInt64(record, "measurement_concept_id"),
		ObservationTypeConceptID: getInt64(record, "measurement_type_concept_id"),
		ObservationSourceValue:   getString(record, "measurement_source_value"),
		UnitSourceValue:          getString(record, "unit_source_value"),
		MappingRule:              getString(record, "mapping_rule") + ":DomainRouted",
	}

	if v, ok := record["measurement_date"].(time.Time); ok {
		obs.ObservationDate = v
	}
	if v := getTimePtr(record, "measurement_datetime"); v != nil {
		obs.ObservationDatetime = v
	}
	if v := getFloat64Ptr(record, "value_as_number"); v != nil {
		obs.ValueAsNumber = v
	}
	if v := getInt64Ptr(record, "value_as_concept_id"); v != nil {
		obs.ValueAsConceptID = v
	}
	// Convert value_source_value to value_as_string for observation
	if v := getString(record, "value_source_value"); v != "" {
		obs.ValueAsString = v
	}

	return obs
}

func (m *RuleBasedMapper) toObservation(record map[string]interface{}) omop.Observation {
	obs := omop.Observation{
		ObservationID:            getInt64(record, "observation_id"),
		PersonID:                 getInt64(record, "person_id"),
		ObservationConceptID:     getInt64(record, "observation_concept_id"),
		ObservationTypeConceptID: getInt64(record, "observation_type_concept_id"),
		ObservationSourceValue:   getString(record, "observation_source_value"),
		ValueAsString:            getString(record, "value_as_string"),
		QualifierSourceValue:     getString(record, "qualifier_source_value"),
		UnitSourceValue:          getString(record, "unit_source_value"),
		MappingRule:              getString(record, "mapping_rule"),
	}

	if v, ok := record["observation_date"].(time.Time); ok {
		obs.ObservationDate = v
	}
	if v := getTimePtr(record, "observation_datetime"); v != nil {
		obs.ObservationDatetime = v
	}
	if v := getFloat64Ptr(record, "value_as_number"); v != nil {
		obs.ValueAsNumber = v
	}
	if v := getInt64Ptr(record, "value_as_concept_id"); v != nil {
		obs.ValueAsConceptID = v
	}

	return obs
}

func (m *RuleBasedMapper) toDeviceExposure(record map[string]interface{}) omop.DeviceExposure {
	d := omop.DeviceExposure{
		DeviceExposureID:    getInt64(record, "device_exposure_id"),
		PersonID:            getInt64(record, "person_id"),
		DeviceConceptID:     getInt64(record, "device_concept_id"),
		DeviceTypeConceptID: getInt64(record, "device_type_concept_id"),
		UniqueDeviceID:      getString(record, "unique_device_id"),
		DeviceSourceValue:   getString(record, "device_source_value"),
		MappingRule:         getString(record, "mapping_rule"),
	}

	if v, ok := record["device_exposure_start_date"].(time.Time); ok {
		d.DeviceExposureStartDate = v
	}
	if v := getTimePtr(record, "device_exposure_start_datetime"); v != nil {
		d.DeviceExposureStartDatetime = v
	}
	if v := getTimePtr(record, "device_exposure_end_date"); v != nil {
		d.DeviceExposureEndDate = v
	}
	if v := getTimePtr(record, "device_exposure_end_datetime"); v != nil {
		d.DeviceExposureEndDatetime = v
	}

	return d
}

// Person and Encounter mapping (kept as direct mapping since they're special)

func (m *RuleBasedMapper) mapPerson(p ccda.Patient, personID int64) omop.Person {
	person := omop.Person{
		PersonID:             personID,
		GenderConceptID:      m.engine.vocab.MapGender(p.Gender.Code),
		YearOfBirth:          p.BirthTime.Year(),
		RaceConceptID:        m.engine.vocab.MapRace(p.Race.Code),
		EthnicityConceptID:   m.engine.vocab.MapEthnicity(p.Ethnicity.Code),
		PersonSourceValue:    p.ID,
		GenderSourceValue:    p.Gender.DisplayName,
		RaceSourceValue:      p.Race.DisplayName,
		EthnicitySourceValue: p.Ethnicity.DisplayName,
		MappingRule:          "RuleMapper:Person",
	}

	if !p.BirthTime.IsZero() {
		month := int(p.BirthTime.Month())
		day := p.BirthTime.Day()
		person.MonthOfBirth = &month
		person.DayOfBirth = &day
		person.BirthDatetime = &p.BirthTime
	}

	return person
}

func (m *RuleBasedMapper) mapEncounter(enc ccda.Encounter, personID int64) omop.VisitOccurrence {
	visitID := omop.GenerateVisitID(personID, enc.ID)

	startDate := enc.EffectiveTime.Low
	if startDate.IsZero() {
		startDate = enc.EffectiveTime.Value
	}

	endDate := enc.EffectiveTime.High
	if endDate.IsZero() {
		endDate = startDate
	}

	return omop.VisitOccurrence{
		VisitOccurrenceID:  visitID,
		PersonID:           personID,
		VisitConceptID:     m.engine.vocab.MapVisitType(enc.Code.Code),
		VisitStartDate:     startDate,
		VisitStartDatetime: timePtrHelper(startDate),
		VisitEndDate:       endDate,
		VisitEndDatetime:   timePtrHelper(endDate),
		VisitTypeConceptID: ConceptEHREncounter,
		VisitSourceValue:   enc.Code.DisplayName,
		MappingRule:        "RuleMapper:Encounter",
	}
}

func timePtrHelper(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// Helper functions for extracting values from maps

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case float64:
			return int64(val)
		}
	}
	return 0
}

func getInt64Ptr(m map[string]interface{}, key string) *int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return &val
		case int:
			i := int64(val)
			return &i
		case float64:
			i := int64(val)
			return &i
		}
	}
	return nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat64Ptr(m map[string]interface{}, key string) *float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return &val
		case float32:
			f := float64(val)
			return &f
		case int:
			f := float64(val)
			return &f
		case int64:
			f := float64(val)
			return &f
		}
	}
	return nil
}

func getIntPtr(m map[string]interface{}, key string) *int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return &val
		case int64:
			i := int(val)
			return &i
		case float64:
			i := int(val)
			return &i
		}
	}
	return nil
}

func getTimePtr(m map[string]interface{}, key string) *time.Time {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case time.Time:
			if !val.IsZero() {
				return &val
			}
		case *time.Time:
			return val
		}
	}
	return nil
}
