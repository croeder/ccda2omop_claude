// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import (
	"fmt"
	"log"
	"time"

	"github.com/ccda2omop/internal/ccda"
	"github.com/ccda2omop/internal/omop"
)

// Mapper transforms C-CDA documents to OMOP CDM structures
type Mapper struct {
	vocab   *VocabularyMapper
	verbose bool
}

// New creates a new Mapper with placeholder vocabulary mappings
func New(verbose bool) *Mapper {
	return &Mapper{
		vocab:   NewVocabularyMapper(),
		verbose: verbose,
	}
}

// NewWithVocabLoader creates a Mapper with a vocabulary loader for real OMOP concept lookups
func NewWithVocabLoader(loader *VocabLoader, verbose bool) *Mapper {
	return &Mapper{
		vocab:   NewVocabularyMapperWithLoader(loader),
		verbose: verbose,
	}
}

// MapDocument transforms a C-CDA document to OMOP data
func (m *Mapper) MapDocument(doc *ccda.Document) (*omop.OMOPData, error) {
	data := &omop.OMOPData{}

	// Generate person ID from patient ID
	personID := omop.GeneratePersonID(doc.Patient.ID, "CCDA")

	// Map patient to person
	person := m.mapPerson(doc.Patient, personID)
	data.Persons = append(data.Persons, person)

	// Map encounters to visit_occurrence
	visitMap := make(map[string]int64) // encounter ID -> visit_occurrence_id
	for _, enc := range doc.Encounters {
		visit := m.mapEncounter(enc, personID)
		visitMap[enc.ID] = visit.VisitOccurrenceID
		data.VisitOccurrences = append(data.VisitOccurrences, visit)
	}
	if m.verbose {
		log.Printf("Mapped %d encounters", len(doc.Encounters))
	}

	// Map problems to condition_occurrence (may produce multiple records per problem)
	for _, prob := range doc.Problems {
		conditions := m.mapProblem(prob, personID, visitMap)
		data.ConditionOccurrences = append(data.ConditionOccurrences, conditions...)
	}
	if m.verbose {
		log.Printf("Mapped %d problems to %d condition records", len(doc.Problems), len(data.ConditionOccurrences))
	}

	// Map medications to drug_exposure (may produce multiple records)
	for _, med := range doc.Medications {
		drugs := m.mapMedication(med, personID, visitMap)
		data.DrugExposures = append(data.DrugExposures, drugs...)
	}
	if m.verbose {
		log.Printf("Mapped %d medications to %d drug records", len(doc.Medications), len(data.DrugExposures))
	}

	// Map immunizations to drug_exposure (may produce multiple records)
	immCount := len(data.DrugExposures)
	for _, imm := range doc.Immunizations {
		drugs := m.mapImmunization(imm, personID, visitMap)
		data.DrugExposures = append(data.DrugExposures, drugs...)
	}
	if m.verbose {
		log.Printf("Mapped %d immunizations to %d drug records", len(doc.Immunizations), len(data.DrugExposures)-immCount)
	}

	// Map procedures to procedure_occurrence (may produce multiple records)
	for _, proc := range doc.Procedures {
		procedures := m.mapProcedure(proc, personID, visitMap)
		data.ProcedureOccurrences = append(data.ProcedureOccurrences, procedures...)
	}
	if m.verbose {
		log.Printf("Mapped %d procedures to %d procedure records", len(doc.Procedures), len(data.ProcedureOccurrences))
	}

	// Map vital signs to measurement (may produce multiple records)
	for _, vital := range doc.VitalSigns {
		measurements := m.mapVitalSign(vital, personID, visitMap)
		data.Measurements = append(data.Measurements, measurements...)
	}
	if m.verbose {
		log.Printf("Mapped %d vital signs to %d measurement records", len(doc.VitalSigns), len(data.Measurements))
	}

	// Map lab results to measurement (may produce multiple records)
	labCount := len(data.Measurements)
	for _, lab := range doc.LabResults {
		measurements := m.mapLabResult(lab, personID, visitMap)
		data.Measurements = append(data.Measurements, measurements...)
	}
	if m.verbose {
		log.Printf("Mapped %d lab results to %d measurement records", len(doc.LabResults), len(data.Measurements)-labCount)
	}

	// Map allergies to observation (may produce multiple records)
	for _, allergy := range doc.Allergies {
		observations := m.mapAllergy(allergy, personID, visitMap)
		data.Observations = append(data.Observations, observations...)
	}
	if m.verbose {
		log.Printf("Mapped %d allergies to %d observation records", len(doc.Allergies), len(data.Observations))
	}

	// Map social observations to observation (may produce multiple records)
	socialCount := len(data.Observations)
	for _, obs := range doc.Observations {
		observations := m.mapSocialObservation(obs, personID, visitMap)
		data.Observations = append(data.Observations, observations...)
	}
	if m.verbose {
		log.Printf("Mapped %d social observations to %d observation records", len(doc.Observations), len(data.Observations)-socialCount)
	}

	// Map devices to device_exposure (may produce multiple records)
	for _, dev := range doc.Devices {
		devices := m.mapDevice(dev, personID, visitMap)
		data.DeviceExposures = append(data.DeviceExposures, devices...)
	}
	if m.verbose {
		log.Printf("Mapped %d devices to %d device records", len(doc.Devices), len(data.DeviceExposures))
	}

	return data, nil
}

