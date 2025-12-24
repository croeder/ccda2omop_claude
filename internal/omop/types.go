package omop

import "time"

// OMOPData holds all OMOP CDM 5.3 tables generated from a C-CDA document
type OMOPData struct {
	Persons              []Person
	VisitOccurrences     []VisitOccurrence
	ConditionOccurrences []ConditionOccurrence
	DrugExposures        []DrugExposure
	ProcedureOccurrences []ProcedureOccurrence
	Measurements         []Measurement
	Observations         []Observation
	DeviceExposures      []DeviceExposure
}

// Person represents the OMOP CDM 5.3 PERSON table
type Person struct {
	PersonID                   int64      `csv:"person_id"`
	GenderConceptID            int64      `csv:"gender_concept_id"`
	YearOfBirth                int        `csv:"year_of_birth"`
	MonthOfBirth               *int       `csv:"month_of_birth"`
	DayOfBirth                 *int       `csv:"day_of_birth"`
	BirthDatetime              *time.Time `csv:"birth_datetime"`
	RaceConceptID              int64      `csv:"race_concept_id"`
	EthnicityConceptID         int64      `csv:"ethnicity_concept_id"`
	LocationID                 *int64     `csv:"location_id"`
	ProviderID                 *int64     `csv:"provider_id"`
	CareSiteID                 *int64     `csv:"care_site_id"`
	PersonSourceValue          string     `csv:"person_source_value"`
	GenderSourceValue          string     `csv:"gender_source_value"`
	GenderSourceConceptID      *int64     `csv:"gender_source_concept_id"`
	RaceSourceValue            string     `csv:"race_source_value"`
	RaceSourceConceptID        *int64     `csv:"race_source_concept_id"`
	EthnicitySourceValue       string     `csv:"ethnicity_source_value"`
	EthnicitySourceConceptID   *int64     `csv:"ethnicity_source_concept_id"`
}

// VisitOccurrence represents the OMOP CDM 5.3 VISIT_OCCURRENCE table
type VisitOccurrence struct {
	VisitOccurrenceID           int64      `csv:"visit_occurrence_id"`
	PersonID                    int64      `csv:"person_id"`
	VisitConceptID              int64      `csv:"visit_concept_id"`
	VisitStartDate              time.Time  `csv:"visit_start_date"`
	VisitStartDatetime          *time.Time `csv:"visit_start_datetime"`
	VisitEndDate                time.Time  `csv:"visit_end_date"`
	VisitEndDatetime            *time.Time `csv:"visit_end_datetime"`
	VisitTypeConceptID          int64      `csv:"visit_type_concept_id"`
	ProviderID                  *int64     `csv:"provider_id"`
	CareSiteID                  *int64     `csv:"care_site_id"`
	VisitSourceValue            string     `csv:"visit_source_value"`
	VisitSourceConceptID        *int64     `csv:"visit_source_concept_id"`
	AdmittedFromConceptID       *int64     `csv:"admitted_from_concept_id"`
	AdmittedFromSourceValue     string     `csv:"admitted_from_source_value"`
	DischargeToConceptID        *int64     `csv:"discharge_to_concept_id"`
	DischargeToSourceValue      string     `csv:"discharge_to_source_value"`
	PrecedingVisitOccurrenceID  *int64     `csv:"preceding_visit_occurrence_id"`
}

