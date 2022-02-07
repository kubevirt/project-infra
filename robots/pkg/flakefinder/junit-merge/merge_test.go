package junit_merge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joshdk/go-junit"
)

func Test_merge(t *testing.T) {
	type args struct {
		suites [][]junit.Suite
	}
	tests := []struct {
		name              string
		args               args
		wantTestsWithState []junit.Test
		wantConflicts      bool
	}{
		{
			name: "test name conflict at merge",
			args: args{
				suites: loadTestData("testdata/conflict"),
			},
			wantTestsWithState: nil,
			wantConflicts:      true,
		},
		{
			name: "network data test",
			args: args{
				suites: loadTestData("testdata/network"),
			},
			wantTestsWithState: nil,
			wantConflicts:      false,
		},
		{
			name: "successful test should override skipped",
			args: args{
				suites: loadTestData("testdata/testoverride"),
			},
			wantTestsWithState: []junit.Test{
				{
					Name:       "[rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]VirtualMachine A valid VirtualMachine given Using virtctl interface Using RunStrategyAlways [test_id:4119]should migrate a running VM",
					Classname:  "",
					Duration:   0,
					Status:     junit.StatusPassed,
					Error:      nil,
					Properties: nil,
				},
			},
			wantConflicts:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, hasConflicts := Merge(tt.args.suites)
			if hasConflicts != tt.wantConflicts {
				t.Errorf("merge() hasConflicts = %v, wantConflicts %v", hasConflicts, tt.wantConflicts)
				return
			}
			if tt.wantTestsWithState != nil {
				testsByName := map[string]junit.Test{}
				for _, test := range got[0].Tests {
					testsByName[test.Name] = test
				}
				for _, expectedTestWithState := range tt.wantTestsWithState {
					if expectedTestWithState.Status != testsByName[expectedTestWithState.Name].Status {
						t.Errorf("merge() got = %v, want %v", testsByName[expectedTestWithState.Name], expectedTestWithState)
					}
				}
			}
		})
	}
}

func loadTestData(directory string) [][]junit.Suite {
	result := [][]junit.Suite{}
	dirEntries, err := os.ReadDir(directory)
	if err != nil {
		panic(err)
	}
	for _, dirEntry := range dirEntries {
		suites, err := junit.IngestFile(filepath.Join(directory, dirEntry.Name()))
		if err != nil {
			panic(err)
		}
		result = append(result, suites)
	}
	return result
}
