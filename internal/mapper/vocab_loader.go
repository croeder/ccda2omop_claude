// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Concept represents a row from the OMOP CONCEPT table
type Concept struct {
	ConceptID       int64
	ConceptName     string
	DomainID        string
	VocabularyID    string
	ConceptClassID  string
	StandardConcept string
	ConceptCode     string
}

// VocabLoader loads and indexes OMOP vocabulary tables
type VocabLoader struct {
	// Index by vocabulary_id + concept_code -> Concept
	conceptIndex map[string]*Concept

	// Index by concept_id -> Concept (for relationship lookups)
	conceptByID map[int64]*Concept

	// Maps source concept_id -> target standard concept_ids (for "Maps to" relationships)
	// A single source concept can map to multiple standard concepts
	mapsTo map[int64][]int64

	// Vocabularies we care about
	relevantVocabs map[string]bool
}

// NewVocabLoader creates a new vocabulary loader
func NewVocabLoader() *VocabLoader {
	return &VocabLoader{
		conceptIndex: make(map[string]*Concept),
		conceptByID:  make(map[int64]*Concept),
		mapsTo:       make(map[int64][]int64),
		relevantVocabs: map[string]bool{
			"SNOMED":    true,
			"RxNorm":    true,
			"LOINC":     true,
			"ICD10CM":   true,
			"ICD9CM":    true,
			"CPT4":      true,
			"HCPCS":     true,
			"CVX":       true,
			"NDC":       true,
			"UNII":      true, // FDA Unique Ingredient Identifier (allergens)
			"NDFRT":     true, // National Drug File Reference Terminology
			"Gender":    true,
			"Race":      true,
			"Ethnicity": true,
			"UCUM":      true, // Units
			"Visit":     true,
		},
	}
}

// conceptKey creates a lookup key from vocabulary and code
func conceptKey(vocabID, code string) string {
	return vocabID + "|" + code
}

// LoadConcepts loads the CONCEPT.csv file
func (vl *VocabLoader) LoadConcepts(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open CONCEPT.csv: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	// Skip header
	if scanner.Scan() {
		// Verify header
		header := scanner.Text()
		if !strings.HasPrefix(header, "concept_id") {
			return fmt.Errorf("unexpected CONCEPT.csv header: %s", header)
		}
	}

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 10 {
			continue
		}

		vocabID := fields[3]
		// Only load relevant vocabularies to save memory
		if !vl.relevantVocabs[vocabID] {
			continue
		}

		conceptID, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}

		// Skip invalid concepts
		if fields[9] != "" { // invalid_reason
			continue
		}

		concept := &Concept{
			ConceptID:       conceptID,
			ConceptName:     fields[1],
			DomainID:        fields[2],
			VocabularyID:    vocabID,
			ConceptClassID:  fields[4],
			StandardConcept: fields[5],
			ConceptCode:     fields[6],
		}

		key := conceptKey(vocabID, concept.ConceptCode)
		vl.conceptIndex[key] = concept
		vl.conceptByID[conceptID] = concept
		count++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading CONCEPT.csv: %w", err)
	}

	log.Printf("Loaded %d concepts from vocabulary tables", count)
	return nil
}

// LoadConceptRelationships loads the CONCEPT_RELATIONSHIP.csv file
// Only loads "Maps to" relationships for mapping source to standard concepts
func (vl *VocabLoader) LoadConceptRelationships(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open CONCEPT_RELATIONSHIP.csv: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	// Skip header
	if scanner.Scan() {
		header := scanner.Text()
		if !strings.HasPrefix(header, "concept_id_1") {
			return fmt.Errorf("unexpected CONCEPT_RELATIONSHIP.csv header: %s", header)
		}
	}

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue
		}

		// Only load "Maps to" relationships
		if fields[2] != "Maps to" {
			continue
		}

		// Skip invalid relationships
		if fields[5] != "" { // invalid_reason
			continue
		}

		sourceID, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}

		targetID, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}

		// Only store if source concept is in our index
		if _, ok := vl.conceptByID[sourceID]; ok {
			vl.mapsTo[sourceID] = append(vl.mapsTo[sourceID], targetID)
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading CONCEPT_RELATIONSHIP.csv: %w", err)
	}

	log.Printf("Loaded %d 'Maps to' relationships", count)
	return nil
}

// LookupConcept finds a concept by vocabulary ID and code
func (vl *VocabLoader) LookupConcept(vocabID, code string) *Concept {
	key := conceptKey(vocabID, code)
	return vl.conceptIndex[key]
}

// LookupConceptByID finds a concept by its ID
func (vl *VocabLoader) LookupConceptByID(conceptID int64) *Concept {
	return vl.conceptByID[conceptID]
}

