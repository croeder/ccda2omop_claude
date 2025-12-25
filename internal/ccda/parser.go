// Copyright 2025 Christophe Roeder. All rights reserved.

package ccda

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
)

// C-CDA Section Template OIDs
const (
	OIDEncounters           = "2.16.840.1.113883.10.20.22.2.22"
	OIDEncountersEntriesReq = "2.16.840.1.113883.10.20.22.2.22.1"
	OIDProblems             = "2.16.840.1.113883.10.20.22.2.5"
	OIDProblemsEntriesReq   = "2.16.840.1.113883.10.20.22.2.5.1"
	OIDMedications          = "2.16.840.1.113883.10.20.22.2.1"
	OIDMedicationsEntriesReq = "2.16.840.1.113883.10.20.22.2.1.1"
	OIDProcedures           = "2.16.840.1.113883.10.20.22.2.7"
	OIDProceduresEntriesReq = "2.16.840.1.113883.10.20.22.2.7.1"
	OIDVitalSigns           = "2.16.840.1.113883.10.20.22.2.4"
	OIDVitalSignsEntriesReq = "2.16.840.1.113883.10.20.22.2.4.1"
	OIDResults              = "2.16.840.1.113883.10.20.22.2.3"
	OIDResultsEntriesReq    = "2.16.840.1.113883.10.20.22.2.3.1"
	OIDAllergies            = "2.16.840.1.113883.10.20.22.2.6"
	OIDAllergiesEntriesReq  = "2.16.840.1.113883.10.20.22.2.6.1"
	OIDImmunizations        = "2.16.840.1.113883.10.20.22.2.2"
	OIDImmunizationsEntriesReq = "2.16.840.1.113883.10.20.22.2.2.1"
	OIDMedicalEquipment     = "2.16.840.1.113883.10.20.22.2.23"
	OIDSocialHistory        = "2.16.840.1.113883.10.20.22.2.17"
)

// ParseFile parses a C-CDA XML file and returns a Document
func ParseFile(filepath string) (*Document, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	doc, err := xmlquery.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return parseDocument(doc)
}

// Parse parses C-CDA XML data and returns a Document
func Parse(data []byte) (*Document, error) {
	doc, err := xmlquery.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return parseDocument(doc)
}

func parseDocument(root *xmlquery.Node) (*Document, error) {
	doc := &Document{
		SectionMeta: make(map[string]SectionMetadata),
	}

	// Parse patient demographics
	doc.Patient = parsePatient(root)

	// Parse author
	doc.Author = parseAuthor(root)

	// Parse custodian
	doc.Custodian = parseCustodian(root)

	// Find and parse each section by template OID
	sections := xmlquery.Find(root, "//component/section")
	for _, section := range sections {
		templateOID := getSectionTemplateOID(section)

		switch templateOID {
		case OIDEncounters, OIDEncountersEntriesReq:
			doc.Encounters = parseEncounters(section)
			doc.SectionMeta["Encounters"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: templateOID == OIDEncountersEntriesReq,
			}
		case OIDProblems, OIDProblemsEntriesReq:
			doc.Problems = parseProblems(section)
			doc.SectionMeta["Problems"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: templateOID == OIDProblemsEntriesReq,
			}
		case OIDMedications, OIDMedicationsEntriesReq:
			doc.Medications = parseMedications(section)
			doc.SectionMeta["Medications"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: templateOID == OIDMedicationsEntriesReq,
			}
		case OIDProcedures, OIDProceduresEntriesReq:
			doc.Procedures = parseProcedures(section)
			doc.SectionMeta["Procedures"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: templateOID == OIDProceduresEntriesReq,
			}
		case OIDVitalSigns, OIDVitalSignsEntriesReq:
			doc.VitalSigns = parseVitalSigns(section)
			doc.SectionMeta["VitalSigns"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: templateOID == OIDVitalSignsEntriesReq,
			}
		case OIDResults, OIDResultsEntriesReq:
			doc.LabResults = parseLabResults(section)
			doc.SectionMeta["LabResults"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: templateOID == OIDResultsEntriesReq,
			}
		case OIDAllergies, OIDAllergiesEntriesReq:
			doc.Allergies = parseAllergies(section)
			doc.SectionMeta["Allergies"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: templateOID == OIDAllergiesEntriesReq,
			}
		case OIDImmunizations, OIDImmunizationsEntriesReq:
			doc.Immunizations = parseImmunizations(section)
			doc.SectionMeta["Immunizations"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: templateOID == OIDImmunizationsEntriesReq,
			}
		case OIDMedicalEquipment:
			doc.Devices = parseDevices(section)
			doc.SectionMeta["Devices"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: false, // No "entries required" variant for this section
			}
		case OIDSocialHistory:
			doc.Observations = parseSocialHistory(section)
			doc.SectionMeta["Observations"] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: false, // No "entries required" variant for this section
			}
		}
	}

	return doc, nil
}

