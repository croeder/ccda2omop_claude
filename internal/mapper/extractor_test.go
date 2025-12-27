// Copyright 2025 Christophe Roeder. All rights reserved.

package mapper

import (
	"strings"
	"testing"
	"time"

	"github.com/antchfx/xmlquery"
)

// Helper to parse XML string into the root element node
func parseXML(t *testing.T, xml string) *xmlquery.Node {
	t.Helper()
	doc, err := xmlquery.Parse(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}
	// Return the first element child (root element), not the document node
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xmlquery.ElementNode {
			return c
		}
	}
	return doc
}

func TestNewExtractor(t *testing.T) {
	e := NewExtractor(false)
	if e == nil {
		t.Fatal("NewExtractor returned nil")
	}
	if e.verbose {
		t.Error("verbose should be false")
	}

	e2 := NewExtractor(true)
	if !e2.verbose {
		t.Error("verbose should be true")
	}
}

func TestParseHL7Time(t *testing.T) {
	e := NewExtractor(false)

	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{"full datetime", "20231201100000", time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)},
		{"datetime no seconds", "202312011000", time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)},
		{"datetime no minutes", "2023120110", time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)},
		{"date only", "20231201", time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)},
		{"year month", "202312", time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)},
		{"year only", "2023", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"with timezone Z", "20231201100000Z", time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)},
		{"with timezone offset", "20231201100000-0500", time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)},
		{"with positive offset", "20231201100000+0100", time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)},
		{"empty string", "", time.Time{}},
		{"invalid", "not-a-date", time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.ParseHL7Time(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("ParseHL7Time(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractString(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root>
		<code code="12345" displayName="Test Code"/>
		<value value="test value"/>
		<text>Inner text content</text>
		<nested><child attr="nested attr">nested text</child></nested>
	</root>`

	doc := parseXML(t, xml)

	tests := []struct {
		name     string
		xpath    string
		expected string
	}{
		{"attribute via /@", "code/@code", "12345"},
		{"displayName attribute", "code/@displayName", "Test Code"},
		{"value attribute", "value/@value", "test value"},
		{"element inner text", "text", "Inner text content"},
		{"nested attribute", "nested/child/@attr", "nested attr"},
		{"nested text", "nested/child", "nested text"},
		{"non-existent path", "nonexistent/@value", ""},
		{"non-existent element", "nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.extractString(doc, tt.xpath)
			if result != tt.expected {
				t.Errorf("extractString(%q) = %q, want %q", tt.xpath, result, tt.expected)
			}
		})
	}
}

func TestExtractFloat(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root>
		<value value="123.45"/>
		<quantity value="99.9" unit="mg"/>
		<text>42.5</text>
		<invalid value="not-a-number"/>
	</root>`

	doc := parseXML(t, xml)

	tests := []struct {
		name     string
		xpath    string
		expected *float64
	}{
		{"value attribute", "value/@value", floatPtr(123.45)},
		{"quantity value", "quantity/@value", floatPtr(99.9)},
		{"inner text", "text", floatPtr(42.5)},
		{"invalid number", "invalid/@value", nil},
		{"non-existent", "nonexistent/@value", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.extractFloat(doc, tt.xpath)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("extractFloat(%q) = %v, want nil", tt.xpath, *result)
				}
			} else {
				if result == nil {
					t.Errorf("extractFloat(%q) = nil, want %v", tt.xpath, *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("extractFloat(%q) = %v, want %v", tt.xpath, *result, *tt.expected)
				}
			}
		})
	}
}

func TestExtractInt(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root>
		<value value="42"/>
		<text>123</text>
		<invalid value="not-a-number"/>
		<float value="3.14"/>
	</root>`

	doc := parseXML(t, xml)

	tests := []struct {
		name     string
		xpath    string
		expected *int64
	}{
		{"value attribute", "value/@value", int64Ptr(42)},
		{"inner text", "text", int64Ptr(123)},
		{"invalid number", "invalid/@value", nil},
		{"float value", "float/@value", nil}, // ParseInt fails on floats
		{"non-existent", "nonexistent/@value", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.extractInt(doc, tt.xpath)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("extractInt(%q) = %v, want nil", tt.xpath, *result)
				}
			} else {
				if result == nil {
					t.Errorf("extractInt(%q) = nil, want %v", tt.xpath, *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("extractInt(%q) = %v, want %v", tt.xpath, *result, *tt.expected)
				}
			}
		})
	}
}

func TestExtractTime(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root>
		<effectiveTime value="20231201"/>
		<nested><time value="20230615100000"/></nested>
		<invalid value="not-a-date"/>
	</root>`

	doc := parseXML(t, xml)

	tests := []struct {
		name     string
		xpath    string
		expected *time.Time
	}{
		{"date value", "effectiveTime/@value", timePtr(time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC))},
		{"nested datetime", "nested/time/@value", timePtr(time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC))},
		{"invalid date", "invalid/@value", nil},
		{"non-existent", "nonexistent/@value", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.extractTime(doc, tt.xpath)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("extractTime(%q) = %v, want nil", tt.xpath, *result)
				}
			} else {
				if result == nil {
					t.Errorf("extractTime(%q) = nil, want %v", tt.xpath, *tt.expected)
				} else if !result.Equal(*tt.expected) {
					t.Errorf("extractTime(%q) = %v, want %v", tt.xpath, *result, *tt.expected)
				}
			}
		})
	}
}

