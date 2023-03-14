package main

import (
	"encoding/json"
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