func getSectionTemplateOID(section *xmlquery.Node) string {
	templates := xmlquery.Find(section, "templateId")
	for _, t := range templates {
		root := attr(t, "root")
		switch root {
		case OIDEncounters, OIDEncountersEntriesReq,
			OIDProblems, OIDProblemsEntriesReq,
			OIDMedications, OIDMedicationsEntriesReq,
			OIDProcedures, OIDProceduresEntriesReq,
			OIDVitalSigns, OIDVitalSignsEntriesReq,
			OIDResults, OIDResultsEntriesReq,
			OIDAllergies, OIDAllergiesEntriesReq,
			OIDImmunizations, OIDImmunizationsEntriesReq,
			OIDMedicalEquipment, OIDSocialHistory:
			return root
		}
	}
	return ""
}

// ============ Patient Parsing ============

func parsePatient(root *xmlquery.Node) Patient {
	p := Patient{}

	// Patient ID
	if id := xmlquery.FindOne(root, "//recordTarget/patientRole/id"); id != nil {
		if ext := attr(id, "extension"); ext != "" {
			p.ID = ext
		} else {
			p.ID = attr(id, "root")
		}
	}

	// Name
	if name := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/name"); name != nil {
		p.Name = parseName(name)
	}

	// Birth time
	if bt := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/birthTime"); bt != nil {
		p.BirthTime = parseHL7Time(attr(bt, "value"))
	}

	// Gender
	if gender := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/administrativeGenderCode"); gender != nil {
		p.Gender = parseCode(gender)
	}

	// Race
	if race := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/raceCode"); race != nil {
		p.Race = parseCode(race)
	}

	// Ethnicity
	if eth := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/ethnicGroupCode"); eth != nil {
		p.Ethnicity = parseCode(eth)
	}

	// Address
	if addr := xmlquery.FindOne(root, "//recordTarget/patientRole/addr"); addr != nil {
		p.Address = parseAddress(addr)
	}

	// Telecom
	telecoms := xmlquery.Find(root, "//recordTarget/patientRole/telecom")
	for _, t := range telecoms {
		p.Telecom = append(p.Telecom, Telecom{
			Use:   attr(t, "use"),
			Value: attr(t, "value"),
		})
	}

	// Marital status
	if ms := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/maritalStatusCode"); ms != nil {
		p.MaritalStatus = parseCode(ms)
	}

	// Language
	if lang := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/languageCommunication/languageCode"); lang != nil {
		p.Language = parseCode(lang)
	}

	return p
}

func parseName(node *xmlquery.Node) Name {
	n := Name{}

	givens := xmlquery.Find(node, "given")
	var givenNames []string
	for _, g := range givens {
		givenNames = append(givenNames, g.InnerText())
	}
	n.Given = strings.Join(givenNames, " ")

	if family := xmlquery.FindOne(node, "family"); family != nil {
		n.Family = family.InnerText()
	}
	if suffix := xmlquery.FindOne(node, "suffix"); suffix != nil {
		n.Suffix = suffix.InnerText()
	}
	if prefix := xmlquery.FindOne(node, "prefix"); prefix != nil {
		n.Prefix = prefix.InnerText()
	}

	return n
}

