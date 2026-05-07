package handler

import (
	"encoding/json"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1/fake"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"
)

// Helper function to create a payload for a pull request event with default values
func createPREventPayload(action github.PullRequestEventAction, prNumber int, org, repo string) []byte {
	prEvent := github.PullRequestEvent{
		Action: action,
		PullRequest: github.PullRequest{
			Number: prNumber,
			Base: github.PullRequestBranch{
				Ref: "main",
				SHA: "sha-1",
				Repo: github.Repo{
					Name:     repo,
					Owner:    github.User{Login: org},
					FullName: fmt.Sprintf("%s/%s", org, repo),
				},
			},
			Head: github.PullRequestBranch{
				SHA: "sha-2",
			},
			User: github.User{Login: "testuser"},
		},
	}
	payload, _ := json.Marshal(prEvent)
	return payload
}

// errorGithubClient is a fake github client that returns an error
type errorGithubClient struct{}

func (e *errorGithubClient) GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error) {
	return nil, fmt.Errorf("GitHub API error")
}

var _ = Describe("detectGoFileChanges", func() {
	DescribeTable("check go file changes",
		func(files []string, expected bool) {
			Expect(detectGoFileChanges(files)).To(BeEquivalentTo(expected))
		},
		Entry("go files only",
			[]string{
				"external-plugins/coverage/plugin/handler/handler.go",
				"releng/release-tool/release-tool.go",
				"robots/flakefinder/main.go",
				"pkg/git/blame_test.go",
			},
			true,
		),
		Entry("no go files",
			[]string{
				"github/ci/prow-deploy/prow-deploy.yaml",
				"README.md",
			},
			false,
		),
		Entry("last is a go file",
			[]string{
				"github/ci/prow-deploy/prow-deploy.yaml",
				"README.md",
				"pkg/git/blame_test.go",
			},
			true,
		),
		Entry("first is a go file",
			[]string{
				"pkg/git/blame_test.go",
				"README.md",
				"docs/prow-clusters.md",
			},
			true,
		),
		Entry("empty list", []string{}, false),
	)
})

var _ = Describe("shouldActOnPREvent", func() {
	DescribeTable("check PR event actions",
		func(action github.PullRequestEventAction, expected bool) {
			Expect(shouldActOnPREvent(string(action))).To(BeEquivalentTo(expected))
		},
		Entry("opened", github.PullRequestActionOpened, true),
		Entry("synchronize", github.PullRequestActionSynchronize, true),
		Entry("closed", github.PullRequestActionClosed, false),
		Entry("labeled", github.PullRequestActionLabeled, false),
		Entry("edited", github.PullRequestActionEdited, false),
	)
})

