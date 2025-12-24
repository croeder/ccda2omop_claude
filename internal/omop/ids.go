package omop

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// GenerateID creates a deterministic int64 ID from input values.
// Uses SHA256 hash truncated to int64 for reproducible IDs.
func GenerateID(values ...string) int64 {
	h := sha256.New()
	for _, v := range values {
		h.Write([]byte(v))
		h.Write([]byte{0}) // separator
	}
	hash := h.Sum(nil)
	// Take first 8 bytes and convert to int64
	// Use absolute value to ensure positive ID
	id := int64(binary.BigEndian.Uint64(hash[:8]))
	if id < 0 {
		id = -id
	}
	return id
}

// GeneratePersonID creates a deterministic person ID from patient identifiers
func GeneratePersonID(patientID string, sourceSystem string) int64 {
	return GenerateID("person", patientID, sourceSystem)
}

// GenerateVisitID creates a deterministic visit ID
func GenerateVisitID(personID int64, encounterID string) int64 {
	return GenerateID("visit", fmt.Sprintf("%d", personID), encounterID)
}

// GenerateConditionID creates a deterministic condition occurrence ID
func GenerateConditionID(personID int64, conditionCode string, startDate string) int64 {
	return GenerateID("condition", fmt.Sprintf("%d", personID), conditionCode, startDate)
}

// GenerateDrugExposureID creates a deterministic drug exposure ID
func GenerateDrugExposureID(personID int64, drugCode string, startDate string) int64 {
	return GenerateID("drug", fmt.Sprintf("%d", personID), drugCode, startDate)
}

// GenerateProcedureID creates a deterministic procedure occurrence ID
func GenerateProcedureID(personID int64, procedureCode string, date string) int64 {
	return GenerateID("procedure", fmt.Sprintf("%d", personID), procedureCode, date)
}

// GenerateMeasurementID creates a deterministic measurement ID
func GenerateMeasurementID(personID int64, measurementCode string, date string, value string) int64 {
	return GenerateID("measurement", fmt.Sprintf("%d", personID), measurementCode, date, value)
}

// GenerateObservationID creates a deterministic observation ID
func GenerateObservationID(personID int64, observationCode string, date string) int64 {
	return GenerateID("observation", fmt.Sprintf("%d", personID), observationCode, date)
}

// GenerateDeviceExposureID creates a deterministic device exposure ID
func GenerateDeviceExposureID(personID int64, deviceCode string, startDate string) int64 {
	return GenerateID("device", fmt.Sprintf("%d", personID), deviceCode, startDate)
}
