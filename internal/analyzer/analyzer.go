// Copyright 2025 Christophe Roeder. All rights reserved.

package analyzer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/ccda2omop/internal/mapper"
)

// CodeMapping represents a single code found in the C-CDA and its OMOP mapping
type CodeMapping struct {
	Section           string
	XPath             string
	SourceCode        string
	SourceCodeSystem  string
	SourceVocabulary  string
	SourceDisplayName string
	OMOPConceptID     int64
	OMOPConceptName   string
	OMOPDomainID      string
	OMOPVocabularyID  string
	IsStandard        bool
	MappingStatus     string // "mapped", "unmapped", "no_vocab"
}

// Analyzer extracts codes from C-CDA files and maps them to OMOP concepts
type Analyzer struct {
	vocabLoader *mapper.VocabLoader
	verbose     bool
}

// New creates a new Analyzer
func New(vocabLoader *mapper.VocabLoader, verbose bool) *Analyzer {
	return &Analyzer{
		vocabLoader: vocabLoader,
		verbose:     verbose,
	}
}

// SectionXPath defines the XPath and code extraction for a C-CDA section
type SectionXPath struct {
	Name      string
	RootXPath string
	CodePaths []CodePath
}

// CodePath defines how to extract a code from within an entry
type CodePath struct {
	Name           string // Description of this code location
	CodeXPath      string // XPath to the code element relative to entry
	CodeAttr       string // Attribute containing the code (usually "code")
	CodeSystemAttr string // Attribute containing the code system OID
	DisplayAttr    string // Attribute containing display name
}

