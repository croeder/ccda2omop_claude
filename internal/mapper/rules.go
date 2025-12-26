// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ccda2omop/internal/ccda"
	"github.com/ccda2omop/internal/omop"
)

// MappingRule defines how to map a C-CDA section to an OMOP table
type MappingRule struct {
	Name   string
	Source SourceSpec
	Target TargetSpec
	Fields []FieldMapping
	IDGen  IDGenSpec
}

// SourceSpec defines the source C-CDA section
type SourceSpec struct {
	Section                  string        // C-CDA section name (Problems, Medications, etc.)
	SectionOID               string        // Section template OID (e.g., "2.16.840.1.113883.10.20.22.2.4")
	SectionOIDEntriesReq     string        // Section template OID for entries-required variant
	EntryXPath               string        // XPath to find entries within the section
	Extraction               []Extraction  // Field extraction specifications
	EntryType                string        // For documentation
	Conditions               []Condition   // Filter conditions - entry must match all conditions
}

// Extraction defines how to extract a field from XML
type Extraction struct {
	Field string // Target field name in the extracted map
	XPath string // XPath expression relative to entry
	Type  string // Value type: "code", "time", "float", "int", "string", "effective_time", "quantity"
}

// Condition defines a filter condition for source entries
type Condition struct {
	Type  string // Condition type: "domain_equals", "domain_not_equals", "field_equals", "field_not_equals"
	Field string // For field conditions: field path to check
	Value string // Value to compare against (domain name or field value)
}

// TargetSpec defines the target OMOP table
type TargetSpec struct {
	Table         string // OMOP table name
	TypeConceptID int64  // *_type_concept_id value
}

// FieldMapping defines how to map a source field to a target field
type FieldMapping struct {
	Source     string // Source field path (dot notation, | for fallback)
	Target     string // Target column/field name
	Transform  string // Transform type: none, date, float, vocab, unit, format_source
	VocabField string // For vocab lookups, field containing code system OID
	Optional   bool   // If true, missing source value is OK
}

// IDGenSpec defines how to generate unique IDs
type IDGenSpec struct {
	BaseFields []string // Fields to use for ID generation
	Generator  string   // Generator function name
}

// MappingContext provides context for transforms
type MappingContext struct {
	PersonID int64
	Vocab    *VocabularyMapper
	VisitMap map[string]int64
	Source   interface{} // The source struct being mapped
}

// TransformFunc is a function that transforms a value
type TransformFunc func(value interface{}, fieldMapping FieldMapping, ctx *MappingContext) (interface{}, error)

