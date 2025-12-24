package ccda

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// C-CDA Section Template OIDs
const (
	OIDEncounters        = "2.16.840.1.113883.10.20.22.2.22"
	OIDEncountersEntriesReq = "2.16.840.1.113883.10.20.22.2.22.1"
	OIDProblems          = "2.16.840.1.113883.10.20.22.2.5"
	OIDProblemsEntriesReq = "2.16.840.1.113883.10.20.22.2.5.1"
	OIDMedications       = "2.16.840.1.113883.10.20.22.2.1"
	OIDMedicationsEntriesReq = "2.16.840.1.113883.10.20.22.2.1.1"
	OIDProcedures        = "2.16.840.1.113883.10.20.22.2.7"
	OIDProceduresEntriesReq = "2.16.840.1.113883.10.20.22.2.7.1"
	OIDVitalSigns        = "2.16.840.1.113883.10.20.22.2.4"
	OIDVitalSignsEntriesReq = "2.16.840.1.113883.10.20.22.2.4.1"
	OIDResults           = "2.16.840.1.113883.10.20.22.2.3"
	OIDResultsEntriesReq = "2.16.840.1.113883.10.20.22.2.3.1"
	OIDAllergies         = "2.16.840.1.113883.10.20.22.2.6"
	OIDAllergiesEntriesReq = "2.16.840.1.113883.10.20.22.2.6.1"
	OIDImmunizations     = "2.16.840.1.113883.10.20.22.2.2"
	OIDImmunizationsEntriesReq = "2.16.840.1.113883.10.20.22.2.2.1"
	OIDMedicalEquipment  = "2.16.840.1.113883.10.20.22.2.23"
	OIDSocialHistory     = "2.16.840.1.113883.10.20.22.2.17"
)

// XML structures for parsing C-CDA

type xmlClinicalDocument struct {
	XMLName      xml.Name         `xml:"ClinicalDocument"`
	RecordTarget xmlRecordTarget  `xml:"recordTarget"`
	Author       xmlAuthor        `xml:"author"`
	Custodian    xmlCustodian     `xml:"custodian"`
	Component    xmlComponent     `xml:"component"`
}

type xmlRecordTarget struct {
	PatientRole xmlPatientRole `xml:"patientRole"`
}

type xmlPatientRole struct {
	ID      []xmlID     `xml:"id"`
	Addr    xmlAddr     `xml:"addr"`
	Telecom []xmlTelecom `xml:"telecom"`
	Patient xmlPatient  `xml:"patient"`
}

type xmlPatient struct {
	Name              xmlName    `xml:"name"`
	AdministrativeGender xmlCode `xml:"administrativeGenderCode"`
	BirthTime         xmlValue   `xml:"birthTime"`
	Race              xmlCode    `xml:"raceCode"`
	Ethnicity         xmlCode    `xml:"ethnicGroupCode"`
	MaritalStatus     xmlCode    `xml:"maritalStatusCode"`
	LanguageComm      xmlLanguageCommunication `xml:"languageCommunication"`
}

type xmlLanguageCommunication struct {
	LanguageCode xmlCode `xml:"languageCode"`
}

type xmlName struct {
	Given  []string `xml:"given"`
	Family string   `xml:"family"`
	Suffix string   `xml:"suffix"`
	Prefix string   `xml:"prefix"`
}

type xmlAddr struct {
	StreetAddressLine []string `xml:"streetAddressLine"`
	City              string   `xml:"city"`
	State             string   `xml:"state"`
	PostalCode        string   `xml:"postalCode"`
	Country           string   `xml:"country"`
}

type xmlTelecom struct {
	Use   string `xml:"use,attr"`
	Value string `xml:"value,attr"`
}

type xmlAuthor struct {
	Time            xmlValue         `xml:"time"`
	AssignedAuthor  xmlAssignedAuthor `xml:"assignedAuthor"`
}

type xmlAssignedAuthor struct {
	ID                   []xmlID `xml:"id"`
	AssignedPerson       xmlAssignedPerson `xml:"assignedPerson"`
	RepresentedOrg       xmlOrganization   `xml:"representedOrganization"`
}

