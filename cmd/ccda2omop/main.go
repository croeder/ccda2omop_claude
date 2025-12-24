package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ccda2omop/internal/analyzer"
	"github.com/ccda2omop/internal/converter"
	"github.com/ccda2omop/internal/mapper"
)

func main() {
	inputFile := flag.String("input", "", "Path to C-CDA XML input file (required)")
	outputDir := flag.String("output", "./output", "Directory for OMOP CSV output files")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	conceptFile := flag.String("concept", "", "Path to OMOP CONCEPT.csv vocabulary file")
	relationshipFile := flag.String("relationship", "", "Path to OMOP CONCEPT_RELATIONSHIP.csv file")
	useRules := flag.Bool("rules", false, "Use rule-based mapper")
	rulesFile := flag.String("rules-file", "", "Path to YAML rules file or directory (implies -rules)")
	analyze := flag.Bool("analyze", false, "Analyze input file and show code mappings (requires -concept)")
	analyzeOutput := flag.String("analyze-output", "", "Output CSV file for analysis (default: stdout)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ccda2omop - Convert C-CDA XML documents to OMOP CDM 5.3 CSV files\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  ccda2omop -input <file.xml> [-output <dir>] [-concept <vocab.csv>] [-relationship <rel.csv>] [-rules] [-rules-file <rules.yaml>] [-verbose]\n")
		fmt.Fprintf(os.Stderr, "  ccda2omop -input <file.xml> -analyze -concept <vocab.csv> [-relationship <rel.csv>] [-analyze-output <file.csv>]\n\n")
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

	// Analyze mode
	if *analyze {
		if err := runAnalyze(*inputFile, *conceptFile, *relationshipFile, *analyzeOutput, *verbose); err != nil {
			log.Fatalf("Analysis failed: %v", err)
		}
		return
	}

	// Convert mode
	cfg := converter.Config{
		InputFile:        *inputFile,
		OutputDir:        *outputDir,
		Verbose:          *verbose,
		ConceptFile:      *conceptFile,
		RelationshipFile: *relationshipFile,
		UseRules:         *useRules || *rulesFile != "",
		RulesFile:        *rulesFile,
	}

	if err := converter.Run(cfg); err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	fmt.Printf("Conversion complete. Output written to: %s\n", *outputDir)
}

func runAnalyze(inputFile, conceptFile, relationshipFile, outputFile string, verbose bool) error {
	// Load vocabulary if provided
	var vocabLoader *mapper.VocabLoader
	if conceptFile != "" {
		if verbose {
			log.Printf("Loading OMOP vocabulary from %s", conceptFile)
		}
		vocabLoader = mapper.NewVocabLoader()
		if err := vocabLoader.LoadConcepts(conceptFile); err != nil {
			return fmt.Errorf("failed to load CONCEPT.csv: %w", err)
		}
		if relationshipFile != "" {
			if verbose {
				log.Printf("Loading concept relationships from %s", relationshipFile)
			}
			if err := vocabLoader.LoadConceptRelationships(relationshipFile); err != nil {
				return fmt.Errorf("failed to load CONCEPT_RELATIONSHIP.csv: %w", err)
			}
		}
	}

	// Create analyzer
	a := analyzer.New(vocabLoader, verbose)

	// Analyze file
	if verbose {
		log.Printf("Analyzing C-CDA file: %s", inputFile)
	}
	mappings, err := a.AnalyzeFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to analyze file: %w", err)
	}

	// Output results
	var output *os.File
	if outputFile != "" {
		output, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	// Write CSV
	if err := a.WriteCSV(mappings, output); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	// Print summary to stderr if output is to file
	if outputFile != "" {
		a.PrintSummary(mappings, os.Stderr)
		fmt.Fprintf(os.Stderr, "\nAnalysis written to: %s\n", outputFile)
	}

	return nil
}
