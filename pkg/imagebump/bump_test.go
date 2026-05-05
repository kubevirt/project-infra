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

package imagebump

import (
	"errors"
	"fmt"
	"testing"
)

func TestReplaceKubevirtCIImage_tagged(t *testing.T) {
	const input = `          - image: quay.io/kubevirtci/bootstrap:v20201119-a5880e0
`
	image := "quay.io/kubevirtci/bootstrap"
	got := ReplaceKubevirtCIImage(input, image, "v20990101-abcdef1")
	want := `          - image: quay.io/kubevirtci/bootstrap:v20990101-abcdef1
`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestReplaceKubevirtCIImage_digest(t *testing.T) {
	const input = `image: quay.io/kubevirtci/golang@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`
	image := "quay.io/kubevirtci/golang"
	got := ReplaceKubevirtCIImage(input, image, "v20990101-abcdef1")
	want := `image: quay.io/kubevirtci/golang:v20990101-abcdef1
`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSplitImageRef(t *testing.T) {
	repo, tag, ok := SplitImageRef("quay.io/kubevirtci/golang:v20260319-c8f1db8")
	if !ok || repo != "quay.io/kubevirtci/golang" || tag != "v20260319-c8f1db8" {
		t.Fatalf("got %q %q %v", repo, tag, ok)
	}
	_, _, ok = SplitImageRef("scratch")
	if ok {
		t.Fatal("expected !ok")
	}
}

func TestErrNoMatchingTag_errorsIs(t *testing.T) {
	wrapped := fmt.Errorf("%w for quay.io/kubevirtci/hello-world (pattern ^x$)", ErrNoMatchingTag)
	if !errors.Is(wrapped, ErrNoMatchingTag) {
		t.Fatalf("expected errors.Is(wrapped, ErrNoMatchingTag), got: %v", wrapped)
	}
}

func TestIsJobConfigPath(t *testing.T) {
	if !IsJobConfigPath("github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml") {
		t.Fatal("expected match for kubevirt-presubmits.yaml")
	}
	if IsJobConfigPath("github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits-0.58.yaml") {
		t.Fatal("expected versioned presubmits path not to match legacy find regex")
	}
}