// ConditionOccurrence represents the OMOP CDM 5.3 CONDITION_OCCURRENCE table
type ConditionOccurrence struct {
	ConditionOccurrenceID       int64      `csv:"condition_occurrence_id"`
	PersonID                    int64      `csv:"person_id"`
	ConditionConceptID          int64      `csv:"condition_concept_id"`
	ConditionStartDate          time.Time  `csv:"condition_start_date"`
	ConditionStartDatetime      *time.Time `csv:"condition_start_datetime"`
	ConditionEndDate            *time.Time `csv:"condition_end_date"`
	ConditionEndDatetime        *time.Time `csv:"condition_end_datetime"`
	ConditionTypeConceptID      int64      `csv:"condition_type_concept_id"`
	ConditionStatusConceptID    *int64     `csv:"condition_status_concept_id"`
	StopReason                  string     `csv:"stop_reason"`
	ProviderID                  *int64     `csv:"provider_id"`
	VisitOccurrenceID           *int64     `csv:"visit_occurrence_id"`
	VisitDetailID               *int64     `csv:"visit_detail_id"`
	ConditionSourceValue        string     `csv:"condition_source_value"`
	ConditionSourceConceptID    *int64     `csv:"condition_source_concept_id"`
	ConditionStatusSourceValue  string     `csv:"condition_status_source_value"`
}

// DrugExposure represents the OMOP CDM 5.3 DRUG_EXPOSURE table
type DrugExposure struct {
	DrugExposureID             int64      `csv:"drug_exposure_id"`
	PersonID                   int64      `csv:"person_id"`
	DrugConceptID              int64      `csv:"drug_concept_id"`
	DrugExposureStartDate      time.Time  `csv:"drug_exposure_start_date"`
	DrugExposureStartDatetime  *time.Time `csv:"drug_exposure_start_datetime"`
	DrugExposureEndDate        time.Time  `csv:"drug_exposure_end_date"`
	DrugExposureEndDatetime    *time.Time `csv:"drug_exposure_end_datetime"`
	VerbatimEndDate            *time.Time `csv:"verbatim_end_date"`
	DrugTypeConceptID          int64      `csv:"drug_type_concept_id"`
	StopReason                 string     `csv:"stop_reason"`
	Refills                    *int       `csv:"refills"`
	Quantity                   *float64   `csv:"quantity"`
	DaysSupply                 *int       `csv:"days_supply"`
	Sig                        string     `csv:"sig"`
	RouteConceptID             *int64     `csv:"route_concept_id"`
	LotNumber                  string     `csv:"lot_number"`
	ProviderID                 *int64     `csv:"provider_id"`
	VisitOccurrenceID          *int64     `csv:"visit_occurrence_id"`
	VisitDetailID              *int64     `csv:"visit_detail_id"`
	DrugSourceValue            string     `csv:"drug_source_value"`
	DrugSourceConceptID        *int64     `csv:"drug_source_concept_id"`
	RouteSourceValue           string     `csv:"route_source_value"`
	DoseUnitSourceValue        string     `csv:"dose_unit_source_value"`
}

// ProcedureOccurrence represents the OMOP CDM 5.3 PROCEDURE_OCCURRENCE table
type ProcedureOccurrence struct {
	ProcedureOccurrenceID     int64      `csv:"procedure_occurrence_id"`
	PersonID                  int64      `csv:"person_id"`
	ProcedureConceptID        int64      `csv:"procedure_concept_id"`
	ProcedureDate             time.Time  `csv:"procedure_date"`
	ProcedureDatetime         *time.Time `csv:"procedure_datetime"`
	ProcedureTypeConceptID    int64      `csv:"procedure_type_concept_id"`
	ModifierConceptID         *int64     `csv:"modifier_concept_id"`
	Quantity                  *int       `csv:"quantity"`
	ProviderID                *int64     `csv:"provider_id"`
	VisitOccurrenceID         *int64     `csv:"visit_occurrence_id"`
	VisitDetailID             *int64     `csv:"visit_detail_id"`
	ProcedureSourceValue      string     `csv:"procedure_source_value"`
	ProcedureSourceConceptID  *int64     `csv:"procedure_source_concept_id"`
	ModifierSourceValue       string     `csv:"modifier_source_value"`
}

