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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package main_test

import (
	prowgithub "k8s.io/test-infra/prow/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strings"

	. "kubevirt.io/project-infra/robots/cmd/flake-issue-creator"
)

var _ = Describe("cluster_failure.go", func() {

	When("extracting cluster failure issues", func() {

		It("returns err on labels not found", func() {
			labels, err := GetFlakeIssuesLabels(DefaultIssueLabels, []prowgithub.Label{}, "kubevirt", "kubevirt")
			Expect(err).To(Not(BeNil()))
			Expect(labels).To(BeNil())
		})

		It("returns found labels", func() {
			labels := []prowgithub.Label{
				{Name: strings.Split(DefaultIssueLabels, ",")[0]},
				{Name: strings.Split(DefaultIssueLabels, ",")[1]},
			}

			issueLabels, err := GetFlakeIssuesLabels(DefaultIssueLabels, labels, "kubevirt", "kubevirt")
			Expect(err).To(BeNil())
			Expect(issueLabels).To(Not(BeNil()))
			Expect(issueLabels).To(HaveLen(2))
		})

	})

})