func parseAddress(node *xmlquery.Node) Address {
	a := Address{}

	streets := xmlquery.Find(node, "streetAddressLine")
	for _, s := range streets {
		a.StreetAddress = append(a.StreetAddress, s.InnerText())
	}

	if city := xmlquery.FindOne(node, "city"); city != nil {
		a.City = city.InnerText()
	}
	if state := xmlquery.FindOne(node, "state"); state != nil {
		a.State = state.InnerText()
	}
	if postal := xmlquery.FindOne(node, "postalCode"); postal != nil {
		a.PostalCode = postal.InnerText()
	}
	if country := xmlquery.FindOne(node, "country"); country != nil {
		a.Country = country.InnerText()
	}

	return a
}

// ============ Author/Custodian Parsing ============

func parseAuthor(root *xmlquery.Node) Author {
	a := Author{}

	if t := xmlquery.FindOne(root, "//author/time"); t != nil {
		a.Time = parseHL7Time(attr(t, "value"))
	}

	if id := xmlquery.FindOne(root, "//author/assignedAuthor/id"); id != nil {
		a.ID = attr(id, "extension")
	}

	if name := xmlquery.FindOne(root, "//author/assignedAuthor/assignedPerson/name"); name != nil {
		a.Name = parseName(name)
	}

	if org := xmlquery.FindOne(root, "//author/assignedAuthor/representedOrganization/name"); org != nil {
		a.Organization = org.InnerText()
	}

	return a
}

func parseCustodian(root *xmlquery.Node) Custodian {
	c := Custodian{}

	base := "//custodian/assignedCustodian/representedCustodianOrganization"

	if id := xmlquery.FindOne(root, base+"/id"); id != nil {
		c.ID = attr(id, "extension")
	}

	if name := xmlquery.FindOne(root, base+"/name"); name != nil {
		c.Name = name.InnerText()
	}

	if addr := xmlquery.FindOne(root, base+"/addr"); addr != nil {
		c.Address = parseAddress(addr)
	}

	if tel := xmlquery.FindOne(root, base+"/telecom"); tel != nil {
		c.Telecom = Telecom{
			Use:   attr(tel, "use"),
			Value: attr(tel, "value"),
		}
	}

	return c
}

// ============ Section Parsers ============

func parseEncounters(section *xmlquery.Node) []Encounter {
	var encounters []Encounter

	entries := xmlquery.Find(section, "entry/encounter")
	for _, enc := range entries {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(enc) {
			continue
		}

		encounter := Encounter{
			ID:            getID(enc),
			Code:          parseCode(xmlquery.FindOne(enc, "code")),
			EffectiveTime: parseEffectiveTime(xmlquery.FindOne(enc, "effectiveTime")),
		}

		// Performer
		if performer := xmlquery.FindOne(enc, "performer/assignedEntity/assignedPerson/name"); performer != nil {
			n := parseName(performer)
			encounter.Performer = strings.TrimSpace(n.Given + " " + n.Family)
		}

		// Location
		if loc := xmlquery.FindOne(enc, "participant[@typeCode='LOC']/participantRole/playingEntity/name"); loc != nil {
			encounter.Location = loc.InnerText()
		}

		encounters = append(encounters, encounter)
	}

	return encounters
}

func parseProblems(section *xmlquery.Node) []Problem {
	var problems []Problem

	// Problems are in act/entryRelationship/observation
	observations := xmlquery.Find(section, "entry/act/entryRelationship/observation")
	for _, obs := range observations {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(obs) {
			continue
		}

		problem := Problem{
			ID:            getID(obs),
			EffectiveTime: parseEffectiveTime(xmlquery.FindOne(obs, "effectiveTime")),
			Status:        parseCode(xmlquery.FindOne(obs, "statusCode")),
		}

		// Problem code is typically in the value element
		if val := xmlquery.FindOne(obs, "value"); val != nil && attr(val, "code") != "" {
			problem.Code = parseCode(val)
		} else {
			problem.Code = parseCode(xmlquery.FindOne(obs, "code"))
		}

		problems = append(problems, problem)
	}

	return problems
}

