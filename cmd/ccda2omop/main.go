package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ccda2omop/internal/converter"
)

func main() {
	inputFile := flag.String("input", "", "Path to C-CDA XML input file (required)")
	outputDir := flag.String("output", "./output", "Directory for OMOP CSV output files")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	conceptFile := flag.String("concept", "", "Path to OMOP CONCEPT.csv vocabulary file")
	relationshipFile := flag.String("relationship", "", "Path to OMOP CONCEPT_RELATIONSHIP.csv file")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ccda2omop - Convert C-CDA XML documents to OMOP CDM 5.3 CSV files\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  ccda2omop -input <file.xml> [-output <dir>] [-concept <vocab.csv>] [-relationship <rel.csv>] [-verbose]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *inputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		log.Fatalf("Input file does not exist: %s", *inputFile)
	}

	cfg := converter.Config{
		InputFile:        *inputFile,
		OutputDir:        *outputDir,
		Verbose:          *verbose,
		ConceptFile:      *conceptFile,
		RelationshipFile: *relationshipFile,
	}

	if err := converter.Run(cfg); err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	fmt.Printf("Conversion complete. Output written to: %s\n", *outputDir)
}