// GetStandardConceptID returns the first standard concept ID for a source concept
// For concepts with multiple mappings, use GetStandardConceptIDs instead
func (vl *VocabLoader) GetStandardConceptID(vocabID, code string) int64 {
	ids := vl.GetStandardConceptIDs(vocabID, code)
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

// GetConceptDomain returns the domain_id for a concept
// Returns empty string if concept not found
func (vl *VocabLoader) GetConceptDomain(conceptID int64) string {
	if concept := vl.conceptByID[conceptID]; concept != nil {
		return concept.DomainID
	}
	return ""
}

// GetStandardConceptIDs returns all standard concept IDs for a source concept
// A single source concept can map to multiple standard concepts
func (vl *VocabLoader) GetStandardConceptIDs(vocabID, code string) []int64 {
	concept := vl.LookupConcept(vocabID, code)
	if concept == nil {
		return nil
	}

	// If already standard, return it
	if concept.StandardConcept == "S" {
		return []int64{concept.ConceptID}
	}

	// Follow "Maps to" relationships
	if targetIDs, ok := vl.mapsTo[concept.ConceptID]; ok && len(targetIDs) > 0 {
		return targetIDs
	}

	// Return the source concept ID if no mapping found
	return []int64{concept.ConceptID}
}

// LoadSupplementaryVocab loads additional vocabulary concepts from a CSV file
// Uses the same format as CONCEPT.csv (tab-separated)
// Lines starting with # are treated as comments and skipped
func (vl *VocabLoader) LoadSupplementaryVocab(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open supplementary vocab file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Skip comment lines and find header
	foundHeader := false
	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Expect header line
		if !strings.HasPrefix(line, "concept_id") {
			return fmt.Errorf("unexpected header in supplementary vocab: %s", line)
		}
		foundHeader = true
		break
	}
	if !foundHeader {
		return fmt.Errorf("no header found in supplementary vocab file")
	}

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}

		conceptID, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}

		// Skip if invalid_reason is set (field 9 if present)
		if len(fields) > 9 && fields[9] != "" {
			continue
		}

		concept := &Concept{
			ConceptID:       conceptID,
			ConceptName:     fields[1],
			DomainID:        fields[2],
			VocabularyID:    fields[3],
			ConceptClassID:  fields[4],
			StandardConcept: fields[5],
			ConceptCode:     fields[6],
		}

		key := conceptKey(concept.VocabularyID, concept.ConceptCode)
		vl.conceptIndex[key] = concept
		vl.conceptByID[conceptID] = concept
		count++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading supplementary vocab file: %w", err)
	}

	log.Printf("Loaded %d supplementary concepts from %s", count, filepath)
	return nil
}

// OIDToVocabularyID maps C-CDA code system OIDs to OMOP vocabulary IDs
// Also accepts direct vocabulary names (e.g., "CPT4", "SNOMED") and returns them as-is
func OIDToVocabularyID(oid string) string {
	switch oid {
	// Standard OIDs
	case "2.16.840.1.113883.6.96":
		return "SNOMED"
	case "2.16.840.1.113883.6.88":
		return "RxNorm"
	case "2.16.840.1.113883.6.1":
		return "LOINC"
	case "2.16.840.1.113883.6.90":
		return "ICD10CM"
	case "2.16.840.1.113883.6.103":
		return "ICD9CM"
	case "2.16.840.1.113883.6.12":
		return "CPT4"
	case "2.16.840.1.113883.6.14":
		return "HCPCS"
	case "2.16.840.1.113883.6.13": // CDT OID sometimes incorrectly used for HCPCS in C-CDA
		return "HCPCS"
	case "2.16.840.1.113883.12.292":
		return "CVX"
	case "2.16.840.1.113883.6.59": // Alternate CVX OID
		return "CVX"
	case "2.16.840.1.113883.6.69":
		return "NDC"
	case "2.16.840.1.113883.4.9":
		return "UNII" // FDA Unique Ingredient Identifier
	case "2.16.840.1.113883.3.26.1.5":
		return "NDFRT" // National Drug File Reference Terminology
	// Direct vocabulary names (some C-CDA files use these instead of OIDs)
	case "SNOMED", "SNOMED CT", "SNOMEDCT":
		return "SNOMED"
	case "RxNorm":
		return "RxNorm"
	case "LOINC":
		return "LOINC"
	case "ICD10CM", "ICD-10-CM", "ICD10":
		return "ICD10CM"
	case "ICD9CM", "ICD-9-CM", "ICD9":
		return "ICD9CM"
	case "CPT4", "CPT", "CPT-4":
		return "CPT4"
	case "HCPCS":
		return "HCPCS"
	case "CVX":
		return "CVX"
	case "NDC":
		return "NDC"
	case "UNII":
		return "UNII"
	case "NDFRT", "NDF-RT":
		return "NDFRT"
	default:
		return ""
	}
}