// RuleEngine executes mapping rules
type RuleEngine struct {
	vocab      *VocabularyMapper
	transforms map[string]TransformFunc
	verbose    bool
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(vocab *VocabularyMapper, verbose bool) *RuleEngine {
	re := &RuleEngine{
		vocab:      vocab,
		transforms: make(map[string]TransformFunc),
		verbose:    verbose,
	}

	// Register built-in transforms
	re.transforms["none"] = transformNone
	re.transforms["date"] = transformDate
	re.transforms["float"] = transformFloat
	re.transforms["int"] = transformInt
	re.transforms["string"] = transformString
	re.transforms["vocab"] = re.transformVocab
	re.transforms["unit"] = re.transformUnit
	re.transforms["route"] = re.transformRoute
	re.transforms["format_source"] = transformFormatSource
	re.transforms["time_ptr"] = transformTimePtr
	re.transforms["value_vocab"] = re.transformValueVocab

	return re
}

// MapEntries maps a slice of source entries using a rule
// If entriesRequired is false, all fields are treated as optional
func (re *RuleEngine) MapEntries(rule MappingRule, sources interface{}, personID int64, visitMap map[string]int64) ([]map[string]interface{}, error) {
	return re.MapEntriesWithOptional(rule, sources, personID, visitMap, true)
}

// MapEntriesWithOptional maps a slice of source entries using a rule
// If entriesRequired is false, all fields are treated as optional
func (re *RuleEngine) MapEntriesWithOptional(rule MappingRule, sources interface{}, personID int64, visitMap map[string]int64, entriesRequired bool) ([]map[string]interface{}, error) {
	srcVal := reflect.ValueOf(sources)
	if srcVal.Kind() != reflect.Slice {
		return nil, fmt.Errorf("sources must be a slice")
	}

	var results []map[string]interface{}
	for i := 0; i < srcVal.Len(); i++ {
		entry := srcVal.Index(i).Interface()
		mapped, err := re.MapEntryWithOptional(rule, entry, personID, visitMap, entriesRequired)
		if err != nil {
			return nil, fmt.Errorf("error mapping entry %d: %w", i, err)
		}
		results = append(results, mapped...)
	}

	return results, nil
}

// MapEntry maps a single source entry using a rule, returning potentially multiple records
func (re *RuleEngine) MapEntry(rule MappingRule, source interface{}, personID int64, visitMap map[string]int64) ([]map[string]interface{}, error) {
	return re.MapEntryWithOptional(rule, source, personID, visitMap, true)
}

// MapEntryWithOptional maps a single source entry using a rule, returning potentially multiple records
// If entriesRequired is false, all fields are treated as optional
func (re *RuleEngine) MapEntryWithOptional(rule MappingRule, source interface{}, personID int64, visitMap map[string]int64, entriesRequired bool) ([]map[string]interface{}, error) {
	ctx := &MappingContext{
		PersonID: personID,
		Vocab:    re.vocab,
		VisitMap: visitMap,
		Source:   source,
	}

	// First, find the concept ID field to determine multi-mapping
	var conceptIDs []int64

	for _, fm := range rule.Fields {
		if fm.Transform == "vocab" {
			// When entries are not required, treat all fields as optional
			isOptional := fm.Optional || !entriesRequired
			value, err := re.extractValue(source, fm.Source)
			if err != nil {
				if isOptional {
					// Optional field missing, use concept 0
					conceptIDs = []int64{0}
					break
				}
				// Required code field missing (e.g., nullFlavor) - skip this entry
				return nil, nil
			}

			codeSystemValue, _ := re.extractValue(source, fm.VocabField)
			codeSystem, _ := codeSystemValue.(string)

			if code, ok := value.(string); ok && code != "" {
				conceptIDs = re.vocab.MapConditionCodes(code, codeSystem)
				if len(conceptIDs) == 0 {
					conceptIDs = []int64{0}
				}
			} else {
				// Empty code value - skip this entry
				return nil, nil
			}
			break
		}
	}

	if len(conceptIDs) == 0 {
		conceptIDs = []int64{0}
	}

	// Check conditions - if any condition fails, skip this entry
	if len(rule.Source.Conditions) > 0 {
		// For domain conditions, we need the first concept ID
		conceptID := conceptIDs[0]
		if !re.checkConditions(rule.Source.Conditions, source, conceptID) {
			return nil, nil // Conditions not met, skip entry
		}
	}

	// Generate base ID
	baseID := re.generateID(rule, source, personID)

	// Create a record for each concept ID (handles multi-mapping)
	var results []map[string]interface{}
	for i, conceptID := range conceptIDs {
		record := make(map[string]interface{})
		record["person_id"] = personID

		// Set the unique ID
		idFieldName := rule.Target.Table + "_id"
		record[idFieldName] = baseID + int64(i)

		// Set the type concept ID
		// OMOP type concept ID field names:
		// - condition_occurrence -> condition_type_concept_id
		// - procedure_occurrence -> procedure_type_concept_id
		// - drug_exposure -> drug_type_concept_id
		// - device_exposure -> device_type_concept_id
		// - measurement -> measurement_type_concept_id
		// - observation -> observation_type_concept_id
		typeFieldName := strings.TrimSuffix(strings.TrimSuffix(rule.Target.Table, "_occurrence"), "_exposure") + "_type_concept_id"
		record[typeFieldName] = rule.Target.TypeConceptID

		// Map each field
		for _, fm := range rule.Fields {
			if fm.Transform == "vocab" {
				// Already handled above
				record[fm.Target] = conceptID
				continue
			}

			// When entries are not required, treat all fields as optional
			isOptional := fm.Optional || !entriesRequired

			value, err := re.extractValue(source, fm.Source)
			if err != nil {
				if isOptional {
					continue
				}
				return nil, fmt.Errorf("error extracting %s: %w", fm.Source, err)
			}

			// Apply transform
			transform := re.transforms[fm.Transform]
			if transform == nil {
				transform = transformNone
			}

			transformed, err := transform(value, fm, ctx)
			if err != nil {
				if isOptional {
					continue
				}
				return nil, fmt.Errorf("error transforming %s: %w", fm.Source, err)
			}

			if transformed != nil {
				record[fm.Target] = transformed
			}
		}

		// Add the rule name for traceability
		record["mapping_rule"] = "RuleMapper:" + rule.Name

		results = append(results, record)
	}

	return results, nil
}

// extractValue extracts a value from a struct using dot notation path
// Supports | for fallback: "Field1|Field2" tries Field1 first, then Field2
func (re *RuleEngine) extractValue(source interface{}, path string) (interface{}, error) {
	// Handle fallback paths
	paths := strings.Split(path, "|")
	for _, p := range paths {
		p = strings.TrimSpace(p)
		val, err := re.extractSingleValue(source, p)
		if err == nil && !isZeroValue(val) {
			return val, nil
		}
	}
	return nil, fmt.Errorf("no value found for path: %s", path)
}

func (re *RuleEngine) extractSingleValue(source interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")

	// Handle map[string]interface{} (from rule-driven extraction)
	if m, ok := source.(map[string]interface{}); ok {
		return re.extractFromMap(m, parts)
	}

	// Handle structs (from typed parser)
	val := reflect.ValueOf(source)

	for _, part := range parts {
		// Handle pointer
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return nil, fmt.Errorf("nil pointer at %s", part)
			}
			val = val.Elem()
		}

		if val.Kind() != reflect.Struct {
			return nil, fmt.Errorf("expected struct at %s, got %s", part, val.Kind())
		}

		val = val.FieldByName(part)
		if !val.IsValid() {
			return nil, fmt.Errorf("field not found: %s", part)
		}
	}

	return val.Interface(), nil
}

