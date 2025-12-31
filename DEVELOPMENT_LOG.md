# CCDA2OMOP Development Log
Copyright 2025 Christophe Roeder. All rights reserved.

This log documents the prompts and changes made during development.

## Session: December 24, 2025

### 1. Directory Input Support
**Prompt**: Request to add batch processing capability for directories of XML files.

**Changes Made**:
- Added `RunBatch()` function in `internal/converter/converter.go` for aggregated output
- Updated `cmd/ccda2omop/main.go` with directory detection and `findXMLFiles()` function
- Commit: 3f61ba1

---

### 2. Analysis Tool Runs
**Prompt**: "could you run the analysis on all the files in the CCDA-data folder and use the CONCEPT.csv file"

**Result**: Ran analysis on 747 XML files in CCDA-data/xml_load_test directory.

---

### 3. CVX Vocabulary Support
**Prompt**: "add CVX vocabulary for immunizations mapping"

**Changes Made**:
- Created `vocab/CVX.csv` with 20 vaccine codes (CVX codes from CDC, synthetic concept_ids)
- Added `-cvx` flag to CLI (later replaced by `-vocab-dir`)
- Added `LoadSupplementaryVocab()` method to `vocab_loader.go` with comment line support
- Immunization mapping improved from 0% to 67%+ depending on data
- Commit: 9056b0f

---

### 4. CPT4 and HCPCS Vocabulary Support
**Prompt**: "add CPT4 vocabulary for procedures mapping"

**Changes Made**:
- Created `vocab/CPT4.csv` with 31 common procedure codes (real CPT codes, synthetic concept_ids)
- Created `vocab/HCPCS.csv` with 2 procedure codes (real HCPCS codes, synthetic concept_ids)
- Changed `-cvx` flag to `-vocab-dir` to load all CSV files from a directory
- Procedure mapping improved from 38.2% to 90.1%
- Overall mapping improved from 79.2% to 85.5%
- Commit: 2857da2

---

### 5. XPath Cleanup
**Prompt**: "could you modify the paths in the analysis so they do not include the instance counts? I mean, the integers in square brackets in the paths should be removed."

**Changes Made**:
- Modified `internal/analyzer/analyzer.go` to remove instance indices like `[1]`, `[2]` from XPaths
- Produces cleaner, more generic paths for analysis
- Commit: 9d31bd9

---

### 6. Vocabulary File Comments
**Prompt**: "keep CPT4.csv, but add comments to the code that make it clear they are synthetic"

**Changes Made**:
- Added header comments to all vocab files (CVX.csv, CPT4.csv, HCPCS.csv)
- Updated `LoadSupplementaryVocab()` to skip lines starting with `#`
- Commit: 7dd85c3

---

### 7. Vocabulary Comment Correction
**Prompt**: "Is it true that the CVX and HCPCS codes here are synthetic?"

**Clarification**: The CVX, CPT4, and HCPCS codes in the vocabulary files are REAL codes from their respective authorities (CDC, AMA, CMS). Only the OMOP concept_ids are synthetic.

**Changes Made**:
- Updated comments in all vocab files to clarify:
  - CVX codes are real (from CDC), concept_ids are synthetic
  - CPT4 codes are real (from AMA), concept_ids are synthetic
  - HCPCS codes are real (from CMS), concept_ids are synthetic

---

## Summary of Supplementary Vocabulary Files

| File | Codes | Source | Concept ID Range |
|------|-------|--------|------------------|
| vocab/CVX.csv | 20 vaccine codes | CDC | 2000000001+ |
| vocab/CPT4.csv | 31 procedure codes | AMA (CPT) | 2100000001+ |
| vocab/HCPCS.csv | 2 procedure codes | CMS | 2200000001+ |

**Note**: These are supplementary vocabularies for demonstration purposes. For production use, obtain official vocabularies from OMOP Athena (https://athena.ohdsi.org/).

---

## Analysis Results (Latest Run)

```
Total C-CDA codes analyzed: 20414
Successfully mapped: 17459 (85.5%)
Unmapped: 2955 (14.5%)

OMOP CDM Tables populated:
  condition_occurrence           2183 records
  device_exposure                3 records
  drug_exposure                  3004 records
  measurement                    10102 records
  observation                    482 records
  procedure_occurrence           1685 records
```

### Section-by-Section Mapping Rates:
- Allergies: 47.0%
- Immunizations: 67.1%
- LabResults: 98.8%
- MedicalEquipment: 100.0%
- Medications: 84.9%
- Problems: 83.8%
- Procedures: 90.1%
- SocialHistory: 4.0%
- VitalSigns: 99.7%

---

## Session: December 30, 2025

### 8. Python Translation Project
**Prompt**: Translate ccda2omop from Go to Python.

**Changes Made**:
- Created sibling project `ccda2omop-py/` with full Python implementation
- Structure: `src/ccda2omop/` with submodules for ccda, omop, mapper, converter, analyzer, report
- Dependencies: lxml for XML/XPath, PyYAML, click for CLI
- Feature parity with Go version including YAML rule support
- Repository: Separate git repository at `../ccda2omop-py/`

---

### 9. Domain-Based Routing Fix (Python)
**Prompt**: Fix Python rule engine to match Go YAML rules behavior for domain-based routing.

**Changes Made**:
- Fixed `rule_engine.py` to strictly enforce domain conditions
- Entries now routed to correct OMOP tables based on concept domain:
  - Procedures with domain "Observation" → observation table
  - Procedures with domain "Measurement" → measurement table
  - Procedures with domain "Procedure" → procedure_occurrence table
- Commit: 006a698 (in ccda2omop-py repo)

---

### 10. Domain Conditions for Go Hardcoded Rules
**Prompt**: Fix Go hardcoded rules to add domain conditions matching YAML rules behavior.

**Changes Made**:
- Added `Conditions` to `SourceSpec` for all existing rules in `rule_definitions.go`:
  - ProblemRule: `domain_equals: Condition`
  - ProcedureRule: `domain_equals: Procedure`
  - VitalSignRule: `domain_equals: Measurement`
  - LabResultRule: `domain_equals: Measurement`
  - AllergyRule: `domain_equals: Observation`
  - SocialObservationRule: `domain_equals: Observation`
- Added 9 new rules for domain-based routing:
  - ProblemToObservationRule (domain: Observation)
  - ProcedureToObservationRule (domain: Observation)
  - ProcedureToMeasurementRule (domain: Measurement)
  - VitalSignToObservationRule (domain: Observation)
  - LabResultToObservationRule (domain: Observation)
  - AllergyToConditionRule (domain: Condition)
  - SocialToConditionRule (domain: Condition)
  - SocialToMeasurementRule (domain: Measurement)
- Updated `AllRules` slice to include all 17 rules
- Commit: abfe15e

**Verification**:
Both Go (hardcoded rules) and Python produce identical output for Patient-620.xml:
- 1 person, 4 visit, 5 condition, 6 drug, 4 procedure, 18 measurement, 1 observation, 0 device