func TestExtractCode(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root>
		<code code="44054006" codeSystem="2.16.840.1.113883.6.96"
			  codeSystemName="SNOMED CT" displayName="Type 2 Diabetes">
			<originalText>Diabetes mellitus type 2</originalText>
		</code>
		<emptyCode/>
		<partialCode code="12345"/>
	</root>`

	doc := parseXML(t, xml)

	t.Run("full code", func(t *testing.T) {
		result := e.extractCode(doc, "code")
		if result == nil {
			t.Fatal("extractCode returned nil")
		}
		if result["Code"] != "44054006" {
			t.Errorf("Code = %v, want 44054006", result["Code"])
		}
		if result["CodeSystem"] != "2.16.840.1.113883.6.96" {
			t.Errorf("CodeSystem = %v, want 2.16.840.1.113883.6.96", result["CodeSystem"])
		}
		if result["CodeSystemName"] != "SNOMED CT" {
			t.Errorf("CodeSystemName = %v, want SNOMED CT", result["CodeSystemName"])
		}
		if result["DisplayName"] != "Type 2 Diabetes" {
			t.Errorf("DisplayName = %v, want Type 2 Diabetes", result["DisplayName"])
		}
		if result["OriginalText"] != "Diabetes mellitus type 2" {
			t.Errorf("OriginalText = %v, want Diabetes mellitus type 2", result["OriginalText"])
		}
	})

	t.Run("empty code", func(t *testing.T) {
		result := e.extractCode(doc, "emptyCode")
		if result != nil {
			t.Errorf("extractCode(emptyCode) = %v, want nil", result)
		}
	})

	t.Run("partial code", func(t *testing.T) {
		result := e.extractCode(doc, "partialCode")
		if result == nil {
			t.Fatal("extractCode(partialCode) returned nil")
		}
		if result["Code"] != "12345" {
			t.Errorf("Code = %v, want 12345", result["Code"])
		}
		if _, ok := result["CodeSystem"]; ok {
			t.Error("CodeSystem should not be present")
		}
	})

	t.Run("non-existent", func(t *testing.T) {
		result := e.extractCode(doc, "nonexistent")
		if result != nil {
			t.Errorf("extractCode(nonexistent) = %v, want nil", result)
		}
	})
}

func TestExtractEffectiveTime(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root>
		<effectiveTime value="20231201"/>
		<range>
			<low value="20230101"/>
			<high value="20231231"/>
		</range>
		<combined value="20230615">
			<low value="20230601"/>
			<high value="20230630"/>
		</combined>
		<empty/>
	</root>`

	doc := parseXML(t, xml)

	t.Run("single value", func(t *testing.T) {
		result := e.extractEffectiveTime(doc, "effectiveTime")
		if result == nil {
			t.Fatal("extractEffectiveTime returned nil")
		}
		v, ok := result["Value"].(time.Time)
		if !ok {
			t.Fatal("Value not a time.Time")
		}
		expected := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
		if !v.Equal(expected) {
			t.Errorf("Value = %v, want %v", v, expected)
		}
	})

	t.Run("range with low/high", func(t *testing.T) {
		result := e.extractEffectiveTime(doc, "range")
		if result == nil {
			t.Fatal("extractEffectiveTime returned nil")
		}
		low, ok := result["Low"].(time.Time)
		if !ok {
			t.Fatal("Low not a time.Time")
		}
		high, ok := result["High"].(time.Time)
		if !ok {
			t.Fatal("High not a time.Time")
		}
		if !low.Equal(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("Low = %v, want 2023-01-01", low)
		}
		if !high.Equal(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("High = %v, want 2023-12-31", high)
		}
	})

	t.Run("combined value and range", func(t *testing.T) {
		result := e.extractEffectiveTime(doc, "combined")
		if result == nil {
			t.Fatal("extractEffectiveTime returned nil")
		}
		if _, ok := result["Value"]; !ok {
			t.Error("Value should be present")
		}
		if _, ok := result["Low"]; !ok {
			t.Error("Low should be present")
		}
		if _, ok := result["High"]; !ok {
			t.Error("High should be present")
		}
	})

	t.Run("empty element", func(t *testing.T) {
		result := e.extractEffectiveTime(doc, "empty")
		if result != nil {
			t.Errorf("extractEffectiveTime(empty) = %v, want nil", result)
		}
	})
}

