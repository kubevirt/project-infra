/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2022 Red Hat, Inc.
 *
 */

package build

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	osexec "os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestNewRating(t *testing.T) {
	type args struct {
		name                   string
		source                 string
		startFrom              time.Duration
		buildNumbersToFailures map[int64]int64
	}
	tests := []struct {
		name string
		args args
		want Rating
	}{
		{
			name: "simple things",
			args: args{
				name:      "test",
				source:    "blah",
				startFrom: 0,
				buildNumbersToFailures: map[int64]int64{
					5: 5,
					4: 4,
					3: 3,
					2: 2,
					1: 1,
				},
			},
			want: Rating{
				Name:      "test",
				Source:    "blah",
				StartFrom: 0,
				BuildNumbers: []int64{
					5,
					4,
					3,
					2,
					1,
				},
				BuildNumbersToData: map[int64]BuildData{
					5: BuildData{5, 5, 2},
					4: BuildData{4, 4, 1},
					3: BuildData{3, 3, 0},
					2: BuildData{2, 2, 1},
					1: BuildData{1, 1, 2},
				},
				TotalCompletedBuilds: 5,
				TotalFailures:        15,
				Mean:                 3,
				Variance:             2,
				StandardDeviation:    1.4142135623730951,
			},
		},
		{
			name: "simple things with outlier",
			args: args{
				name:      "test",
				source:    "blah",
				startFrom: 0,
				buildNumbersToFailures: map[int64]int64{
					5: 15,
					4: 4,
					3: 3,
					2: 2,
					1: 1,
				},
			},
			want: Rating{
				Name:      "test",
				Source:    "blah",
				StartFrom: 0,
				BuildNumbers: []int64{
					5,
					4,
					3,
					2,
					1,
				},
				BuildNumbersToData: map[int64]BuildData{
					5: BuildData{5, 15, 2},
					4: BuildData{4, 4, 1},
					3: BuildData{3, 3, 1},
					2: BuildData{2, 2, 1},
					1: BuildData{1, 1, 1},
				},
				TotalCompletedBuilds: 5,
				TotalFailures:        25,
				Mean:                 5,
				Variance:             26.000000,
				StandardDeviation:    5.099020,
			},
		},
		{
			name: "compute with one bad build",
			args: args{
				name:      "test-kubevirt-cnv-4.11-compute-ocs",
				source:    "blah",
				startFrom: 0,
				buildNumbersToFailures: map[int64]int64{
					318: 0,
					317: 0,
					316: 0,
					315: 0,
					314: 0,
					313: 3,
					312: 55,
					311: 1,
					310: 0,
					309: 0,
					308: 0,
					307: 0,
					306: 0,
					305: 0,
					304: 0,
					303: 0,
					302: 0,
					301: 0,
					300: 0,
					299: 0,
					298: 0,
					297: 0,
					296: 0,
					294: 0,
					293: 0,
					292: 2,
					291: 0,
					290: 0,
					289: 0,
					288: 0,
					287: 0,
					286: 0,
					285: 0,
				},
			},
			want: Rating{
				Name:      "test-kubevirt-cnv-4.11-compute-ocs",
				Source:    "blah",
				StartFrom: 0,
				BuildNumbers: []int64{
					318,
					317,
					316,
					315,
					314,
					313,
					312,
					311,
					310,
					309,
					308,
					307,
					306,
					305,
					304,
					303,
					302,
					301,
					300,
					299,
					298,
					297,
					296,
					294,
					293,
					292,
					291,
					290,
					289,
					288,
					287,
					286,
					285,
				},
				BuildNumbersToData: map[int64]BuildData{
					318: BuildData{318, 0, 1},
					317: BuildData{317, 0, 1},
					316: BuildData{316, 0, 1},
					315: BuildData{315, 0, 1},
					314: BuildData{314, 0, 1},
					313: BuildData{313, 3, 1},
					312: BuildData{312, 55, 6},
					311: BuildData{311, 1, 1},
					310: BuildData{310, 0, 1},
					309: BuildData{309, 0, 1},
					308: BuildData{308, 0, 1},
					307: BuildData{307, 0, 1},
					306: BuildData{306, 0, 1},
					305: BuildData{305, 0, 1},
					304: BuildData{304, 0, 1},
					303: BuildData{303, 0, 1},
					302: BuildData{302, 0, 1},
					301: BuildData{301, 0, 1},
					300: BuildData{300, 0, 1},
					299: BuildData{299, 0, 1},
					298: BuildData{298, 0, 1},
					297: BuildData{297, 0, 1},
					296: BuildData{296, 0, 1},
					294: BuildData{294, 0, 1},
					293: BuildData{293, 0, 1},
					292: BuildData{292, 2, 1},
					291: BuildData{291, 0, 1},
					290: BuildData{290, 0, 1},
					289: BuildData{289, 0, 1},
					288: BuildData{288, 0, 1},
					287: BuildData{287, 0, 1},
					286: BuildData{286, 0, 1},
					285: BuildData{285, 0, 1},
				},
				TotalCompletedBuilds: 33,
				TotalFailures:        61,
				Mean:                 1.8484848484848484,
				Variance:             88.67401285583105,
				StandardDeviation:    9.416687998220555,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRating(tt.args.name, tt.args.source, tt.args.startFrom, tt.args.buildNumbersToFailures)
			if areRatingsEqual(got, tt.want) {
				return
			}
			gotIndent, err := json.MarshalIndent(got, "", "\t")
			if err != nil {
				t.Errorf(fmt.Sprintf("%v", err))
			}
			wantIndent, err := json.MarshalIndent(tt.want, "", "\t")
			if err != nil {
				t.Errorf(fmt.Sprintf("%v", err))
			}
			t.Errorf("NewRating() = %v\n, want\n %v\n, diff:\n %v", string(gotIndent), string(wantIndent), diff(string(gotIndent), string(wantIndent)))
		})
	}
}

