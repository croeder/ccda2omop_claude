// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

// Mapping rules for C-CDA to OMOP CDM conversion
// Each rule defines how to map a C-CDA section to an OMOP table

var (
	// ProblemRule maps Problems to condition_occurrence
	ProblemRule = MappingRule{
		Name: "problems_to_conditions",
		Source: SourceSpec{
			Section:   "Problems",
			EntryType: "Problem",
		},
		Target: TargetSpec{
			Table:         "condition_occurrence",
			TypeConceptID: ConceptEHRProblemList,
		},
		Fields: []FieldMapping{
			{Source: "Code.Code", Target: "condition_concept_id", Transform: "vocab", VocabField: "Code.CodeSystem"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "condition_start_date", Transform: "date"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "condition_start_datetime", Transform: "time_ptr"},
			{Source: "EffectiveTime.High", Target: "condition_end_date", Transform: "time_ptr", Optional: true},
			{Source: "EffectiveTime.High", Target: "condition_end_datetime", Transform: "time_ptr", Optional: true},
			{Source: "Code", Target: "condition_source_value", Transform: "format_source"},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Code.Code", "EffectiveTime.Low"},
			Generator:  "condition",
		},
	}

	// MedicationRule maps Medications to drug_exposure
	MedicationRule = MappingRule{
		Name: "medications_to_drugs",
		Source: SourceSpec{
			Section:   "Medications",
			EntryType: "Medication",
		},
		Target: TargetSpec{
			Table:         "drug_exposure",
			TypeConceptID: ConceptEHRPrescription,
		},
		Fields: []FieldMapping{
			{Source: "Code.Code", Target: "drug_concept_id", Transform: "vocab", VocabField: "Code.CodeSystem"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "drug_exposure_start_date", Transform: "date"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "drug_exposure_start_datetime", Transform: "time_ptr"},
			{Source: "EffectiveTime.High|EffectiveTime.Low|EffectiveTime.Value", Target: "drug_exposure_end_date", Transform: "date"},
			{Source: "EffectiveTime.High|EffectiveTime.Low|EffectiveTime.Value", Target: "drug_exposure_end_datetime", Transform: "time_ptr"},
			{Source: "Code", Target: "drug_source_value", Transform: "format_source"},
			{Source: "RouteCode.DisplayName", Target: "route_source_value", Transform: "string", Optional: true},
			{Source: "RouteCode.Code", Target: "route_concept_id", Transform: "route", VocabField: "RouteCode.CodeSystem", Optional: true},
			{Source: "DoseQuantity.Value", Target: "quantity", Transform: "float", Optional: true},
			{Source: "DoseQuantity.Unit", Target: "dose_unit_source_value", Transform: "string", Optional: true},
			{Source: "DaysSupply", Target: "days_supply", Transform: "int", Optional: true},
			{Source: "Refills", Target: "refills", Transform: "int", Optional: true},
			{Source: "Instructions", Target: "sig", Transform: "string", Optional: true},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Code.Code", "EffectiveTime.Low"},
			Generator:  "drug",
		},
	}

	// ImmunizationRule maps Immunizations to drug_exposure
	ImmunizationRule = MappingRule{
		Name: "immunizations_to_drugs",
		Source: SourceSpec{
			Section:   "Immunizations",
			EntryType: "Immunization",
		},
		Target: TargetSpec{
			Table:         "drug_exposure",
			TypeConceptID: ConceptEHRPrescription,
		},
		Fields: []FieldMapping{
			{Source: "Code.Code", Target: "drug_concept_id", Transform: "vocab", VocabField: "Code.CodeSystem"},
			{Source: "EffectiveTime", Target: "drug_exposure_start_date", Transform: "date"},
			{Source: "EffectiveTime", Target: "drug_exposure_start_datetime", Transform: "time_ptr"},
			{Source: "EffectiveTime", Target: "drug_exposure_end_date", Transform: "date"},
			{Source: "EffectiveTime", Target: "drug_exposure_end_datetime", Transform: "time_ptr"},
			{Source: "Code", Target: "drug_source_value", Transform: "format_source"},
			{Source: "LotNumber", Target: "lot_number", Transform: "string", Optional: true},
			{Source: "RouteCode.DisplayName", Target: "route_source_value", Transform: "string", Optional: true},
			{Source: "RouteCode.Code", Target: "route_concept_id", Transform: "route", VocabField: "RouteCode.CodeSystem", Optional: true},
			{Source: "DoseQuantity.Value", Target: "quantity", Transform: "float", Optional: true},
			{Source: "DoseQuantity.Unit", Target: "dose_unit_source_value", Transform: "string", Optional: true},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Code.Code", "EffectiveTime"},
			Generator:  "drug",
		},
	}

	// ProcedureRule maps Procedures to procedure_occurrence
	ProcedureRule = MappingRule{
		Name: "procedures_to_procedures",
		Source: SourceSpec{
			Section:   "Procedures",
			EntryType: "Procedure",
		},
		Target: TargetSpec{
			Table:         "procedure_occurrence",
			TypeConceptID: ConceptEHRProcedure,
		},
		Fields: []FieldMapping{
			{Source: "Code.Code", Target: "procedure_concept_id", Transform: "vocab", VocabField: "Code.CodeSystem"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "procedure_date", Transform: "date"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "procedure_datetime", Transform: "time_ptr"},
			{Source: "Code", Target: "procedure_source_value", Transform: "format_source"},
			{Source: "TargetSite.DisplayName", Target: "modifier_source_value", Transform: "string", Optional: true},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Code.Code", "EffectiveTime.Low"},
			Generator:  "procedure",
		},
	}

	// VitalSignRule maps VitalSigns to measurement
	VitalSignRule = MappingRule{
		Name: "vitals_to_measurements",
		Source: SourceSpec{
			Section:   "VitalSigns",
			EntryType: "VitalSign",
		},
		Target: TargetSpec{
			Table:         "measurement",
			TypeConceptID: ConceptEHRObservation,
		},
		Fields: []FieldMapping{
			{Source: "Code.Code", Target: "measurement_concept_id", Transform: "vocab", VocabField: "Code.CodeSystem"},
			{Source: "EffectiveTime", Target: "measurement_date", Transform: "date"},
			{Source: "EffectiveTime", Target: "measurement_datetime", Transform: "time_ptr"},
			{Source: "Code", Target: "measurement_source_value", Transform: "format_source"},
			{Source: "Value", Target: "value_as_number", Transform: "float", Optional: true},
			{Source: "Unit", Target: "unit_source_value", Transform: "string", Optional: true},
			{Source: "Unit", Target: "unit_concept_id", Transform: "unit", Optional: true},
			{Source: "Value", Target: "value_source_value", Transform: "string", Optional: true},
			{Source: "Interpretation.Code", Target: "value_as_concept_id", Transform: "value_vocab", VocabField: "Interpretation.CodeSystem", Optional: true},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Code.Code", "EffectiveTime", "Value"},
			Generator:  "measurement",
		},
	}

	// LabResultRule maps LabResults to measurement
	LabResultRule = MappingRule{
		Name: "labs_to_measurements",
		Source: SourceSpec{
			Section:   "LabResults",
			EntryType: "LabResult",
		},
		Target: TargetSpec{
			Table:         "measurement",
			TypeConceptID: ConceptEHRObservation,
		},
		Fields: []FieldMapping{
			{Source: "Code.Code", Target: "measurement_concept_id", Transform: "vocab", VocabField: "Code.CodeSystem"},
			{Source: "EffectiveTime", Target: "measurement_date", Transform: "date"},
			{Source: "EffectiveTime", Target: "measurement_datetime", Transform: "time_ptr"},
			{Source: "Code", Target: "measurement_source_value", Transform: "format_source"},
			{Source: "Value", Target: "value_as_number", Transform: "float", Optional: true},
			{Source: "Value|ValueString", Target: "value_source_value", Transform: "string", Optional: true},
			{Source: "Unit", Target: "unit_source_value", Transform: "string", Optional: true},
			{Source: "Unit", Target: "unit_concept_id", Transform: "unit", Optional: true},
			{Source: "ReferenceRange.Low", Target: "range_low", Transform: "float", Optional: true},
			{Source: "ReferenceRange.High", Target: "range_high", Transform: "float", Optional: true},
			{Source: "Interpretation.Code", Target: "value_as_concept_id", Transform: "value_vocab", VocabField: "Interpretation.CodeSystem", Optional: true},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Code.Code", "EffectiveTime", "Value"},
			Generator:  "measurement",
		},
	}

	// AllergyRule maps Allergies to observation
	AllergyRule = MappingRule{
		Name: "allergies_to_observations",
		Source: SourceSpec{
			Section:   "Allergies",
			EntryType: "Allergy",
		},
		Target: TargetSpec{
			Table:         "observation",
			TypeConceptID: ConceptEHRObservation,
		},
		Fields: []FieldMapping{
			{Source: "Substance.Code|Code.Code", Target: "observation_concept_id", Transform: "vocab", VocabField: "Substance.CodeSystem|Code.CodeSystem", Optional: true},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "observation_date", Transform: "date", Optional: true},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "observation_datetime", Transform: "time_ptr", Optional: true},
			{Source: "Substance|Code", Target: "observation_source_value", Transform: "format_source", Optional: true},
			{Source: "Reaction.DisplayName", Target: "value_as_string", Transform: "string", Optional: true},
			{Source: "Reaction.Code", Target: "value_as_concept_id", Transform: "value_vocab", VocabField: "Reaction.CodeSystem", Optional: true},
			{Source: "Severity.DisplayName", Target: "qualifier_source_value", Transform: "string", Optional: true},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Substance.Code", "EffectiveTime.Low"},
			Generator:  "observation",
		},
	}

	// SocialObservationRule maps SocialHistory observations to observation
	SocialObservationRule = MappingRule{
		Name: "social_to_observations",
		Source: SourceSpec{
			Section:   "Observations",
			EntryType: "SocialObservation",
		},
		Target: TargetSpec{
			Table:         "observation",
			TypeConceptID: ConceptEHRObservation,
		},
		Fields: []FieldMapping{
			{Source: "Code.Code", Target: "observation_concept_id", Transform: "vocab", VocabField: "Code.CodeSystem"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "observation_date", Transform: "date"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "observation_datetime", Transform: "time_ptr"},
			{Source: "Code", Target: "observation_source_value", Transform: "format_source"},
			{Source: "Value.DisplayName", Target: "value_as_string", Transform: "string", Optional: true},
			{Source: "Value.Code", Target: "value_as_concept_id", Transform: "value_vocab", VocabField: "Value.CodeSystem", Optional: true},
			{Source: "ValueQuantity.Value", Target: "value_as_number", Transform: "float", Optional: true},
			{Source: "ValueQuantity.Unit", Target: "unit_source_value", Transform: "string", Optional: true},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Code.Code", "EffectiveTime.Low"},
			Generator:  "observation",
		},
	}

	// DeviceRule maps Devices to device_exposure
	DeviceRule = MappingRule{
		Name: "devices_to_device_exposure",
		Source: SourceSpec{
			Section:   "Devices",
			EntryType: "Device",
		},
		Target: TargetSpec{
			Table:         "device_exposure",
			TypeConceptID: ConceptEHRObservation,
		},
		Fields: []FieldMapping{
			{Source: "Code.Code", Target: "device_concept_id", Transform: "vocab", VocabField: "Code.CodeSystem"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "device_exposure_start_date", Transform: "date"},
			{Source: "EffectiveTime.Low|EffectiveTime.Value", Target: "device_exposure_start_datetime", Transform: "time_ptr"},
			{Source: "EffectiveTime.High", Target: "device_exposure_end_date", Transform: "time_ptr", Optional: true},
			{Source: "EffectiveTime.High", Target: "device_exposure_end_datetime", Transform: "time_ptr", Optional: true},
			{Source: "Code", Target: "device_source_value", Transform: "format_source"},
			{Source: "UDI", Target: "unique_device_id", Transform: "string", Optional: true},
		},
		IDGen: IDGenSpec{
			BaseFields: []string{"Code.Code", "EffectiveTime.Low"},
			Generator:  "device",
		},
	}

	// AllRules contains all mapping rules
	AllRules = []MappingRule{
		ProblemRule,
		MedicationRule,
		ImmunizationRule,
		ProcedureRule,
		VitalSignRule,
		LabResultRule,
		AllergyRule,
		SocialObservationRule,
		DeviceRule,
	}
)

// GetRuleBySection returns the mapping rule for a given section name
func GetRuleBySection(section string) *MappingRule {
	for _, rule := range AllRules {
		if rule.Source.Section == section {
			return &rule
		}
	}
	return nil
}

// GetRuleByName returns the mapping rule by its name
func GetRuleByName(name string) *MappingRule {
	for _, rule := range AllRules {
		if rule.Name == name {
			return &rule
		}
	}
	return nil
}
