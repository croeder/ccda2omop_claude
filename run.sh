   ./ccda2omop -input /Users/croeder/claude_play/CCDA-data/xml_load_test
     -analyze \
     -concept /Users/croeder/claude_play/CCDA_OMOP_Private/CONCEPT.csv \
     -relationship /Users/croeder/claude_play/CCDA_OMOP_Private/CONCEPT_RELATIONSHIP.csv \
     -vocab-dir /Users/croeder/claude_play/ccda2omop/vocab \
     -analyze-output /tmp/code_analysis.csv 2>&1 | tail -5
