package junit_merge

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/joshdk/go-junit"
)

func Test_merge(t *testing.T) {
	type args struct {
		suites [][]junit.Suite
	}
	tests := []struct {
		name          string
		args          args
		want          []junit.Suite
		wantConflicts bool
	}{
		{
			name: "test name conflict at merge",
			args: args{
				suites: loadTestData("testdata/conflict"),
			},
			want:          nil,
			wantConflicts: true,
		},
		{
			name: "network data test",
			args: args{
				suites: loadTestData("testdata/network"),
			},
			want:          nil,
			wantConflicts: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, hasConflicts := Merge(tt.args.suites)
			if hasConflicts != tt.wantConflicts {
				t.Errorf("merge() hasConflicts = %v, wantConflicts %v", hasConflicts, tt.wantConflicts)
				return
			}
			if tt.want != nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("merge() got = %v, want %v", got, tt.want)
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