func parseMedications(section *xmlquery.Node) []Medication {
	var medications []Medication

	entries := xmlquery.Find(section, "entry/substanceAdministration")
	for _, sa := range entries {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(sa) {
			continue
		}

		med := Medication{
			ID:           getID(sa),
			Status:       parseCode(xmlquery.FindOne(sa, "statusCode")),
			RouteCode:    parseCode(xmlquery.FindOne(sa, "routeCode")),
			DoseQuantity: parseQuantity(xmlquery.FindOne(sa, "doseQuantity")),
			RateQuantity: parseQuantity(xmlquery.FindOne(sa, "rateQuantity")),
		}

		// Drug code from consumable
		if code := xmlquery.FindOne(sa, "consumable/manufacturedProduct/manufacturedMaterial/code"); code != nil {
			med.Code = parseCode(code)
		}

		// Effective time - find the one with low/high or value
		effTimes := xmlquery.Find(sa, "effectiveTime")
		for _, et := range effTimes {
			if xmlquery.FindOne(et, "low") != nil || xmlquery.FindOne(et, "high") != nil {
				med.EffectiveTime = parseEffectiveTime(et)
				break
			} else if attr(et, "value") != "" {
				med.EffectiveTime.Value = parseHL7Time(attr(et, "value"))
			}
		}

		// Days supply from entryRelationship/supply
		if supply := xmlquery.FindOne(sa, "entryRelationship/supply/quantity"); supply != nil {
			if days, err := strconv.Atoi(attr(supply, "value")); err == nil {
				med.DaysSupply = days
			}
		}

		medications = append(medications, med)
	}

	return medications
}

func parseProcedures(section *xmlquery.Node) []Procedure {
	var procedures []Procedure

	entries := xmlquery.Find(section, "entry/procedure")
	for _, proc := range entries {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(proc) {
			continue
		}

		procedure := Procedure{
			ID:            getID(proc),
			Code:          parseCode(xmlquery.FindOne(proc, "code")),
			EffectiveTime: parseEffectiveTime(xmlquery.FindOne(proc, "effectiveTime")),
			Status:        parseCode(xmlquery.FindOne(proc, "statusCode")),
			TargetSite:    parseCode(xmlquery.FindOne(proc, "targetSiteCode")),
		}

		if performer := xmlquery.FindOne(proc, "performer/assignedEntity/assignedPerson/name"); performer != nil {
			n := parseName(performer)
			procedure.Performer = strings.TrimSpace(n.Given + " " + n.Family)
		}

		procedures = append(procedures, procedure)
	}

	return procedures
}

func parseVitalSigns(section *xmlquery.Node) []VitalSign {
	var vitals []VitalSign

	// Vital signs are in organizer/component/observation
	observations := xmlquery.Find(section, "entry/organizer/component/observation")
	for _, obs := range observations {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(obs) {
			continue
		}

		vital := VitalSign{
			ID:             getID(obs),
			Code:           parseCode(xmlquery.FindOne(obs, "code")),
			Interpretation: parseCode(xmlquery.FindOne(obs, "interpretationCode")),
		}

		if et := xmlquery.FindOne(obs, "effectiveTime"); et != nil {
			vital.EffectiveTime = parseHL7Time(attr(et, "value"))
		}

		if val := xmlquery.FindOne(obs, "value"); val != nil {
			vital.Value, _ = strconv.ParseFloat(attr(val, "value"), 64)
			vital.Unit = attr(val, "unit")
		}

		vitals = append(vitals, vital)
	}

	return vitals
}

func parseLabResults(section *xmlquery.Node) []LabResult {
	var results []LabResult

	// Lab results are in organizer/component/observation
	observations := xmlquery.Find(section, "entry/organizer/component/observation")
	for _, obs := range observations {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(obs) {
			continue
		}

		result := LabResult{
			ID:             getID(obs),
			Code:           parseCode(xmlquery.FindOne(obs, "code")),
			Status:         parseCode(xmlquery.FindOne(obs, "statusCode")),
			Interpretation: parseCode(xmlquery.FindOne(obs, "interpretationCode")),
		}

		if et := xmlquery.FindOne(obs, "effectiveTime"); et != nil {
			result.EffectiveTime = parseHL7Time(attr(et, "value"))
		}

		if val := xmlquery.FindOne(obs, "value"); val != nil {
			if v, err := strconv.ParseFloat(attr(val, "value"), 64); err == nil {
				result.Value = v
			} else {
				result.ValueString = attr(val, "value")
			}
			result.Unit = attr(val, "unit")
		}

		// Reference range
		if rr := xmlquery.FindOne(obs, "referenceRange/observationRange"); rr != nil {
			if text := xmlquery.FindOne(rr, "text"); text != nil {
				result.ReferenceRange.Text = text.InnerText()
			}
			if low := xmlquery.FindOne(rr, "value/low"); low != nil {
				result.ReferenceRange.Low, _ = strconv.ParseFloat(attr(low, "value"), 64)
			}
			if high := xmlquery.FindOne(rr, "value/high"); high != nil {
				result.ReferenceRange.High, _ = strconv.ParseFloat(attr(high, "value"), 64)
			}
		}

		results = append(results, result)
	}

	return results
}