type xmlAssignedPerson struct {
	Name xmlName `xml:"name"`
}

type xmlOrganization struct {
	Name string `xml:"name"`
}

type xmlCustodian struct {
	AssignedCustodian xmlAssignedCustodian `xml:"assignedCustodian"`
}

type xmlAssignedCustodian struct {
	RepresentedCustodianOrg xmlRepresentedCustodianOrg `xml:"representedCustodianOrganization"`
}

type xmlRepresentedCustodianOrg struct {
	ID      []xmlID    `xml:"id"`
	Name    string     `xml:"name"`
	Telecom xmlTelecom `xml:"telecom"`
	Addr    xmlAddr    `xml:"addr"`
}

type xmlComponent struct {
	StructuredBody xmlStructuredBody `xml:"structuredBody"`
}

type xmlStructuredBody struct {
	Components []xmlComponentSection `xml:"component"`
}

type xmlComponentSection struct {
	Section xmlSection `xml:"section"`
}

type xmlSection struct {
	TemplateID []xmlTemplateID `xml:"templateId"`
	Code       xmlCode         `xml:"code"`
	Title      string          `xml:"title"`
	Text       xmlText         `xml:"text"`
	Entry      []xmlEntry      `xml:"entry"`
}

type xmlTemplateID struct {
	Root      string `xml:"root,attr"`
	Extension string `xml:"extension,attr"`
}

type xmlText struct {
	Content string `xml:",innerxml"`
}

type xmlEntry struct {
	TypeCode         string           `xml:"typeCode,attr"`
	Encounter        *xmlEncounter    `xml:"encounter"`
	Act              *xmlAct          `xml:"act"`
	SubstanceAdmin   *xmlSubstanceAdministration `xml:"substanceAdministration"`
	Procedure        *xmlProcedure    `xml:"procedure"`
	Organizer        *xmlOrganizer    `xml:"organizer"`
	Observation      *xmlObservation  `xml:"observation"`
	Supply           *xmlSupply       `xml:"supply"`
}

type xmlEncounter struct {
	ClassCode     string            `xml:"classCode,attr"`
	MoodCode      string            `xml:"moodCode,attr"`
	ID            []xmlID           `xml:"id"`
	Code          xmlCode           `xml:"code"`
	EffectiveTime xmlEffectiveTime  `xml:"effectiveTime"`
	Performer     []xmlPerformer    `xml:"performer"`
	Participant   []xmlParticipant  `xml:"participant"`
	EntryRelationship []xmlEntryRelationship `xml:"entryRelationship"`
}

type xmlAct struct {
	ClassCode     string           `xml:"classCode,attr"`
	MoodCode      string           `xml:"moodCode,attr"`
	ID            []xmlID          `xml:"id"`
	Code          xmlCode          `xml:"code"`
	StatusCode    xmlCode          `xml:"statusCode"`
	EffectiveTime xmlEffectiveTime `xml:"effectiveTime"`
	EntryRelationship []xmlEntryRelationship `xml:"entryRelationship"`
}

type xmlEntryRelationship struct {
	TypeCode    string          `xml:"typeCode,attr"`
	Observation *xmlObservation `xml:"observation"`
	Act         *xmlAct         `xml:"act"`
	Supply      *xmlSupply      `xml:"supply"`
}

type xmlObservation struct {
	ClassCode     string           `xml:"classCode,attr"`
	MoodCode      string           `xml:"moodCode,attr"`
	ID            []xmlID          `xml:"id"`
	Code          xmlCode          `xml:"code"`
	StatusCode    xmlCode          `xml:"statusCode"`
	EffectiveTime xmlEffectiveTime `xml:"effectiveTime"`
	Value         xmlValue         `xml:"value"`
	InterpretationCode xmlCode     `xml:"interpretationCode"`
	ReferenceRange xmlReferenceRange `xml:"referenceRange"`
	Participant   []xmlParticipant  `xml:"participant"`
	EntryRelationship []xmlEntryRelationship `xml:"entryRelationship"`
}

