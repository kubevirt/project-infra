package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type ResultType string

const (
	// rest_client_requests_total
	ResultTypePatchVMICount   ResultType = "PATCH-virtualmachineinstances-count"
	ResultTypeUpdateVMICount  ResultType = "UPDATE-virtualmachineinstances-count"
	ResultTypeCreatePodsCount ResultType = "CREATE-pods-count"

	// kubevirt_vmi_phase_transition_time_from_creation_seconds_bucket
	ResultTypeVMICreationToRunningP99   ResultType = "vmiCreationToRunningSecondsP99"
	ResultTypeVMICreationToRunningP95   ResultType = "vmiCreationToRunningSecondsP95"
	ResultTypeVMICreationToRunningP50   ResultType = "vmiCreationToRunningSecondsP50"
	ResultTypeVMIDeletionToSucceededP99 ResultType = "vmiDeletionToSucceededSecondsP99"
	ResultTypeVMIDeletionToSucceededP95 ResultType = "vmiDeletionToSucceededSecondsP95"
	ResultTypeVMIDeletionToSucceededP50 ResultType = "vmiDeletionToSucceededSecondsP50"
	ResultTypeVMIDeletionToFailedP99    ResultType = "vmiDeletionToFailedSecondsP99"
	ResultTypeVMIDeletionToFailedP95    ResultType = "vmiDeletionToFailedSecondsP95"
	ResultTypeVMIDeletionToFailedP50    ResultType = "vmiDeletionToFailedSecondsP50"
)

const (
	ResultTypeResourceOperationCountFormat = "%s-%s-count"
)

const (
	ResultTypePhaseCountFormat = "%s-phase-count"
)

type ThresholdResult struct {
	ThresholdValue    float64    `json:"thresholdValue"`
	ThresholdMetric   ResultType `json:"thresholdMetric,omitempty"`
	ThresholdRatio    float64    `json:"thresholdRatio,omitempty"`
	ThresholdExceeded bool       `json:"thresholdExceeded"`
}

type ResultValue struct {
	Value           float64          `json:"value"`
	ThresholdResult *ThresholdResult `json:"thresholdResult,omitempty"`
}

type Result struct {
	Values map[ResultType]ResultValue
}

type ResultWithDate struct {
	Values map[ResultType]ResultValue
	Date   *time.Time
}

func (r *Result) toString() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *Result) DumpToFile(filePath string) error {
	str, err := r.toString()
	if err != nil {
		return err
	}

	log.Printf("Writing results to file at path %s", filePath)

	return os.WriteFile(filePath, []byte(str), 0644)
}

func (r *Result) DumpToStdout() error {
	str, err := r.toString()
	if err != nil {
		return err
	}
	fmt.Println(str)
	return nil
}