func parseAllergies(section *xmlquery.Node) []Allergy {
	var allergies []Allergy

	// Allergies are in act/entryRelationship/observation
	observations := xmlquery.Find(section, "entry/act/entryRelationship/observation")
	for _, obs := range observations {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(obs) {
			continue
		}

		allergy := Allergy{
			ID:            getID(obs),
			EffectiveTime: parseEffectiveTime(xmlquery.FindOne(obs, "effectiveTime")),
			Status:        parseCode(xmlquery.FindOne(obs, "statusCode")),
		}

		// Substance from participant
		if substance := xmlquery.FindOne(obs, "participant[@typeCode='CSM']/participantRole/playingEntity/code"); substance != nil {
			allergy.Substance = parseCode(substance)
		}

		// Reaction from nested observation
		// Severity from nested observation

		allergies = append(allergies, allergy)
	}

	return allergies
}

func parseImmunizations(section *xmlquery.Node) []Immunization {
	var immunizations []Immunization

	entries := xmlquery.Find(section, "entry/substanceAdministration")
	for _, sa := range entries {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(sa) {
			continue
		}

		imm := Immunization{
			ID:           getID(sa),
			Status:       parseCode(xmlquery.FindOne(sa, "statusCode")),
			RouteCode:    parseCode(xmlquery.FindOne(sa, "routeCode")),
			DoseQuantity: parseQuantity(xmlquery.FindOne(sa, "doseQuantity")),
		}

		// Vaccine code
		if code := xmlquery.FindOne(sa, "consumable/manufacturedProduct/manufacturedMaterial/code"); code != nil {
			imm.Code = parseCode(code)
		}

		// Lot number
		if lot := xmlquery.FindOne(sa, "consumable/manufacturedProduct/manufacturedMaterial/lotNumberText"); lot != nil {
			imm.LotNumber = lot.InnerText()
		}

		// Effective time
		if et := xmlquery.FindOne(sa, "effectiveTime"); et != nil {
			imm.EffectiveTime = parseHL7Time(attr(et, "value"))
		}

		immunizations = append(immunizations, imm)
	}

	return immunizations
}

func parseDevices(section *xmlquery.Node) []Device {
	var devices []Device

	entries := xmlquery.Find(section, "entry/supply")
	for _, sup := range entries {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(sup) {
			continue
		}

		device := Device{
			ID:            getID(sup),
			EffectiveTime: parseEffectiveTime(xmlquery.FindOne(sup, "effectiveTime")),
			Status:        parseCode(xmlquery.FindOne(sup, "statusCode")),
		}

		// Device code from product or participant/playingDevice
		if code := xmlquery.FindOne(sup, "product/manufacturedProduct/manufacturedMaterial/code"); code != nil && attr(code, "code") != "" {
			device.Code = parseCode(code)
		}
		if code := xmlquery.FindOne(sup, "participant/participantRole/playingDevice/code"); code != nil && attr(code, "code") != "" {
			device.Code = parseCode(code)
		}

		// UDI from participant
		if udi := xmlquery.FindOne(sup, "participant/participantRole/id"); udi != nil {
			device.UDI = attr(udi, "extension")
		}

		devices = append(devices, device)
	}

	return devices
}