type xmlReferenceRange struct {
	ObservationRange xmlObservationRange `xml:"observationRange"`
}

type xmlObservationRange struct {
	Value xmlValue `xml:"value"`
	Text  string   `xml:"text"`
}

type xmlSubstanceAdministration struct {
	ClassCode     string           `xml:"classCode,attr"`
	MoodCode      string           `xml:"moodCode,attr"`
	ID            []xmlID          `xml:"id"`
	StatusCode    xmlCode          `xml:"statusCode"`
	EffectiveTime []xmlEffectiveTime `xml:"effectiveTime"`
	RouteCode     xmlCode          `xml:"routeCode"`
	DoseQuantity  xmlQuantity      `xml:"doseQuantity"`
	RateQuantity  xmlQuantity      `xml:"rateQuantity"`
	Consumable    xmlConsumable    `xml:"consumable"`
	EntryRelationship []xmlEntryRelationship `xml:"entryRelationship"`
}

type xmlConsumable struct {
	ManufacturedProduct xmlManufacturedProduct `xml:"manufacturedProduct"`
}

type xmlManufacturedProduct struct {
	ManufacturedMaterial xmlManufacturedMaterial `xml:"manufacturedMaterial"`
}

type xmlManufacturedMaterial struct {
	Code     xmlCode `xml:"code"`
	LotNumber string `xml:"lotNumberText"`
}

type xmlProcedure struct {
	ClassCode     string           `xml:"classCode,attr"`
	MoodCode      string           `xml:"moodCode,attr"`
	ID            []xmlID          `xml:"id"`
	Code          xmlCode          `xml:"code"`
	StatusCode    xmlCode          `xml:"statusCode"`
	EffectiveTime xmlEffectiveTime `xml:"effectiveTime"`
	TargetSiteCode xmlCode         `xml:"targetSiteCode"`
	Performer     []xmlPerformer   `xml:"performer"`
}

type xmlOrganizer struct {
	ClassCode     string           `xml:"classCode,attr"`
	MoodCode      string           `xml:"moodCode,attr"`
	ID            []xmlID          `xml:"id"`
	Code          xmlCode          `xml:"code"`
	StatusCode    xmlCode          `xml:"statusCode"`
	EffectiveTime xmlEffectiveTime `xml:"effectiveTime"`
	Component     []xmlOrganizerComponent `xml:"component"`
}

type xmlOrganizerComponent struct {
	Observation *xmlObservation `xml:"observation"`
}

type xmlSupply struct {
	ClassCode     string           `xml:"classCode,attr"`
	MoodCode      string           `xml:"moodCode,attr"`
	ID            []xmlID          `xml:"id"`
	StatusCode    xmlCode          `xml:"statusCode"`
	EffectiveTime xmlEffectiveTime `xml:"effectiveTime"`
	Quantity      xmlQuantity      `xml:"quantity"`
	Product       xmlProduct       `xml:"product"`
	Participant   []xmlParticipant `xml:"participant"`
}

type xmlProduct struct {
	ManufacturedProduct xmlManufacturedProduct `xml:"manufacturedProduct"`
}

type xmlPerformer struct {
	AssignedEntity xmlAssignedEntity `xml:"assignedEntity"`
}

type xmlAssignedEntity struct {
	ID               []xmlID `xml:"id"`
	AssignedPerson   xmlAssignedPerson `xml:"assignedPerson"`
}

type xmlParticipant struct {
	TypeCode        string           `xml:"typeCode,attr"`
	ParticipantRole xmlParticipantRole `xml:"participantRole"`
}

type xmlParticipantRole struct {
	ClassCode   string  `xml:"classCode,attr"`
	ID          []xmlID `xml:"id"`
	Code        xmlCode `xml:"code"`
	PlayingDevice xmlPlayingDevice `xml:"playingDevice"`
	PlayingEntity xmlPlayingEntity `xml:"playingEntity"`
}

