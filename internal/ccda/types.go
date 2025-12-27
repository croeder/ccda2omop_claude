// Copyright 2025 Christophe Roeder. All rights reserved.

package ccda

import (
	"encoding/xml"
	"time"

	"github.com/antchfx/xmlquery"
)

// SectionMetadata contains metadata about a parsed C-CDA section
type SectionMetadata struct {
	TemplateOID     string // The template OID that was matched
	EntriesRequired bool   // True if the section uses "entries required" template (OID ends in .1)
}

// Document represents a parsed C-CDA clinical document
type Document struct {
	XMLName       xml.Name
	Patient       Patient
	Author        Author
	Custodian     Custodian
	Encounters    []Encounter
	Problems      []Problem
	Medications   []Medication
	Procedures    []Procedure
	VitalSigns    []VitalSign
	LabResults    []LabResult
	Allergies     []Allergy
	Immunizations []Immunization
	Devices       []Device
	Observations  []SocialObservation

	// Section metadata - tracks which template type was used for each section
	SectionMeta map[string]SectionMetadata

	// XMLRoot stores the raw XML root node for xpath-based extraction
	XMLRoot *xmlquery.Node
}

// Patient represents patient demographics from the C-CDA recordTarget
type Patient struct {
	ID            string
	Name          Name
	BirthTime     time.Time
	Gender        CodedValue
	Race          CodedValue
	Ethnicity     CodedValue
	Address       Address
	Telecom       []Telecom
	MaritalStatus CodedValue
	Language      CodedValue
}

// Name represents a person's name
type Name struct {
	Given  string
	Family string
	Suffix string
	Prefix string
}

// Address represents a postal address
type Address struct {
	StreetAddress []string
	City          string
	State         string
	PostalCode    string
	Country       string
}

// Telecom represents a telecommunication address (phone, email, etc.)
type Telecom struct {
	Use   string
	Value string
}

// Author represents the document author
type Author struct {
	Time         time.Time
	ID           string
	Name         Name
	Organization string
}

// Custodian represents the document custodian organization
type Custodian struct {
	ID           string
	Name         string
	Address      Address
	Telecom      Telecom
}

// CodedValue represents a coded entry with code system information
type CodedValue struct {
	Code           string
	CodeSystem     string
	CodeSystemName string
	DisplayName    string
	OriginalText   string
}

// EffectiveTime represents a time or time range
type EffectiveTime struct {
	Low   time.Time
	High  time.Time
	Value time.Time
}

// Encounter represents an encounter from the Encounters section
type Encounter struct {
	ID            string
	Code          CodedValue
	EffectiveTime EffectiveTime
	Performer     string
	Location      string
	DischargeCode CodedValue
}

// Problem represents a problem/condition from the Problems section
type Problem struct {
	ID            string
	Code          CodedValue
	EffectiveTime EffectiveTime
	Status        CodedValue
	Severity      CodedValue
}

// Medication represents a medication from the Medications section
type Medication struct {
	ID            string
	Code          CodedValue
	EffectiveTime EffectiveTime
	DoseQuantity  Quantity
	RateQuantity  Quantity
	RouteCode     CodedValue
	Status        CodedValue
	Instructions  string
	Refills       int
	DaysSupply    int
}

// Quantity represents a quantity with unit
type Quantity struct {
	Value float64
	Unit  string
}

// Procedure represents a procedure from the Procedures section
type Procedure struct {
	ID            string
	Code          CodedValue
	EffectiveTime EffectiveTime
	Status        CodedValue
	TargetSite    CodedValue
	Performer     string
}

// VitalSign represents a vital sign measurement
type VitalSign struct {
	ID            string
	Code          CodedValue
	EffectiveTime time.Time
	Value         float64
	Unit          string
	Interpretation CodedValue
}

// LabResult represents a laboratory result
type LabResult struct {
	ID             string
	Code           CodedValue
	EffectiveTime  time.Time
	Value          float64
	ValueString    string
	Unit           string
	ReferenceRange ReferenceRange
	Interpretation CodedValue
	Status         CodedValue
}

// ReferenceRange represents a lab result reference range
type ReferenceRange struct {
	Low  float64
	High float64
	Text string
}

// Allergy represents an allergy or intolerance
type Allergy struct {
	ID            string
	Code          CodedValue
	EffectiveTime EffectiveTime
	Status        CodedValue
	Severity      CodedValue
	Reaction      CodedValue
	Substance     CodedValue
}

// Immunization represents an immunization administration
type Immunization struct {
	ID            string
	Code          CodedValue
	EffectiveTime time.Time
	Status        CodedValue
	RouteCode     CodedValue
	DoseQuantity  Quantity
	LotNumber     string
	Manufacturer  string
}

// Device represents a medical device
type Device struct {
	ID            string
	Code          CodedValue
	EffectiveTime EffectiveTime
	Status        CodedValue
	UDI           string // Unique Device Identifier
}

// SocialObservation represents a social history observation
type SocialObservation struct {
	ID            string
	Code          CodedValue
	EffectiveTime EffectiveTime
	Value         CodedValue
	ValueQuantity Quantity
	Status        CodedValue
}
