package handler

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"kubevirt.io/project-infra/external-plugins/testutils"
	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/git/localgit"
	gitv2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"
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
				gitClientFactory.Clean()
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
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
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

				canUserRehearse, message := testable.canUserRehearse("testorg", "testrepo", pr, testData.GetUserNameOrDefault(), testData.ChangedFiles)

				Expect(canUserRehearse).To(BeEquivalentTo(testData.ExpectedCanUserRehearse))
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
					ChangedFiles:            []string{},
					TopLevelApprovers:       sets.Set[string]{},
					LeafApprovers:           map[string]sets.Set[string]{},
					ExpectedCanUserRehearse: false,
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
					ExpectedCanUserRehearse: false,
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
					ExpectedCanUserRehearse: true,
					IssueLabelsExisting:     []string{fmt.Sprintf("testorg/testrepo#0:%s", OKToRehearse)},
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
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userName: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					ExpectedCanUserRehearse: true,
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
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userName: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					UserName:                userNameUpperCase,
					ExpectedCanUserRehearse: true,
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
					ExpectedCanUserRehearse: true,
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
					UserName:                userNameUpperCase,
					ExpectedCanUserRehearse: true,
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
					ExpectedCanUserRehearse: true,
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
					ExpectedCanUserRehearse: false,
				},
			),
			Entry("only user in org - user is top level approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userName: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("only user in org - user is top level approver - case in OWNERS and GitHub don't match - topLeverApprovers lowercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userName: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					UserName:                userNameUpperCase,
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("only user in org - user is top level approver - case in OWNERS and GitHub don't match - topLeverApprovers uppercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userNameUpperCase: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					UserName:                userName,
					ExpectedCanUserRehearse: true,
				},
			),
		)
	})

})

type CanUserRehearseOrgAndUserTestData struct {
	OrgMembers              map[string][]string
	ChangedFiles            []string
	TopLevelApprovers       sets.Set[string]
	LeafApprovers           map[string]sets.Set[string]
	ExpectedCanUserRehearse bool
	ExpectedMessageParts    []string
	UserName                string
	IssueLabelsExisting     []string
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
