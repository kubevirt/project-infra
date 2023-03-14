package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Record struct {
	StartDate    string
	NumberOfDays int
	Data         RecordData
}

type RecordData struct {
	Average    float64
	DataPoints []RecordDataPoint
}

type RecordDataPoint struct {
	Value float64
	Date  time.Time `json:",omitempty"`
}

func NewRecordDateWithAverage(rdps []RecordDataPoint) RecordData {
	r := RecordData{DataPoints: rdps}
	sum, count := 0.0, 0.0
	for _, rdp := range r.DataPoints {
		sum = sum + rdp.Value
		count += 1
	}
	if count == 0 {
		r.Average = 0.0
		return r
	}
	r.Average = sum / count
	return r
}

func calculateAVGAndWriteOutput(results map[YearWeek][]Result, objType string, metrics ...ResultType) error {
	for _, metric := range metrics {
		for yw, _ := range results {
			record := Record{
				StartDate:    getMondayOfWeekDate(yw.Year, yw.Week),
				NumberOfDays: 0,
			}
			outputDirPath := filepath.Join(outputDir, objType, string(metric), getMondayOfWeekDate(yw.Year, yw.Week), "data")
			err := os.MkdirAll(outputDirPath, 0755)
			if err != nil {
				return err
			}
			outputPath := filepath.Join(outputDirPath, "results.json")
			rdp := []RecordDataPoint{}
			for _, result := range results[yw] {
				rdp = append(rdp, RecordDataPoint{
					Value: result.Values[metric].Value,
					// todo: find a way to populate date
				})
			}
			record.Data = NewRecordDateWithAverage(rdp)
			f, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			err = json.NewEncoder(f).Encode(&record)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
