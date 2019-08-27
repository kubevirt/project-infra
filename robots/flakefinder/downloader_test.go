package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/test-infra/prow/github"
	. "kubevirt.io/project-infra/robots/flakefinder"
)

var _ = Describe("downloader.go", func() {

	When("testing isLatestCommit", func() {

		singleRepoJsonText :=
			`{
	"timestamp":1562772668,
	"pull":"2473",
	"repo-version":"f3bb83f4377b8b45bd47d33373edfacf85361f0e",
	"repos": {
		"kubevirt/kubevirt":"release-0.13:577e95c340e1b21ff431cbba25ad33c891554e38,2473:8c33c116def661c69b4a8eb08fac9ca07dfbf03c"
	}
}`

		It("finds a commit", func() {
			pullRequest := github.PullRequest{Number: 2473, Head: github.PullRequestBranch{SHA: "8c33c116def661c69b4a8eb08fac9ca07dfbf03c"}}
			isLatestCommit := IsLatestCommit([]byte(singleRepoJsonText), &pullRequest)
			Expect(isLatestCommit).To(BeTrue())
		})

		It("does not find a commit", func() {
			pullRequest := github.PullRequest{Number: 2474, Head: github.PullRequestBranch{SHA: "8c33c116def661c69b4a8eb08fac9ca07dfbf03c"}}
			isLatestCommit := IsLatestCommit([]byte(singleRepoJsonText), &pullRequest)
			Expect(isLatestCommit).To(BeFalse())
		})

		It("finds the second commit", func() {
			jsonTextWithTwoRepos :=
				`{
	"timestamp":1562772668,
	"pull":"2473",
	"repo-version":"f3bb83f4377b8b45bd47d33373edfacf85361f0e",
	"repos": {
		"kubevirt/kubevirt1":"release-0.13:577e95c340e1b21ff431cbba25ad33c891554e38,2473:8c33c116def661c69b4a8eb08fac9ca07dfbf03c",
		"kubevirt/kubevirt2":"release-0.11:577e95c340e1b21ff431cbba25ad33c891554e38,2474:8c33c116def661c69b4a8eb08fac9ca07dfbf03c"
	}
}`
			pullRequest := github.PullRequest{Number: 2474, Head: github.PullRequestBranch{SHA: "8c33c116def661c69b4a8eb08fac9ca07dfbf03c"}}
			isLatestCommit := IsLatestCommit([]byte(jsonTextWithTwoRepos), &pullRequest)
			Expect(isLatestCommit).To(BeTrue())
		})

	})

})