func TestExtractQuantity(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root>
		<value value="120" unit="mm[Hg]"/>
		<valueOnly value="42"/>
		<unitOnly unit="mg"/>
		<empty/>
	</root>`

	doc := parseXML(t, xml)

	t.Run("full quantity", func(t *testing.T) {
		result := e.extractQuantity(doc, "value")
		if result == nil {
			t.Fatal("extractQuantity returned nil")
		}
		if result["Value"] != float64(120) {
			t.Errorf("Value = %v, want 120", result["Value"])
		}
		if result["Unit"] != "mm[Hg]" {
			t.Errorf("Unit = %v, want mm[Hg]", result["Unit"])
		}
	})

	t.Run("value only", func(t *testing.T) {
		result := e.extractQuantity(doc, "valueOnly")
		if result == nil {
			t.Fatal("extractQuantity returned nil")
		}
		if result["Value"] != float64(42) {
			t.Errorf("Value = %v, want 42", result["Value"])
		}
	})

	t.Run("unit only", func(t *testing.T) {
		result := e.extractQuantity(doc, "unitOnly")
		if result == nil {
			t.Fatal("extractQuantity returned nil")
		}
		if result["Unit"] != "mg" {
			t.Errorf("Unit = %v, want mg", result["Unit"])
		}
	})

	t.Run("empty", func(t *testing.T) {
		result := e.extractQuantity(doc, "empty")
		if result != nil {
			t.Errorf("extractQuantity(empty) = %v, want nil", result)
		}
	})
}

func TestExtractPatient(t *testing.T) {
	e := NewExtractor(false)

	xml := `<ClinicalDocument>
		<recordTarget>
			<patientRole>
				<id extension="123-45-6789" root="2.16.840.1.113883.4.1"/>
				<patient>
					<name>
						<given>John</given>
						<given>Q</given>
						<family>Doe</family>
						<suffix>Jr</suffix>
					</name>
					<birthTime value="19800515"/>
					<administrativeGenderCode code="M" codeSystem="2.16.840.1.113883.5.1" displayName="Male"/>
					<raceCode code="2106-3" codeSystem="2.16.840.1.113883.6.238" displayName="White"/>
					<ethnicGroupCode code="2186-5" codeSystem="2.16.840.1.113883.6.238" displayName="Not Hispanic or Latino"/>
				</patient>
			</patientRole>
		</recordTarget>
	</ClinicalDocument>`

	// extractPatient uses absolute XPaths, so we need the document node
	doc, _ := xmlquery.Parse(strings.NewReader(xml))
	result := e.extractPatient(doc)

	if result["ID"] != "123-45-6789" {
		t.Errorf("ID = %v, want 123-45-6789", result["ID"])
	}

	name, ok := result["Name"].(map[string]interface{})
	if !ok {
		t.Fatal("Name not a map")
	}
	if name["Given"] != "John Q" {
		t.Errorf("Given = %v, want John Q", name["Given"])
	}
	if name["Family"] != "Doe" {
		t.Errorf("Family = %v, want Doe", name["Family"])
	}
	if name["Suffix"] != "Jr" {
		t.Errorf("Suffix = %v, want Jr", name["Suffix"])
	}

	bt, ok := result["BirthTime"].(time.Time)
	if !ok {
		t.Fatal("BirthTime not a time.Time")
	}
	expected := time.Date(1980, 5, 15, 0, 0, 0, 0, time.UTC)
	if !bt.Equal(expected) {
		t.Errorf("BirthTime = %v, want %v", bt, expected)
	}

	gender, ok := result["Gender"].(map[string]interface{})
	if !ok {
		t.Fatal("Gender not a map")
	}
	if gender["Code"] != "M" {
		t.Errorf("Gender Code = %v, want M", gender["Code"])
	}
}

func TestShouldIncludeEntry(t *testing.T) {
	e := NewExtractor(false)

	tests := []struct {
		name     string
		xml      string
		expected bool
	}{
		{
			"EVN mood completed status",
			`<entry moodCode="EVN"><statusCode code="completed"/></entry>`,
			true,
		},
		{
			"EVN mood active status",
			`<entry moodCode="EVN"><statusCode code="active"/></entry>`,
			true,
		},
		{
			"no moodCode no statusCode",
			`<entry></entry>`,
			true,
		},
		{
			"INT mood (intent)",
			`<entry moodCode="INT"><statusCode code="completed"/></entry>`,
			false,
		},
		{
			"RQO mood (request)",
			`<entry moodCode="RQO"><statusCode code="completed"/></entry>`,
			false,
		},
		{
			"cancelled status",
			`<entry moodCode="EVN"><statusCode code="cancelled"/></entry>`,
			false,
		},
		{
			"aborted status",
			`<entry moodCode="EVN"><statusCode code="aborted"/></entry>`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// parseXML returns the root element directly
			entry := parseXML(t, tt.xml)
			result := e.ShouldIncludeEntry(entry)
			if result != tt.expected {
				t.Errorf("ShouldIncludeEntry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestXPathExtractStringWithFallback(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root>
		<primary value="primary value"/>
		<fallback value="fallback value"/>
	</root>`

	doc := parseXML(t, xml)

	t.Run("primary exists", func(t *testing.T) {
		result := e.XPathExtractString(doc, "primary/@value", "fallback/@value")
		if result != "primary value" {
			t.Errorf("XPathExtractString = %q, want %q", result, "primary value")
		}
	})

	t.Run("primary missing uses fallback", func(t *testing.T) {
		result := e.XPathExtractString(doc, "nonexistent/@value", "fallback/@value")
		if result != "fallback value" {
			t.Errorf("XPathExtractString = %q, want %q", result, "fallback value")
		}
	})

	t.Run("both missing", func(t *testing.T) {
		result := e.XPathExtractString(doc, "nonexistent/@value", "alsoNonexistent/@value")
		if result != "" {
			t.Errorf("XPathExtractString = %q, want empty", result)
		}
	})
}

