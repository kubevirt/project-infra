package flakefinder_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/pkg/flakefinder"
	ghapi "kubevirt.io/project-infra/pkg/flakefinder/github"
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

		prNumber := 2473
		oneOffPrNumber := 2474

		headSHA := "8c33c116def661c69b4a8eb08fac9ca07dfbf03c"
		secondHeadSHA := "8c33c116def661c69b4a8eb08fac9ca07dfbf03d"

		It("finds a commit", func() {
			pullRequest := &ghapi.PullRequest{Number: prNumber, SHA: headSHA}
			isLatestCommit := flakefinder.IsLatestCommit([]byte(singleRepoJsonText), pullRequest)
			Expect(isLatestCommit).To(BeTrue())
		})

		It("does not find a commit if pr number wrong", func() {
			pullRequest := &ghapi.PullRequest{Number: oneOffPrNumber, SHA: headSHA}
			isLatestCommit := flakefinder.IsLatestCommit([]byte(singleRepoJsonText), pullRequest)
			Expect(isLatestCommit).To(BeFalse())
		})

		It("does not find a commit if sha wrong", func() {
			pullRequest := &ghapi.PullRequest{Number: prNumber, SHA: secondHeadSHA}
			isLatestCommit := flakefinder.IsLatestCommit([]byte(singleRepoJsonText), pullRequest)
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
			pullRequest := &ghapi.PullRequest{Number: oneOffPrNumber, SHA: headSHA}
			isLatestCommit := flakefinder.IsLatestCommit([]byte(jsonTextWithTwoRepos), pullRequest)
			Expect(isLatestCommit).To(BeTrue())
		})

	})

})