type xmlPlayingDevice struct {
	Code xmlCode `xml:"code"`
}

type xmlPlayingEntity struct {
	Code xmlCode `xml:"code"`
	Name string  `xml:"name"`
}

type xmlID struct {
	Root      string `xml:"root,attr"`
	Extension string `xml:"extension,attr"`
}

type xmlCode struct {
	Code           string `xml:"code,attr"`
	CodeSystem     string `xml:"codeSystem,attr"`
	CodeSystemName string `xml:"codeSystemName,attr"`
	DisplayName    string `xml:"displayName,attr"`
	OriginalText   xmlOriginalText `xml:"originalText"`
}

type xmlOriginalText struct {
	Reference xmlReference `xml:"reference"`
	Content   string       `xml:",chardata"`
}

type xmlReference struct {
	Value string `xml:"value,attr"`
}

type xmlEffectiveTime struct {
	Value    string   `xml:"value,attr"`
	Low      xmlValue `xml:"low"`
	High     xmlValue `xml:"high"`
	Operator string   `xml:"operator,attr"`
}

type xmlValue struct {
	Type      string  `xml:"type,attr"`
	Value     string  `xml:"value,attr"`
	Unit      string  `xml:"unit,attr"`
	Code      string  `xml:"code,attr"`
	CodeSystem string `xml:"codeSystem,attr"`
	CodeSystemName string `xml:"codeSystemName,attr"`
	DisplayName string `xml:"displayName,attr"`
	Low       *xmlValue `xml:"low"`
	High      *xmlValue `xml:"high"`
}

type xmlQuantity struct {
	Value string `xml:"value,attr"`
	Unit  string `xml:"unit,attr"`
}

// ParseFile parses a C-CDA XML file and returns a Document
func ParseFile(filepath string) (*Document, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return Parse(data)
}

// Parse parses C-CDA XML data and returns a Document
func Parse(data []byte) (*Document, error) {
	var xmlDoc xmlClinicalDocument
	if err := xml.Unmarshal(data, &xmlDoc); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	doc := &Document{}

	// Parse patient demographics
	doc.Patient = parsePatient(xmlDoc.RecordTarget.PatientRole)

	// Parse author
	doc.Author = parseAuthor(xmlDoc.Author)

	// Parse custodian
	doc.Custodian = parseCustodian(xmlDoc.Custodian)

	// Parse sections
	for _, comp := range xmlDoc.Component.StructuredBody.Components {
		section := comp.Section
		templateOID := getSectionTemplateOID(section.TemplateID)

		switch templateOID {
		case OIDEncounters, OIDEncountersEntriesReq:
			doc.Encounters = parseEncounters(section)
		case OIDProblems, OIDProblemsEntriesReq:
			doc.Problems = parseProblems(section)
		case OIDMedications, OIDMedicationsEntriesReq:
			doc.Medications = parseMedications(section)
		case OIDProcedures, OIDProceduresEntriesReq:
			doc.Procedures = parseProcedures(section)
		case OIDVitalSigns, OIDVitalSignsEntriesReq:
			doc.VitalSigns = parseVitalSigns(section)
		case OIDResults, OIDResultsEntriesReq:
			doc.LabResults = parseLabResults(section)
		case OIDAllergies, OIDAllergiesEntriesReq:
			doc.Allergies = parseAllergies(section)
		case OIDImmunizations, OIDImmunizationsEntriesReq:
			doc.Immunizations = parseImmunizations(section)
		case OIDMedicalEquipment:
			doc.Devices = parseDevices(section)
		case OIDSocialHistory:
			doc.Observations = parseSocialHistory(section)
		}
	}

	return doc, nil
}

