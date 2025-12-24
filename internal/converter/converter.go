package converter

import (
	"fmt"
	"log"
	"os"

	"github.com/ccda2omop/internal/ccda"
	"github.com/ccda2omop/internal/mapper"
	"github.com/ccda2omop/internal/omop"
)

type Config struct {
	InputFile        string
	OutputDir        string
	Verbose          bool
	ConceptFile      string // Path to CONCEPT.csv
	RelationshipFile string // Path to CONCEPT_RELATIONSHIP.csv
	UseRules         bool   // Use rule-based mapper instead of hardcoded mapper
	RulesFile        string // Path to YAML rules file (optional, uses Go-defined rules if empty)
}

// Shared vocab loader for batch processing
var sharedVocabLoader *mapper.VocabLoader

// LoadVocabulary loads vocabulary files and caches them for reuse
func LoadVocabulary(conceptFile, relationshipFile string, verbose bool) error {
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

	sharedVocabLoader = loader
	return nil
}

// RunBatch processes multiple C-CDA files and aggregates results into a single output
func RunBatch(files []string, cfg Config) error {
	// Load vocabulary if specified and not already loaded
	if cfg.ConceptFile != "" && sharedVocabLoader == nil {
		if err := LoadVocabulary(cfg.ConceptFile, cfg.RelationshipFile, cfg.Verbose); err != nil {
			return err
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Aggregate all OMOP data
	aggregatedData := &omop.OMOPData{}

	for i, inputFile := range files {
		if cfg.Verbose {
			log.Printf("Processing file %d/%d: %s", i+1, len(files), inputFile)
		}

		omopData, err := processFile(inputFile, cfg)
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", inputFile, err)
		}

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
		return fmt.Errorf("failed to write OMOP CSV files: %w", err)
	}

	if cfg.Verbose {
		log.Printf("Wrote %d person records", len(aggregatedData.Persons))
		log.Printf("Wrote %d visit_occurrence records", len(aggregatedData.VisitOccurrences))
		log.Printf("Wrote %d condition_occurrence records", len(aggregatedData.ConditionOccurrences))
		log.Printf("Wrote %d drug_exposure records", len(aggregatedData.DrugExposures))
		log.Printf("Wrote %d procedure_occurrence records", len(aggregatedData.ProcedureOccurrences))
		log.Printf("Wrote %d measurement records", len(aggregatedData.Measurements))
		log.Printf("Wrote %d observation records", len(aggregatedData.Observations))
		log.Printf("Wrote %d device_exposure records", len(aggregatedData.DeviceExposures))
	}

	return nil
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

	// Map C-CDA to OMOP
	if cfg.UseRules {
		// Use rule-based mapper
		var rm *mapper.RuleBasedMapper
		var err error

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

	// Use traditional mapper
	var m *mapper.Mapper
	if sharedVocabLoader != nil {
		m = mapper.NewWithVocabLoader(sharedVocabLoader, cfg.Verbose)
	} else {
		m = mapper.New(cfg.Verbose)
	}
	return m.MapDocument(doc)
}

func Run(cfg Config) error {
	// Load vocabulary if specified and not already loaded
	if cfg.ConceptFile != "" && sharedVocabLoader == nil {
		if err := LoadVocabulary(cfg.ConceptFile, cfg.RelationshipFile, cfg.Verbose); err != nil {
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

	// Map C-CDA to OMOP
	var omopData *omop.OMOPData

	if cfg.UseRules {
		// Use rule-based mapper
		var rm *mapper.RuleBasedMapper
		var err error

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

		omopData, err = rm.MapDocument(doc)
		if err != nil {
			return fmt.Errorf("failed to map C-CDA to OMOP (rules): %w", err)
		}
	} else {
		// Use traditional mapper
		var m *mapper.Mapper
		if sharedVocabLoader != nil {
			m = mapper.NewWithVocabLoader(sharedVocabLoader, cfg.Verbose)
		} else {
			m = mapper.New(cfg.Verbose)
		}
		var err error
		omopData, err = m.MapDocument(doc)
		if err != nil {
			return fmt.Errorf("failed to map C-CDA to OMOP: %w", err)
		}
	}

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
