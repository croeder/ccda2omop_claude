// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLRulesFile represents the top-level structure of the YAML rules file
type YAMLRulesFile struct {
	Rules []YAMLRule `yaml:"rules"`
}

// YAMLRule represents a single mapping rule in YAML format
type YAMLRule struct {
	Name   string         `yaml:"name"`
	Source YAMLSourceSpec `yaml:"source"`
	Target YAMLTargetSpec `yaml:"target"`
	Fields []YAMLField    `yaml:"fields"`
	IDGen  YAMLIDGenSpec  `yaml:"id_gen"`
}

// YAMLSourceSpec represents the source specification in YAML
type YAMLSourceSpec struct {
	Section              string           `yaml:"section"`
	SectionOID           string           `yaml:"section_oid"`
	SectionOIDEntriesReq string           `yaml:"section_oid_entries_required"`
	EntryXPath           string           `yaml:"entry_xpath"`
	Extraction           []YAMLExtraction `yaml:"extraction,omitempty"`
	EntryType            string           `yaml:"entry_type"`
	Conditions           []YAMLCondition  `yaml:"conditions,omitempty"`
}

// YAMLExtraction represents a field extraction specification in YAML
type YAMLExtraction struct {
	Field string `yaml:"field"` // Target field name
	XPath string `yaml:"xpath"` // XPath expression relative to entry
	Type  string `yaml:"type"`  // Value type: code, time, float, int, string, effective_time, quantity
}

// YAMLCondition represents a filter condition in YAML
type YAMLCondition struct {
	Type  string `yaml:"type"`  // domain_equals, domain_not_equals, field_equals, field_not_equals, field_contains
	Field string `yaml:"field"` // For field conditions: field path to check
	Value string `yaml:"value"` // Value to compare against
}

// YAMLTargetSpec represents the target specification in YAML
type YAMLTargetSpec struct {
	Table         string `yaml:"table"`
	TypeConceptID int64  `yaml:"type_concept_id"`
}

// YAMLField represents a field mapping in YAML
type YAMLField struct {
	Source     string `yaml:"source"`
	Target     string `yaml:"target"`
	Transform  string `yaml:"transform"`
	VocabField string `yaml:"vocab_field,omitempty"`
	Optional   bool   `yaml:"optional,omitempty"`
}

// YAMLIDGenSpec represents ID generation specification in YAML
type YAMLIDGenSpec struct {
	BaseFields []string `yaml:"base_fields"`
	Generator  string   `yaml:"generator"`
}

// LoadRulesFromYAML loads mapping rules from a YAML file or directory.
// If path is a directory, it loads all .yaml/.yml files from that directory.
// If path is a file, it loads rules from that single file (supports both
// single-rule format and multi-rule format with "rules:" key).
func LoadRulesFromYAML(path string) ([]MappingRule, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access rules path: %w", err)
	}

	if info.IsDir() {
		return loadRulesFromDirectory(path)
	}
	return loadRulesFromFile(path)
}

// loadRulesFromDirectory loads all YAML rule files from a directory
func loadRulesFromDirectory(dir string) ([]MappingRule, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules directory: %w", err)
	}

	var rules []MappingRule
	var filenames []string

	// Collect and sort filenames for deterministic ordering
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			filenames = append(filenames, name)
		}
	}
	sort.Strings(filenames)

	// Load each file
	for _, filename := range filenames {
		filepath := filepath.Join(dir, filename)
		fileRules, err := loadRulesFromFile(filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", filename, err)
		}
		rules = append(rules, fileRules...)
	}

	return rules, nil
}

// loadRulesFromFile loads rules from a single YAML file.
// Supports both single-rule format (rule at top level) and
// multi-rule format (rules under "rules:" key).
func loadRulesFromFile(filename string) ([]MappingRule, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	// Try parsing as a single rule first (new format)
	var singleRule YAMLRule
	if err := yaml.Unmarshal(data, &singleRule); err == nil && singleRule.Name != "" {
		return []MappingRule{convertYAMLRule(singleRule)}, nil
	}

	// Try parsing as multi-rule file (old format with "rules:" key)
	var yamlFile YAMLRulesFile
	if err := yaml.Unmarshal(data, &yamlFile); err != nil {
		return nil, fmt.Errorf("failed to parse rules file: %w", err)
	}

	rules := make([]MappingRule, 0, len(yamlFile.Rules))
	for _, yr := range yamlFile.Rules {
		rule := convertYAMLRule(yr)
		rules = append(rules, rule)
	}

	return rules, nil
}

// convertYAMLRule converts a YAMLRule to a MappingRule
func convertYAMLRule(yr YAMLRule) MappingRule {
	fields := make([]FieldMapping, 0, len(yr.Fields))
	for _, yf := range yr.Fields {
		fields = append(fields, FieldMapping{
			Source:     yf.Source,
			Target:     yf.Target,
			Transform:  yf.Transform,
			VocabField: yf.VocabField,
			Optional:   yf.Optional,
		})
	}

	conditions := make([]Condition, 0, len(yr.Source.Conditions))
	for _, yc := range yr.Source.Conditions {
		conditions = append(conditions, Condition{
			Type:  yc.Type,
			Field: yc.Field,
			Value: yc.Value,
		})
	}

	extractions := make([]Extraction, 0, len(yr.Source.Extraction))
	for _, ye := range yr.Source.Extraction {
		extractions = append(extractions, Extraction{
			Field: ye.Field,
			XPath: ye.XPath,
			Type:  ye.Type,
		})
	}

	return MappingRule{
		Name: yr.Name,
		Source: SourceSpec{
			Section:              yr.Source.Section,
			SectionOID:           yr.Source.SectionOID,
			SectionOIDEntriesReq: yr.Source.SectionOIDEntriesReq,
			EntryXPath:           yr.Source.EntryXPath,
			Extraction:           extractions,
			EntryType:            yr.Source.EntryType,
			Conditions:           conditions,
		},
		Target: TargetSpec{
			Table:         yr.Target.Table,
			TypeConceptID: yr.Target.TypeConceptID,
		},
		Fields: fields,
		IDGen: IDGenSpec{
			BaseFields: yr.IDGen.BaseFields,
			Generator:  yr.IDGen.Generator,
		},
	}
}

// IndexRulesBySection converts a slice of rules to a map keyed by section name
func IndexRulesBySection(rules []MappingRule) map[string][]MappingRule {
	result := make(map[string][]MappingRule)
	for _, rule := range rules {
		section := rule.Source.Section
		result[section] = append(result[section], rule)
	}
	return result
}

// GetRuleBySectionFromList returns a rule by section name from a list of rules
func GetRuleBySectionFromList(rules []MappingRule, section string) *MappingRule {
	for i := range rules {
		if rules[i].Source.Section == section {
			return &rules[i]
		}
	}
	return nil
}

// GetRuleByNameFromList returns a rule by name from a list of rules
func GetRuleByNameFromList(rules []MappingRule, name string) *MappingRule {
	for i := range rules {
		if rules[i].Name == name {
			return &rules[i]
		}
	}
	return nil
}
