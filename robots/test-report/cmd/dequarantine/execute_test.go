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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package dequarantine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	testreport "kubevirt.io/project-infra/pkg/test-report"
)

func Test_applyData(t *testing.T) {
	type args struct {
		options dequarantineExecuteOpts
		values  []*quarantinedTestsRunData
	}
	tests := []struct {
		name                          string
		args                          args
		wantErr                       error
		wantRemainingQuarantinedTests []*testreport.FilterTestRecord
	}{
		{
			name: "no input data",
			args: args{
				options: dequarantineExecuteOpts{},
				values:  loadTestFileOrPanic("testdata/no-results.json"),
			},
			wantErr:                       fmt.Errorf("no input data"),
			wantRemainingQuarantinedTests: nil,
		},
		{
			name: "no test results",
			args: args{
				options: dequarantineExecuteOpts{},
				values:  loadTestFileOrPanic("testdata/no-test-results.json"),
			},
			wantErr: nil,
			wantRemainingQuarantinedTests: []*testreport.FilterTestRecord{
				{
					Id:     "Storage Starting a VirtualMachineInstance with error disk  should pause VMI on IO error",
					Reason: "Failed continously - Tracked in https://issues.redhat.com/browse/CNV-17044, https://issues.redhat.com/browse/CNV-19692",
				},
			},
		},
		{
			name: "test failing",
			args: args{
				options: dequarantineExecuteOpts{},
				values:  loadTestFileOrPanic("testdata/test-failing.json"),
			},
			wantErr: nil,
			wantRemainingQuarantinedTests: []*testreport.FilterTestRecord{
				{
					Id:     "with a dedicated migration network Should migrate over that network",
					Reason: "Failed continously - Tracked in https://issues.redhat.com/browse/CNV-17143",
				},
			},
		},
		{
			name: "test passing",
			args: args{
				options: dequarantineExecuteOpts{},
				values:  loadTestFileOrPanic("testdata/test-passing.json"),
			},
			wantErr:                       nil,
			wantRemainingQuarantinedTests: nil,
		},
		{
			name: "test some passing some failing",
			args: args{
				options: dequarantineExecuteOpts{},
				values:  loadTestFileOrPanic("testdata/test-some-passing-some-failing.json"),
			},
			wantErr: nil,
			wantRemainingQuarantinedTests: []*testreport.FilterTestRecord{
				{
					Id:     "with a dedicated migration network Should migrate over that network",
					Reason: "Failed continously - Tracked in https://issues.redhat.com/browse/CNV-17143",
				},
			},
		},
		{
			name: "some tests skipped",
			args: args{
				options: dequarantineExecuteOpts{
					minimumPassedRunsPerTest: 2,
				},
				values: loadTestFileOrPanic("testdata/some-tests-skipped.json"),
			},
			wantErr: nil,
			wantRemainingQuarantinedTests: []*testreport.FilterTestRecord{
				{
					Id:     "should reach the vmi",
					Reason: "Failed continously - Tracked in https://issues.redhat.com/browse/CNV-17143",
				},
			},
		},
		{
			name: "one test passing, no minimum",
			args: args{
				options: dequarantineExecuteOpts{},
				values:  loadTestFileOrPanic("testdata/one-test-passing.json"),
			},
			wantErr:                       nil,
			wantRemainingQuarantinedTests: nil,
		},
		{
			name: "one test passing, minimum passing two",
			args: args{
				options: dequarantineExecuteOpts{
					minimumPassedRunsPerTest: 2,
				},
				values: loadTestFileOrPanic("testdata/one-test-passing.json"),
			},
			wantErr: nil,
			wantRemainingQuarantinedTests: []*testreport.FilterTestRecord{
				{
					Id:     "Storage Starting a VirtualMachineInstance with lun disk should run the VMI",
					Reason: "Failed continously - Tracked in https://issues.redhat.com/browse/CNV-17044, https://issues.redhat.com/browse/CNV-19692",
				},
			},
		},
		{
			name: "one test fixed, one test passing, minimum passing two",
			args: args{
				options: dequarantineExecuteOpts{
					minimumPassedRunsPerTest: 2,
				},
				values: loadTestFileOrPanic("testdata/test-fixed.json"),
			},
			wantErr:                       nil,
			wantRemainingQuarantinedTests: nil,
		},
		{
			name: "some tests skipped, but passing later on",
			args: args{
				options: dequarantineExecuteOpts{
					minimumPassedRunsPerTest: 2,
				},
				values: loadTestFileOrPanic("testdata/some-tests-skipped-but-two-passing-in-skipped.json"),
			},
			wantErr:                       nil,
			wantRemainingQuarantinedTests: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRemainingQuarantinedTests, gotErr := filterUnstableTestRecords(tt.args.options, tt.args.values)
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("filterUnstableTestRecords() gotErr = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotRemainingQuarantinedTests, tt.wantRemainingQuarantinedTests) {
				t.Errorf("filterUnstableTestRecords() gotRemainingQuarantinedTests = %v, want %v", gotRemainingQuarantinedTests, tt.wantRemainingQuarantinedTests)
			}
		})
	}
}

func loadTestFileOrPanic(filename string) []*quarantinedTestsRunData {
	byteSlice, err := os.ReadFile(filename)
	panicOnError(err)
	buffer := bytes.NewBuffer(byteSlice)
	var result []*quarantinedTestsRunData
	err = json.NewDecoder(buffer).Decode(&result)
	panicOnError(err)
	return result
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