var _ = Describe("generateCoverageJob", func() {
	var (
		handler *GitHubEventsHandler
		pr      *github.PullRequest
		job     prowapi.ProwJob
	)

	BeforeEach(func() {
		cfg := &Config{
			Defaults: JobConfig{
				Namespace: "kubevirt-prow-jobs",
				Image:     "quay.io/kubevirtci/covreport:latest",
				Cluster:   "kubevirt-prow-control-plane",
				Env: map[string]string{
					"GO_MOD_PATH": "go.mod",
					"GOTOOLCHAIN": "local",
				},
				TimeoutMinutes:     120,
				GracePeriodSeconds: 15,
				UtilityImages: UtilityImagesConfig{
					CloneRefs:  "us-docker.pkg.dev/k8s-infra-prow/images/clonerefs:v20260401-f6cc3990c",
					InitUpload: "us-docker.pkg.dev/k8s-infra-prow/images/initupload:v20260401-f6cc3990c",
					Entrypoint: "us-docker.pkg.dev/k8s-infra-prow/images/entrypoint:v20260401-f6cc3990c",
					Sidecar:    "us-docker.pkg.dev/k8s-infra-prow/images/sidecar:v20260401-f6cc3990c",
				},
				GCS: GCSConfig{
					Bucket:            "kubevirt-prow",
					PathStrategy:      "explicit",
					CredentialsSecret: "gcs",
				},
			},
			Repos: map[string]JobConfig{
				"kubevirt/project-infra": {
					TestPackages: "./external-plugins/... ./releng/... ./robots/... ./cmd/... ./pkg/...",
				},
			},
		}
		handler = &GitHubEventsHandler{config: cfg}

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

		jobCfg, _ := cfg.RepoConfig("kubevirt/project-infra")
		job = handler.generateCoverageJob(pr, "test-event-123", jobCfg)
	})

	DescribeTable("Should set the correct configuration",
		func(getField func(prowapi.ProwJob) string, expectedValue string) {
			Expect(getField(job)).To(Equal(expectedValue))
		},
		Entry("job name", func(j prowapi.ProwJob) string { return j.Spec.Job }, "coverage-auto"),
		Entry("agent", func(j prowapi.ProwJob) string { return string(j.Spec.Agent) }, "kubernetes"),
		Entry("cluster", func(j prowapi.ProwJob) string { return j.Spec.Cluster }, "kubevirt-prow-control-plane"),
		Entry("coverage-plugin label", func(j prowapi.ProwJob) string { return j.Labels["coverage-plugin"] }, "true"),
		Entry("container image", func(j prowapi.ProwJob) string { return j.Spec.PodSpec.Containers[0].Image }, "quay.io/kubevirtci/covreport:latest"),
		Entry("GCS bucket", func(j prowapi.ProwJob) string { return j.Spec.DecorationConfig.GCSConfiguration.Bucket }, "kubevirt-prow"),
		Entry("GCS path strategy", func(j prowapi.ProwJob) string { return string(j.Spec.DecorationConfig.GCSConfiguration.PathStrategy) }, string(prowapi.PathStrategyExplicit)),
		Entry("GCS credentials secret", func(j prowapi.ProwJob) string { return *j.Spec.DecorationConfig.GCSCredentialsSecret }, "gcs"),
		Entry("GO_MOD_PATH env", func(j prowapi.ProwJob) string {
			for _, env := range j.Spec.PodSpec.Containers[0].Env {
				if env.Name == "GO_MOD_PATH" {
					return env.Value
				}
			}
			return ""
		}, "go.mod"),
		Entry("GOTOOLCHAIN env", func(j prowapi.ProwJob) string {
			for _, env := range j.Spec.PodSpec.Containers[0].Env {
				if env.Name == "GOTOOLCHAIN" {
					return env.Value
				}
			}
			return ""
		}, "local"),
	)

	DescribeTable("Should include specific coverage targets",
		func(target string) {
			args := job.Spec.PodSpec.Containers[0].Args
			Expect(args).To(HaveLen(1))
			Expect(args[0]).To(ContainSubstring(target))
		},
		Entry("external-plugins", "./external-plugins/..."),
		Entry("releng", "./releng/..."),
		Entry("robots", "./robots/..."),
		Entry("cmd", "./cmd/..."),
		Entry("pkg", "./pkg/..."),
	)

	DescribeTable("Should not include paths with e2e tests",
		func(target string) {
			args := job.Spec.PodSpec.Containers[0].Args
			Expect(args[0]).NotTo(ContainSubstring(target))
		},
		Entry("should not run all tests", "go test ./..."),
		Entry("should not include github/ci/services e2e tests", "./github/ci/services/..."),
	)

	DescribeTable("Should set decoration utility images",
		func(getField func(prowapi.ProwJob) string) {
			Expect(getField(job)).NotTo(BeEmpty())
		},
		Entry("clonerefs", func(j prowapi.ProwJob) string { return j.Spec.DecorationConfig.UtilityImages.CloneRefs }),
		Entry("initupload", func(j prowapi.ProwJob) string { return j.Spec.DecorationConfig.UtilityImages.InitUpload }),
		Entry("entrypoint", func(j prowapi.ProwJob) string { return j.Spec.DecorationConfig.UtilityImages.Entrypoint }),
		Entry("sidecar", func(j prowapi.ProwJob) string { return j.Spec.DecorationConfig.UtilityImages.Sidecar }),
	)

	It("Should output coverage artifacts matching Spyglass config", func() {
		args := job.Spec.PodSpec.Containers[0].Args
		Expect(args).To(HaveLen(1))
		Expect(args[0]).To(ContainSubstring("-coverprofile=${ARTIFACTS}/filtered.cov"))
		Expect(args[0]).To(ContainSubstring("-o ${ARTIFACTS}/filtered.html"))
	})
})

