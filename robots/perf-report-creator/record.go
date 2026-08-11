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

func readRecord(path string) (*Record, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var record Record
	if err := json.NewDecoder(f).Decode(&record); err != nil {
		return nil, err
	}
	return &record, nil
}

func dataPointKey(dp RecordDataPoint) string {
	if dp.Date == nil {
		return ""
	}
	return dp.Date.UTC().Format(time.RFC3339Nano)
}

func mergeDataPoints(existing, incoming []RecordDataPoint) []RecordDataPoint {
	byDate := map[string]RecordDataPoint{}
	order := make([]string, 0, len(existing)+len(incoming))

	add := func(dp RecordDataPoint) {
		key := dataPointKey(dp)
		if key == "" {
			return
		}
		if _, ok := byDate[key]; !ok {
			order = append(order, key)
		}
		byDate[key] = dp
	}

	for _, dp := range existing {
		add(dp)
	}
	for _, dp := range incoming {
		add(dp)
	}

	merged := make([]RecordDataPoint, 0, len(order))
	for _, key := range order {
		merged = append(merged, byDate[key])
	}
	return merged
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

			existing, err := readRecord(outputPath)
			if err != nil {
				return err
			}
			if existing != nil {
				rdp = mergeDataPoints(existing.Data.DataPoints, rdp)
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
			closeErr := f.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	return nil
}