// extractFromMap extracts a value from nested maps using dot notation
func (re *RuleEngine) extractFromMap(m map[string]interface{}, parts []string) (interface{}, error) {
	var current interface{} = m

	for i, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("key not found: %s", part)
			}
			current = val
		case *float64:
			if v == nil {
				return nil, fmt.Errorf("nil pointer at %s", part)
			}
			if i == len(parts)-1 {
				return *v, nil
			}
			return nil, fmt.Errorf("cannot traverse into float at %s", part)
		case *int64:
			if v == nil {
				return nil, fmt.Errorf("nil pointer at %s", part)
			}
			if i == len(parts)-1 {
				return *v, nil
			}
			return nil, fmt.Errorf("cannot traverse into int at %s", part)
		case *time.Time:
			if v == nil {
				return nil, fmt.Errorf("nil pointer at %s", part)
			}
			if i == len(parts)-1 {
				return *v, nil
			}
			return nil, fmt.Errorf("cannot traverse into time at %s", part)
		default:
			return nil, fmt.Errorf("cannot traverse into %T at %s", current, part)
		}
	}

	return current, nil
}

func isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	return val.IsZero()
}

// generateID generates a unique ID for a record
func (re *RuleEngine) generateID(rule MappingRule, source interface{}, personID int64) int64 {
	// Build a string from base fields
	var parts []string
	parts = append(parts, fmt.Sprintf("%d", personID))

	for _, field := range rule.IDGen.BaseFields {
		val, err := re.extractValue(source, field)
		if err == nil {
			parts = append(parts, fmt.Sprintf("%v", val))
		}
	}

	// Use the appropriate generator based on target table
	switch rule.Target.Table {
	case "condition_occurrence":
		code, _ := re.extractValue(source, "Code.Code")
		date, _ := re.extractValue(source, "EffectiveTime.Low")
		if t, ok := date.(time.Time); ok && !t.IsZero() {
			return omop.GenerateConditionID(personID, fmt.Sprintf("%v", code), t.Format("2006-01-02"))
		}
		date, _ = re.extractValue(source, "EffectiveTime.Value")
		if t, ok := date.(time.Time); ok && !t.IsZero() {
			return omop.GenerateConditionID(personID, fmt.Sprintf("%v", code), t.Format("2006-01-02"))
		}
		return omop.GenerateConditionID(personID, fmt.Sprintf("%v", code), time.Now().Format("2006-01-02"))

	case "drug_exposure":
		code, _ := re.extractValue(source, "Code.Code")
		date := re.getEffectiveDate(source)
		return omop.GenerateDrugExposureID(personID, fmt.Sprintf("%v", code), date.Format("2006-01-02"))

	case "procedure_occurrence":
		code, _ := re.extractValue(source, "Code.Code")
		date := re.getEffectiveDate(source)
		return omop.GenerateProcedureID(personID, fmt.Sprintf("%v", code), date.Format("2006-01-02"))

	case "measurement":
		code, _ := re.extractValue(source, "Code.Code")
		date := re.getEffectiveDate(source)
		value, _ := re.extractValue(source, "Value")
		return omop.GenerateMeasurementID(personID, fmt.Sprintf("%v", code), date.Format("2006-01-02"), fmt.Sprintf("%v", value))

	case "observation":
		// Try Substance.Code for allergies, Code.Code for others
		code, err := re.extractValue(source, "Substance.Code")
		if err != nil {
			code, _ = re.extractValue(source, "Code.Code")
		}
		date := re.getEffectiveDate(source)
		return omop.GenerateObservationID(personID, fmt.Sprintf("%v", code), date.Format("2006-01-02"))

	case "device_exposure":
		code, _ := re.extractValue(source, "Code.Code")
		date := re.getEffectiveDate(source)
		return omop.GenerateDeviceExposureID(personID, fmt.Sprintf("%v", code), date.Format("2006-01-02"))

	default:
		// Generic hash
		return omop.GenerateConditionID(personID, strings.Join(parts, ":"), time.Now().Format("2006-01-02"))
	}
}