// Measurement represents the OMOP CDM 5.3 MEASUREMENT table
type Measurement struct {
	MeasurementID             int64      `csv:"measurement_id"`
	PersonID                  int64      `csv:"person_id"`
	MeasurementConceptID      int64      `csv:"measurement_concept_id"`
	MeasurementDate           time.Time  `csv:"measurement_date"`
	MeasurementDatetime       *time.Time `csv:"measurement_datetime"`
	MeasurementTime           string     `csv:"measurement_time"`
	MeasurementTypeConceptID  int64      `csv:"measurement_type_concept_id"`
	OperatorConceptID         *int64     `csv:"operator_concept_id"`
	ValueAsNumber             *float64   `csv:"value_as_number"`
	ValueAsConceptID          *int64     `csv:"value_as_concept_id"`
	UnitConceptID             *int64     `csv:"unit_concept_id"`
	RangeLow                  *float64   `csv:"range_low"`
	RangeHigh                 *float64   `csv:"range_high"`
	ProviderID                *int64     `csv:"provider_id"`
	VisitOccurrenceID         *int64     `csv:"visit_occurrence_id"`
	VisitDetailID             *int64     `csv:"visit_detail_id"`
	MeasurementSourceValue    string     `csv:"measurement_source_value"`
	MeasurementSourceConceptID *int64    `csv:"measurement_source_concept_id"`
	UnitSourceValue           string     `csv:"unit_source_value"`
	ValueSourceValue          string     `csv:"value_source_value"`
}

// Observation represents the OMOP CDM 5.3 OBSERVATION table
type Observation struct {
	ObservationID             int64      `csv:"observation_id"`
	PersonID                  int64      `csv:"person_id"`
	ObservationConceptID      int64      `csv:"observation_concept_id"`
	ObservationDate           time.Time  `csv:"observation_date"`
	ObservationDatetime       *time.Time `csv:"observation_datetime"`
	ObservationTypeConceptID  int64      `csv:"observation_type_concept_id"`
	ValueAsNumber             *float64   `csv:"value_as_number"`
	ValueAsString             string     `csv:"value_as_string"`
	ValueAsConceptID          *int64     `csv:"value_as_concept_id"`
	QualifierConceptID        *int64     `csv:"qualifier_concept_id"`
	UnitConceptID             *int64     `csv:"unit_concept_id"`
	ProviderID                *int64     `csv:"provider_id"`
	VisitOccurrenceID         *int64     `csv:"visit_occurrence_id"`
	VisitDetailID             *int64     `csv:"visit_detail_id"`
	ObservationSourceValue    string     `csv:"observation_source_value"`
	ObservationSourceConceptID *int64    `csv:"observation_source_concept_id"`
	UnitSourceValue           string     `csv:"unit_source_value"`
	QualifierSourceValue      string     `csv:"qualifier_source_value"`
}

// DeviceExposure represents the OMOP CDM 5.3 DEVICE_EXPOSURE table
type DeviceExposure struct {
	DeviceExposureID           int64      `csv:"device_exposure_id"`
	PersonID                   int64      `csv:"person_id"`
	DeviceConceptID            int64      `csv:"device_concept_id"`
	DeviceExposureStartDate    time.Time  `csv:"device_exposure_start_date"`
	DeviceExposureStartDatetime *time.Time `csv:"device_exposure_start_datetime"`
	DeviceExposureEndDate      *time.Time `csv:"device_exposure_end_date"`
	DeviceExposureEndDatetime  *time.Time `csv:"device_exposure_end_datetime"`
	DeviceTypeConceptID        int64      `csv:"device_type_concept_id"`
	UniqueDeviceID             string     `csv:"unique_device_id"`
	Quantity                   *int       `csv:"quantity"`
	ProviderID                 *int64     `csv:"provider_id"`
	VisitOccurrenceID          *int64     `csv:"visit_occurrence_id"`
	VisitDetailID              *int64     `csv:"visit_detail_id"`
	DeviceSourceValue          string     `csv:"device_source_value"`
	DeviceSourceConceptID      *int64     `csv:"device_source_concept_id"`
}