func TestXPathExtractWithPredicates(t *testing.T) {
	e := NewExtractor(false)

	// Test the fix for XPath predicates containing /@
	xml := `<root>
		<observation>
			<code code="ASSERTION"/>
			<value code="266919005" displayName="Never smoker"/>
		</observation>
		<observation>
			<code code="OTHER"/>
			<value code="12345" displayName="Other value"/>
		</observation>
	</root>`

	doc := parseXML(t, xml)

	t.Run("predicate with attribute test", func(t *testing.T) {
		// This XPath has /@ inside the predicate - tests our LastIndex fix
		result := e.extractString(doc, "observation[code/@code='ASSERTION']/value/@displayName")
		if result != "Never smoker" {
			t.Errorf("extractString with predicate = %q, want %q", result, "Never smoker")
		}
	})

	t.Run("predicate with different value", func(t *testing.T) {
		result := e.extractString(doc, "observation[code/@code='OTHER']/value/@displayName")
		if result != "Other value" {
			t.Errorf("extractString with predicate = %q, want %q", result, "Other value")
		}
	})

	t.Run("predicate no match", func(t *testing.T) {
		result := e.extractString(doc, "observation[code/@code='NONEXISTENT']/value/@displayName")
		if result != "" {
			t.Errorf("extractString with non-matching predicate = %q, want empty", result)
		}
	})
}

func TestGetAttr(t *testing.T) {
	e := NewExtractor(false)

	xml := `<element attr1="value1" attr2="value2"/>`
	// parseXML returns the root element directly
	elem := parseXML(t, xml)

	if e.GetAttr(elem, "attr1") != "value1" {
		t.Errorf("GetAttr(attr1) = %q, want value1", e.GetAttr(elem, "attr1"))
	}
	if e.GetAttr(elem, "attr2") != "value2" {
		t.Errorf("GetAttr(attr2) = %q, want value2", e.GetAttr(elem, "attr2"))
	}
	if e.GetAttr(elem, "nonexistent") != "" {
		t.Errorf("GetAttr(nonexistent) = %q, want empty", e.GetAttr(elem, "nonexistent"))
	}
	if e.GetAttr(nil, "attr1") != "" {
		t.Error("GetAttr(nil, attr1) should return empty")
	}
}

func TestXPathNode(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root><child><grandchild/></child></root>`
	doc := parseXML(t, xml)

	t.Run("existing node", func(t *testing.T) {
		node := e.XPathNode(doc, "//child")
		if node == nil {
			t.Error("XPathNode returned nil for existing path")
		}
	})

	t.Run("non-existent node", func(t *testing.T) {
		node := e.XPathNode(doc, "//nonexistent")
		if node != nil {
			t.Error("XPathNode should return nil for non-existent path")
		}
	})
}

func TestXPathNodes(t *testing.T) {
	e := NewExtractor(false)

	xml := `<root><item>1</item><item>2</item><item>3</item></root>`
	doc := parseXML(t, xml)

	t.Run("multiple nodes", func(t *testing.T) {
		nodes := e.XPathNodes(doc, "//item")
		if len(nodes) != 3 {
			t.Errorf("XPathNodes returned %d nodes, want 3", len(nodes))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		nodes := e.XPathNodes(doc, "//nonexistent")
		if len(nodes) != 0 {
			t.Errorf("XPathNodes returned %d nodes, want 0", len(nodes))
		}
	})
}

// Helper functions for creating pointers
func floatPtr(f float64) *float64 {
	return &f
}

func int64Ptr(i int64) *int64 {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}
