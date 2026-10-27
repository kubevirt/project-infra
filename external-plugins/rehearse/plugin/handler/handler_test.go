package handler

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"kubevirt.io/project-infra/pkg/testutils"
	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/git/localgit"
	gitv2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"
	"sigs.k8s.io/prow/pkg/kube"
)

var _ = Describe("Events", func() {

	Context("With a git repo", func() {
		var gitrepo *localgit.LocalGit
		var gitClientFactory gitv2.ClientFactory
		var eventsServer *GitHubEventsHandler
		var dummyLog *logrus.Logger

		BeforeEach(func() {

			var err error
			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred(), "Could not create local git repo and client factory")
			dummyLog = logrus.New()
			foc := &testutils.FakeOwnersClient{
				ExistingTopLevelApprovers: sets.New[string]("testuser"),
			}
			froc := &testutils.FakeRepoownersClient{
				Foc: foc,
			}
			eventsServer = NewGitHubEventsHandler(nil, dummyLog, nil, nil, "prow-config.yaml", "", true, gitClientFactory, froc)
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				_ = gitClientFactory.Clean()
			}
		})

		It("Should load jobs from git refspec", func() {
			prowConfig := config.ProwConfig{}
			jobsConfig := config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					"foo/bar": {
						{
							JobBase: config.JobBase{
								Name: "a-presubmit",
								Spec: &v1.PodSpec{
									Containers: []v1.Container{
										{
											Image:   "foo/var",
											Command: []string{"/bin/foo"},
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(gitrepo.MakeFakeRepo("foo", "bar")).Should(Succeed())
			prowConfigBytes, err := json.Marshal(prowConfig)
			Expect(err).ShouldNot(HaveOccurred())
			jobsConfigBytes, err := json.Marshal(jobsConfig)
			Expect(err).ShouldNot(HaveOccurred())
			files := map[string][]byte{
				"prow-config.yaml": prowConfigBytes,
				"jobs-config.yaml": jobsConfigBytes,
			}
			Expect(gitrepo.AddCommit("foo", "bar", files)).Should(Succeed())
			headref, err := gitrepo.RevParse("foo", "bar", "HEAD")
			Expect(err).ShouldNot(HaveOccurred())
			gitClient, err := gitClientFactory.ClientFor("foo", "bar")
			Expect(err).ShouldNot(HaveOccurred())
			out, err := eventsServer.loadConfigsAtRef([]string{"jobs-config.yaml"}, gitClient, headref)
			Expect(err).ShouldNot(HaveOccurred())
			outConfig, exists := out["jobs-config.yaml"]
			Expect(exists).To(BeTrue())
			outJobs, exists := outConfig.PresubmitsStatic["foo/bar"]
			Expect(exists).To(BeTrue())
			Expect(outJobs[0].Name).To(Equal(jobsConfig.PresubmitsStatic["foo/bar"][0].Name))
		})

		It("Should reset SourcePath for postsubmits and periodics", func() {
			prowConfig := config.ProwConfig{}
			jobsConfig := config.JobConfig{
				PostsubmitsStatic: map[string][]config.Postsubmit{
					"foo/bar": {
						{
							JobBase: config.JobBase{
								Name: "a-postsubmit",
								Spec: &v1.PodSpec{
									Containers: []v1.Container{
										{
											Image:   "foo/var",
											Command: []string{"/bin/foo"},
										},
									},
								},
							},
						},
					},
				},
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Name: "a-periodic",
							Spec: &v1.PodSpec{
								Containers: []v1.Container{
									{
										Image:   "foo/var",
										Command: []string{"/bin/foo"},
									},
								},
							},
						},
						Cron: "0 0 * * *",
					},
				},
			}

			Expect(gitrepo.MakeFakeRepo("foo", "bar")).Should(Succeed())
			prowConfigBytes, err := json.Marshal(prowConfig)
			Expect(err).ShouldNot(HaveOccurred())
			jobsConfigBytes, err := json.Marshal(jobsConfig)
			Expect(err).ShouldNot(HaveOccurred())
			files := map[string][]byte{
				"prow-config.yaml": prowConfigBytes,
				"jobs-config.yaml": jobsConfigBytes,
			}
			Expect(gitrepo.AddCommit("foo", "bar", files)).Should(Succeed())
			headref, err := gitrepo.RevParse("foo", "bar", "HEAD")
			Expect(err).ShouldNot(HaveOccurred())
			gitClient, err := gitClientFactory.ClientFor("foo", "bar")
			Expect(err).ShouldNot(HaveOccurred())
			out, err := eventsServer.loadConfigsAtRef([]string{"jobs-config.yaml"}, gitClient, headref)
			Expect(err).ShouldNot(HaveOccurred())
			outConfig, exists := out["jobs-config.yaml"]
			Expect(exists).To(BeTrue())
			outPostsubmits, exists := outConfig.PostsubmitsStatic["foo/bar"]
			Expect(exists).To(BeTrue())
			Expect(outPostsubmits[0].JobBase.SourcePath).To(HaveSuffix("jobs-config.yaml"))
			Expect(outConfig.Periodics[0].JobBase.SourcePath).To(HaveSuffix("jobs-config.yaml"))
		})

	})

	Context("Utility functions", func() {

		It("Should return correct repo from job key", func() {
			ret := repoFromJobKey("foo/bar#baz-something/something-else")
			Expect(ret).To(Equal("foo/bar"))
		})

		DescribeTable(
			"Should calculate extra refs",
			func(refs []prowapi.Refs, expected prowapi.Refs) {
				ret := makeTargetRepoRefs(refs, "foo", "bar", "baz")
				Expect(ret).To(Equal(expected))
				Expect(refs).ToNot(Equal(expected), "Input refs should not be modified")
			},
			Entry(
				"Refs exists and there is no workdir defined",
				[]prowapi.Refs{
					{
						WorkDir: false,
					},
				},
				prowapi.Refs{
					Org:     "foo",
					Repo:    "bar",
					WorkDir: true,
					BaseRef: "baz",
				},
			),
			Entry(
				"Refs is nil",
				nil,
				prowapi.Refs{
					Org:     "foo",
					Repo:    "bar",
					WorkDir: true,
					BaseRef: "baz",
				},
			),
		)

		DescribeTable(
			"Should calculate if a workdir is already defined",
			func(refs []prowapi.Refs, expected bool) {
				Expect(workdirAlreadyDefined(refs)).To(Equal(expected))
			},
			Entry(
				"When workdir is already defined",
				[]prowapi.Refs{
					{
						WorkDir: false,
					},
					{
						WorkDir: true,
					},
				},
				true),
			Entry(
				"When workdir is not defined",
				[]prowapi.Refs{
					{
						WorkDir: false,
					},
					{
						WorkDir: false,
					},
				},
				false),
		)

		It("Should discover HEAD branch name from remote", func() {
			headBranchName, err := discoverHeadBranchName("kubevirt", "kubevirt", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(headBranchName).To(Equal("main"))
		})

		It("Should discover HEAD branch name from cloneURI", func() {
			headBranchName, err := discoverHeadBranchName("foo", "bar", "https://github.com/nmstate/nmstate")
			Expect(err).ToNot(HaveOccurred())
			Expect(headBranchName).To(Equal("base"))
		})

	})

})

var _ = Describe("PR filtering", func() {

	Context("Handler filtering jobs", func() {

		var handler *GitHubEventsHandler
		var headConfig *config.Config
		var headConfigPresubmit config.Presubmit
		var baseConfig *config.Config
		var baseConfigPresubmit config.Presubmit
		var pr *github.PullRequest

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			headConfigPresubmit = config.Presubmit{
				JobBase: config.JobBase{
					Name: "testJob",
					Spec: newPodSpec(),
				},
				AlwaysRun:           false,
				Optional:            false,
				Trigger:             "",
				RerunCommand:        "",
				Brancher:            config.Brancher{},
				RegexpChangeMatcher: config.RegexpChangeMatcher{},
				Reporter:            config.Reporter{},
				JenkinsSpec:         nil,
			}
			headConfig = &config.Config{
				JobConfig: config.JobConfig{
					Presets: nil,
					PresubmitsStatic: map[string][]config.Presubmit{
						"kubevirt/kubevirt": {
							headConfigPresubmit,
						},
					},
					PostsubmitsStatic: nil,
					Periodics:         nil,
					AllRepos:          nil,
					ProwYAMLGetter:    nil,
					DecorateAllJobs:   false,
				},
			}
			baseConfigPresubmit = config.Presubmit{
				JobBase: config.JobBase{
					Name: "testJob",
					Spec: newPodSpec(),
				},
				AlwaysRun:           false,
				Optional:            false,
				Trigger:             "",
				RerunCommand:        "",
				Brancher:            config.Brancher{},
				RegexpChangeMatcher: config.RegexpChangeMatcher{},
				Reporter:            config.Reporter{},
				JenkinsSpec:         nil,
			}
			baseConfig = &config.Config{
				JobConfig: config.JobConfig{
					Presets: nil,
					PresubmitsStatic: map[string][]config.Presubmit{
						"kubevirt/kubevirt": {
							baseConfigPresubmit,
						},
					},
					PostsubmitsStatic: nil,
					Periodics:         nil,
					AllRepos:          nil,
					ProwYAMLGetter:    nil,
					DecorateAllJobs:   false,
				},
			}
			pr = &github.PullRequest{
				Number: 17,
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner: github.User{
							Login: "kubevirt",
						},
						Name: "project-infra",
					},
				},
			}
		})

		It("doesn't generate a prowjob without changes", func() {
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42")
			Expect(presubmits).To(BeEmpty())
		})

		It("generates a prowjob if spec changes", func() {
			headConfigPresubmit.Spec.Containers[0].Image = "v2/test37"
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42")
			Expect(presubmits).ToNot(BeEmpty())
		})

		It("generates a prowjob if context changes", func() {
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Cluster = "new-cluster"
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42")
			Expect(presubmits).ToNot(BeEmpty())
		})

		It("generates a prowjob for branch if context changes", func() {
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Cluster = "new-cluster"
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Branches = []string{"release-42"}
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42")
			Expect(presubmits).ToNot(BeEmpty())
			Expect(presubmits[0].Spec.ExtraRefs[0].BaseRef).To(BeEquivalentTo("release-42"))
		})

		It("strips extra-refs matching the PR repo", func() {
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Cluster = "new-cluster"
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].ExtraRefs = []prowapi.Refs{
				{Org: "kubevirt", Repo: "project-infra", BaseRef: "main"},
			}
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42")
			Expect(presubmits).ToNot(BeEmpty())
			for _, ref := range presubmits[0].Spec.ExtraRefs {
				Expect(ref.Org + "/" + ref.Repo).ToNot(Equal("kubevirt/project-infra"))
			}
		})
	})

	Context("Handler filtering postsubmit jobs", func() {

		var handler *GitHubEventsHandler
		var headConfig *config.Config
		var baseConfig *config.Config
		var pr *github.PullRequest

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			headConfig = &config.Config{
				JobConfig: config.JobConfig{
					PostsubmitsStatic: map[string][]config.Postsubmit{
						"kubevirt/kubevirt": {
							{
								JobBase: config.JobBase{
									Name: "testPostsubmitJob",
									Spec: newPodSpec(),
								},
							},
						},
					},
				},
			}
			baseConfig = &config.Config{
				JobConfig: config.JobConfig{
					PostsubmitsStatic: map[string][]config.Postsubmit{
						"kubevirt/kubevirt": {
							{
								JobBase: config.JobBase{
									Name: "testPostsubmitJob",
									Spec: newPodSpec(),
								},
							},
						},
					},
				},
			}
			pr = &github.PullRequest{
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner: github.User{
							Login: "kubevirt",
						},
						Name: "project-infra",
					},
				},
			}
		})

		It("doesn't generate a prowjob without changes", func() {
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42")
			Expect(postsubmits).To(BeEmpty())
		})

		It("generates a prowjob if spec changes", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42")
			Expect(postsubmits).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.Type).To(Equal(prowapi.PostsubmitJob))
		})

		It("generates a prowjob with extra refs for cross-repo job", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42")
			Expect(postsubmits).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs[0].Org).To(Equal("kubevirt"))
			Expect(postsubmits[0].Spec.ExtraRefs[0].Repo).To(Equal("kubevirt"))
		})

		It("generates a prowjob for branch if spec changes", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Branches = []string{"release-42"}
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42")
			Expect(postsubmits).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs[0].BaseRef).To(Equal("release-42"))
		})

		It("unwraps a ^branch$ pattern for extra refs", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Branches = []string{"^main$"}
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42")
			Expect(postsubmits).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs[0].BaseRef).To(Equal("main"))
		})

		It("creates a single rehearsal for a same-repo postsubmit with multiple branches", func() {
			job := config.Postsubmit{
				JobBase: config.JobBase{
					Name: "testPostsubmitJob",
					Spec: newPodSpec(),
				},
			}
			headConfig.PostsubmitsStatic = map[string][]config.Postsubmit{
				"kubevirt/project-infra": {job},
			}
			baseConfig.PostsubmitsStatic = map[string][]config.Postsubmit{
				"kubevirt/project-infra": {job},
			}
			headConfig.PostsubmitsStatic["kubevirt/project-infra"][0].Spec.Containers[0].Image = "v2/test37"
			headConfig.PostsubmitsStatic["kubevirt/project-infra"][0].Branches = []string{"main", "release-1.6"}
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42")
			Expect(postsubmits).To(HaveLen(1))
		})

		It("skips a restricted postsubmit", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Annotations = map[string]string{
				rehearsalRestrictedAnnotation: "true",
			}
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42")
			Expect(postsubmits).To(BeEmpty())
		})

		It("strips extra-refs matching the PR repo", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].ExtraRefs = []prowapi.Refs{
				{Org: "kubevirt", Repo: "project-infra", BaseRef: "main"},
			}
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42")
			Expect(postsubmits).ToNot(BeEmpty())
			for _, ref := range postsubmits[0].Spec.ExtraRefs {
				Expect(ref.Org + "/" + ref.Repo).ToNot(Equal("kubevirt/project-infra"))
			}
		})
	})

	Context("Handler filtering periodic jobs", func() {

		var handler *GitHubEventsHandler
		var headConfig *config.Config
		var baseConfig *config.Config
		var pr *github.PullRequest

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			headConfig = &config.Config{
				JobConfig: config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Name: "testPeriodicJob",
								Spec: newPodSpec(),
							},
							Cron: "0 0 * * *",
						},
					},
				},
			}
			baseConfig = &config.Config{
				JobConfig: config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Name: "testPeriodicJob",
								Spec: newPodSpec(),
							},
							Cron: "0 0 * * *",
						},
					},
				},
			}
			pr = &github.PullRequest{
				Number: 17,
				User:   github.User{Login: "testauthor"},
				Base: github.PullRequestBranch{
					Ref: "main",
					SHA: "basesha",
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner: github.User{
							Login: "kubevirt",
						},
						Name: "project-infra",
					},
				},
				Head: github.PullRequestBranch{
					Ref: "feature",
					SHA: "headsha",
				},
			}
		})

		It("doesn't generate a prowjob without changes", func() {
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42")
			Expect(periodics).To(BeEmpty())
		})

		It("generates a periodic job type before conversion", func() {
			headConfig.Periodics[0].Spec.Containers[0].Image = "v2/test37"
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42")
			Expect(periodics).ToNot(BeEmpty())
			Expect(periodics[0].Spec.Type).To(Equal(prowapi.PeriodicJob))
		})

		It("strips extra-refs matching the PR repo", func() {
			headConfig.Periodics[0].Spec.Containers[0].Image = "v2/test37"
			headConfig.Periodics[0].ExtraRefs = []prowapi.Refs{
				{Org: "kubevirt", Repo: "project-infra", BaseRef: "main"},
			}
			baseConfig.Periodics[0].ExtraRefs = []prowapi.Refs{
				{Org: "kubevirt", Repo: "project-infra", BaseRef: "main"},
			}
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42")
			Expect(periodics).ToNot(BeEmpty())
			for _, ref := range periodics[0].Spec.ExtraRefs {
				Expect(ref.Org + "/" + ref.Repo).ToNot(Equal("kubevirt/project-infra"))
			}
		})

		It("converts to presubmit with PR refs after conversion", func() {
			headConfig.Periodics[0].Spec.Containers[0].Image = "v2/test37"
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42")
			Expect(periodics).ToNot(BeEmpty())
			converted := convertPeriodicsToPresubmits(periodics, pr)
			Expect(converted[0].Spec.Type).To(Equal(prowapi.PresubmitJob))
			Expect(converted[0].Spec.Report).To(BeTrue())
			Expect(converted[0].Spec.Refs).ToNot(BeNil())
			Expect(converted[0].Spec.Refs.Org).To(Equal("kubevirt"))
			Expect(converted[0].Spec.Refs.Repo).To(Equal("project-infra"))
			Expect(converted[0].Spec.Refs.Pulls).To(HaveLen(1))
			Expect(converted[0].Labels[kube.ProwJobTypeLabel]).To(Equal(string(prowapi.PresubmitJob)))
			Expect(converted[0].Labels[kube.OrgLabel]).To(Equal("kubevirt"))
			Expect(converted[0].Labels[kube.RepoLabel]).To(Equal("project-infra"))
			Expect(converted[0].Labels[kube.BaseRefLabel]).To(Equal("main"))
			Expect(converted[0].Labels[kube.PullLabel]).To(Equal("17"))
		})

		It("skips a restricted periodic", func() {
			headConfig.Periodics[0].Spec.Containers[0].Image = "v2/test37"
			headConfig.Periodics[0].Annotations = map[string]string{
				rehearsalRestrictedAnnotation: "true",
			}
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42")
			Expect(periodics).To(BeEmpty())
		})

		It("generates a prowjob for new periodic", func() {
			headConfig.Periodics = append(headConfig.Periodics, config.Periodic{
				JobBase: config.JobBase{
					Name: "newPeriodicJob",
					Spec: newPodSpec(),
				},
				Cron: "0 0 * * *",
			})
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42")
			Expect(periodics).To(HaveLen(1))
			Expect(periodics[0].Spec.Job).To(Equal("newPeriodicJob"))
		})
	})

	Context("extracting job names from PR comments", func() {

		var handler *GitHubEventsHandler

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
		})

		It("extracts job names from comment body", func() {
			commentBody := `/rehearse jobname1 
/rehearse jobname2

Gna meh whatever 

/rehearse jobname3    
`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeEquivalentTo([]string{
				"jobname1",
				"jobname2",
				"jobname3",
			}))
		})

		It("extracts no job names from comment body if all is found", func() {
			commentBody := `Gna meh whatever 

/rehearse all


`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeNil())
		})

		It("extracts no job names from comment body if no element is found since only whitespace after command", func() {
			commentBody := `Gna meh whatever 

/rehearse    


`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeNil())
		})

		It("extracts no job names from comment body", func() {
			commentBody := `Gna meh whatever 

/rehearse


`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeNil())
		})

		It("extracts question mark from comment body", func() {
			commentBody := `Gna meh whatever 

/rehearse ?


`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeEquivalentTo([]string{"?"}))
		})
	})

	Context("filtering jobs by name", func() {

		var handler *GitHubEventsHandler
		var prowJobs []prowapi.ProwJob

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			prowJobs = []prowapi.ProwJob{
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob1",
					},
				},
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob2",
					},
				},
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob3",
					},
				},
			}
		})

		It("filters nothing if slice is nil", func() {
			Expect(handler.filterProwJobsByJobNames(prowJobs, nil)).To(BeEquivalentTo(prowJobs))
		})

		It("filters one job", func() {
			expected := []prowapi.ProwJob{
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob1",
					},
				},
			}
			Expect(handler.filterProwJobsByJobNames(prowJobs, []string{"prowJob1"})).To(BeEquivalentTo(expected))
		})

		It("filters two jobs", func() {
			expected := []prowapi.ProwJob{
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob1",
					},
				},
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob3",
					},
				},
			}
			Expect(handler.filterProwJobsByJobNames(prowJobs, []string{"prowJob1", "prowJob3"})).To(BeEquivalentTo(expected))
		})

	})

	Context("filtering jobs by allowed types", func() {

		var prowJobs []prowapi.ProwJob

		BeforeEach(func() {
			prowJobs = []prowapi.ProwJob{
				{
					Spec: prowapi.ProwJobSpec{
						Job:  "presubmit-job",
						Type: prowapi.PresubmitJob,
					},
				},
				{
					Spec: prowapi.ProwJobSpec{
						Job:  "postsubmit-job",
						Type: prowapi.PostsubmitJob,
					},
				},
				{
					Spec: prowapi.ProwJobSpec{
						Job:  "periodic-job",
						Type: prowapi.PeriodicJob,
					},
				},
			}
		})

		DescribeTable("partitionProwJobsByAllowedTypes",
			func(allowedTypes []prowapi.ProwJobType, expectedAllowed []string, expectedSkipped []string) {
				allowed, skipped := partitionProwJobsByAllowedTypes(prowJobs, allowedTypes)
				var allowedNames []string
				for _, pj := range allowed {
					allowedNames = append(allowedNames, pj.Spec.Job)
				}
				var skippedNames []string
				for _, pj := range skipped {
					skippedNames = append(skippedNames, pj.Spec.Job)
				}
				if expectedAllowed == nil {
					Expect(allowedNames).To(BeNil())
				} else {
					Expect(allowedNames).To(Equal(expectedAllowed))
				}
				if expectedSkipped == nil {
					Expect(skippedNames).To(BeNil())
				} else {
					Expect(skippedNames).To(Equal(expectedSkipped))
				}
			},
			Entry("all types allowed",
				[]prowapi.ProwJobType{prowapi.PresubmitJob, prowapi.PostsubmitJob, prowapi.PeriodicJob},
				[]string{"presubmit-job", "postsubmit-job", "periodic-job"},
				nil,
			),
			Entry("only presubmit allowed",
				[]prowapi.ProwJobType{prowapi.PresubmitJob},
				[]string{"presubmit-job"},
				[]string{"postsubmit-job", "periodic-job"},
			),
			Entry("no types allowed",
				[]prowapi.ProwJobType{},
				nil,
				[]string{"presubmit-job", "postsubmit-job", "periodic-job"},
			),
			Entry("nil allowed types",
				nil,
				nil,
				[]string{"presubmit-job", "postsubmit-job", "periodic-job"},
			),
		)
	})

	Context("rehearsal helpers", func() {
		DescribeTable("tryResolveBranchPattern",
			func(branch, expected string, ok bool) {
				resolved, resolvedOK := tryResolveBranchPattern(branch)
				Expect(resolvedOK).To(Equal(ok))
				Expect(resolved).To(Equal(expected))
			},
			Entry("unwraps ^main$", "^main$", "main", true),
			Entry("keeps a plain branch name", "release-42", "release-42", true),
			Entry("keeps a dotted branch name", "release-1.6", "release-1.6", true),
			Entry("rejects a regex branch", "^release-.*$", "", false),
		)

		DescribeTable("rehearsalBranches",
			func(jobBranches []string, jobRepoOrg, prRepoFullName string, expected []string) {
				Expect(rehearsalBranches(jobBranches, jobRepoOrg, prRepoFullName)).To(Equal(expected))
			},
			Entry("same repo collapses to HEAD", []string{"main", "release-1.6"}, "kubevirt/project-infra", "kubevirt/project-infra", []string{"HEAD"}),
			Entry("empty branches is HEAD", nil, "kubevirt/kubevirt", "kubevirt/project-infra", []string{"HEAD"}),
			Entry("simple names are kept", []string{"main", "release-1.6"}, "kubevirt/kubevirt", "kubevirt/project-infra", []string{"main", "release-1.6"}),
			Entry("unwraps ^main$", []string{"^main$"}, "kubevirt/kubevirt", "kubevirt/project-infra", []string{"main"}),
			Entry("regex falls back to HEAD", []string{"^release-.*$"}, "kubevirt/kubevirt", "kubevirt/project-infra", []string{"HEAD"}),
			Entry("keeps simple names and drops regex", []string{"^main$", "^release-.*$"}, "kubevirt/kubevirt", "kubevirt/project-infra", []string{"main"}),
		)

		It("disables slack reporting on a rehearsal job", func() {
			job := prowapi.ProwJob{}
			disableSlackReporting(&job)
			Expect(job.Spec.ReporterConfig).ToNot(BeNil())
			Expect(job.Spec.ReporterConfig.Slack).ToNot(BeNil())
			Expect(job.Spec.ReporterConfig.Slack.Report).ToNot(BeNil())
			Expect(*job.Spec.ReporterConfig.Slack.Report).To(BeFalse())
		})

		It("lists unauthorized jobs when /rehearse all is used by a leaf approver", func() {
			skipped := []prowapi.ProwJob{
				{Spec: prowapi.ProwJobSpec{Job: "periodic-job"}},
			}
			comment := formatCreatedJobsComment([]string{"rehearsal-presubmit-job"}, skipped)
			Expect(comment).To(ContainSubstring("rehearsal-presubmit-job"))
			Expect(comment).To(ContainSubstring("periodic-job"))
			Expect(comment).To(ContainSubstring("top-level approver"))
		})

		It("explains that periodics require a top-level approver when none can be created", func() {
			skipped := []prowapi.ProwJob{
				{Spec: prowapi.ProwJobSpec{Job: "periodic-job"}},
			}
			comment := formatNoJobsComment(skipped)
			Expect(comment).To(ContainSubstring("periodic-job"))
			Expect(comment).To(ContainSubstring("top-level approver"))
		})
	})

	Context("canUserRehearse", func() {
		var testable *GitHubEventsHandler
		var fakeGHC *fakegithub.FakeClient
		var pr *github.PullRequest
		var fakeOwnersClient *testutils.FakeOwnersClient
		const userName = "testuser"
		const userNameUpperCase = "TestUser"
		BeforeEach(func() {
			fakeGHC = &fakegithub.FakeClient{}
			fakeGHC.OrgMembers = map[string][]string{
				"testorg": {
					"testauthor",
					userName,
				},
			}
			fakeOwnersClient = &testutils.FakeOwnersClient{}
			fakeRepoownersClient := &testutils.FakeRepoownersClient{
				Foc: fakeOwnersClient,
			}
			testable = &GitHubEventsHandler{
				ghClient:     fakeGHC,
				ownersClient: fakeRepoownersClient,
			}
			pr = &github.PullRequest{
				User: github.User{Login: "testauthor"},
			}
		})

		DescribeTable("org and user",
			func(testData CanUserRehearseOrgAndUserTestData) {
				fakeGHC.OrgMembers = testData.OrgMembers
				fakeGHC.IssueLabelsExisting = testData.IssueLabelsExisting
				fakeOwnersClient.ExistingTopLevelApprovers = testData.TopLevelApprovers
				fakeOwnersClient.CurrentLeafApprovers = testData.LeafApprovers

				allowedTypes, message := testable.canUserRehearse("testorg", "testrepo", pr, testData.GetUserNameOrDefault(), testData.ChangedFiles)

				if testData.ExpectedAllowedTypes == nil {
					Expect(allowedTypes).To(BeNil())
				} else {
					Expect(allowedTypes).To(ConsistOf(testData.ExpectedAllowedTypes))
				}
				for _, part := range testData.ExpectedMessageParts {
					Expect(message).To(ContainSubstring(part))
				}
			},
			Entry("only author in org",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
						},
					},
					ChangedFiles:         []string{},
					TopLevelApprovers:    sets.Set[string]{},
					LeafApprovers:        map[string]sets.Set[string]{},
					ExpectedAllowedTypes: nil,
				},
			),
			Entry("user and author in org - user is not an approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles: []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{
						"topLevelApprover": struct{}{},
					},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {
							"leafApprover": struct{}{},
						},
					},
					ExpectedAllowedTypes: nil,
					ExpectedMessageParts: []string{
						"@testuser",
						"you need to be an approver",
						"leafApprover",
						"topLevelApprover",
					},
				},
			),
			Entry("user and author in org - user is not an approver, but ok-to-rehearse label is present",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles: []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{
						"topLevelApprover": struct{}{},
					},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {
							"leafApprover": struct{}{},
						},
					},
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob},
					IssueLabelsExisting:  []string{fmt.Sprintf("testorg/testrepo#0:%s", OKToRehearse)},
				},
			),
			Entry("user and author in org - user is a top level approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:         []string{"changedFile"},
					TopLevelApprovers:    sets.Set[string]{userName: struct{}{}},
					LeafApprovers:        map[string]sets.Set[string]{},
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob, prowapi.PostsubmitJob, prowapi.PeriodicJob},
				},
			),
			Entry("user and author in org - user is a top level approver (case in OWNERS and GitHub don't match)",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:         []string{"changedFile"},
					TopLevelApprovers:    sets.Set[string]{userName: struct{}{}},
					LeafApprovers:        map[string]sets.Set[string]{},
					UserName:             userNameUpperCase,
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob, prowapi.PostsubmitJob, prowapi.PeriodicJob},
				},
			),
			Entry("user and author in org - user is a leaf approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:      []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {userName: struct{}{}},
					},
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob},
				},
			),
			Entry("user and author in org - user is a leaf approver - case in OWNERS and GitHub is different for user - OWNERS lowercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:      []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {userName: struct{}{}},
					},
					UserName:             userNameUpperCase,
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob},
				},
			),
			Entry("user and author in org - user is a leaf approver - case in OWNERS and GitHub is different for user - OWNERS uppercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:      []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {userNameUpperCase: struct{}{}},
					},
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob},
				},
			),
			Entry("user and author in org - user is not a leaf approver for all files",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles: []string{
						"changedFile",
						"otherChangedFile",
					},
					TopLevelApprovers: sets.Set[string]{},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {userName: struct{}{}},
					},
					ExpectedAllowedTypes: nil,
				},
			),
			Entry("only user in org - user is top level approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:         []string{"changedFile"},
					TopLevelApprovers:    sets.Set[string]{userName: struct{}{}},
					LeafApprovers:        map[string]sets.Set[string]{},
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob, prowapi.PostsubmitJob, prowapi.PeriodicJob},
				},
			),
			Entry("only user in org - user is top level approver - case in OWNERS and GitHub don't match - topLeverApprovers lowercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:         []string{"changedFile"},
					TopLevelApprovers:    sets.Set[string]{userName: struct{}{}},
					LeafApprovers:        map[string]sets.Set[string]{},
					UserName:             userNameUpperCase,
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob, prowapi.PostsubmitJob, prowapi.PeriodicJob},
				},
			),
			Entry("only user in org - user is top level approver - case in OWNERS and GitHub don't match - topLeverApprovers uppercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:         []string{"changedFile"},
					TopLevelApprovers:    sets.Set[string]{userNameUpperCase: struct{}{}},
					LeafApprovers:        map[string]sets.Set[string]{},
					UserName:             userName,
					ExpectedAllowedTypes: []prowapi.ProwJobType{prowapi.PresubmitJob, prowapi.PostsubmitJob, prowapi.PeriodicJob},
				},
			),
		)
	})

})

