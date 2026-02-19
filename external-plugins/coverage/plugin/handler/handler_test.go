// TODO: Add unit tests using Ginkgo

package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/github"
)

var _ = Describe("detectGoFileChanges", func() {
	DescribeTable("returns true when a file is in a watched path",
		func(files []string) {
			Expect(detectGoFileChanges(files)).To(BeTrue())
		},
		Entry("external-plugins path", []string{"external-plugins/coverage/plugin/handler/handler.go"}),
		Entry("releng path", []string{"releng/release-tool/release-tool.go"}),
		Entry("robots path", []string{"robots/flakefinder/flakefinder.go"}),
		Entry("pkg path", []string{"pkg/git/blame_test.go"}),
	)

	DescribeTable("returns false when no relevant file is in the list",
		func(files []string) {
			Expect(detectGoFileChanges(files)).To(BeFalse())
		},
		Entry("github path", []string{"github/ci/prow-deploy/config.yaml"}),
		Entry("root level file", []string{"README.md"}),
		Entry("empty list", []string{}),
	)
})

var _ = Describe("actOnPrEvent", func() {
	DescribeTable("returns true when the action should trigger the coverage plugin",
		func(action github.PullRequestEventAction) {
			event := &github.PullRequestEvent{Action: action}
			Expect(actOnPrEvent(event)).To(BeTrue())
		},
		Entry("opened", github.PullRequestActionOpened),
		Entry("synchronize", github.PullRequestActionSynchronize),
	)

	DescribeTable("returns false when the action should not trigger the coverage plugin",
		func(action github.PullRequestEventAction) {
			event := &github.PullRequestEvent{Action: action}
			Expect(actOnPrEvent(event)).To(BeFalse())
		},
		Entry("closed", github.PullRequestActionClosed),
		Entry("labeled", github.PullRequestActionLabeled),
		Entry("edited", github.PullRequestActionEdited),
	)
})

var _ = Describe("generateCoverageJob", func() {
	var (
		handler *GitHubEventsHandler
		pr      *github.PullRequest
		job     prowapi.ProwJob
	)

	BeforeEach(func() {
		handler = &GitHubEventsHandler{
			jobsNamespace: "kubevirt-prow-jobs",
		}

		pr = &github.PullRequest{
			Number: 123,
			Base: github.PullRequestBranch{
				Ref: "main",
				SHA: "sha-1",
				Repo: github.Repo{
					Name:     "project-infra",
					Owner:    github.User{Login: "kubevirt"},
					FullName: "kubevirt/project-infra",
				},
			},
			Head: github.PullRequestBranch{
				SHA: "sha-2",
			},
			User: github.User{Login: "testuser"},
		}

		job = handler.generateCoverageJob(pr, "test-event-123")
	})

	DescribeTable("Should set the correct configuration",
		func(getField func(prowapi.ProwJob) string, expectedValue string) {
			Expect(getField(job)).To(Equal(expectedValue))

		},
		Entry("job name", func(j prowapi.ProwJob) string { return j.Spec.Job }, "coverage-auto"),
		Entry("cluster", func(j prowapi.ProwJob) string { return j.Spec.Cluster }, "kubevirt-prow-control-plane"),
		Entry("coverage-plugin label", func(j prowapi.ProwJob) string { return j.Labels["coverage-plugin"] }, "true"),
		Entry("container image", func(j prowapi.ProwJob) string { return j.Spec.PodSpec.Containers[0].Image }, "quay.io/kubevirtci/golang:v20251218-e7a7fc9"),
	)
	It("Should run make coverage command", func() {
		Expect(job.Spec.PodSpec.Containers[0].Args).To(ContainElement("make coverage"))
	})
	It("Should set go version env to 1.25.1", func() {
		Expect(job.Spec.PodSpec.Containers[0].Env).To(ContainElement(corev1.EnvVar{
			Name:  "GO_MOD_PATH",
			Value: "go.mod",
		}))
	})
})
