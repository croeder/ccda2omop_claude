// Copyright 2025 Christophe Roeder. All rights reserved.

package omop

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"
)

// CSVWriter writes OMOP data to CSV files
type CSVWriter struct {
	outputDir string
}

// NewCSVWriter creates a new CSV writer for the specified output directory
func NewCSVWriter(outputDir string) *CSVWriter {
	return &CSVWriter{outputDir: outputDir}
}

// WriteAll writes all OMOP tables to CSV files
func (w *CSVWriter) WriteAll(data *OMOPData) error {
	if err := w.writeTable("person.csv", data.Persons); err != nil {
		return err
	}
	if err := w.writeTable("visit_occurrence.csv", data.VisitOccurrences); err != nil {
		return err
	}
	if err := w.writeTable("condition_occurrence.csv", data.ConditionOccurrences); err != nil {
		return err
	}
	if err := w.writeTable("drug_exposure.csv", data.DrugExposures); err != nil {
		return err
	}
	if err := w.writeTable("procedure_occurrence.csv", data.ProcedureOccurrences); err != nil {
		return err
	}
	if err := w.writeTable("measurement.csv", data.Measurements); err != nil {
		return err
	}
	if err := w.writeTable("observation.csv", data.Observations); err != nil {
		return err
	}
	if err := w.writeTable("device_exposure.csv", data.DeviceExposures); err != nil {
		return err
	}
	return nil
}

// writeTable writes a slice of structs to a CSV file
func (w *CSVWriter) writeTable(filename string, data interface{}) error {
	filepath := filepath.Join(w.outputDir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Use reflection to get headers and values
	sliceVal := reflect.ValueOf(data)
	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("expected slice, got %s", sliceVal.Kind())
	}

	// Get headers from struct tags
	if sliceVal.Len() == 0 {
		// Write empty file with headers only if we can determine the type
		sliceType := sliceVal.Type().Elem()
		headers := getHeaders(sliceType)
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("failed to write headers: %w", err)
		}
		return nil
	}

	// Get headers from first element
	firstElem := sliceVal.Index(0)
	headers := getHeaders(firstElem.Type())
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Write each row
	for i := 0; i < sliceVal.Len(); i++ {
		row := getRow(sliceVal.Index(i))
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row %d: %w", i, err)
		}
	}

	return nil
}

// getHeaders extracts CSV column names from struct tags
func getHeaders(t reflect.Type) []string {
	var headers []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("csv")
		if tag != "" {
			headers = append(headers, tag)
		} else {
			headers = append(headers, field.Name)
		}
	}
	return headers
}

// getRow extracts values from a struct as strings
func getRow(v reflect.Value) []string {
	var row []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		row = append(row, formatValue(field))
	}
	return row
}

// formatValue converts a reflect.Value to a CSV string
func formatValue(v reflect.Value) string {
	// Handle pointer types
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	case reflect.Bool:
		if v.Bool() {
			return "1"
		}
		return "0"
	default:
		// Handle time.Time
		if v.Type() == reflect.TypeOf(time.Time{}) {
			t := v.Interface().(time.Time)
			if t.IsZero() {
				return ""
			}
			// Check if time component is midnight (date only)
			if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
				return t.Format("2006-01-02")
			}
			return t.Format("2006-01-02 15:04:05")
		}
		return fmt.Sprintf("%v", v.Interface())
	}
}
