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
	InputFile string
	OutputDir string
	Verbose   bool
}

func Run(cfg Config) error {
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
	m := mapper.New(cfg.Verbose)
	omopData, err := m.MapDocument(doc)
	if err != nil {
		return fmt.Errorf("failed to map C-CDA to OMOP: %w", err)
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
