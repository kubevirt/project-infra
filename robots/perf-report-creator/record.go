package main

import (
	"encoding/json"
	"fmt"
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
	Date  *time.Time `json:",omitempty"`
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

func calculateAVGAndWriteOutput(results map[YearWeek][]ResultWithDate, objType string, outputDir string, metrics ...string) error {
	for _, metric := range metrics {
		for yw := range results {
			record := Record{
				StartDate:    getMondayOfWeekDate(yw.Year, yw.Week),
				NumberOfDays: 0,
			}
			outputDirPath := filepath.Join(outputDir, objType, metric, getMondayOfWeekDate(yw.Year, yw.Week), "data")
			err := os.MkdirAll(outputDirPath, 0755)
			if err != nil {
				return err
			}
			outputPath := filepath.Join(outputDirPath, "results.json")
			fmt.Println("writing output to", outputPath)
			rdp := []RecordDataPoint{}
			for _, result := range results[yw] {
				result := result
				rdp = append(rdp, RecordDataPoint{
					Value: result.Values[ResultType(metric)].Value,
					Date:  result.Date,
				})
			}
			record.NumberOfDays = 7
			record.Data = NewRecordDateWithAverage(rdp)
			f, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			e := json.NewEncoder(f)
			e.SetIndent("", "  ")
			err = e.Encode(&record)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
