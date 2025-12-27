// Copyright 2025 Christophe Roeder. All rights reserved.

package converter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccda2omop/internal/ccda"
	"github.com/ccda2omop/internal/mapper"
	"github.com/ccda2omop/internal/omop"
	"github.com/ccda2omop/internal/report"
)

type Config struct {
	InputFile        string
	OutputDir        string
	Verbose          bool
	ConceptFile      string // Path to CONCEPT.csv
	RelationshipFile string // Path to CONCEPT_RELATIONSHIP.csv
	VocabDir         string // Path to directory with supplementary vocabulary files (e.g., CVX.csv)
	RulesFile        string // Path to YAML rules file (optional, uses Go-defined rules if empty)
	GenerateReport   bool   // Generate conversion report
}

// ConversionSummary holds counts of records created during conversion
type ConversionSummary struct {
	Persons              int
	VisitOccurrences     int
	ConditionOccurrences int
	DrugExposures        int
	ProcedureOccurrences int
	Measurements         int
	Observations         int
	DeviceExposures      int
	Report               *report.ConversionReport
}

// Shared vocab loader for batch processing
var sharedVocabLoader *mapper.VocabLoader

// LoadVocabulary loads vocabulary files and caches them for reuse
func LoadVocabulary(conceptFile, relationshipFile, vocabDir string, verbose bool) error {
	if sharedVocabLoader != nil {
		return nil // Already loaded
	}

	if conceptFile == "" {
		return nil // No vocabulary files specified
	}

	if verbose {
		log.Printf("Loading OMOP vocabulary from %s", conceptFile)
	}

	loader := mapper.NewVocabLoader()

	if err := loader.LoadConcepts(conceptFile); err != nil {
		return fmt.Errorf("failed to load CONCEPT.csv: %w", err)
	}

	if relationshipFile != "" {
		if verbose {
			log.Printf("Loading concept relationships from %s", relationshipFile)
		}
		if err := loader.LoadConceptRelationships(relationshipFile); err != nil {
			return fmt.Errorf("failed to load CONCEPT_RELATIONSHIP.csv: %w", err)
		}
	}

	// Load supplementary vocabularies from directory if provided
	if vocabDir != "" {
		if err := loadSupplementaryVocabs(loader, vocabDir, verbose); err != nil {
			return fmt.Errorf("failed to load supplementary vocabularies: %w", err)
		}
	}

	sharedVocabLoader = loader
	return nil
}

// loadSupplementaryVocabs loads all CSV files from a directory as supplementary vocabularies
func loadSupplementaryVocabs(vocabLoader *mapper.VocabLoader, dir string, verbose bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read vocab directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".csv") {
			filePath := filepath.Join(dir, name)
			if err := vocabLoader.LoadSupplementaryVocab(filePath); err != nil {
				return fmt.Errorf("failed to load %s: %w", name, err)
			}
		}
	}
	return nil
}

