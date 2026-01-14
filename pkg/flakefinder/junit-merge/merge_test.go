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
		name               string
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
			wantConflicts: true,
		},
		{
			name: "skipped test should NOT override successful",
			args: args{
				suites: [][]junit.Suite{
					{
						{
							Name: "testsuite where test was successful",
							Tests: []junit.Test{
								{
									Name:   "successful test",
									Status: junit.StatusPassed,
								},
							},
						},
					},
					{
						{
							Name: "testsuite where test was skipped",
							Tests: []junit.Test{
								{
									Name:   "successful test",
									Status: junit.StatusSkipped,
								},
							},
						},
					},
				},
			},
			wantTestsWithState: []junit.Test{
				{
					Name:   "successful test",
					Status: junit.StatusPassed,
				},
			},
			wantConflicts: false,
		},
		{
			name: "skipped test should NOT override failed",
			args: args{
				suites: [][]junit.Suite{
					{
						{
							Name: "testsuite where test failed",
							Tests: []junit.Test{
								{
									Name:   "failed test",
									Status: junit.StatusFailed,
								},
							},
						},
					},
					{
						{
							Name: "testsuite where test was skipped",
							Tests: []junit.Test{
								{
									Name:   "failed test",
									Status: junit.StatusSkipped,
								},
							},
						},
					},
				},
			},
			wantTestsWithState: []junit.Test{
				{
					Name:   "failed test",
					Status: junit.StatusFailed,
				},
			},
			wantConflicts: false,
		},
		{
			name: "skipped test should NOT override errored",
			args: args{
				suites: [][]junit.Suite{
					{
						{
							Name: "testsuite where test errored",
							Tests: []junit.Test{
								{
									Name:   "errored test",
									Status: junit.StatusError,
								},
							},
						},
					},
					{
						{
							Name: "testsuite where test was skipped",
							Tests: []junit.Test{
								{
									Name:   "errored test",
									Status: junit.StatusSkipped,
								},
							},
						},
					},
				},
			},
			wantTestsWithState: []junit.Test{
				{
					Name:   "errored test",
					Status: junit.StatusError,
				},
			},
			wantConflicts: false,
		},
		{
			name: "failed test SHOULD override passed",
			args: args{
				suites: [][]junit.Suite{
					{
						{
							Name: "testsuite where test passed",
							Tests: []junit.Test{
								{
									Name:   "failed test",
									Status: junit.StatusPassed,
								},
							},
						},
					},
					{
						{
							Name: "testsuite where test failed",
							Tests: []junit.Test{
								{
									Name:   "failed test",
									Status: junit.StatusFailed,
								},
							},
						},
					},
				},
			},
			wantTestsWithState: []junit.Test{
				{
					Name:   "failed test",
					Status: junit.StatusFailed,
				},
			},
			wantConflicts: true,
		},
		{
			name: "errored test SHOULD override passed",
			args: args{
				suites: [][]junit.Suite{
					{
						{
							Name: "testsuite where test passed",
							Tests: []junit.Test{
								{
									Name:   "errored test",
									Status: junit.StatusPassed,
								},
							},
						},
					},
					{
						{
							Name: "testsuite where test failed",
							Tests: []junit.Test{
								{
									Name:   "errored test",
									Status: junit.StatusError,
								},
							},
						},
					},
				},
			},
			wantTestsWithState: []junit.Test{
				{
					Name:   "errored test",
					Status: junit.StatusError,
				},
			},
			wantConflicts: true,
		},
		{
			name: "passed test should NOT override failed",
			args: args{
				suites: [][]junit.Suite{
					{
						{
							Name: "testsuite where test failed",
							Tests: []junit.Test{
								{
									Name:   "failed test",
									Status: junit.StatusFailed,
								},
							},
						},
					},
					{
						{
							Name: "testsuite where test passed",
							Tests: []junit.Test{
								{
									Name:   "failed test",
									Status: junit.StatusPassed,
								},
							},
						},
					},
				},
			},
			wantTestsWithState: []junit.Test{
				{
					Name:   "failed test",
					Status: junit.StatusFailed,
				},
			},
			wantConflicts: true,
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