func getSectionTemplateOID(templates []xmlTemplateID) string {
	for _, t := range templates {
		// Return the first recognized template OID
		switch t.Root {
		case OIDEncounters, OIDEncountersEntriesReq,
			OIDProblems, OIDProblemsEntriesReq,
			OIDMedications, OIDMedicationsEntriesReq,
			OIDProcedures, OIDProceduresEntriesReq,
			OIDVitalSigns, OIDVitalSignsEntriesReq,
			OIDResults, OIDResultsEntriesReq,
			OIDAllergies, OIDAllergiesEntriesReq,
			OIDImmunizations, OIDImmunizationsEntriesReq,
			OIDMedicalEquipment, OIDSocialHistory:
			return t.Root
		}
	}
	return ""
}

func parsePatient(pr xmlPatientRole) Patient {
	p := Patient{}

	// Get patient ID
	if len(pr.ID) > 0 {
		if pr.ID[0].Extension != "" {
			p.ID = pr.ID[0].Extension
		} else {
			p.ID = pr.ID[0].Root
		}
	}

	// Parse name
	if len(pr.Patient.Name.Given) > 0 {
		p.Name.Given = strings.Join(pr.Patient.Name.Given, " ")
	}
	p.Name.Family = pr.Patient.Name.Family
	p.Name.Suffix = pr.Patient.Name.Suffix
	p.Name.Prefix = pr.Patient.Name.Prefix

	// Parse birth time
	p.BirthTime = parseHL7Time(pr.Patient.BirthTime.Value)

	// Parse gender
	p.Gender = parseCodedValue(pr.Patient.AdministrativeGender)

	// Parse race
	p.Race = parseCodedValue(pr.Patient.Race)

	// Parse ethnicity
	p.Ethnicity = parseCodedValue(pr.Patient.Ethnicity)

	// Parse address
	p.Address = parseAddress(pr.Addr)

	// Parse telecom
	for _, t := range pr.Telecom {
		p.Telecom = append(p.Telecom, Telecom{
			Use:   t.Use,
			Value: t.Value,
		})
	}

	// Parse marital status
	p.MaritalStatus = parseCodedValue(pr.Patient.MaritalStatus)

	// Parse language
	p.Language = parseCodedValue(pr.Patient.LanguageComm.LanguageCode)

	return p
}

func parseAuthor(a xmlAuthor) Author {
	author := Author{}
	author.Time = parseHL7Time(a.Time.Value)

	if len(a.AssignedAuthor.ID) > 0 {
		author.ID = a.AssignedAuthor.ID[0].Extension
	}

	if len(a.AssignedAuthor.AssignedPerson.Name.Given) > 0 {
		author.Name.Given = strings.Join(a.AssignedAuthor.AssignedPerson.Name.Given, " ")
	}
	author.Name.Family = a.AssignedAuthor.AssignedPerson.Name.Family

	author.Organization = a.AssignedAuthor.RepresentedOrg.Name

	return author
}

func parseCustodian(c xmlCustodian) Custodian {
	cust := Custodian{}
	org := c.AssignedCustodian.RepresentedCustodianOrg

	if len(org.ID) > 0 {
		cust.ID = org.ID[0].Extension
	}
	cust.Name = org.Name
	cust.Address = parseAddress(org.Addr)
	cust.Telecom = Telecom{
		Use:   org.Telecom.Use,
		Value: org.Telecom.Value,
	}

	return cust
}

func parseAddress(a xmlAddr) Address {
	return Address{
		StreetAddress: a.StreetAddressLine,
		City:          a.City,
		State:         a.State,
		PostalCode:    a.PostalCode,
		Country:       a.Country,
	}
}

func parseCodedValue(c xmlCode) CodedValue {
	return CodedValue{
		Code:           c.Code,
		CodeSystem:     c.CodeSystem,
		CodeSystemName: c.CodeSystemName,
		DisplayName:    c.DisplayName,
		OriginalText:   c.OriginalText.Content,
	}
}

func parseEffectiveTime(et xmlEffectiveTime) EffectiveTime {
	return EffectiveTime{
		Value: parseHL7Time(et.Value),
		Low:   parseHL7Time(et.Low.Value),
		High:  parseHL7Time(et.High.Value),
	}
}