func (re *RuleEngine) getEffectiveDate(source interface{}) time.Time {
	// Try various date field patterns
	if date, err := re.extractValue(source, "EffectiveTime.Low"); err == nil {
		if t, ok := date.(time.Time); ok && !t.IsZero() {
			return t
		}
	}
	if date, err := re.extractValue(source, "EffectiveTime.Value"); err == nil {
		if t, ok := date.(time.Time); ok && !t.IsZero() {
			return t
		}
	}
	if date, err := re.extractValue(source, "EffectiveTime"); err == nil {
		if t, ok := date.(time.Time); ok && !t.IsZero() {
			return t
		}
	}
	return time.Now()
}

// Transform functions

func transformNone(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	return value, nil
}

func transformDate(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	if t, ok := value.(time.Time); ok {
		if t.IsZero() {
			return nil, nil
		}
		return t, nil
	}
	return nil, fmt.Errorf("expected time.Time, got %T", value)
}

func transformTimePtr(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	if t, ok := value.(time.Time); ok {
		if t.IsZero() {
			return nil, nil
		}
		return &t, nil
	}
	return nil, nil
}

func transformFloat(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	switch v := value.(type) {
	case float64:
		if v == 0 {
			return nil, nil
		}
		return v, nil
	case float32:
		if v == 0 {
			return nil, nil
		}
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if v == "" {
			return nil, nil
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to float", value)
	}
}

func transformInt(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case string:
		if v == "" {
			return nil, nil
		}
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		return i, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to int", value)
	}
}

func transformString(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	return fmt.Sprintf("%v", value), nil
}