func (m *Mapper) mapPerson(p ccda.Patient, personID int64) omop.Person {
	person := omop.Person{
		PersonID:           personID,
		GenderConceptID:    m.vocab.MapGender(p.Gender.Code),
		YearOfBirth:        p.BirthTime.Year(),
		RaceConceptID:      m.vocab.MapRace(p.Race.Code),
		EthnicityConceptID: m.vocab.MapEthnicity(p.Ethnicity.Code),
		PersonSourceValue:  p.ID,
		GenderSourceValue:  p.Gender.DisplayName,
		RaceSourceValue:    p.Race.DisplayName,
		EthnicitySourceValue: p.Ethnicity.DisplayName,
		MappingRule:        "DirectMapper:Person",
	}

	// Set month and day if available
	if !p.BirthTime.IsZero() {
		month := int(p.BirthTime.Month())
		day := p.BirthTime.Day()
		person.MonthOfBirth = &month
		person.DayOfBirth = &day
		person.BirthDatetime = &p.BirthTime
	}

	return person
}

func (m *Mapper) mapEncounter(enc ccda.Encounter, personID int64) omop.VisitOccurrence {
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
		VisitConceptID:     m.vocab.MapVisitType(enc.Code.Code),
		VisitStartDate:     startDate,
		VisitStartDatetime: timePtr(startDate),
		VisitEndDate:       endDate,
		VisitEndDatetime:   timePtr(endDate),
		VisitTypeConceptID: ConceptEHREncounter,
		VisitSourceValue:   enc.Code.DisplayName,
		MappingRule:        "DirectMapper:Encounter",
	}
}

func (m *Mapper) mapProblem(prob ccda.Problem, personID int64, visitMap map[string]int64) []omop.ConditionOccurrence {
	startDate := prob.EffectiveTime.Low
	if startDate.IsZero() {
		startDate = prob.EffectiveTime.Value
	}

	// Get all mapped concept IDs (a single source code may map to multiple standard concepts)
	conceptIDs := m.vocab.MapConditionCodes(prob.Code.Code, prob.Code.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0} // No mapping found
	}

	var conditions []omop.ConditionOccurrence
	for i, conceptID := range conceptIDs {
		// Generate unique ID for each mapping (append index for multiple mappings)
		baseID := omop.GenerateConditionID(personID, prob.Code.Code, startDate.Format("2006-01-02"))
		conditionID := baseID + int64(i)

		condition := omop.ConditionOccurrence{
			ConditionOccurrenceID:  conditionID,
			PersonID:               personID,
			ConditionConceptID:     conceptID,
			ConditionStartDate:     startDate,
			ConditionStartDatetime: timePtr(startDate),
			ConditionTypeConceptID: ConceptEHRProblemList,
			ConditionSourceValue:   formatSourceValue(prob.Code),
			MappingRule:            "DirectMapper:Problems",
		}

		if !prob.EffectiveTime.High.IsZero() {
			condition.ConditionEndDate = timePtr(prob.EffectiveTime.High)
			condition.ConditionEndDatetime = timePtr(prob.EffectiveTime.High)
		}

		conditions = append(conditions, condition)
	}

	return conditions
}

