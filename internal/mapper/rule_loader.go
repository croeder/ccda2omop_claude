package mapper

import (
	"fmt"
	"os"

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

// LoadRulesFromYAML loads mapping rules from a YAML file
func LoadRulesFromYAML(filename string) ([]MappingRule, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

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