// RunBatch processes multiple C-CDA files and aggregates results into a single output
func RunBatch(files []string, cfg Config) (*ConversionSummary, error) {
	// Load vocabulary if specified and not already loaded
	if cfg.ConceptFile != "" && sharedVocabLoader == nil {
		if err := LoadVocabulary(cfg.ConceptFile, cfg.RelationshipFile, cfg.VocabDir, cfg.Verbose); err != nil {
			return nil, err
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Aggregate all OMOP data
	aggregatedData := &omop.OMOPData{}

	// Initialize report if requested
	var convReport *report.ConversionReport
	if cfg.GenerateReport {
		convReport = report.NewConversionReport()
	}

	for i, inputFile := range files {
		if cfg.Verbose {
			log.Printf("Processing file %d/%d: %s", i+1, len(files), inputFile)
		}

		omopData, err := processFile(inputFile, cfg)
		if err != nil {
			if convReport != nil {
				convReport.AddDocument(true)
			}
			return nil, fmt.Errorf("failed to process %s: %w", inputFile, err)
		}

		if convReport != nil {
			convReport.AddDocument(false)
		}

		// Set source file on all records (use base filename for readability)
		sourceFile := filepath.Base(inputFile)
		setSourceFile(omopData, sourceFile)

		// Aggregate the data
		aggregatedData.Persons = append(aggregatedData.Persons, omopData.Persons...)
		aggregatedData.VisitOccurrences = append(aggregatedData.VisitOccurrences, omopData.VisitOccurrences...)
		aggregatedData.ConditionOccurrences = append(aggregatedData.ConditionOccurrences, omopData.ConditionOccurrences...)
		aggregatedData.DrugExposures = append(aggregatedData.DrugExposures, omopData.DrugExposures...)
		aggregatedData.ProcedureOccurrences = append(aggregatedData.ProcedureOccurrences, omopData.ProcedureOccurrences...)
		aggregatedData.Measurements = append(aggregatedData.Measurements, omopData.Measurements...)
		aggregatedData.Observations = append(aggregatedData.Observations, omopData.Observations...)
		aggregatedData.DeviceExposures = append(aggregatedData.DeviceExposures, omopData.DeviceExposures...)
	}

	// Write aggregated OMOP CSV files
	writer := omop.NewCSVWriter(cfg.OutputDir)
	if err := writer.WriteAll(aggregatedData); err != nil {
		return nil, fmt.Errorf("failed to write OMOP CSV files: %w", err)
	}

	// Calculate report from aggregated data if requested
	if convReport != nil {
		convReport.CalculateFromOMOPData(aggregatedData)
	}

	// Build summary
	summary := &ConversionSummary{
		Persons:              len(aggregatedData.Persons),
		VisitOccurrences:     len(aggregatedData.VisitOccurrences),
		ConditionOccurrences: len(aggregatedData.ConditionOccurrences),
		DrugExposures:        len(aggregatedData.DrugExposures),
		ProcedureOccurrences: len(aggregatedData.ProcedureOccurrences),
		Measurements:         len(aggregatedData.Measurements),
		Observations:         len(aggregatedData.Observations),
		DeviceExposures:      len(aggregatedData.DeviceExposures),
		Report:               convReport,
	}

	if cfg.Verbose {
		log.Printf("Wrote %d person records", summary.Persons)
		log.Printf("Wrote %d visit_occurrence records", summary.VisitOccurrences)
		log.Printf("Wrote %d condition_occurrence records", summary.ConditionOccurrences)
		log.Printf("Wrote %d drug_exposure records", summary.DrugExposures)
		log.Printf("Wrote %d procedure_occurrence records", summary.ProcedureOccurrences)
		log.Printf("Wrote %d measurement records", summary.Measurements)
		log.Printf("Wrote %d observation records", summary.Observations)
		log.Printf("Wrote %d device_exposure records", summary.DeviceExposures)
	}

	return summary, nil
}

// processFile processes a single C-CDA file and returns OMOP data without writing
func processFile(inputFile string, cfg Config) (*omop.OMOPData, error) {
	if cfg.Verbose {
		log.Printf("Parsing C-CDA file: %s", inputFile)
	}

	// Parse the C-CDA document
	doc, err := ccda.ParseFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse C-CDA file: %w", err)
	}

	if cfg.Verbose {
		log.Printf("Successfully parsed C-CDA document for patient: %s %s",
			doc.Patient.Name.Given, doc.Patient.Name.Family)
	}

	// Map C-CDA to OMOP using rule-based mapper
	var rm *mapper.RuleBasedMapper

	if cfg.RulesFile != "" {
		// Load rules from YAML file
		if cfg.Verbose {
			log.Printf("Loading mapping rules from %s", cfg.RulesFile)
		}
		if sharedVocabLoader != nil {
			rm, err = mapper.NewRuleBasedMapperFromYAMLWithLoader(cfg.RulesFile, sharedVocabLoader, cfg.Verbose)
		} else {
			rm, err = mapper.NewRuleBasedMapperFromYAML(cfg.RulesFile, mapper.NewVocabularyMapper(), cfg.Verbose)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load rules file: %w", err)
		}
	} else {
		// Use Go-defined rules
		if sharedVocabLoader != nil {
			rm = mapper.NewRuleBasedMapperWithLoader(sharedVocabLoader, cfg.Verbose)
		} else {
			rm = mapper.NewRuleBasedMapper(mapper.NewVocabularyMapper(), cfg.Verbose)
		}
	}

	return rm.MapDocument(doc)
}

func Run(cfg Config) error {
	// Load vocabulary if specified and not already loaded
	if cfg.ConceptFile != "" && sharedVocabLoader == nil {
		if err := LoadVocabulary(cfg.ConceptFile, cfg.RelationshipFile, cfg.VocabDir, cfg.Verbose); err != nil {
			return err
		}
	}

	if cfg.Verbose {
		log.Printf("Parsing C-CDA file: %s", cfg.InputFile)
	}

	// Parse the C-CDA document
	doc, err := ccda.ParseFile(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("failed to parse C-CDA file: %w", err)
	}

	if cfg.Verbose {
		log.Printf("Successfully parsed C-CDA document for patient: %s %s",
			doc.Patient.Name.Given, doc.Patient.Name.Family)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Map C-CDA to OMOP using rule-based mapper
	var rm *mapper.RuleBasedMapper

	if cfg.RulesFile != "" {
		// Load rules from YAML file
		if cfg.Verbose {
			log.Printf("Loading mapping rules from %s", cfg.RulesFile)
		}
		if sharedVocabLoader != nil {
			rm, err = mapper.NewRuleBasedMapperFromYAMLWithLoader(cfg.RulesFile, sharedVocabLoader, cfg.Verbose)
		} else {
			rm, err = mapper.NewRuleBasedMapperFromYAML(cfg.RulesFile, mapper.NewVocabularyMapper(), cfg.Verbose)
		}
		if err != nil {
			return fmt.Errorf("failed to load rules file: %w", err)
		}
	} else {
		// Use Go-defined rules
		if sharedVocabLoader != nil {
			rm = mapper.NewRuleBasedMapperWithLoader(sharedVocabLoader, cfg.Verbose)
		} else {
			rm = mapper.NewRuleBasedMapper(mapper.NewVocabularyMapper(), cfg.Verbose)
		}
	}

	omopData, err := rm.MapDocument(doc)
	if err != nil {
		return fmt.Errorf("failed to map C-CDA to OMOP: %w", err)
	}

	// Set source file on all records
	sourceFile := filepath.Base(cfg.InputFile)
	setSourceFile(omopData, sourceFile)

	// Write OMOP CSV files
	writer := omop.NewCSVWriter(cfg.OutputDir)
	if err := writer.WriteAll(omopData); err != nil {
		return fmt.Errorf("failed to write OMOP CSV files: %w", err)
	}

	if cfg.Verbose {
		log.Printf("Wrote %d person records", len(omopData.Persons))
		log.Printf("Wrote %d visit_occurrence records", len(omopData.VisitOccurrences))
		log.Printf("Wrote %d condition_occurrence records", len(omopData.ConditionOccurrences))
		log.Printf("Wrote %d drug_exposure records", len(omopData.DrugExposures))
		log.Printf("Wrote %d procedure_occurrence records", len(omopData.ProcedureOccurrences))
		log.Printf("Wrote %d measurement records", len(omopData.Measurements))
		log.Printf("Wrote %d observation records", len(omopData.Observations))
		log.Printf("Wrote %d device_exposure records", len(omopData.DeviceExposures))
	}

	return nil
}

// setSourceFile sets the SourceFile field on all records in the OMOP data
func setSourceFile(data *omop.OMOPData, sourceFile string) {
	for i := range data.Persons {
		data.Persons[i].SourceFile = sourceFile
	}
	for i := range data.VisitOccurrences {
		data.VisitOccurrences[i].SourceFile = sourceFile
	}
	for i := range data.ConditionOccurrences {
		data.ConditionOccurrences[i].SourceFile = sourceFile
	}
	for i := range data.DrugExposures {
		data.DrugExposures[i].SourceFile = sourceFile
	}
	for i := range data.ProcedureOccurrences {
		data.ProcedureOccurrences[i].SourceFile = sourceFile
	}
	for i := range data.Measurements {
		data.Measurements[i].SourceFile = sourceFile
	}
	for i := range data.Observations {
		data.Observations[i].SourceFile = sourceFile
	}
	for i := range data.DeviceExposures {
		data.DeviceExposures[i].SourceFile = sourceFile
	}
}