// Standard C-CDA section definitions
var sectionDefinitions = []SectionXPath{
	{
		Name:      "Problems",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.5.1']/entry/act/entryRelationship/observation",
		CodePaths: []CodePath{
			{Name: "Problem Code", CodeXPath: "value", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
	{
		Name:      "Medications",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.1.1']/entry/substanceAdministration",
		CodePaths: []CodePath{
			{Name: "Medication Code", CodeXPath: "consumable/manufacturedProduct/manufacturedMaterial/code", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
			{Name: "Route Code", CodeXPath: "routeCode", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
	{
		Name:      "Immunizations",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.2.1']/entry/substanceAdministration",
		CodePaths: []CodePath{
			{Name: "Vaccine Code", CodeXPath: "consumable/manufacturedProduct/manufacturedMaterial/code", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
	{
		Name:      "Procedures",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.7.1']/entry/procedure",
		CodePaths: []CodePath{
			{Name: "Procedure Code", CodeXPath: "code", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
	{
		Name:      "VitalSigns",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.4.1']/entry/organizer/component/observation",
		CodePaths: []CodePath{
			{Name: "Vital Sign Code", CodeXPath: "code", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
	{
		Name:      "LabResults",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.3.1']/entry/organizer/component/observation",
		CodePaths: []CodePath{
			{Name: "Lab Code", CodeXPath: "code", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
	{
		Name:      "Allergies",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.6.1']/entry/act/entryRelationship/observation",
		CodePaths: []CodePath{
			{Name: "Allergy Code", CodeXPath: "participant/participantRole/playingEntity/code", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
			{Name: "Reaction Code", CodeXPath: "entryRelationship/observation[templateId/@root='2.16.840.1.113883.10.20.22.4.9']/value", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
	{
		Name:      "SocialHistory",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.17']/entry/observation",
		CodePaths: []CodePath{
			{Name: "Social Observation Code", CodeXPath: "code", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
	{
		Name:      "MedicalEquipment",
		RootXPath: "//component/section[templateId/@root='2.16.840.1.113883.10.20.22.2.23']/entry",
		CodePaths: []CodePath{
			{Name: "Device Code", CodeXPath: ".//participant/participantRole/playingDevice/code", CodeAttr: "code", CodeSystemAttr: "codeSystem", DisplayAttr: "displayName"},
		},
	},
}

// AnalyzeFile analyzes a C-CDA file and returns all code mappings
func (a *Analyzer) AnalyzeFile(filepath string) ([]CodeMapping, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	doc, err := xmlquery.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var mappings []CodeMapping

	for _, section := range sectionDefinitions {
		entries := xmlquery.Find(doc, section.RootXPath)
		for _, entry := range entries {
			for _, codePath := range section.CodePaths {
				// Find code elements within this entry
				var codeNodes []*xmlquery.Node
				if codePath.CodeXPath == "" || codePath.CodeXPath == "." {
					codeNodes = []*xmlquery.Node{entry}
				} else {
					codeNodes = xmlquery.Find(entry, codePath.CodeXPath)
				}

				for _, codeNode := range codeNodes {
					code := codeNode.SelectAttr(codePath.CodeAttr)
					if code == "" {
						continue
					}

					codeSystem := codeNode.SelectAttr(codePath.CodeSystemAttr)
					displayName := codeNode.SelectAttr(codePath.DisplayAttr)

					// Build XPath for this code (without instance counts for cleaner output)
					xpath := section.RootXPath
					if codePath.CodeXPath != "" && codePath.CodeXPath != "." {
						xpath = xpath + "/" + codePath.CodeXPath
					}

					mapping := a.mapCode(section.Name, xpath, code, codeSystem, displayName, codePath.Name)
					mappings = append(mappings, mapping)
				}
			}
		}
	}

	return mappings, nil
}

// mapCode maps a single code to OMOP concepts
func (a *Analyzer) mapCode(section, xpath, code, codeSystem, displayName, codeLocation string) CodeMapping {
	mapping := CodeMapping{
		Section:           section,
		XPath:             xpath,
		SourceCode:        code,
		SourceCodeSystem:  codeSystem,
		SourceDisplayName: displayName,
	}

	// Convert OID to vocabulary ID
	vocabID := mapper.OIDToVocabularyID(codeSystem)
	mapping.SourceVocabulary = vocabID

	if vocabID == "" {
		mapping.MappingStatus = "no_vocab"
		return mapping
	}

	if a.vocabLoader == nil {
		mapping.MappingStatus = "no_vocab_loader"
		return mapping
	}

	// Look up the source concept
	sourceConcept := a.vocabLoader.LookupConcept(vocabID, code)
	if sourceConcept == nil {
		mapping.MappingStatus = "unmapped"
		return mapping
	}

	// Get standard concept mappings
	standardIDs := a.vocabLoader.GetStandardConceptIDs(vocabID, code)
	if len(standardIDs) == 0 {
		mapping.MappingStatus = "unmapped"
		mapping.OMOPConceptID = sourceConcept.ConceptID
		mapping.OMOPConceptName = sourceConcept.ConceptName
		mapping.OMOPDomainID = sourceConcept.DomainID
		mapping.OMOPVocabularyID = sourceConcept.VocabularyID
		return mapping
	}

	// Use the first standard concept (could return multiple for multi-mapping)
	standardConcept := a.vocabLoader.LookupConceptByID(standardIDs[0])
	if standardConcept != nil {
		mapping.OMOPConceptID = standardConcept.ConceptID
		mapping.OMOPConceptName = standardConcept.ConceptName
		mapping.OMOPDomainID = standardConcept.DomainID
		mapping.OMOPVocabularyID = standardConcept.VocabularyID
		mapping.IsStandard = standardConcept.StandardConcept == "S"
		mapping.MappingStatus = "mapped"

		// Note if there are multiple mappings
		if len(standardIDs) > 1 {
			mapping.MappingStatus = fmt.Sprintf("mapped (%d targets)", len(standardIDs))
		}
	} else {
		mapping.MappingStatus = "unmapped"
	}

	return mapping
}

// WriteCSV writes the mappings to a CSV file
func (a *Analyzer) WriteCSV(mappings []CodeMapping, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{
		"Section",
		"XPath",
		"Source_Code",
		"Source_CodeSystem_OID",
		"Source_Vocabulary",
		"Source_DisplayName",
		"OMOP_Concept_ID",
		"OMOP_Concept_Name",
		"OMOP_Domain_ID",
		"OMOP_Vocabulary_ID",
		"Is_Standard",
		"Mapping_Status",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, m := range mappings {
		row := []string{
			m.Section,
			m.XPath,
			m.SourceCode,
			m.SourceCodeSystem,
			m.SourceVocabulary,
			m.SourceDisplayName,
			fmt.Sprintf("%d", m.OMOPConceptID),
			m.OMOPConceptName,
			m.OMOPDomainID,
			m.OMOPVocabularyID,
			fmt.Sprintf("%t", m.IsStandard),
			m.MappingStatus,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// PrintSummary prints a summary of the analysis to the writer
func (a *Analyzer) PrintSummary(mappings []CodeMapping, w io.Writer) {
	// Count by section and mapping status
	sectionCounts := make(map[string]int)
	statusCounts := make(map[string]int)
	domainCounts := make(map[string]int)

	for _, m := range mappings {
		sectionCounts[m.Section]++
		statusCounts[m.MappingStatus]++
		if m.OMOPDomainID != "" {
			domainCounts[m.OMOPDomainID]++
		}
	}

	fmt.Fprintf(w, "\n=== Analysis Summary ===\n\n")
	fmt.Fprintf(w, "Total codes found: %d\n\n", len(mappings))

	fmt.Fprintf(w, "By Section:\n")
	for section, count := range sectionCounts {
		fmt.Fprintf(w, "  %-20s %d\n", section, count)
	}

	fmt.Fprintf(w, "\nBy Mapping Status:\n")
	for status, count := range statusCounts {
		fmt.Fprintf(w, "  %-20s %d\n", status, count)
	}

	fmt.Fprintf(w, "\nBy OMOP Domain:\n")
	for domain, count := range domainCounts {
		fmt.Fprintf(w, "  %-20s %d\n", domain, count)
	}

	// Show unmapped codes
	fmt.Fprintf(w, "\n=== Unmapped Codes ===\n")
	for _, m := range mappings {
		if strings.HasPrefix(m.MappingStatus, "unmapped") || m.MappingStatus == "no_vocab" {
			fmt.Fprintf(w, "  [%s] %s (%s) - %s\n", m.Section, m.SourceCode, m.SourceVocabulary, m.SourceDisplayName)
		}
	}
}

// domainToTable maps OMOP domain_id to the primary CDM table
func domainToTable(domainID string) string {
	switch domainID {
	case "Condition":
		return "condition_occurrence"
	case "Drug":
		return "drug_exposure"
	case "Procedure":
		return "procedure_occurrence"
	case "Measurement":
		return "measurement"
	case "Observation":
		return "observation"
	case "Device":
		return "device_exposure"
	case "Visit":
		return "visit_occurrence"
	case "Specimen":
		return "specimen"
	case "Note":
		return "note"
	default:
		if domainID == "" {
			return "(unmapped)"
		}
		return domainID
	}
}

// SectionMapping tracks the mapping from a C-CDA section/path to OMOP tables
type SectionMapping struct {
	Section     string
	CodePath    string
	TotalCodes  int
	MappedCodes int
	OMOPTables  map[string]int // table name -> count
}

// WriteMappingSummary writes a summary showing C-CDA sections/paths and their OMOP table mappings
func (a *Analyzer) WriteMappingSummary(mappings []CodeMapping, w io.Writer) {
	// Group by section and extract code path from XPath
	sectionPaths := make(map[string]*SectionMapping)

	for _, m := range mappings {
		// Extract a simplified path identifier from the XPath
		pathKey := extractCodePath(m.XPath)
		key := m.Section + "|" + pathKey

		sm, exists := sectionPaths[key]
		if !exists {
			sm = &SectionMapping{
				Section:    m.Section,
				CodePath:   pathKey,
				OMOPTables: make(map[string]int),
			}
			sectionPaths[key] = sm
		}

		sm.TotalCodes++
		if m.OMOPDomainID != "" {
			sm.MappedCodes++
			table := domainToTable(m.OMOPDomainID)
			sm.OMOPTables[table]++
		}
	}

	// Print header
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "================================================================================\n")
	fmt.Fprintf(w, "C-CDA to OMOP Mapping Summary\n")
	fmt.Fprintf(w, "================================================================================\n\n")

	// Collect and sort sections
	var sections []string
	sectionMap := make(map[string][]*SectionMapping)
	for _, sm := range sectionPaths {
		if _, exists := sectionMap[sm.Section]; !exists {
			sections = append(sections, sm.Section)
		}
		sectionMap[sm.Section] = append(sectionMap[sm.Section], sm)
	}
	sort.Strings(sections)

	// Print by section
	for _, section := range sections {
		paths := sectionMap[section]
		sort.Slice(paths, func(i, j int) bool {
			return paths[i].CodePath < paths[j].CodePath
		})

		// Calculate section totals
		sectionTotal := 0
		sectionMapped := 0
		sectionTables := make(map[string]int)
		for _, p := range paths {
			sectionTotal += p.TotalCodes
			sectionMapped += p.MappedCodes
			for table, count := range p.OMOPTables {
				sectionTables[table] += count
			}
		}

		fmt.Fprintf(w, "C-CDA Section: %s\n", section)
		fmt.Fprintf(w, "  Total codes: %d, Mapped: %d (%.1f%%)\n",
			sectionTotal, sectionMapped, percentage(sectionMapped, sectionTotal))

		// Show OMOP tables for this section
		fmt.Fprintf(w, "  OMOP Tables:\n")
		tables := sortedKeys(sectionTables)
		for _, table := range tables {
			count := sectionTables[table]
			fmt.Fprintf(w, "    → %-25s %d codes\n", table, count)
		}

		// Show code paths within section
		fmt.Fprintf(w, "  Code Paths:\n")
		for _, p := range paths {
			tables := sortedKeys(p.OMOPTables)
			tableStr := strings.Join(tables, ", ")
			if tableStr == "" {
				tableStr = "(no mapping)"
			}
			fmt.Fprintf(w, "    %-40s → %s (%d codes)\n", p.CodePath, tableStr, p.TotalCodes)
		}
		fmt.Fprintf(w, "\n")
	}

	// Overall summary
	fmt.Fprintf(w, "================================================================================\n")
	fmt.Fprintf(w, "Overall Summary\n")
	fmt.Fprintf(w, "================================================================================\n\n")

	totalCodes := 0
	mappedCodes := 0
	allTables := make(map[string]int)
	for _, sm := range sectionPaths {
		totalCodes += sm.TotalCodes
		mappedCodes += sm.MappedCodes
		for table, count := range sm.OMOPTables {
			allTables[table] += count
		}
	}

	fmt.Fprintf(w, "Total C-CDA codes analyzed: %d\n", totalCodes)
	fmt.Fprintf(w, "Successfully mapped: %d (%.1f%%)\n", mappedCodes, percentage(mappedCodes, totalCodes))
	fmt.Fprintf(w, "Unmapped: %d (%.1f%%)\n\n", totalCodes-mappedCodes, percentage(totalCodes-mappedCodes, totalCodes))

	fmt.Fprintf(w, "OMOP CDM Tables populated:\n")
	tables := sortedKeys(allTables)
	for _, table := range tables {
		count := allTables[table]
		fmt.Fprintf(w, "  %-30s %d records\n", table, count)
	}
}

// extractCodePath extracts a simplified code path from a full XPath
func extractCodePath(xpath string) string {
	// Extract the code-specific part after the entry index
	// e.g., ".../observation[1]/value[1]" -> "value"
	// e.g., ".../substanceAdministration[1]/consumable/.../code[1]" -> "consumable/.../code"

	parts := strings.Split(xpath, "/")
	var codeParts []string
	foundEntry := false

	for _, part := range parts {
		// Skip empty parts
		if part == "" {
			continue
		}
		// Look for entry-level elements
		if strings.HasPrefix(part, "observation[") ||
			strings.HasPrefix(part, "substanceAdministration[") ||
			strings.HasPrefix(part, "procedure[") ||
			strings.HasPrefix(part, "act[") ||
			strings.HasPrefix(part, "organizer[") {
			foundEntry = true
			continue
		}
		if foundEntry {
			// Remove index suffixes for cleaner display
			cleanPart := part
			if idx := strings.Index(part, "["); idx > 0 {
				cleanPart = part[:idx]
			}
			codeParts = append(codeParts, cleanPart)
		}
	}

	if len(codeParts) == 0 {
		return "(root)"
	}
	return strings.Join(codeParts, "/")
}

func percentage(part, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(part) / float64(total) * 100.0
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
