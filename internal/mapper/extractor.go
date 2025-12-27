// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
)

// Extractor extracts data from C-CDA XML using rules
type Extractor struct {
	verbose bool
}

// NewExtractor creates a new XML extractor
func NewExtractor(verbose bool) *Extractor {
	return &Extractor{verbose: verbose}
}

// ExtractedData holds extracted data from a C-CDA document
type ExtractedData struct {
	Patient     map[string]interface{}
	Sections    map[string][]map[string]interface{} // Section name -> entries
	SectionMeta map[string]SectionMetadata
}

// SectionMetadata holds metadata about a section
type SectionMetadata struct {
	TemplateOID     string
	EntriesRequired bool
}

// ExtractFromFile extracts data from a C-CDA file using the provided rules
func (e *Extractor) ExtractFromFile(filepath string, rules []MappingRule) (*ExtractedData, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	doc, err := xmlquery.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return e.ExtractFromDocument(doc, rules)
}

// ExtractFromDocument extracts data from an xmlquery document using rules
func (e *Extractor) ExtractFromDocument(root *xmlquery.Node, rules []MappingRule) (*ExtractedData, error) {
	data := &ExtractedData{
		Patient:     make(map[string]interface{}),
		Sections:    make(map[string][]map[string]interface{}),
		SectionMeta: make(map[string]SectionMetadata),
	}

	// Extract patient demographics (special case - not rule-driven yet)
	data.Patient = e.extractPatient(root)

	// Index rules by section OID for faster lookup
	rulesByOID := make(map[string][]MappingRule)
	for _, rule := range rules {
		if rule.Source.SectionOID != "" {
			rulesByOID[rule.Source.SectionOID] = append(rulesByOID[rule.Source.SectionOID], rule)
		}
		if rule.Source.SectionOIDEntriesReq != "" {
			rulesByOID[rule.Source.SectionOIDEntriesReq] = append(rulesByOID[rule.Source.SectionOIDEntriesReq], rule)
		}
	}

	// Find and process each section
	sections := xmlquery.Find(root, "//component/section")
	for _, section := range sections {
		templateOID := e.getSectionTemplateOID(section, rulesByOID)
		if templateOID == "" {
			continue
		}

		// Find rules for this section
		sectionRules, ok := rulesByOID[templateOID]
		if !ok {
			continue
		}

		// Process each rule for this section
		for _, rule := range sectionRules {
			if rule.Source.EntryXPath == "" || len(rule.Source.Extraction) == 0 {
				continue // Skip rules without extraction config
			}

			// Determine if entries are required
			entriesRequired := templateOID == rule.Source.SectionOIDEntriesReq

			// Store section metadata
			data.SectionMeta[rule.Source.Section] = SectionMetadata{
				TemplateOID:     templateOID,
				EntriesRequired: entriesRequired,
			}

			// Extract entries using the rule's XPath
			entries := xmlquery.Find(section, rule.Source.EntryXPath)
			for _, entry := range entries {
				// Filter by moodCode and statusCode
				if !e.shouldIncludeEntry(entry) {
					continue
				}

				// Extract fields from entry
				extracted := e.extractEntry(entry, rule.Source.Extraction)
				if extracted != nil {
					data.Sections[rule.Source.Section] = append(data.Sections[rule.Source.Section], extracted)
				}
			}
		}
	}

	return data, nil
}

// getSectionTemplateOID finds the template OID for a section that matches our rules
func (e *Extractor) getSectionTemplateOID(section *xmlquery.Node, rulesByOID map[string][]MappingRule) string {
	templates := xmlquery.Find(section, "templateId")
	for _, t := range templates {
		root := e.attr(t, "root")
		if _, ok := rulesByOID[root]; ok {
			return root
		}
	}
	return ""
}

// extractEntry extracts fields from an entry node using extraction specifications
func (e *Extractor) extractEntry(entry *xmlquery.Node, extractions []Extraction) map[string]interface{} {
	result := make(map[string]interface{})

	for _, ext := range extractions {
		value := e.extractField(entry, ext)
		if value != nil {
			result[ext.Field] = value
		}
	}

	// Also extract the entry ID
	if id := e.getID(entry); id != "" {
		result["ID"] = id
	}

	return result
}

