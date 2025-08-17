package main_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/testing"

	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1/fake"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/git/localgit"
	git2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"

	"kubevirt.io/project-infra/external-plugins/test-subset/plugin/handler"
)

const (
	org      = "kubevirt"
	repo     = "kubevirt"
	baseRef  = "main"
	orgRepo  = org + "/" + repo
	prNumber = 17
	testuser = "testuser"
)

var _ = Describe("Test-subset", func() {
	Context("A valid pull request comment event", func() {
		var gitrepo *localgit.LocalGit
		var gitClientFactory git2.ClientFactory
		var eventsHandler *handler.GitHubEventsHandler
		var prowc *fake.FakeProwV1

		BeforeEach(func() {
			var err error
			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred())

			// Create a fake git repo
			makeRepoWithEmptyProwConfig(gitrepo, org, repo)

			// Generate a base commit with jobs
			baseConfig, err := json.Marshal(&config.Config{
				JobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						orgRepo: {
							{
								JobBase: config.JobBase{
									Name: "job1",
									Spec: &v1.PodSpec{
										Containers: []v1.Container{
											{
												Image: "image1",
											},
										},
									},
								},
							},
							{
								JobBase: config.JobBase{
									Name: "job2",
									Spec: &v1.PodSpec{
										Containers: []v1.Container{
											{
												Image: "image2",
											},
										},
									},
								},
							},
						},
					},
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
			err = gitrepo.AddCommit(org, repo, map[string][]byte{
				"jobs-config.yaml": baseConfig,
			})
			Expect(err).ShouldNot(HaveOccurred())
			baseref, err := gitrepo.RevParse(org, repo, "HEAD")
			Expect(err).ShouldNot(HaveOccurred())

			// Setup fake GitHub client
			gh := fakegithub.NewFakeClient()
			gh.OrgMembers = map[string][]string{
				repo: {testuser},
			}

			// Setup PR
			gh.PullRequests = map[int]*github.PullRequest{
				prNumber: {
					Number: prNumber,
					State:  "open",
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Name:     repo,
							FullName: orgRepo,
						},
						Ref: baseRef,
						SHA: baseref,
					},
					Head: github.PullRequestBranch{
						Repo: github.Repo{
							Name:     repo,
							FullName: orgRepo,
						},
						Ref: baseRef,
						SHA: baseref,
					},
				},
			}

			// Setup events handler
			fakelog := logrus.New()
			eventsChan := make(chan *handler.GitHubEvent)
			prowc = &fake.FakeProwV1{
				Fake: &testing.Fake{},
			}
			eventsHandler = handler.NewGitHubEventsHandler(
				eventsChan,
				fakelog,
				prowc.ProwJobs("test-ns"),
				gh,
				"prowconfig.yaml",
				"jobs-config.yaml",
				"",
				gitClientFactory)
			eventsHandler.SetLocalConfLoad()
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				gitClientFactory.Clean()
			}
		})

		Context("a member comments a test-subset command", func() {
			It("Should set KUBEVIRT_LABEL_FILTER for --filter parameter", func() {
				handleTestSubsetCommand(eventsHandler, `/test-subset job1 --filter "USB"`)
				validateJobEnvironmentVars(prowc, map[string]string{
					"KUBEVIRT_LABEL_FILTER": "(USB)",
				})
			})

			It("Should set KUBEVIRT_LABEL_FOCUS for --focus parameter", func() {
				handleTestSubsetCommand(eventsHandler, `/test-subset job1 --focus "FocusString"`)
				validateJobEnvironmentVars(prowc, map[string]string{
					"KUBEVIRT_LABEL_FOCUS": "FocusString",
				})
			})

			It("Should set KUBEVIRT_VERBOSITY for --verbosity parameter", func() {
				handleTestSubsetCommand(eventsHandler, `/test-subset job1 --verbosity=virtLauncher:3,virtHandler:3`)
				validateJobEnvironmentVars(prowc, map[string]string{
					"KUBEVIRT_VERBOSITY": "virtLauncher:3,virtHandler:3",
				})
			})

			It("Should set multiple environment variables when multiple parameters provided", func() {
				handleTestSubsetCommand(eventsHandler, `/test-subset job1 --filter "(USB)" --focus "FocusString" --verbosity=virtLauncher:2`)
				validateJobEnvironmentVars(prowc, map[string]string{
					"KUBEVIRT_LABEL_FILTER": "(USB)",
					"KUBEVIRT_LABEL_FOCUS":  "FocusString",
					"KUBEVIRT_VERBOSITY":    "virtLauncher:2",
				})
			})

			It("Should auto-add parentheses for filter values without them", func() {
				handleTestSubsetCommand(eventsHandler, `/test-subset job1 --filter=USB`)
				validateJobEnvironmentVars(prowc, map[string]string{
					"KUBEVIRT_LABEL_FILTER": "(USB)",
				})
			})

			It("Should handle mixed quotes and parameter formats", func() {
				handleTestSubsetCommand(eventsHandler, `/test-subset job1 --filter='(USB)' --verbosity="virtLauncher:2" --focus "Focus String"`)
				validateJobEnvironmentVars(prowc, map[string]string{
					"KUBEVIRT_LABEL_FILTER": "(USB)",
					"KUBEVIRT_LABEL_FOCUS":  "Focus String",
					"KUBEVIRT_VERBOSITY":    "virtLauncher:2",
				})
			})

			It("Should handle values with spaces in quotes", func() {
				handleTestSubsetCommand(eventsHandler, `/test-subset job1 --filter="(Label With Spaces)" --verbosity="virtLauncher:3,virtHandler:2"`)
				validateJobEnvironmentVars(prowc, map[string]string{
					"KUBEVIRT_LABEL_FILTER": "(Label With Spaces)",
					"KUBEVIRT_VERBOSITY":    "virtLauncher:3,virtHandler:2",
				})
			})

			It("Should handle commands with leading and trailing whitespace", func() {
				handleTestSubsetCommand(eventsHandler, `   /test-subset job1 --filter=USB --verbosity=virtLauncher:3   `)
				validateJobEnvironmentVars(prowc, map[string]string{
					"KUBEVIRT_LABEL_FILTER": "(USB)",
					"KUBEVIRT_VERBOSITY":    "virtLauncher:3",
				})
			})
		})
	})
})