var _ = Describe("Handle", func() {
	var (
		handler          *GitHubEventsHandler
		fakeGithubClient *fakegithub.FakeClient
		fakeProwClient   *fake.FakeProwV1
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.SetOutput(io.Discard) //Discard logs for testing

		fakeGithubClient = fakegithub.NewFakeClient() //Create fake github client

		//Set fake prow client
		fakeProwClient = &fake.FakeProwV1{
			Fake: &testing.Fake{},
		}

		//Create handler
		handler = &GitHubEventsHandler{
			logger:        logger,
			githubClient:  fakeGithubClient,
			prowJobClient: fakeProwClient.ProwJobs("test-namespace"),
			config: &Config{
				Defaults: JobConfig{
					Namespace: "test-namespace",
					Image:     "test-image:latest",
					Cluster:   "test-cluster",
					GCS:       GCSConfig{Bucket: "test-bucket", CredentialsSecret: "gcs"},
					UtilityImages: UtilityImagesConfig{
						CloneRefs:  "clonerefs:latest",
						InitUpload: "initupload:latest",
						Entrypoint: "entrypoint:latest",
						Sidecar:    "sidecar:latest",
					},
				},
				Repos: map[string]JobConfig{
					"kubevirt/project-infra": {
						TestPackages: "./...",
					},
				},
			},
			dryrun: false,
		}
	})

	Context("When a PR is opened with Go file changes", func() {
		It("Should create a coverage ProwJob", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-1",
				Payload: createPREventPayload(github.PullRequestActionOpened, 100, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[100] = []github.PullRequestChange{
				{Filename: "external-plugins/coverage/plugin/handler/handler.go"},
				{Filename: "pkg/git/blame_test.go"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(HaveLen(1))
			Expect(fakeProwClient.Actions()[0].GetVerb()).To(Equal("create"))

			createAction := fakeProwClient.Actions()[0].(testing.CreateAction)
			prowJob := createAction.GetObject().(*prowapi.ProwJob)
			Expect(prowJob.Spec.Job).To(Equal("coverage-auto"))

		})

	})

	Context("When a PR is synchronized with Go file changes", func() {
		It("Should create a coverage ProwJob", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-2",
				Payload: createPREventPayload(github.PullRequestActionSynchronize, 101, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[101] = []github.PullRequestChange{
				{Filename: "releng/release-tool/release-tool.go"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(HaveLen(1))
			Expect(fakeProwClient.Actions()[0].GetVerb()).To(Equal("create"))

			createAction := fakeProwClient.Actions()[0].(testing.CreateAction)
			prowJob := createAction.GetObject().(*prowapi.ProwJob)
			Expect(prowJob.Spec.Job).To(Equal("coverage-auto"))
		})

	})

	Context("When a PR has mixed go and non go file changes", func() {
		It("Should create a coverage ProwJob", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-3",
				Payload: createPREventPayload(github.PullRequestActionOpened, 102, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[102] = []github.PullRequestChange{
				{Filename: "external-plugins/coverage/plugin/handler/handler.go"},
				{Filename: "releng/release-tool/release-tool.go"},
				{Filename: "README.md"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(HaveLen(1))
			Expect(fakeProwClient.Actions()[0].GetVerb()).To(Equal("create"))
		})
	})

	Context("When the PR is from an unconfigured repo", func() {
		It("Should not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-unknown-repo",
				Payload: createPREventPayload(github.PullRequestActionOpened, 600, "kubevirt", "unknown-repo"),
			}
			fakeGithubClient.PullRequestChanges[600] = []github.PullRequestChange{
				{Filename: "main.go"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the PR action is not opened or synchronize", func() {
		DescribeTable("Should not create a job",
			func(action github.PullRequestEventAction, prNumber int) {
				event := &GitHubEvent{
					Type:    "pull_request",
					GUID:    "event-guid-4",
					Payload: createPREventPayload(action, prNumber, "kubevirt", "project-infra"),
				}
				fakeGithubClient.PullRequestChanges[prNumber] = []github.PullRequestChange{
					{Filename: "pkg/test.go"},
				}
				handler.Handle(event)

				Expect(fakeProwClient.Actions()).To(BeEmpty())
			},
			Entry("PR closed", github.PullRequestActionClosed, 200),
			Entry("PR labeled", github.PullRequestActionLabeled, 201),
			Entry("PR edited", github.PullRequestActionEdited, 202),
		)
	})

	Context("When the event type is not a pull_request", func() {
		It("Should not create a job", func() {
			event := &GitHubEvent{
				Type:    "push",
				GUID:    "event-guid-push",
				Payload: []byte("{}"),
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the PR has no Go file changes", func() {
		It("Should not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-5",
				Payload: createPREventPayload(github.PullRequestActionOpened, 300, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[300] = []github.PullRequestChange{
				{Filename: "README.md"},
				{Filename: "docs/guide.txt"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the PR has no file changes", func() {
		It("Should not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-7",
				Payload: createPREventPayload(github.PullRequestActionOpened, 302, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[302] = []github.PullRequestChange{}

			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the payload is invalid JSON", func() {
		It("Should not panic and not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-bad",
				Payload: []byte("invalid json{{{"),
			}

			Expect(func() { handler.Handle(event) }).NotTo(Panic())
			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the payload is empty", func() {
		It("Should not panic and not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-empty",
				Payload: []byte{},
			}

			Expect(func() { handler.Handle(event) }).NotTo(Panic())
			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the GitHub API fails to get PR changes", func() {
		It("Should not panic and not create a job", func() {
			handler.githubClient = &errorGithubClient{}

			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-github-error",
				Payload: createPREventPayload(github.PullRequestActionOpened, 500, "kubevirt", "project-infra"),
			}

			Expect(func() { handler.Handle(event) }).NotTo(Panic())
			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When ProwJob creation fails", func() {
		It("Should not panic", func() {
			fakeProwClient.Fake.PrependReactor("create", "prowjobs", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("api server unavailable")
			})

			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-prowjob-error",
				Payload: createPREventPayload(github.PullRequestActionOpened, 501, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[501] = []github.PullRequestChange{
				{Filename: "handler.go"},
			}

			Expect(func() { handler.Handle(event) }).NotTo(Panic())
		})
	})

	Context("When dry-run mode is enabled", func() {
		BeforeEach(func() {
			handler.dryrun = true
		})

		It("Should not create a ProwJob even with valid Go file changes", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-dryrun",
				Payload: createPREventPayload(github.PullRequestActionOpened, 400, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[400] = []github.PullRequestChange{
				{Filename: "external-plugins/coverage/plugin/handler/handler.go"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})
})
