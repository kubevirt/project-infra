/*
 * Copyright 2026 The KubeVirt Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package prowjobconfigs

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"sigs.k8s.io/prow/pkg/config"
)

func TestDeriveSelectionLabels(t *testing.T) {
	tests := []struct {
		name string
		want map[string]string
	}{
		{name: "pull-kubevirt-build"},
		{
			name: "pull-kubevirt-e2e-k8s-1.37-sig-compute",
			want: map[string]string{LabelK8sVersion: "1.37", LabelSIG: "compute", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.37-sig-compute-serial",
			want: map[string]string{LabelK8sVersion: "1.37", LabelSIG: "compute", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.37-sig-compute-migrations",
			want: map[string]string{LabelK8sVersion: "1.37", LabelSIG: "compute", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.37-ipv6-sig-network",
			want: map[string]string{LabelK8sVersion: "1.37", LabelSIG: "network", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.36-windows2016",
			want: map[string]string{LabelK8sVersion: "1.36", LabelSIG: SelectionLabelNone, LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.36-windows2016-1.10",
			want: map[string]string{LabelK8sVersion: "1.36", LabelSIG: SelectionLabelNone, LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.10-sig-compute",
			want: map[string]string{LabelK8sVersion: "1.10", LabelSIG: "compute", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-kind-1.37-vgpu",
			want: map[string]string{LabelK8sVersion: "1.37", LabelSIG: SelectionLabelNone, LabelProvider: "kind"},
		},
		{
			name: "pull-kubevirt-e2e-kind-sriov",
			want: map[string]string{LabelK8sVersion: SelectionLabelNone, LabelSIG: SelectionLabelNone, LabelProvider: "kind"},
		},
		{
			name: "pull-kubevirt-e2e-windows2016",
			want: map[string]string{LabelK8sVersion: SelectionLabelNone, LabelSIG: SelectionLabelNone, LabelProvider: SelectionLabelNone},
		},
		{
			name: "periodic-kubevirt-e2e-k8s-1.35-sig-storage",
			want: map[string]string{LabelK8sVersion: "1.35", LabelSIG: "storage", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.24-operator",
			want: map[string]string{LabelK8sVersion: "1.24", LabelSIG: "operator", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.24-operator-root",
			want: map[string]string{LabelK8sVersion: "1.24", LabelSIG: "operator", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.37-sig-performance-kwok",
			want: map[string]string{LabelK8sVersion: "1.37", LabelSIG: "performance", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.37-sig-monitoring",
			want: map[string]string{LabelK8sVersion: "1.37", LabelSIG: "monitoring", LabelProvider: "k8s"},
		},
		{
			name: "pull-kubevirt-e2e-k8s-1.33-sig-compute-1.6",
			want: map[string]string{LabelK8sVersion: "1.33", LabelSIG: "compute", LabelProvider: "k8s"},
		},
		{
			name: "periodic-kubevirt-e2e-sysprep",
			want: map[string]string{LabelK8sVersion: SelectionLabelNone, LabelSIG: SelectionLabelNone, LabelProvider: SelectionLabelNone},
		},
		{
			name: "pull-kubevirt-e2e-kind-1.35-sig-compute-arm64",
			want: map[string]string{LabelK8sVersion: "1.35", LabelSIG: "compute", LabelProvider: "kind"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveSelectionLabels(tt.name)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeriveSelectionLabels() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSelectionLabelErrors(t *testing.T) {
	job := "pull-kubevirt-e2e-k8s-1.37-sig-compute"
	if errs := SelectionLabelErrors(job, DeriveSelectionLabels(job)); len(errs) != 0 {
		t.Fatalf("matching labels returned %v", errs)
	}
	if errs := SelectionLabelErrors(job, nil); len(errs) != 3 {
		t.Fatalf("missing labels returned %v", errs)
	}
	wrong := DeriveSelectionLabels(job)
	wrong[LabelK8sVersion] = "1.36"
	errs := SelectionLabelErrors(job, wrong)
	if len(errs) != 1 || !strings.Contains(errs[0], `expected "1.37"`) {
		t.Fatalf("wrong version returned %v", errs)
	}
	errs = SelectionLabelErrors("pull-kubevirt-build", map[string]string{LabelSIG: "compute"})
	if len(errs) != 1 || !strings.Contains(errs[0], "not a kubevirt e2e lane") {
		t.Fatalf("stray label returned %v", errs)
	}
}

func TestInsertSelectionLabels(t *testing.T) {
	input := `presubmits:
  kubevirt/kubevirt:
  - always_run: false
    labels:
      ci.kubevirt.io/prometheus: ""
      preset-docker-mirror-proxy: "true"
    max_concurrency: 11
    name: pull-kubevirt-e2e-k8s-1.37-sig-compute
    spec:
      containers:
      - command:
        - echo
  - always_run: true
    labels:
      preset-bazel-cache: "true"
    name: pull-kubevirt-unit-test
  - always_run: false
    name: pull-kubevirt-e2e-windows2016
periodics:
- labels:
    preset-shared-images: "true"
  name: periodic-kubevirt-e2e-kind-1.34-vgpu
  spec:
    containers:
    - name: test
- name: periodic-other
`
	want := `presubmits:
  kubevirt/kubevirt:
  - always_run: false
    labels:
      ci.kubevirt.io/k8s-version: "1.37"
      ci.kubevirt.io/prometheus: ""
      ci.kubevirt.io/provider: "k8s"
      ci.kubevirt.io/sig: "compute"
      preset-docker-mirror-proxy: "true"
    max_concurrency: 11
    name: pull-kubevirt-e2e-k8s-1.37-sig-compute
    spec:
      containers:
      - command:
        - echo
  - always_run: true
    labels:
      preset-bazel-cache: "true"
    name: pull-kubevirt-unit-test
  - always_run: false
    labels:
      ci.kubevirt.io/k8s-version: "none"
      ci.kubevirt.io/provider: "none"
      ci.kubevirt.io/sig: "none"
    name: pull-kubevirt-e2e-windows2016
periodics:
- labels:
    ci.kubevirt.io/k8s-version: "1.34"
    ci.kubevirt.io/provider: "kind"
    ci.kubevirt.io/sig: "none"
    preset-shared-images: "true"
  name: periodic-kubevirt-e2e-kind-1.34-vgpu
  spec:
    containers:
    - name: test
- name: periodic-other
`
	got := InsertSelectionLabels(input)
	if got != want {
		t.Fatalf("InsertSelectionLabels() =\n%s\nwant:\n%s", got, want)
	}
	if again := InsertSelectionLabels(got); again != got {
		t.Fatal("InsertSelectionLabels is not idempotent")
	}
}

func TestKubevirtJobConfigsHaveSelectionLabels(t *testing.T) {
	matches, err := filepath.Glob("../../../github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("no kubevirt job config files found")
	}
	for _, path := range matches {
		jobConfig, err := config.ReadJobConfig(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		var errs []string
		for _, jobs := range jobConfig.PresubmitsStatic {
			for _, job := range jobs {
				errs = append(errs, SelectionLabelErrors(job.Name, job.Labels)...)
			}
		}
		for _, jobs := range jobConfig.PostsubmitsStatic {
			for _, job := range jobs {
				errs = append(errs, SelectionLabelErrors(job.Name, job.Labels)...)
			}
		}
		for _, job := range jobConfig.Periodics {
			errs = append(errs, SelectionLabelErrors(job.Name, job.Labels)...)
		}
		if len(errs) > 0 {
			t.Errorf("%s:\n%s", path, strings.Join(errs, "\n"))
		}
	}
}
