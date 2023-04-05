package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var resultsDir = "output/results"
var weeklyResultsDir = "output/weekly"

func Test_readLinesAndMatchRegex(t *testing.T) {
	tests := []struct {
		name         string
		filepath     string
		wantErr      bool
		regex        string
		stringLength int
	}{
		{
			name:     "test valid build-log.txt to read vm data",
			filepath: "test-build-log.txt",
			wantErr:  false,
			regex:    "create a batch of 100 running VMs should sucessfully create all VMS",
		},
		{
			name:     "test valid build-log.txt to read vmi data",
			filepath: "test-build-log.txt",
			wantErr:  false,
			regex:    "create a batch of 100 VMIs should sucessfully create all VMIS",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.filepath)
			if err != nil {
				t.Errorf("error opening file: %#v", err)
				return
			}

			got, err := readLinesAndMatchRegex(f, tt.regex)
			if (err != nil) != tt.wantErr {
				t.Errorf("readLinesAndMatchRegex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			r := Result{}
			if err := json.Unmarshal([]byte(got), &r); err != nil {
				t.Errorf("unable to unmarshal data from json text: %#v", err)
			}
		})
	}
}

func Test_getWeeklyVMResults(t *testing.T) {
	tests := []struct {
		name    string
		results Collection
		//want    map[string][]Result
		wantErr bool
	}{
		{
			name: "test two days in different Year",
			results: Collection{
				"job-123-0": {
					JobDirCreationTime: time.Date(2022, 10, 13, 0, 0, 0, 0, time.UTC),
					VMResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 111.11111111111111},
						},
					},
				},
				"job-123-1": {
					JobDirCreationTime: time.Date(2022, 10, 14, 0, 0, 0, 0, time.UTC),
					VMResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 115.11111111111111},
						},
					},
				},
				"job-125-0": {
					JobDirCreationTime: time.Date(2023, 02, 15, 0, 0, 0, 0, time.UTC),
					VMResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 150.11111111111111},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getWeeklyVMResults(&tt.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("getWeeklyVMResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("getWeeklyVMResults() got = %v, want %v", got, tt.want)
			//}
			//fmt.Println(got)

			results := map[string][]ResultWithDate{}
			for yw, result := range got {
				date := getMondayOfWeekDate(yw.Year, yw.Week)
				//fmt.Println(date, result)
				if _, ok := results[date]; ok {
					results[date] = append(results[date], result...)
					continue
				}
				results[date] = result
			}
			f, err := os.Create("results-1.json")
			if err != nil {
				t.Errorf("unable to open file %+v", err)
				return
			}
			e := json.NewEncoder(f)
			err = e.Encode(results)
			if err != nil {
				t.Errorf("unable to encode json to file %+v", err)
			}
		})
	}
}

func Test_getWeeklyVMIResults(t *testing.T) {
	tests := []struct {
		name    string
		results Collection
		//want    map[string][]Result
		wantErr bool
	}{
		{
			name: "test two days in different Year",
			results: Collection{
				"job-123-0": {
					JobDirCreationTime: time.Date(2022, 10, 13, 0, 0, 0, 0, time.UTC),
					VMIResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 111.11111111111111},
						},
					},
				},
				"job-123-1": {
					JobDirCreationTime: time.Date(2022, 10, 14, 0, 0, 0, 0, time.UTC),
					VMIResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 115.11111111111111},
						},
					},
				},
				"job-125-0": {
					JobDirCreationTime: time.Date(2023, 02, 15, 0, 0, 0, 0, time.UTC),
					VMIResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 150.11111111111111},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getWeeklyVMIResults(&tt.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("getWeeklyVMResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("getWeeklyVMResults() got = %v, want %v", got, tt.want)
			//}
			//fmt.Println(got)
			for yw, result := range got {
				date := getMondayOfWeekDate(yw.Year, yw.Week)
				fmt.Println(date, result)
			}
		})
	}
}

func Test_writeCollection(t *testing.T) {
	tests := []struct {
		name               string
		collection         *Collection
		performanceJobName string
		outputDir          string
		wantErr            bool
	}{
		{
			name:               "check results directory for a example result",
			performanceJobName: "test-job-name",
			collection: &Collection{
				"job-123-0": {
					JobDirCreationTime: time.Date(2022, 10, 13, 0, 0, 0, 0, time.UTC),
					VMIResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 111.11111111111111},
						},
					},
					VMResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 111.11111111111111},
						},
					},
				},
				"job-123-1": {
					JobDirCreationTime: time.Date(2022, 10, 14, 0, 0, 0, 0, time.UTC),
					VMIResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 115.11111111111111},
						},
					},
					VMResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 115.11111111111111},
						},
					},
				},
				"job-125-0": {
					JobDirCreationTime: time.Date(2023, 02, 15, 0, 0, 0, 0, time.UTC),
					VMIResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 150.11111111111111},
						},
					},
					VMResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 150.11111111111111},
						},
					},
				},
			},
			outputDir: resultsDir,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := writeCollection(tt.collection, tt.outputDir, tt.performanceJobName); (err != nil) != tt.wantErr {
				t.Errorf("writeCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
			dirs, err := os.ReadDir(tt.outputDir)
			if err != nil {
				t.Errorf("unable to read output directory: %+v", err)
			}
			foundDirs := map[string]int{}
			for _, entry := range dirs {
				if _, ok := (*tt.collection)[entry.Name()]; ok {
					_, err := os.Open(filepath.Join(tt.outputDir, entry.Name(), "results.json"))
					if err == nil {
						foundDirs[entry.Name()] = 0
					}
				}
			}
			if len(foundDirs) != len(*tt.collection) {
				t.Errorf("expeted to find some directories/files which doent exist")
			}

			//for _, entry := range dirs {
			//	if _, ok := (*tt.collection)[entry.Name()]; ok {
			//		err := os.RemoveAll(filepath.Join(tt.outputDir, entry.Name()))
			//		if err != nil {
			//			t.Errorf("error cleaning up the output directory: %+v", err)
			//		}
			//	}
			//}
		})
	}
}

func Test_runWeeklyReports(t *testing.T) {
	tests := []struct {
		name    string
		w       weeklyReportOpts
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "example test",
			w: weeklyReportOpts{
				since:          0,
				resultsDir:     resultsDir,
				outputDir:      weeklyResultsDir,
				vmMetricsList:  string(ResultTypeCreatePodsCount),
				vmiMetricsList: string(ResultTypeCreatePodsCount),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runWeeklyReports(tt.w); (err != nil) != tt.wantErr {
				t.Errorf("runWeeklyReports() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
