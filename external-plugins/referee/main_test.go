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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("main", func() {
	Context("options", func() {
		When("initialRetestRepositories", func() {
			const errorExpected = true
			const noErrorExpected = false
			DescribeTable("validation",
				func(o options, errorExpected bool) {
					actual := o.Validate()
					if errorExpected {
						Expect(actual).ToNot(BeNil())
					} else {
						Expect(actual).ToNot(HaveOccurred())
					}
				},
				Entry("one repo, wrong identifier",
					options{initialRetestRepositories: "kubevirt"},
					errorExpected,
				),
				Entry("one repo, correct identifier",
					options{initialRetestRepositories: "kubevirt/kubevirt"},
					noErrorExpected,
				),
				Entry("two repos, incorrect identifier",
					options{initialRetestRepositories: "kubevirt/kubevirt,kubevirt//project-infra"},
					errorExpected,
				),
				Entry("two repos, correct identifiers",
					options{initialRetestRepositories: "kubevirt/kubevirt,kubevirt/project-infra"},
					noErrorExpected,
				),
			)
			DescribeTable("repo identifiers",
				func(o options, expectedRepos []RepoIdentifier) {
					repositories := o.InitialRetestRepositories()
					for index, expectedRepoId := range expectedRepos {
						Expect(repositories[index].Org).To(BeEquivalentTo(expectedRepoId.Org), "org doesn't match")
						Expect(repositories[index].Repo).To(BeEquivalentTo(expectedRepoId.Repo), "repo doesn't match")
					}
				},
				Entry("one incomplete identifier",
					options{initialRetestRepositories: "kubevirt"},
					nil,
				),
				Entry("one complete identifier",
					options{initialRetestRepositories: "kubevirt/kubevirt"},
					[]RepoIdentifier{
						{Org: "kubevirt", Repo: "kubevirt"},
					},
				),
				Entry("two incomplete identifiers",
					options{initialRetestRepositories: "kubevirt/kubevirt,kubevirt//project-infra"},
					nil,
				),
				Entry("two complete identifiers",
					options{initialRetestRepositories: "kubevirt/kubevirt,kubevirt/project-infra"},
					[]RepoIdentifier{
						{Org: "kubevirt", Repo: "kubevirt"},
						{Org: "kubevirt", Repo: "project-infra"},
					},
				),
			)
		})
	})
})

func TestMainSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}