type CanUserRehearseOrgAndUserTestData struct {
	OrgMembers           map[string][]string
	ChangedFiles         []string
	TopLevelApprovers    sets.Set[string]
	LeafApprovers        map[string]sets.Set[string]
	ExpectedAllowedTypes []prowapi.ProwJobType
	ExpectedMessageParts []string
	UserName             string
	IssueLabelsExisting  []string
}

func (td CanUserRehearseOrgAndUserTestData) GetUserNameOrDefault() string {
	if td.UserName == "" {
		return "testuser"
	}
	return td.UserName
}

func newPodSpec() *v1.PodSpec {
	return &v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:                     "blah",
				Image:                    "v2/test42",
				Command:                  nil,
				Args:                     nil,
				WorkingDir:               "",
				Ports:                    nil,
				EnvFrom:                  nil,
				Env:                      nil,
				Resources:                v1.ResourceRequirements{},
				VolumeMounts:             nil,
				VolumeDevices:            nil,
				LivenessProbe:            nil,
				ReadinessProbe:           nil,
				StartupProbe:             nil,
				Lifecycle:                nil,
				TerminationMessagePath:   "",
				TerminationMessagePolicy: "",
				ImagePullPolicy:          "",
				SecurityContext:          nil,
				Stdin:                    false,
				StdinOnce:                false,
				TTY:                      false,
			},
		},
	}
}