func parseSocialHistory(section *xmlquery.Node) []SocialObservation {
	var observations []SocialObservation

	entries := xmlquery.Find(section, "entry/observation")
	for _, obs := range entries {
		// Only include actual events (moodCode=EVN) with completed/active status
		if !shouldIncludeEntry(obs) {
			continue
		}

		socialObs := SocialObservation{
			ID:            getID(obs),
			Code:          parseCode(xmlquery.FindOne(obs, "code")),
			EffectiveTime: parseEffectiveTime(xmlquery.FindOne(obs, "effectiveTime")),
			Status:        parseCode(xmlquery.FindOne(obs, "statusCode")),
		}

		// Value can be coded or quantity
		if val := xmlquery.FindOne(obs, "value"); val != nil {
			if attr(val, "code") != "" {
				socialObs.Value = parseCode(val)
			} else if attr(val, "value") != "" {
				v, _ := strconv.ParseFloat(attr(val, "value"), 64)
				socialObs.ValueQuantity = Quantity{
					Value: v,
					Unit:  attr(val, "unit"),
				}
			}
		}

		observations = append(observations, socialObs)
	}

	return observations
}

// ============ Helper Functions ============

// isActualEvent checks if an entry has moodCode="EVN" (event/actual occurrence)
// Only EVN entries represent things that actually happened.
// Other moodCodes like INT (intent), RQO (request), PRMS (promise) represent plans.
func isActualEvent(node *xmlquery.Node) bool {
	if node == nil {
		return false
	}
	moodCode := attr(node, "moodCode")
	// EVN = Event (actual occurrence)
	// Empty moodCode defaults to EVN for observations
	return moodCode == "EVN" || moodCode == ""
}

// hasCompletedStatus checks if an entry has a status that indicates completion.
// Valid statuses: "completed", "active" (ongoing events that have started)
// Invalid statuses: "cancelled", "aborted", "new", "held", "suspended", "nullified"
func hasCompletedStatus(node *xmlquery.Node) bool {
	if node == nil {
		return true // No node means no status restriction
	}
	statusNode := xmlquery.FindOne(node, "statusCode")
	if statusNode == nil {
		return true // No status code means we accept it
	}
	status := attr(statusNode, "code")
	// Accept completed and active (in-progress events that have started)
	return status == "completed" || status == "active" || status == ""
}

// shouldIncludeEntry checks both moodCode and statusCode to determine
// if an entry represents an actual event that occurred
func shouldIncludeEntry(node *xmlquery.Node) bool {
	return isActualEvent(node) && hasCompletedStatus(node)
}

// attr safely gets an attribute value from a node
func attr(node *xmlquery.Node, name string) string {
	if node == nil {
		return ""
	}
	return node.SelectAttr(name)
}

// getID extracts the ID from an element's id child
func getID(node *xmlquery.Node) string {
	if node == nil {
		return ""
	}
	if id := xmlquery.FindOne(node, "id"); id != nil {
		if ext := attr(id, "extension"); ext != "" {
			return ext
		}
		return attr(id, "root")
	}
	return ""
}

// parseCode extracts a CodedValue from an element with code attributes
func parseCode(node *xmlquery.Node) CodedValue {
	if node == nil {
		return CodedValue{}
	}
	cv := CodedValue{
		Code:           attr(node, "code"),
		CodeSystem:     attr(node, "codeSystem"),
		CodeSystemName: attr(node, "codeSystemName"),
		DisplayName:    attr(node, "displayName"),
	}
	if ot := xmlquery.FindOne(node, "originalText"); ot != nil {
		cv.OriginalText = ot.InnerText()
	}
	return cv
}

// parseEffectiveTime extracts an EffectiveTime from an effectiveTime element
func parseEffectiveTime(node *xmlquery.Node) EffectiveTime {
	if node == nil {
		return EffectiveTime{}
	}
	et := EffectiveTime{
		Value: parseHL7Time(attr(node, "value")),
	}
	if low := xmlquery.FindOne(node, "low"); low != nil {
		et.Low = parseHL7Time(attr(low, "value"))
	}
	if high := xmlquery.FindOne(node, "high"); high != nil {
		et.High = parseHL7Time(attr(high, "value"))
	}
	return et
}

// parseQuantity extracts a Quantity from a quantity element
func parseQuantity(node *xmlquery.Node) Quantity {
	if node == nil {
		return Quantity{}
	}
	val, _ := strconv.ParseFloat(attr(node, "value"), 64)
	return Quantity{
		Value: val,
		Unit:  attr(node, "unit"),
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