// extractField extracts a single field from a node
func (e *Extractor) extractField(node *xmlquery.Node, ext Extraction) interface{} {
	switch ext.Type {
	case "code":
		return e.extractCode(node, ext.XPath)
	case "time":
		return e.extractTime(node, ext.XPath)
	case "effective_time":
		return e.extractEffectiveTime(node, ext.XPath)
	case "float":
		return e.extractFloat(node, ext.XPath)
	case "int":
		return e.extractInt(node, ext.XPath)
	case "string":
		return e.extractString(node, ext.XPath)
	case "quantity":
		return e.extractQuantity(node, ext.XPath)
	default:
		return e.extractString(node, ext.XPath)
	}
}

// extractCode extracts a coded value (code, codeSystem, displayName, etc.)
func (e *Extractor) extractCode(node *xmlquery.Node, xpath string) map[string]interface{} {
	target := xmlquery.FindOne(node, xpath)
	if target == nil {
		return nil
	}

	result := make(map[string]interface{})
	if code := e.attr(target, "code"); code != "" {
		result["Code"] = code
	}
	if cs := e.attr(target, "codeSystem"); cs != "" {
		result["CodeSystem"] = cs
	}
	if csn := e.attr(target, "codeSystemName"); csn != "" {
		result["CodeSystemName"] = csn
	}
	if dn := e.attr(target, "displayName"); dn != "" {
		result["DisplayName"] = dn
	}
	if ot := xmlquery.FindOne(target, "originalText"); ot != nil {
		result["OriginalText"] = ot.InnerText()
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// extractTime extracts a time value from an attribute or element
func (e *Extractor) extractTime(node *xmlquery.Node, xpath string) *time.Time {
	// Handle attribute XPath - split on LAST /@ to handle predicates with /@ inside
	if lastIdx := strings.LastIndex(xpath, "/@"); lastIdx != -1 {
		pathPart := xpath[:lastIdx]
		attrPart := xpath[lastIdx+2:]
		target := xmlquery.FindOne(node, pathPart)
		if target == nil {
			return nil
		}
		value := e.attr(target, attrPart)
		t := e.parseHL7Time(value)
		if t.IsZero() {
			return nil
		}
		return &t
	}

	// Handle element XPath
	target := xmlquery.FindOne(node, xpath)
	if target == nil {
		return nil
	}
	value := e.attr(target, "value")
	t := e.parseHL7Time(value)
	if t.IsZero() {
		return nil
	}
	return &t
}

// extractEffectiveTime extracts an effective time with low/high/value
func (e *Extractor) extractEffectiveTime(node *xmlquery.Node, xpath string) map[string]interface{} {
	target := xmlquery.FindOne(node, xpath)
	if target == nil {
		return nil
	}

	result := make(map[string]interface{})

	if v := e.attr(target, "value"); v != "" {
		t := e.parseHL7Time(v)
		if !t.IsZero() {
			result["Value"] = t
		}
	}
	if low := xmlquery.FindOne(target, "low"); low != nil {
		t := e.parseHL7Time(e.attr(low, "value"))
		if !t.IsZero() {
			result["Low"] = t
		}
	}
	if high := xmlquery.FindOne(target, "high"); high != nil {
		t := e.parseHL7Time(e.attr(high, "value"))
		if !t.IsZero() {
			result["High"] = t
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// extractFloat extracts a float value
func (e *Extractor) extractFloat(node *xmlquery.Node, xpath string) *float64 {
	// Handle attribute XPath - split on LAST /@ to handle predicates with /@ inside
	if lastIdx := strings.LastIndex(xpath, "/@"); lastIdx != -1 {
		pathPart := xpath[:lastIdx]
		attrPart := xpath[lastIdx+2:]
		target := xmlquery.FindOne(node, pathPart)
		if target == nil {
			return nil
		}
		value := e.attr(target, attrPart)
		if value == "" {
			return nil
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil
		}
		return &f
	}

	// Handle element XPath
	target := xmlquery.FindOne(node, xpath)
	if target == nil {
		return nil
	}
	value := e.attr(target, "value")
	if value == "" {
		value = target.InnerText()
	}
	if value == "" {
		return nil
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &f
}

// extractInt extracts an integer value
func (e *Extractor) extractInt(node *xmlquery.Node, xpath string) *int64 {
	// Handle attribute XPath - split on LAST /@ to handle predicates with /@ inside
	if lastIdx := strings.LastIndex(xpath, "/@"); lastIdx != -1 {
		pathPart := xpath[:lastIdx]
		attrPart := xpath[lastIdx+2:]
		target := xmlquery.FindOne(node, pathPart)
		if target == nil {
			return nil
		}
		value := e.attr(target, attrPart)
		if value == "" {
			return nil
		}
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil
		}
		return &i
	}

	target := xmlquery.FindOne(node, xpath)
	if target == nil {
		return nil
	}
	value := e.attr(target, "value")
	if value == "" {
		value = target.InnerText()
	}
	if value == "" {
		return nil
	}
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	return &i
}

// extractString extracts a string value
func (e *Extractor) extractString(node *xmlquery.Node, xpath string) string {
	// Handle attribute XPath - split on LAST /@ to handle predicates with /@ inside
	if lastIdx := strings.LastIndex(xpath, "/@"); lastIdx != -1 {
		pathPart := xpath[:lastIdx]
		attrPart := xpath[lastIdx+2:] // skip "/@"
		target := xmlquery.FindOne(node, pathPart)
		if target == nil {
			return ""
		}
		return e.attr(target, attrPart)
	}

	target := xmlquery.FindOne(node, xpath)
	if target == nil {
		return ""
	}
	// Try attribute value first, then inner text
	if v := e.attr(target, "value"); v != "" {
		return v
	}
	return target.InnerText()
}

// extractQuantity extracts a quantity (value + unit)
func (e *Extractor) extractQuantity(node *xmlquery.Node, xpath string) map[string]interface{} {
	target := xmlquery.FindOne(node, xpath)
	if target == nil {
		return nil
	}

	result := make(map[string]interface{})
	if v := e.attr(target, "value"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			result["Value"] = f
		}
	}
	if u := e.attr(target, "unit"); u != "" {
		result["Unit"] = u
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// extractPatient extracts patient demographics (kept as special case for now)
func (e *Extractor) extractPatient(root *xmlquery.Node) map[string]interface{} {
	p := make(map[string]interface{})

	// Patient ID
	if id := xmlquery.FindOne(root, "//recordTarget/patientRole/id"); id != nil {
		if ext := e.attr(id, "extension"); ext != "" {
			p["ID"] = ext
		} else {
			p["ID"] = e.attr(id, "root")
		}
	}

	// Name
	if name := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/name"); name != nil {
		p["Name"] = e.extractName(name)
	}

	// Birth time
	if bt := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/birthTime"); bt != nil {
		t := e.parseHL7Time(e.attr(bt, "value"))
		if !t.IsZero() {
			p["BirthTime"] = t
		}
	}

	// Gender
	if gender := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/administrativeGenderCode"); gender != nil {
		p["Gender"] = e.extractCode(root, "//recordTarget/patientRole/patient/administrativeGenderCode")
	}

	// Race
	if race := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/raceCode"); race != nil {
		p["Race"] = e.extractCode(root, "//recordTarget/patientRole/patient/raceCode")
	}

	// Ethnicity
	if eth := xmlquery.FindOne(root, "//recordTarget/patientRole/patient/ethnicGroupCode"); eth != nil {
		p["Ethnicity"] = e.extractCode(root, "//recordTarget/patientRole/patient/ethnicGroupCode")
	}

	return p
}

// extractName extracts a name structure
func (e *Extractor) extractName(node *xmlquery.Node) map[string]interface{} {
	result := make(map[string]interface{})

	givens := xmlquery.Find(node, "given")
	var givenNames []string
	for _, g := range givens {
		givenNames = append(givenNames, g.InnerText())
	}
	if len(givenNames) > 0 {
		result["Given"] = strings.Join(givenNames, " ")
	}

	if family := xmlquery.FindOne(node, "family"); family != nil {
		result["Family"] = family.InnerText()
	}
	if suffix := xmlquery.FindOne(node, "suffix"); suffix != nil {
		result["Suffix"] = suffix.InnerText()
	}
	if prefix := xmlquery.FindOne(node, "prefix"); prefix != nil {
		result["Prefix"] = prefix.InnerText()
	}

	return result
}

// Helper functions

// shouldIncludeEntry checks moodCode and statusCode
func (e *Extractor) shouldIncludeEntry(node *xmlquery.Node) bool {
	return e.isActualEvent(node) && e.hasCompletedStatus(node)
}

// isActualEvent checks if moodCode indicates an actual event
func (e *Extractor) isActualEvent(node *xmlquery.Node) bool {
	if node == nil {
		return false
	}
	moodCode := e.attr(node, "moodCode")
	return moodCode == "EVN" || moodCode == ""
}

// hasCompletedStatus checks if statusCode indicates completion
func (e *Extractor) hasCompletedStatus(node *xmlquery.Node) bool {
	if node == nil {
		return true
	}
	statusNode := xmlquery.FindOne(node, "statusCode")
	if statusNode == nil {
		return true
	}
	status := e.attr(statusNode, "code")
	return status == "completed" || status == "active" || status == ""
}

// attr safely gets an attribute value
func (e *Extractor) attr(node *xmlquery.Node, name string) string {
	if node == nil {
		return ""
	}
	return node.SelectAttr(name)
}

// getID extracts the ID from an element
func (e *Extractor) getID(node *xmlquery.Node) string {
	if node == nil {
		return ""
	}
	if id := xmlquery.FindOne(node, "id"); id != nil {
		if ext := e.attr(id, "extension"); ext != "" {
			return ext
		}
		return e.attr(id, "root")
	}
	return ""
}

// parseHL7Time parses HL7 datetime format
func (e *Extractor) parseHL7Time(s string) time.Time {
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

// =============================================================================
// Exported XPath extraction functions for use by the rule engine
// =============================================================================

// XPathExtractString extracts a string value from a node using xpath
// Supports fallback xpath if primary returns empty
func (e *Extractor) XPathExtractString(node *xmlquery.Node, xpath, fallbackXPath string) string {
	result := e.extractString(node, xpath)
	if result == "" && fallbackXPath != "" {
		result = e.extractString(node, fallbackXPath)
	}
	return result
}

// XPathExtractFloat extracts a float value from a node using xpath
func (e *Extractor) XPathExtractFloat(node *xmlquery.Node, xpath, fallbackXPath string) *float64 {
	result := e.extractFloat(node, xpath)
	if result == nil && fallbackXPath != "" {
		result = e.extractFloat(node, fallbackXPath)
	}
	return result
}

// XPathExtractInt extracts an integer value from a node using xpath
func (e *Extractor) XPathExtractInt(node *xmlquery.Node, xpath, fallbackXPath string) *int64 {
	result := e.extractInt(node, xpath)
	if result == nil && fallbackXPath != "" {
		result = e.extractInt(node, fallbackXPath)
	}
	return result
}

// XPathExtractTime extracts a time value from a node using xpath
func (e *Extractor) XPathExtractTime(node *xmlquery.Node, xpath, fallbackXPath string) *time.Time {
	result := e.extractTime(node, xpath)
	if result == nil && fallbackXPath != "" {
		result = e.extractTime(node, fallbackXPath)
	}
	return result
}

// XPathExtractCode extracts a coded value map from a node using xpath
func (e *Extractor) XPathExtractCode(node *xmlquery.Node, xpath string) map[string]interface{} {
	return e.extractCode(node, xpath)
}

// XPathExtractEffectiveTime extracts an effective time structure from a node
func (e *Extractor) XPathExtractEffectiveTime(node *xmlquery.Node, xpath string) map[string]interface{} {
	return e.extractEffectiveTime(node, xpath)
}

// XPathExtractQuantity extracts a quantity (value + unit) from a node
func (e *Extractor) XPathExtractQuantity(node *xmlquery.Node, xpath string) map[string]interface{} {
	return e.extractQuantity(node, xpath)
}

// XPathNode returns the XML node for a given xpath, or nil if not found
func (e *Extractor) XPathNode(node *xmlquery.Node, xpath string) *xmlquery.Node {
	return xmlquery.FindOne(node, xpath)
}

// XPathNodes returns all XML nodes matching a given xpath
func (e *Extractor) XPathNodes(node *xmlquery.Node, xpath string) []*xmlquery.Node {
	return xmlquery.Find(node, xpath)
}

// ParseHL7Time parses an HL7 datetime string (exported for use by rule engine)
func (e *Extractor) ParseHL7Time(s string) time.Time {
	return e.parseHL7Time(s)
}

// GetAttr safely gets an attribute value from a node
func (e *Extractor) GetAttr(node *xmlquery.Node, name string) string {
	return e.attr(node, name)
}

// ShouldIncludeEntry checks if an entry should be included based on moodCode and statusCode
func (e *Extractor) ShouldIncludeEntry(node *xmlquery.Node) bool {
	return e.shouldIncludeEntry(node)
}
