package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ccda2omop/internal/analyzer"
	"github.com/ccda2omop/internal/converter"
	"github.com/ccda2omop/internal/mapper"
)

func main() {
	inputPath := flag.String("input", "", "Path to C-CDA XML file or directory of XML files (required)")
	outputDir := flag.String("output", "./output", "Directory for OMOP CSV output files")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	conceptFile := flag.String("concept", "", "Path to OMOP CONCEPT.csv vocabulary file")
	relationshipFile := flag.String("relationship", "", "Path to OMOP CONCEPT_RELATIONSHIP.csv file")
	useRules := flag.Bool("rules", false, "Use rule-based mapper")
	rulesFile := flag.String("rules-file", "", "Path to YAML rules file or directory (implies -rules)")
	analyzeFlag := flag.Bool("analyze", false, "Analyze input file(s) and show code mappings (requires -concept)")
	analyzeOutput := flag.String("analyze-output", "", "Output CSV file for analysis (default: stdout)")
	summary := flag.Bool("summary", false, "Show summary of C-CDA sections to OMOP table mappings (use with -analyze)")
	vocabDir := flag.String("vocab-dir", "", "Path to directory containing supplementary vocabulary CSV files (e.g., CVX.csv, CPT4.csv)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "ccda2omop - Convert C-CDA XML documents to OMOP CDM 5.3 CSV files\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  ccda2omop -input <file.xml|dir> [-output <dir>] [-concept <vocab.csv>] [-relationship <rel.csv>] [-rules] [-rules-file <rules.yaml>] [-verbose]\n")
		fmt.Fprintf(os.Stderr, "  ccda2omop -input <file.xml|dir> -analyze -concept <vocab.csv> [-relationship <rel.csv>] [-analyze-output <file.csv>]\n")
		fmt.Fprintf(os.Stderr, "  ccda2omop -input <file.xml|dir> -analyze -summary -concept <vocab.csv> [-relationship <rel.csv>]\n\n")
		fmt.Fprintf(os.Stderr, "The -input flag accepts either a single XML file or a directory containing XML files.\n")
		fmt.Fprintf(os.Stderr, "When a directory is specified, all .xml files are processed and results are aggregated.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *inputPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	info, err := os.Stat(*inputPath)
	if os.IsNotExist(err) {
		log.Fatalf("Input path does not exist: %s", *inputPath)
	}
	if err != nil {
		log.Fatalf("Error accessing input path: %v", err)
	}

	// Collect XML files
	var xmlFiles []string
	if info.IsDir() {
		xmlFiles, err = findXMLFiles(*inputPath)
		if err != nil {
			log.Fatalf("Failed to find XML files: %v", err)
		}
		if len(xmlFiles) == 0 {
			log.Fatalf("No XML files found in directory: %s", *inputPath)
		}
		if *verbose {
			log.Printf("Found %d XML files in %s", len(xmlFiles), *inputPath)
		}
	} else {
		xmlFiles = []string{*inputPath}
	}

	// Analyze mode
	if *analyzeFlag {
		if err := runAnalyze(xmlFiles, *conceptFile, *relationshipFile, *vocabDir, *analyzeOutput, *summary, *verbose); err != nil {
			log.Fatalf("Analysis failed: %v", err)
		}
		return
	}

	// Convert mode - process all files and aggregate output
	cfg := converter.Config{
		OutputDir:        *outputDir,
		Verbose:          *verbose,
		ConceptFile:      *conceptFile,
		RelationshipFile: *relationshipFile,
		UseRules:         *useRules || *rulesFile != "",
		RulesFile:        *rulesFile,
	}

	if err := converter.RunBatch(xmlFiles, cfg); err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	fmt.Printf("Conversion complete. Processed %d file(s). Output written to: %s\n", len(xmlFiles), *outputDir)
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
			if verbose {
				log.Printf("Loading supplementary vocabulary from %s", filePath)
			}
			if err := vocabLoader.LoadSupplementaryVocab(filePath); err != nil {
				return fmt.Errorf("failed to load %s: %w", name, err)
			}
		}
	}
	return nil
}

// findXMLFiles returns a sorted list of XML files from a directory
func findXMLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".xml") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	return files, nil
}

func runAnalyze(inputFiles []string, conceptFile, relationshipFile, vocabDir, outputFile string, showSummary, verbose bool) error {
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
		// Load supplementary vocabularies from directory if provided
		if vocabDir != "" {
			if err := loadSupplementaryVocabs(vocabLoader, vocabDir, verbose); err != nil {
				return fmt.Errorf("failed to load supplementary vocabularies: %w", err)
			}
		}
	}

	// Create analyzer
	a := analyzer.New(vocabLoader, verbose)

	// Analyze all files and aggregate mappings
	var allMappings []analyzer.CodeMapping
	for i, inputFile := range inputFiles {
		if verbose {
			log.Printf("Analyzing file %d/%d: %s", i+1, len(inputFiles), inputFile)
		}
		mappings, err := a.AnalyzeFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to analyze file %s: %w", inputFile, err)
		}
		allMappings = append(allMappings, mappings...)
	}

	if verbose && len(inputFiles) > 1 {
		log.Printf("Aggregated %d mappings from %d files", len(allMappings), len(inputFiles))
	}

	// Summary mode - show C-CDA to OMOP table mapping summary
	if showSummary {
		a.WriteMappingSummary(allMappings, os.Stdout)
		return nil
	}

	// Output results
	var output *os.File
	var err error
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
	if err := a.WriteCSV(allMappings, output); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	// Print summary to stderr if output is to file
	if outputFile != "" {
		a.PrintSummary(allMappings, os.Stderr)
		fmt.Fprintf(os.Stderr, "\nAnalysis written to: %s\n", outputFile)
	}

	return nil
}
