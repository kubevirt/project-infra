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
 * Copyright the KubeVirt Authors.
 *
 */

package main

import (
	"reflect"
	"testing"

	"sigs.k8s.io/prow/pkg/config"
)

func periodics(names ...string) []config.Periodic {
	var result []config.Periodic
	for _, name := range names {
		result = append(result, config.Periodic{JobBase: config.JobBase{Name: name}})
	}
	return result
}

func TestLatestSIGLanes(t *testing.T) {
	tests := []struct {
		name      string
		periodics []config.Periodic
		want      jobNames
	}{
		{
			name:      "no periodics",
			periodics: nil,
			want:      nil,
		},
		{
			name: "returns the SIG lanes of the latest Kubernetes version",
			periodics: periodics(
				"periodic-kubevirt-e2e-k8s-1.36-sig-compute",
				"periodic-kubevirt-e2e-k8s-1.37-sig-storage",
				"periodic-kubevirt-e2e-k8s-1.37-sig-compute",
				"periodic-kubevirt-e2e-k8s-1.36-sig-network",
				"periodic-kubevirt-e2e-k8s-1.37-sig-network",
				"periodic-kubevirt-e2e-k8s-1.37-sig-operator",
			),
			want: jobNames{
				"periodic-kubevirt-e2e-k8s-1.37-sig-compute",
				"periodic-kubevirt-e2e-k8s-1.37-sig-network",
				"periodic-kubevirt-e2e-k8s-1.37-sig-operator",
				"periodic-kubevirt-e2e-k8s-1.37-sig-storage",
			},
		},
		{
			name: "keeps the latest lane of each SIG while lanes for a new Kubernetes version are added",
			periodics: periodics(
				"periodic-kubevirt-e2e-k8s-1.37-sig-compute",
				"periodic-kubevirt-e2e-k8s-1.37-sig-network",
				"periodic-kubevirt-e2e-k8s-1.38-sig-compute",
			),
			want: jobNames{
				"periodic-kubevirt-e2e-k8s-1.37-sig-network",
				"periodic-kubevirt-e2e-k8s-1.38-sig-compute",
			},
		},
		{
			name: "compares versions numerically",
			periodics: periodics(
				"periodic-kubevirt-e2e-k8s-1.100-sig-compute",
				"periodic-kubevirt-e2e-k8s-1.99-sig-compute",
			),
			want: jobNames{"periodic-kubevirt-e2e-k8s-1.100-sig-compute"},
		},
		{
			name: "ignores lanes that are not plain SIG lanes",
			periodics: periodics(
				"periodic-kubevirt-e2e-k8s-1.37-sig-compute",
				"periodic-kubevirt-e2e-k8s-1.38-sig-compute-migrations",
				"periodic-kubevirt-e2e-k8s-1.38-sig-network-with-dnc",
				"periodic-kubevirt-e2e-k8s-1.38-ipv6-sig-network",
				"periodic-kubevirt-e2e-k8s-1.38-sig-monitoring",
				"periodic-kubevirt-push-test-metrics",
			),
			want: jobNames{"periodic-kubevirt-e2e-k8s-1.37-sig-compute"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := latestSIGLanes(tt.periodics); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("latestSIGLanes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobNamesFlag(t *testing.T) {
	var j jobNames
	for _, value := range []string{"job-a", "job-b"} {
		if err := j.Set(value); err != nil {
			t.Fatalf("Set(%q) returned error: %v", value, err)
		}
	}
	if want := (jobNames{"job-a", "job-b"}); !reflect.DeepEqual(j, want) {
		t.Errorf("jobNames = %v, want %v", j, want)
	}
	if got, want := j.String(), "job-a,job-b"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
