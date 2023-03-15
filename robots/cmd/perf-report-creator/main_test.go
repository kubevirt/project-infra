package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

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
				"2022-10-13": {
					VMResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 111.11111111111111},
						},
					},
				},
				"2022-10-14": {
					VMResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 115.11111111111111},
						},
					},
				},
				"2023-02-15": {
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
			got, err := getWeeklyVMResults(tt.results)
			if (err != nil) != tt.wantErr {
				t.Errorf("getWeeklyVMResults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if !reflect.DeepEqual(got, tt.want) {
			//	t.Errorf("getWeeklyVMResults() got = %v, want %v", got, tt.want)
			//}
			//fmt.Println(got)

			results := map[string][]Result{}
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
				"2022-10-13": {
					VMIResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 111.11111111111111},
						},
					},
				},
				"2022-10-14": {
					VMIResult: Result{
						Values: map[ResultType]ResultValue{
							"CREATE-pods-count": {Value: 115.11111111111111},
						},
					},
				},
				"2023-02-15": {
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
			got, err := getWeeklyVMIResults(tt.results)
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