// parseHL7Time parses HL7 datetime format (YYYYMMDDHHMMSS, YYYYMMDD, etc.)
func parseHL7Time(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	// Remove timezone suffix if present
	s = strings.TrimSuffix(s, "Z")
	if idx := strings.IndexAny(s, "+-"); idx > 0 && idx < len(s)-1 {
		s = s[:idx]
	}

	formats := []string{
		"20060102150405",
		"200601021504",
		"2006010215",
		"20060102",
		"200601",
		"2006",
	}

	for _, format := range formats {
		if len(s) >= len(format) {
			if t, err := time.Parse(format, s[:len(format)]); err == nil {
				return t
			}
		}
	}

	return time.Time{}
}

func parseQuantity(q xmlQuantity) Quantity {
	val, _ := strconv.ParseFloat(q.Value, 64)
	return Quantity{
		Value: val,
		Unit:  q.Unit,
	}
}

func getIDString(ids []xmlID) string {
	if len(ids) > 0 {
		if ids[0].Extension != "" {
			return ids[0].Extension
		}
		return ids[0].Root
	}
	return ""
}

// Section parsers - implemented in separate files for clarity
// but declared here as package-level functions

func parseEncounters(section xmlSection) []Encounter {
	var encounters []Encounter

	for _, entry := range section.Entry {
		if entry.Encounter == nil {
			continue
		}

		enc := entry.Encounter
		encounter := Encounter{
			ID:            getIDString(enc.ID),
			Code:          parseCodedValue(enc.Code),
			EffectiveTime: parseEffectiveTime(enc.EffectiveTime),
		}

		// Get performer name
		if len(enc.Performer) > 0 {
			p := enc.Performer[0].AssignedEntity.AssignedPerson.Name
			encounter.Performer = strings.TrimSpace(strings.Join(p.Given, " ") + " " + p.Family)
		}

		// Get location from participant
		for _, part := range enc.Participant {
			if part.TypeCode == "LOC" {
				encounter.Location = part.ParticipantRole.PlayingEntity.Name
				break
			}
		}

		encounters = append(encounters, encounter)
	}

	return encounters
}

func parseProblems(section xmlSection) []Problem {
	var problems []Problem

	for _, entry := range section.Entry {
		if entry.Act == nil {
			continue
		}

		// Problems are nested in act/entryRelationship/observation
		for _, er := range entry.Act.EntryRelationship {
			if er.Observation == nil {
				continue
			}

			obs := er.Observation
			problem := Problem{
				ID:            getIDString(obs.ID),
				EffectiveTime: parseEffectiveTime(obs.EffectiveTime),
				Status:        parseCodedValue(obs.StatusCode),
			}

			// The problem code is in the value element
			if obs.Value.Code != "" {
				problem.Code = CodedValue{
					Code:           obs.Value.Code,
					CodeSystem:     obs.Value.CodeSystem,
					CodeSystemName: obs.Value.CodeSystemName,
					DisplayName:    obs.Value.DisplayName,
				}
			} else {
				problem.Code = parseCodedValue(obs.Code)
			}

			problems = append(problems, problem)
		}
	}

	return problems
}

func parseMedications(section xmlSection) []Medication {
	var medications []Medication

	for _, entry := range section.Entry {
		if entry.SubstanceAdmin == nil {
			continue
		}

		sa := entry.SubstanceAdmin
		med := Medication{
			ID:           getIDString(sa.ID),
			Code:         parseCodedValue(sa.Consumable.ManufacturedProduct.ManufacturedMaterial.Code),
			DoseQuantity: parseQuantity(sa.DoseQuantity),
			RateQuantity: parseQuantity(sa.RateQuantity),
			RouteCode:    parseCodedValue(sa.RouteCode),
			Status:       parseCodedValue(sa.StatusCode),
		}

		// Parse effective times (may have multiple for different purposes)
		for _, et := range sa.EffectiveTime {
			if et.Low.Value != "" || et.High.Value != "" {
				med.EffectiveTime = parseEffectiveTime(et)
				break
			} else if et.Value != "" {
				med.EffectiveTime.Value = parseHL7Time(et.Value)
			}
		}

		// Check for supply info in entry relationships
		for _, er := range sa.EntryRelationship {
			if er.Supply != nil {
				if er.Supply.Quantity.Value != "" {
					days, _ := strconv.Atoi(er.Supply.Quantity.Value)
					med.DaysSupply = days
				}
			}
		}

		medications = append(medications, med)
	}

	return medications
}