func (m *Mapper) mapMedication(med ccda.Medication, personID int64, visitMap map[string]int64) []omop.DrugExposure {
	startDate := med.EffectiveTime.Low
	if startDate.IsZero() {
		startDate = med.EffectiveTime.Value
	}
	if startDate.IsZero() {
		startDate = time.Now() // fallback
	}

	endDate := med.EffectiveTime.High
	if endDate.IsZero() {
		endDate = startDate
	}

	// Get all mapped concept IDs
	conceptIDs := m.vocab.MapDrugCodes(med.Code.Code, med.Code.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	var drugs []omop.DrugExposure
	for i, conceptID := range conceptIDs {
		baseID := omop.GenerateDrugExposureID(personID, med.Code.Code, startDate.Format("2006-01-02"))
		drugID := baseID + int64(i)

		drug := omop.DrugExposure{
			DrugExposureID:            drugID,
			PersonID:                  personID,
			DrugConceptID:             conceptID,
			DrugExposureStartDate:     startDate,
			DrugExposureStartDatetime: timePtr(startDate),
			DrugExposureEndDate:       endDate,
			DrugExposureEndDatetime:   timePtr(endDate),
			DrugTypeConceptID:         ConceptEHRPrescription,
			DrugSourceValue:           formatSourceValue(med.Code),
			RouteSourceValue:          med.RouteCode.DisplayName,
			MappingRule:               "DirectMapper:Medications",
		}

		if med.DoseQuantity.Value != 0 {
			qty := med.DoseQuantity.Value
			drug.Quantity = &qty
			drug.DoseUnitSourceValue = med.DoseQuantity.Unit
		}

		if med.DaysSupply > 0 {
			days := med.DaysSupply
			drug.DaysSupply = &days
		}

		if med.Refills > 0 {
			refills := med.Refills
			drug.Refills = &refills
		}

		drug.Sig = med.Instructions

		if med.RouteCode.Code != "" {
			routeID := m.vocab.MapRouteCode(med.RouteCode.Code, med.RouteCode.CodeSystem)
			drug.RouteConceptID = &routeID
		}

		drugs = append(drugs, drug)
	}

	return drugs
}

func (m *Mapper) mapImmunization(imm ccda.Immunization, personID int64, visitMap map[string]int64) []omop.DrugExposure {
	date := imm.EffectiveTime
	if date.IsZero() {
		date = time.Now()
	}

	// Get all mapped concept IDs
	conceptIDs := m.vocab.MapDrugCodes(imm.Code.Code, imm.Code.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	var drugs []omop.DrugExposure
	for i, conceptID := range conceptIDs {
		baseID := omop.GenerateDrugExposureID(personID, imm.Code.Code, date.Format("2006-01-02"))
		drugID := baseID + int64(i)

		drug := omop.DrugExposure{
			DrugExposureID:            drugID,
			PersonID:                  personID,
			DrugConceptID:             conceptID,
			DrugExposureStartDate:     date,
			DrugExposureStartDatetime: timePtr(date),
			DrugExposureEndDate:       date,
			DrugExposureEndDatetime:   timePtr(date),
			DrugTypeConceptID:         ConceptEHRPrescription,
			DrugSourceValue:           formatSourceValue(imm.Code),
			LotNumber:                 imm.LotNumber,
			RouteSourceValue:          imm.RouteCode.DisplayName,
			MappingRule:               "DirectMapper:Immunizations",
		}

		if imm.DoseQuantity.Value != 0 {
			qty := imm.DoseQuantity.Value
			drug.Quantity = &qty
			drug.DoseUnitSourceValue = imm.DoseQuantity.Unit
		}

		if imm.RouteCode.Code != "" {
			routeID := m.vocab.MapRouteCode(imm.RouteCode.Code, imm.RouteCode.CodeSystem)
			drug.RouteConceptID = &routeID
		}

		drugs = append(drugs, drug)
	}

	return drugs
}

func (m *Mapper) mapProcedure(proc ccda.Procedure, personID int64, visitMap map[string]int64) []omop.ProcedureOccurrence {
	date := proc.EffectiveTime.Low
	if date.IsZero() {
		date = proc.EffectiveTime.Value
	}
	if date.IsZero() {
		date = time.Now()
	}

	// Get all mapped concept IDs
	conceptIDs := m.vocab.MapProcedureCodes(proc.Code.Code, proc.Code.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	var procedures []omop.ProcedureOccurrence
	for i, conceptID := range conceptIDs {
		baseID := omop.GenerateProcedureID(personID, proc.Code.Code, date.Format("2006-01-02"))
		procID := baseID + int64(i)

		procedures = append(procedures, omop.ProcedureOccurrence{
			ProcedureOccurrenceID:  procID,
			PersonID:               personID,
			ProcedureConceptID:     conceptID,
			ProcedureDate:          date,
			ProcedureDatetime:      timePtr(date),
			ProcedureTypeConceptID: ConceptEHRProcedure,
			ProcedureSourceValue:   formatSourceValue(proc.Code),
			ModifierSourceValue:    proc.TargetSite.DisplayName,
			MappingRule:            "DirectMapper:Procedures",
		})
	}

	return procedures
}

func (m *Mapper) mapVitalSign(vital ccda.VitalSign, personID int64, visitMap map[string]int64) []omop.Measurement {
	date := vital.EffectiveTime
	if date.IsZero() {
		date = time.Now()
	}

	valueStr := ""
	if vital.Value != 0 {
		valueStr = formatFloat(vital.Value)
	}

	// Get all mapped concept IDs
	conceptIDs := m.vocab.MapMeasurementCodes(vital.Code.Code, vital.Code.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	var measurements []omop.Measurement
	for i, conceptID := range conceptIDs {
		baseID := omop.GenerateMeasurementID(personID, vital.Code.Code, date.Format("2006-01-02"), valueStr)
		measID := baseID + int64(i)

		meas := omop.Measurement{
			MeasurementID:            measID,
			PersonID:                 personID,
			MeasurementConceptID:     conceptID,
			MeasurementDate:          date,
			MeasurementDatetime:      timePtr(date),
			MeasurementTypeConceptID: ConceptEHRObservation,
			MeasurementSourceValue:   formatSourceValue(vital.Code),
			UnitSourceValue:          vital.Unit,
			ValueSourceValue:         valueStr,
			MappingRule:              "DirectMapper:VitalSigns",
		}

		if vital.Value != 0 {
			meas.ValueAsNumber = &vital.Value
		}

		if vital.Unit != "" {
			unitID := m.vocab.MapUnitCode(vital.Unit)
			meas.UnitConceptID = &unitID
		}

		measurements = append(measurements, meas)
	}

	return measurements
}

func (m *Mapper) mapLabResult(lab ccda.LabResult, personID int64, visitMap map[string]int64) []omop.Measurement {
	date := lab.EffectiveTime
	if date.IsZero() {
		date = time.Now()
	}

	valueStr := lab.ValueString
	if lab.Value != 0 {
		valueStr = formatFloat(lab.Value)
	}

	// Get all mapped concept IDs
	conceptIDs := m.vocab.MapMeasurementCodes(lab.Code.Code, lab.Code.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	var measurements []omop.Measurement
	for i, conceptID := range conceptIDs {
		baseID := omop.GenerateMeasurementID(personID, lab.Code.Code, date.Format("2006-01-02"), valueStr)
		measID := baseID + int64(i)

		meas := omop.Measurement{
			MeasurementID:            measID,
			PersonID:                 personID,
			MeasurementConceptID:     conceptID,
			MeasurementDate:          date,
			MeasurementDatetime:      timePtr(date),
			MeasurementTypeConceptID: ConceptEHRObservation,
			MeasurementSourceValue:   formatSourceValue(lab.Code),
			UnitSourceValue:          lab.Unit,
			ValueSourceValue:         valueStr,
			MappingRule:              "DirectMapper:LabResults",
		}

		if lab.Value != 0 {
			meas.ValueAsNumber = &lab.Value
		}

		if lab.Unit != "" {
			unitID := m.vocab.MapUnitCode(lab.Unit)
			meas.UnitConceptID = &unitID
		}

		if lab.ReferenceRange.Low != 0 {
			meas.RangeLow = &lab.ReferenceRange.Low
		}
		if lab.ReferenceRange.High != 0 {
			meas.RangeHigh = &lab.ReferenceRange.High
		}

		measurements = append(measurements, meas)
	}

	return measurements
}

func (m *Mapper) mapAllergy(allergy ccda.Allergy, personID int64, visitMap map[string]int64) []omop.Observation {
	date := allergy.EffectiveTime.Low
	if date.IsZero() {
		date = allergy.EffectiveTime.Value
	}
	if date.IsZero() {
		date = time.Now()
	}

	// Get all mapped concept IDs
	conceptIDs := m.vocab.MapObservationCodes(allergy.Substance.Code, allergy.Substance.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	var observations []omop.Observation
	for i, conceptID := range conceptIDs {
		baseID := omop.GenerateObservationID(personID, allergy.Substance.Code, date.Format("2006-01-02"))
		obsID := baseID + int64(i)

		obs := omop.Observation{
			ObservationID:            obsID,
			PersonID:                 personID,
			ObservationConceptID:     conceptID,
			ObservationDate:          date,
			ObservationDatetime:      timePtr(date),
			ObservationTypeConceptID: ConceptEHRObservation,
			ObservationSourceValue:   formatSourceValue(allergy.Substance),
			ValueAsString:            allergy.Reaction.DisplayName,
			QualifierSourceValue:     allergy.Severity.DisplayName,
			MappingRule:              "DirectMapper:Allergies",
		}

		observations = append(observations, obs)
	}

	return observations
}

func (m *Mapper) mapSocialObservation(socialObs ccda.SocialObservation, personID int64, visitMap map[string]int64) []omop.Observation {
	date := socialObs.EffectiveTime.Low
	if date.IsZero() {
		date = socialObs.EffectiveTime.Value
	}
	if date.IsZero() {
		date = time.Now()
	}

	// Get all mapped concept IDs
	conceptIDs := m.vocab.MapObservationCodes(socialObs.Code.Code, socialObs.Code.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	var observations []omop.Observation
	for i, conceptID := range conceptIDs {
		baseID := omop.GenerateObservationID(personID, socialObs.Code.Code, date.Format("2006-01-02"))
		obsID := baseID + int64(i)

		obs := omop.Observation{
			ObservationID:            obsID,
			PersonID:                 personID,
			ObservationConceptID:     conceptID,
			ObservationDate:          date,
			ObservationDatetime:      timePtr(date),
			ObservationTypeConceptID: ConceptEHRObservation,
			ObservationSourceValue:   formatSourceValue(socialObs.Code),
			MappingRule:              "DirectMapper:SocialHistory",
		}

		if socialObs.Value.Code != "" {
			obs.ValueAsString = socialObs.Value.DisplayName
		} else if socialObs.ValueQuantity.Value != 0 {
			obs.ValueAsNumber = &socialObs.ValueQuantity.Value
			obs.UnitSourceValue = socialObs.ValueQuantity.Unit
		}

		observations = append(observations, obs)
	}

	return observations
}

func (m *Mapper) mapDevice(dev ccda.Device, personID int64, visitMap map[string]int64) []omop.DeviceExposure {
	startDate := dev.EffectiveTime.Low
	if startDate.IsZero() {
		startDate = dev.EffectiveTime.Value
	}
	if startDate.IsZero() {
		startDate = time.Now()
	}

	// Get all mapped concept IDs
	conceptIDs := m.vocab.MapDeviceCodes(dev.Code.Code, dev.Code.CodeSystem)
	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	var devices []omop.DeviceExposure
	for i, conceptID := range conceptIDs {
		baseID := omop.GenerateDeviceExposureID(personID, dev.Code.Code, startDate.Format("2006-01-02"))
		deviceID := baseID + int64(i)

		device := omop.DeviceExposure{
			DeviceExposureID:            deviceID,
			PersonID:                    personID,
			DeviceConceptID:             conceptID,
			DeviceExposureStartDate:     startDate,
			DeviceExposureStartDatetime: timePtr(startDate),
			DeviceTypeConceptID:         ConceptEHRObservation,
			UniqueDeviceID:              dev.UDI,
			DeviceSourceValue:           formatSourceValue(dev.Code),
			MappingRule:                 "DirectMapper:Devices",
		}

		if !dev.EffectiveTime.High.IsZero() {
			device.DeviceExposureEndDate = timePtr(dev.EffectiveTime.High)
			device.DeviceExposureEndDatetime = timePtr(dev.EffectiveTime.High)
		}

		devices = append(devices, device)
	}

	return devices
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func formatSourceValue(cv ccda.CodedValue) string {
	if cv.DisplayName != "" {
		return cv.DisplayName
	}
	if cv.Code != "" {
		return cv.Code
	}
	return cv.OriginalText
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%g", f)
}