// Helper function to create and handle test-subset commands
func handleTestSubsetCommand(eventsHandler *handler.GitHubEventsHandler, commandBody string) {
	event := &github.IssueCommentEvent{
		Action: github.IssueCommentActionCreated,
		Comment: github.IssueComment{
			Body: commandBody,
			User: github.User{
				Login: testuser,
			},
		},
		GUID: "guid",
		Repo: github.Repo{
			FullName: orgRepo,
		},
		Issue: github.Issue{
			Number: prNumber,
			State:  "open",
			User: github.User{
				Login: testuser,
			},
			PullRequest: &struct{}{},
		},
	}

	handlerEvent, err := makeHandlerIssueCommentEvent(event)
	Expect(err).ShouldNot(HaveOccurred())

	eventsHandler.Handle(handlerEvent)
}

// Helper function to validate environment variables in the created job
func validateJobEnvironmentVars(prowc *fake.FakeProwV1, expectedEnvVars map[string]string) {
	Expect(prowc.Actions()).Should(HaveLen(1))

	createAction := prowc.Actions()[0]
	Expect(createAction.GetVerb()).To(Equal("create"))

	// Get the created object using reflection on the fake client
	obj := createAction.(testing.CreateAction).GetObject()
	prowJob, ok := obj.(*prowapi.ProwJob)
	Expect(ok).To(BeTrue())

	envVars := prowJob.Spec.PodSpec.Containers[0].Env

	// Check for each expected environment variable
	for expectedName, expectedValue := range expectedEnvVars {
		found := false
		for _, env := range envVars {
			if env.Name == expectedName {
				Expect(env.Value).To(Equal(expectedValue), fmt.Sprintf("Environment variable %s should have value %s", expectedName, expectedValue))
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), fmt.Sprintf("Environment variable %s should be set", expectedName))
	}

	// Ensure no unexpected environment variables are set
	expectedNames := []string{"KUBEVIRT_LABEL_FILTER", "KUBEVIRT_LABEL_FOCUS", "KUBEVIRT_VERBOSITY"}
	for _, env := range envVars {
		for _, expectedName := range expectedNames {
			if env.Name == expectedName {
				_, shouldBeSet := expectedEnvVars[expectedName]
				Expect(shouldBeSet).To(BeTrue(), fmt.Sprintf("Unexpected environment variable %s was set", expectedName))
			}
		}
	}
}

func makeRepoWithEmptyProwConfig(lg *localgit.LocalGit, org, repo string) error {
	err := lg.MakeFakeRepo(org, repo)
	if err != nil {
		return err
	}
	prowConfig, err := json.Marshal(&config.ProwConfig{})
	if err != nil {
		return err
	}
	return lg.AddCommit(org, repo, map[string][]byte{
		"prowconfig.yaml": prowConfig,
	})
}

func makeHandlerIssueCommentEvent(event *github.IssueCommentEvent) (*handler.GitHubEvent, error) {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	handlerEvent := &handler.GitHubEvent{
		Type:    "issue_comment",
		GUID:    event.GUID,
		Payload: eventBytes,
	}
	return handlerEvent, nil
}