func parseProcedures(section xmlSection) []Procedure {
	var procedures []Procedure

	for _, entry := range section.Entry {
		if entry.Procedure == nil {
			continue
		}

		proc := entry.Procedure
		procedure := Procedure{
			ID:            getIDString(proc.ID),
			Code:          parseCodedValue(proc.Code),
			EffectiveTime: parseEffectiveTime(proc.EffectiveTime),
			Status:        parseCodedValue(proc.StatusCode),
			TargetSite:    parseCodedValue(proc.TargetSiteCode),
		}

		if len(proc.Performer) > 0 {
			p := proc.Performer[0].AssignedEntity.AssignedPerson.Name
			procedure.Performer = strings.TrimSpace(strings.Join(p.Given, " ") + " " + p.Family)
		}

		procedures = append(procedures, procedure)
	}

	return procedures
}

func parseVitalSigns(section xmlSection) []VitalSign {
	var vitals []VitalSign

	for _, entry := range section.Entry {
		if entry.Organizer == nil {
			continue
		}

		// Vital signs are in organizer/component/observation
		for _, comp := range entry.Organizer.Component {
			if comp.Observation == nil {
				continue
			}

			obs := comp.Observation
			vital := VitalSign{
				ID:             getIDString(obs.ID),
				Code:           parseCodedValue(obs.Code),
				EffectiveTime:  parseHL7Time(obs.EffectiveTime.Value),
				Interpretation: parseCodedValue(obs.InterpretationCode),
			}

			// Parse value
			if obs.Value.Value != "" {
				vital.Value, _ = strconv.ParseFloat(obs.Value.Value, 64)
				vital.Unit = obs.Value.Unit
			}

			vitals = append(vitals, vital)
		}
	}

	return vitals
}

func parseLabResults(section xmlSection) []LabResult {
	var results []LabResult

	for _, entry := range section.Entry {
		if entry.Organizer == nil {
			continue
		}

		// Lab results are in organizer/component/observation
		for _, comp := range entry.Organizer.Component {
			if comp.Observation == nil {
				continue
			}

			obs := comp.Observation
			result := LabResult{
				ID:             getIDString(obs.ID),
				Code:           parseCodedValue(obs.Code),
				EffectiveTime:  parseHL7Time(obs.EffectiveTime.Value),
				Interpretation: parseCodedValue(obs.InterpretationCode),
				Status:         parseCodedValue(obs.StatusCode),
			}

			// Parse value (numeric or string)
			if obs.Value.Value != "" {
				if val, err := strconv.ParseFloat(obs.Value.Value, 64); err == nil {
					result.Value = val
				} else {
					result.ValueString = obs.Value.Value
				}
				result.Unit = obs.Value.Unit
			}

			// Parse reference range
			if obs.ReferenceRange.ObservationRange.Text != "" {
				result.ReferenceRange.Text = obs.ReferenceRange.ObservationRange.Text
			}
			if obs.ReferenceRange.ObservationRange.Value.Low != nil {
				result.ReferenceRange.Low, _ = strconv.ParseFloat(obs.ReferenceRange.ObservationRange.Value.Low.Value, 64)
			}
			if obs.ReferenceRange.ObservationRange.Value.High != nil {
				result.ReferenceRange.High, _ = strconv.ParseFloat(obs.ReferenceRange.ObservationRange.Value.High.Value, 64)
			}

			results = append(results, result)
		}
	}

	return results
}

