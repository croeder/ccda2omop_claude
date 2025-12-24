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

// New creates a new Mapper
func New(verbose bool) *Mapper {
	return &Mapper{
		vocab:   NewVocabularyMapper(),
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

	// Map problems to condition_occurrence
	for _, prob := range doc.Problems {
		condition := m.mapProblem(prob, personID, visitMap)
		data.ConditionOccurrences = append(data.ConditionOccurrences, condition)
	}
	if m.verbose {
		log.Printf("Mapped %d problems", len(doc.Problems))
	}

	// Map medications to drug_exposure
	for _, med := range doc.Medications {
		drug := m.mapMedication(med, personID, visitMap)
		data.DrugExposures = append(data.DrugExposures, drug)
	}
	if m.verbose {
		log.Printf("Mapped %d medications", len(doc.Medications))
	}

	// Map immunizations to drug_exposure
	for _, imm := range doc.Immunizations {
		drug := m.mapImmunization(imm, personID, visitMap)
		data.DrugExposures = append(data.DrugExposures, drug)
	}
	if m.verbose {
		log.Printf("Mapped %d immunizations", len(doc.Immunizations))
	}

	// Map procedures to procedure_occurrence
	for _, proc := range doc.Procedures {
		procedure := m.mapProcedure(proc, personID, visitMap)
		data.ProcedureOccurrences = append(data.ProcedureOccurrences, procedure)
	}
	if m.verbose {
		log.Printf("Mapped %d procedures", len(doc.Procedures))
	}

	// Map vital signs to measurement
	for _, vital := range doc.VitalSigns {
		measurement := m.mapVitalSign(vital, personID, visitMap)
		data.Measurements = append(data.Measurements, measurement)
	}
	if m.verbose {
		log.Printf("Mapped %d vital signs", len(doc.VitalSigns))
	}

	// Map lab results to measurement
	for _, lab := range doc.LabResults {
		measurement := m.mapLabResult(lab, personID, visitMap)
		data.Measurements = append(data.Measurements, measurement)
	}
	if m.verbose {
		log.Printf("Mapped %d lab results", len(doc.LabResults))
	}

	// Map allergies to observation
	for _, allergy := range doc.Allergies {
		observation := m.mapAllergy(allergy, personID, visitMap)
		data.Observations = append(data.Observations, observation)
	}
	if m.verbose {
		log.Printf("Mapped %d allergies", len(doc.Allergies))
	}

	// Map social observations to observation
	for _, obs := range doc.Observations {
		observation := m.mapSocialObservation(obs, personID, visitMap)
		data.Observations = append(data.Observations, observation)
	}
	if m.verbose {
		log.Printf("Mapped %d observations", len(doc.Observations))
	}

	// Map devices to device_exposure
	for _, dev := range doc.Devices {
		device := m.mapDevice(dev, personID, visitMap)
		data.DeviceExposures = append(data.DeviceExposures, device)
	}
	if m.verbose {
		log.Printf("Mapped %d devices", len(doc.Devices))
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
	}
}

func (m *Mapper) mapProblem(prob ccda.Problem, personID int64, visitMap map[string]int64) omop.ConditionOccurrence {
	startDate := prob.EffectiveTime.Low
	if startDate.IsZero() {
		startDate = prob.EffectiveTime.Value
	}

	conditionID := omop.GenerateConditionID(personID, prob.Code.Code, startDate.Format("2006-01-02"))

	condition := omop.ConditionOccurrence{
		ConditionOccurrenceID:  conditionID,
		PersonID:               personID,
		ConditionConceptID:     m.vocab.MapConditionCode(prob.Code.Code, prob.Code.CodeSystem),
		ConditionStartDate:     startDate,
		ConditionStartDatetime: timePtr(startDate),
		ConditionTypeConceptID: ConceptEHRProblemList,
		ConditionSourceValue:   formatSourceValue(prob.Code),
	}

	if !prob.EffectiveTime.High.IsZero() {
		condition.ConditionEndDate = timePtr(prob.EffectiveTime.High)
		condition.ConditionEndDatetime = timePtr(prob.EffectiveTime.High)
	}

	return condition
}

func (m *Mapper) mapMedication(med ccda.Medication, personID int64, visitMap map[string]int64) omop.DrugExposure {
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

	drugID := omop.GenerateDrugExposureID(personID, med.Code.Code, startDate.Format("2006-01-02"))

	drug := omop.DrugExposure{
		DrugExposureID:            drugID,
		PersonID:                  personID,
		DrugConceptID:             m.vocab.MapDrugCode(med.Code.Code, med.Code.CodeSystem),
		DrugExposureStartDate:     startDate,
		DrugExposureStartDatetime: timePtr(startDate),
		DrugExposureEndDate:       endDate,
		DrugExposureEndDatetime:   timePtr(endDate),
		DrugTypeConceptID:         ConceptEHRPrescription,
		DrugSourceValue:           formatSourceValue(med.Code),
		RouteSourceValue:          med.RouteCode.DisplayName,
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

	return drug
}

func (m *Mapper) mapImmunization(imm ccda.Immunization, personID int64, visitMap map[string]int64) omop.DrugExposure {
	date := imm.EffectiveTime
	if date.IsZero() {
		date = time.Now()
	}

	drugID := omop.GenerateDrugExposureID(personID, imm.Code.Code, date.Format("2006-01-02"))

	drug := omop.DrugExposure{
		DrugExposureID:            drugID,
		PersonID:                  personID,
		DrugConceptID:             m.vocab.MapDrugCode(imm.Code.Code, imm.Code.CodeSystem),
		DrugExposureStartDate:     date,
		DrugExposureStartDatetime: timePtr(date),
		DrugExposureEndDate:       date,
		DrugExposureEndDatetime:   timePtr(date),
		DrugTypeConceptID:         ConceptEHRPrescription,
		DrugSourceValue:           formatSourceValue(imm.Code),
		LotNumber:                 imm.LotNumber,
		RouteSourceValue:          imm.RouteCode.DisplayName,
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

	return drug
}

func (m *Mapper) mapProcedure(proc ccda.Procedure, personID int64, visitMap map[string]int64) omop.ProcedureOccurrence {
	date := proc.EffectiveTime.Low
	if date.IsZero() {
		date = proc.EffectiveTime.Value
	}
	if date.IsZero() {
		date = time.Now()
	}

	procID := omop.GenerateProcedureID(personID, proc.Code.Code, date.Format("2006-01-02"))

	return omop.ProcedureOccurrence{
		ProcedureOccurrenceID:  procID,
		PersonID:               personID,
		ProcedureConceptID:     m.vocab.MapProcedureCode(proc.Code.Code, proc.Code.CodeSystem),
		ProcedureDate:          date,
		ProcedureDatetime:      timePtr(date),
		ProcedureTypeConceptID: ConceptEHRProcedure,
		ProcedureSourceValue:   formatSourceValue(proc.Code),
		ModifierSourceValue:    proc.TargetSite.DisplayName,
	}
}

func (m *Mapper) mapVitalSign(vital ccda.VitalSign, personID int64, visitMap map[string]int64) omop.Measurement {
	date := vital.EffectiveTime
	if date.IsZero() {
		date = time.Now()
	}

	valueStr := ""
	if vital.Value != 0 {
		valueStr = formatFloat(vital.Value)
	}

	measID := omop.GenerateMeasurementID(personID, vital.Code.Code, date.Format("2006-01-02"), valueStr)

	meas := omop.Measurement{
		MeasurementID:            measID,
		PersonID:                 personID,
		MeasurementConceptID:     m.vocab.MapMeasurementCode(vital.Code.Code, vital.Code.CodeSystem),
		MeasurementDate:          date,
		MeasurementDatetime:      timePtr(date),
		MeasurementTypeConceptID: ConceptEHRObservation,
		MeasurementSourceValue:   formatSourceValue(vital.Code),
		UnitSourceValue:          vital.Unit,
		ValueSourceValue:         valueStr,
	}

	if vital.Value != 0 {
		meas.ValueAsNumber = &vital.Value
	}

	if vital.Unit != "" {
		unitID := m.vocab.MapUnitCode(vital.Unit)
		meas.UnitConceptID = &unitID
	}

	return meas
}

func (m *Mapper) mapLabResult(lab ccda.LabResult, personID int64, visitMap map[string]int64) omop.Measurement {
	date := lab.EffectiveTime
	if date.IsZero() {
		date = time.Now()
	}

	valueStr := lab.ValueString
	if lab.Value != 0 {
		valueStr = formatFloat(lab.Value)
	}

	measID := omop.GenerateMeasurementID(personID, lab.Code.Code, date.Format("2006-01-02"), valueStr)

	meas := omop.Measurement{
		MeasurementID:            measID,
		PersonID:                 personID,
		MeasurementConceptID:     m.vocab.MapMeasurementCode(lab.Code.Code, lab.Code.CodeSystem),
		MeasurementDate:          date,
		MeasurementDatetime:      timePtr(date),
		MeasurementTypeConceptID: ConceptEHRObservation,
		MeasurementSourceValue:   formatSourceValue(lab.Code),
		UnitSourceValue:          lab.Unit,
		ValueSourceValue:         valueStr,
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

	return meas
}

func (m *Mapper) mapAllergy(allergy ccda.Allergy, personID int64, visitMap map[string]int64) omop.Observation {
	date := allergy.EffectiveTime.Low
	if date.IsZero() {
		date = allergy.EffectiveTime.Value
	}
	if date.IsZero() {
		date = time.Now()
	}

	obsID := omop.GenerateObservationID(personID, allergy.Substance.Code, date.Format("2006-01-02"))

	obs := omop.Observation{
		ObservationID:            obsID,
		PersonID:                 personID,
		ObservationConceptID:     m.vocab.MapObservationCode(allergy.Substance.Code, allergy.Substance.CodeSystem),
		ObservationDate:          date,
		ObservationDatetime:      timePtr(date),
		ObservationTypeConceptID: ConceptEHRObservation,
		ObservationSourceValue:   formatSourceValue(allergy.Substance),
		ValueAsString:            allergy.Reaction.DisplayName,
		QualifierSourceValue:     allergy.Severity.DisplayName,
	}

	return obs
}

func (m *Mapper) mapSocialObservation(socialObs ccda.SocialObservation, personID int64, visitMap map[string]int64) omop.Observation {
	date := socialObs.EffectiveTime.Low
	if date.IsZero() {
		date = socialObs.EffectiveTime.Value
	}
	if date.IsZero() {
		date = time.Now()
	}

	obsID := omop.GenerateObservationID(personID, socialObs.Code.Code, date.Format("2006-01-02"))

	obs := omop.Observation{
		ObservationID:            obsID,
		PersonID:                 personID,
		ObservationConceptID:     m.vocab.MapObservationCode(socialObs.Code.Code, socialObs.Code.CodeSystem),
		ObservationDate:          date,
		ObservationDatetime:      timePtr(date),
		ObservationTypeConceptID: ConceptEHRObservation,
		ObservationSourceValue:   formatSourceValue(socialObs.Code),
	}

	if socialObs.Value.Code != "" {
		obs.ValueAsString = socialObs.Value.DisplayName
	} else if socialObs.ValueQuantity.Value != 0 {
		obs.ValueAsNumber = &socialObs.ValueQuantity.Value
		obs.UnitSourceValue = socialObs.ValueQuantity.Unit
	}

	return obs
}

func (m *Mapper) mapDevice(dev ccda.Device, personID int64, visitMap map[string]int64) omop.DeviceExposure {
	startDate := dev.EffectiveTime.Low
	if startDate.IsZero() {
		startDate = dev.EffectiveTime.Value
	}
	if startDate.IsZero() {
		startDate = time.Now()
	}

	deviceID := omop.GenerateDeviceExposureID(personID, dev.Code.Code, startDate.Format("2006-01-02"))

	device := omop.DeviceExposure{
		DeviceExposureID:            deviceID,
		PersonID:                    personID,
		DeviceConceptID:             m.vocab.MapDeviceCode(dev.Code.Code, dev.Code.CodeSystem),
		DeviceExposureStartDate:     startDate,
		DeviceExposureStartDatetime: timePtr(startDate),
		DeviceTypeConceptID:         ConceptEHRObservation,
		UniqueDeviceID:              dev.UDI,
		DeviceSourceValue:           formatSourceValue(dev.Code),
	}

	if !dev.EffectiveTime.High.IsZero() {
		device.DeviceExposureEndDate = timePtr(dev.EffectiveTime.High)
		device.DeviceExposureEndDatetime = timePtr(dev.EffectiveTime.High)
	}

	return device
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