func (re *RuleEngine) transformVocab(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	code, ok := value.(string)
	if !ok || code == "" {
		return int64(0), nil
	}

	codeSystem := ""
	if fm.VocabField != "" {
		// VocabField can use | for fallbacks just like Source
		if cs, err := re.extractValue(ctx.Source, fm.VocabField); err == nil {
			codeSystem, _ = cs.(string)
		}
	}

	// Return first concept ID (multi-mapping handled at higher level)
	return re.vocab.MapConditionCode(code, codeSystem), nil
}

func (re *RuleEngine) transformUnit(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	unit, ok := value.(string)
	if !ok || unit == "" {
		return nil, nil
	}
	return re.vocab.MapUnitCode(unit), nil
}

func (re *RuleEngine) transformRoute(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	code, ok := value.(string)
	if !ok || code == "" {
		return nil, nil
	}

	codeSystem := ""
	if fm.VocabField != "" {
		if cs, err := re.extractValue(ctx.Source, fm.VocabField); err == nil {
			codeSystem, _ = cs.(string)
		}
	}

	return re.vocab.MapRouteCode(code, codeSystem), nil
}

// transformValueVocab maps a coded value (e.g., observation value, interpretation) to an OMOP concept ID
// This is used for value_as_concept_id fields in MEASUREMENT and OBSERVATION tables
func (re *RuleEngine) transformValueVocab(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	code, ok := value.(string)
	if !ok || code == "" {
		return nil, nil
	}

	codeSystem := ""
	if fm.VocabField != "" {
		if cs, err := re.extractValue(ctx.Source, fm.VocabField); err == nil {
			codeSystem, _ = cs.(string)
		}
	}

	// Use the appropriate mapper based on target field
	conceptID := re.vocab.MapObservationValueCode(code, codeSystem)
	if conceptID == 0 {
		return nil, nil
	}
	return conceptID, nil
}

func transformFormatSource(value interface{}, fm FieldMapping, ctx *MappingContext) (interface{}, error) {
	// Handle ccda.CodedValue (from typed parser)
	if cv, ok := value.(ccda.CodedValue); ok {
		if cv.DisplayName != "" {
			return cv.DisplayName, nil
		}
		if cv.Code != "" {
			return cv.Code, nil
		}
		return cv.OriginalText, nil
	}
	// Handle map[string]interface{} (from rule-driven extraction)
	if m, ok := value.(map[string]interface{}); ok {
		if dn, ok := m["DisplayName"].(string); ok && dn != "" {
			return dn, nil
		}
		if code, ok := m["Code"].(string); ok && code != "" {
			return code, nil
		}
		if ot, ok := m["OriginalText"].(string); ok && ot != "" {
			return ot, nil
		}
	}
	return fmt.Sprintf("%v", value), nil
}

// checkConditions evaluates all conditions for an entry
// Returns true if all conditions pass, false if any condition fails
func (re *RuleEngine) checkConditions(conditions []Condition, source interface{}, conceptID int64) bool {
	for _, cond := range conditions {
		if !re.checkCondition(cond, source, conceptID) {
			return false
		}
	}
	return true
}

// checkCondition evaluates a single condition
func (re *RuleEngine) checkCondition(cond Condition, source interface{}, conceptID int64) bool {
	switch cond.Type {
	case "domain_equals":
		domain := re.vocab.GetConceptDomain(conceptID)
		return domain == cond.Value

	case "domain_not_equals":
		domain := re.vocab.GetConceptDomain(conceptID)
		return domain != cond.Value

	case "field_equals":
		value, err := re.extractValue(source, cond.Field)
		if err != nil {
			return false
		}
		return fmt.Sprintf("%v", value) == cond.Value

	case "field_not_equals":
		value, err := re.extractValue(source, cond.Field)
		if err != nil {
			return true // Field not found means it's not equal
		}
		return fmt.Sprintf("%v", value) != cond.Value

	case "field_contains":
		value, err := re.extractValue(source, cond.Field)
		if err != nil {
			return false
		}
		return strings.Contains(fmt.Sprintf("%v", value), cond.Value)

	default:
		// Unknown condition type - pass by default
		return true
	}
}