func parseAllergies(section xmlSection) []Allergy {
	var allergies []Allergy

	for _, entry := range section.Entry {
		if entry.Act == nil {
			continue
		}

		// Allergies are nested in act/entryRelationship/observation
		for _, er := range entry.Act.EntryRelationship {
			if er.Observation == nil {
				continue
			}

			obs := er.Observation
			allergy := Allergy{
				ID:            getIDString(obs.ID),
				EffectiveTime: parseEffectiveTime(obs.EffectiveTime),
				Status:        parseCodedValue(obs.StatusCode),
			}

			// The allergen is in participant/participantRole/playingEntity
			for _, part := range obs.Participant {
				if part.TypeCode == "CSM" {
					allergy.Substance = parseCodedValue(part.ParticipantRole.PlayingEntity.Code)
					break
				}
			}

			// Get reaction and severity from nested observations
			for _, ner := range obs.EntryRelationship {
				if ner.Observation != nil {
					if ner.Observation.Code.Code == "ASSERTION" {
						allergy.Code = CodedValue{
							Code:           ner.Observation.Value.Code,
							CodeSystem:     ner.Observation.Value.CodeSystem,
							DisplayName:    ner.Observation.Value.DisplayName,
						}
					}
				}
			}

			allergies = append(allergies, allergy)
		}
	}

	return allergies
}

func parseImmunizations(section xmlSection) []Immunization {
	var immunizations []Immunization

	for _, entry := range section.Entry {
		if entry.SubstanceAdmin == nil {
			continue
		}

		sa := entry.SubstanceAdmin
		imm := Immunization{
			ID:           getIDString(sa.ID),
			Code:         parseCodedValue(sa.Consumable.ManufacturedProduct.ManufacturedMaterial.Code),
			Status:       parseCodedValue(sa.StatusCode),
			RouteCode:    parseCodedValue(sa.RouteCode),
			DoseQuantity: parseQuantity(sa.DoseQuantity),
			LotNumber:    sa.Consumable.ManufacturedProduct.ManufacturedMaterial.LotNumber,
		}

		// Parse effective time
		for _, et := range sa.EffectiveTime {
			if et.Value != "" {
				imm.EffectiveTime = parseHL7Time(et.Value)
				break
			}
		}

		immunizations = append(immunizations, imm)
	}

	return immunizations
}

func parseDevices(section xmlSection) []Device {
	var devices []Device

	for _, entry := range section.Entry {
		// Devices can be in supply or organizer
		if entry.Supply != nil {
			sup := entry.Supply
			device := Device{
				ID:            getIDString(sup.ID),
				Code:          parseCodedValue(sup.Product.ManufacturedProduct.ManufacturedMaterial.Code),
				EffectiveTime: parseEffectiveTime(sup.EffectiveTime),
				Status:        parseCodedValue(sup.StatusCode),
			}

			// Get UDI from participant
			for _, part := range sup.Participant {
				if len(part.ParticipantRole.ID) > 0 {
					device.UDI = part.ParticipantRole.ID[0].Extension
					break
				}
			}

			devices = append(devices, device)
		}
	}

	return devices
}

func parseSocialHistory(section xmlSection) []SocialObservation {
	var observations []SocialObservation

	for _, entry := range section.Entry {
		if entry.Observation == nil {
			continue
		}

		obs := entry.Observation
		socialObs := SocialObservation{
			ID:            getIDString(obs.ID),
			Code:          parseCodedValue(obs.Code),
			EffectiveTime: parseEffectiveTime(obs.EffectiveTime),
			Status:        parseCodedValue(obs.StatusCode),
		}

		// Value can be coded or quantity
		if obs.Value.Code != "" {
			socialObs.Value = CodedValue{
				Code:           obs.Value.Code,
				CodeSystem:     obs.Value.CodeSystem,
				CodeSystemName: obs.Value.CodeSystemName,
				DisplayName:    obs.Value.DisplayName,
			}
		} else if obs.Value.Value != "" {
			val, _ := strconv.ParseFloat(obs.Value.Value, 64)
			socialObs.ValueQuantity = Quantity{
				Value: val,
				Unit:  obs.Value.Unit,
			}
		}

		observations = append(observations, socialObs)
	}

	return observations
}
