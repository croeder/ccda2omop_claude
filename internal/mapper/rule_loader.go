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
	Section   string `yaml:"section"`
	EntryType string `yaml:"entry_type"`
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

	return MappingRule{
		Name: yr.Name,
		Source: SourceSpec{
			Section:   yr.Source.Section,
			EntryType: yr.Source.EntryType,
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