func areRatingsEqual(a, b Rating) bool {
	if a.Name != b.Name ||
		!reflect.DeepEqual(a.BuildNumbers, b.BuildNumbers) ||
		!isSimilar(a.Variance, b.Variance, 0.000001) ||
		!isSimilar(a.StandardDeviation, b.StandardDeviation, 0.000001) ||
		!isSimilar(a.Mean, b.Mean, 0.000001) ||
		a.Source != b.Source ||
		a.StartFrom != b.StartFrom ||
		a.TotalCompletedBuilds != b.TotalCompletedBuilds ||
		a.TotalFailures != b.TotalFailures {
		return false
	}
	if len(a.BuildNumbersToData) != len(b.BuildNumbersToData) {
		return false
	}
	for buildNo, data := range a.BuildNumbersToData {
		buildData, exists := b.BuildNumbersToData[buildNo]
		if !exists {
			return false
		}
		if data.Number != buildData.Number ||
			data.Failures != buildData.Failures ||
			!isSimilar(data.Sigma, buildData.Sigma, 0.1) {
			return false
		}
	}
	return true
}

func isSimilar(a, b, r float64) bool {
	return math.Abs(a-b) <= r
}

func diff(a, b string) string {
	if a == b {
		return "(strings are identical)"
	}
	temp, err := os.MkdirTemp("", "diff")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temp)
	fileNameA := filepath.Join(temp, "a")
	os.WriteFile(fileNameA, []byte(a), 0666)
	fileNameB := filepath.Join(temp, "b")
	os.WriteFile(fileNameB, []byte(b), 0666)
	command := osexec.Command("diff", "-y", "-W", "200", fileNameA, fileNameB)
	output, _ := command.CombinedOutput()
	return string(output)
}

func Test_buildDeviationMap(t *testing.T) {
	type args struct {
		buildNumbersToFailures map[int64]int64
		mean                   float64
		standardDeviation      float64
	}
	tests := []struct {
		name                   string
		args                   args
		wantBuildNumbersToData map[int64]BuildData
	}{
		{
			name: "some failures",
			args: args{
				buildNumbersToFailures: map[int64]int64{
					314: 0,
					312: 55,
				},
				mean:              27.5,
				standardDeviation: 27.5,
			},
			wantBuildNumbersToData: map[int64]BuildData{
				314: {
					Number:   314,
					Failures: 0,
					Sigma:    1,
				},
				312: {
					Number:   312,
					Failures: 55,
					Sigma:    1,
				},
			},
		},
		{
			name: "no failures",
			args: args{
				buildNumbersToFailures: map[int64]int64{
					318: 0,
					317: 0,
					316: 0,
				},
				mean:              0,
				standardDeviation: 0,
			},
			wantBuildNumbersToData: map[int64]BuildData{
				318: {
					Number:   318,
					Failures: 0,
					Sigma:    0,
				},
				317: {
					Number:   317,
					Failures: 0,
					Sigma:    0,
				},
				316: {
					Number:   316,
					Failures: 0,
					Sigma:    0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotBuildNumbersToData := buildDeviationMap(tt.args.buildNumbersToFailures, tt.args.mean, tt.args.standardDeviation); !reflect.DeepEqual(gotBuildNumbersToData, tt.wantBuildNumbersToData) {
				t.Errorf("buildDeviationMap() = %v, want %v", gotBuildNumbersToData, tt.wantBuildNumbersToData)
			}
		})
	}
}

func TestRating_ShouldFilterBuild(t *testing.T) {
	type fields struct {
		Name                 string
		Source               string
		StartFrom            time.Duration
		BuildNumbers         []int64
		BuildNumbersToData   map[int64]BuildData
		TotalCompletedBuilds int64
		TotalFailures        int64
		Mean                 float64
		Variance             float64
		StandardDeviation    float64
	}
	type args struct {
		buildNumber int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "should filter build",
			fields: fields{
				Mean: 42,
				BuildNumbersToData: map[int64]BuildData{
					int64(37): {
						Number:   37,
						Failures: 77,
						Sigma:    4,
					},
				},
			},
			args: args{
				buildNumber: 37,
			},
			want: true,
		},
		{
			name: "should not filter build even though sigma would indicate, but no of tests is lower than mean",
			fields: fields{
				Mean: 42,
				BuildNumbersToData: map[int64]BuildData{
					int64(37): {
						Number:   37,
						Failures: 1,
						Sigma:    4,
					},
				},
			},
			args: args{
				buildNumber: 37,
			},
			want: false,
		},
		{
			name: "should not filter build since sigma doesn't indicate",
			fields: fields{
				Mean: 42,
				BuildNumbersToData: map[int64]BuildData{
					int64(37): {
						Number:   37,
						Failures: 69,
						Sigma:    3,
					},
				},
			},
			args: args{
				buildNumber: 37,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Rating{
				Name:                 tt.fields.Name,
				Source:               tt.fields.Source,
				StartFrom:            tt.fields.StartFrom,
				BuildNumbers:         tt.fields.BuildNumbers,
				BuildNumbersToData:   tt.fields.BuildNumbersToData,
				TotalCompletedBuilds: tt.fields.TotalCompletedBuilds,
				TotalFailures:        tt.fields.TotalFailures,
				Mean:                 tt.fields.Mean,
				Variance:             tt.fields.Variance,
				StandardDeviation:    tt.fields.StandardDeviation,
			}
			if got := r.ShouldFilterBuild(tt.args.buildNumber); got != tt.want {
				t.Errorf("ShouldFilterBuild() = %v, want %v", got, tt.want)
			}
		})
	}
}
