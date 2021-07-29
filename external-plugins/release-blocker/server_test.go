package main

import (
	"testing"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/github"

	"kubevirt.io/project-infra/external-plugins/testutils"
)

func TestHandlePREvent(t *testing.T) {
	var tests = []struct {
		name          string
		user          string
		eventLabel    string
		eventAction   github.PullRequestEventAction
		shouldLabel   bool
		shouldUnlabel bool
		expectedLabel string
		hasLabel      bool
		shouldComment bool
	}{
		{
			name:        "ignore bot during label",
			user:        "bot",
			eventAction: github.PullRequestActionLabeled,
			eventLabel:  "release-blocker/release-v0.1",
		},
		{
			name:        "ignore bot during unlabel",
			user:        "bot",
			eventAction: github.PullRequestActionLabeled,
			eventLabel:  "release-blocker/release-v0.1",
		},
		{
			name:          "should correct manually labeling release blocker",
			user:          "randomuser",
			eventAction:   github.PullRequestActionLabeled,
			shouldUnlabel: true,
			hasLabel:      true,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1",
			eventLabel:    "release-blocker/release-v0.1",
			shouldComment: true,
		},
		{
			name:          "should correct manually unlabeling release blocker",
			user:          "randomuser",
			eventAction:   github.PullRequestActionUnlabeled,
			shouldLabel:   true,
			hasLabel:      false,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1",
			eventLabel:    "release-blocker/release-v0.1",
			shouldComment: true,
		},
		{
			name:        "should ignore none release-blocker unlabel",
			user:        "randomuser",
			eventAction: github.PullRequestActionUnlabeled,
			eventLabel:  "someotherlabel/release-v0.1",
		},
		{
			name:        "should ignore none release-blocker label",
			user:        "randomuser",
			eventAction: github.PullRequestActionLabeled,
			eventLabel:  "someotherlabel/release-v0.1",
		},
	}

	org := "someorg"

	for _, tc := range tests {

		t.Logf("test case %s", tc.name)
		fc := &testutils.FakeClient{
			PullRequests:        make(map[int]*github.PullRequest),
			IssueComments:       make(map[int][]github.IssueComment),
			IssueCommentsAdded:  []string{},
			IssueLabelsExisting: []string{},
			OrgMembers:          make(map[string][]string),
			RepoBranches: []github.Branch{
				{
					Name: "release-v0.1",
				},
			},
		}
		s := &Server{
			ghc:     fc,
			botName: "bot",
			log:     logrus.WithField("plugin", pluginName),
		}

		if tc.hasLabel {
			fc.IssueLabelsExisting = []string{tc.expectedLabel}
		}

		pr := github.PullRequestEvent{
			Action: tc.eventAction,
			Repo: github.Repo{
				Owner: github.User{
					Login: org,
				},
				Name: org,
			},
			Number: 1,
			Label: github.Label{
				Name: tc.eventLabel,
			},
			Sender: github.User{
				Login: tc.user,
			},
		}

		err := s.handlePR(logrus.WithField("testcase", tc.name), pr)
		if err != nil {
			t.Errorf("For case %s, didn't expect error from release-blocker: %v", tc.name, err)
			continue
		}
		if tc.shouldLabel {
			if fc.IssueLabelsAdded[0] != tc.expectedLabel {
				t.Errorf("For case %s, didn't add expected release-blocker label: %v", tc.name, fc.IssueLabelsAdded)
				continue
			}
		} else if len(fc.IssueLabelsAdded) != 0 {
			t.Errorf("For case %s, didn't add expected release-blocker label", tc.name)
			continue
		}

		if tc.shouldUnlabel {
			if fc.IssueLabelsRemoved[0] != tc.expectedLabel {
				t.Errorf("For case %s, didn't remove expected release-blocker label", tc.name)
				continue
			}

		} else if len(fc.IssueLabelsRemoved) != 0 {
			t.Errorf("For case %s, unexpected label removed", tc.name)
			continue
		}

		if tc.shouldComment {
			if len(fc.IssueCommentsAdded) == 0 {
				t.Errorf("For case %s, didn't add expected comment", tc.name)
				continue
			}
		} else if len(fc.IssueCommentsAdded) != 0 {
			t.Errorf("For case %s, unexpected comment", tc.name)
			continue
		}
	}

}

func TestHandleIssueEvent(t *testing.T) {
	var tests = []struct {
		name          string
		user          string
		eventLabel    string
		eventAction   github.IssueEventAction
		shouldLabel   bool
		shouldUnlabel bool
		expectedLabel string
		hasLabel      bool
		shouldComment bool
	}{
		{
			name:        "ignore bot during label",
			user:        "bot",
			eventAction: github.IssueActionLabeled,
			eventLabel:  "release-blocker/release-v0.1",
		},
		{
			name:        "ignore bot during unlabel",
			user:        "bot",
			eventAction: github.IssueActionLabeled,
			eventLabel:  "release-blocker/release-v0.1",
		},
		{
			name:          "should correct manually labeling release blocker",
			user:          "randomuser",
			eventAction:   github.IssueActionLabeled,
			shouldUnlabel: true,
			hasLabel:      true,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1",
			eventLabel:    "release-blocker/release-v0.1",
			shouldComment: true,
		},
		{
			name:          "should correct manually unlabeling release blocker",
			user:          "randomuser",
			eventAction:   github.IssueActionUnlabeled,
			shouldLabel:   true,
			hasLabel:      false,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1",
			eventLabel:    "release-blocker/release-v0.1",
			shouldComment: true,
		},
		{
			name:        "should ignore none release-blocker unlabel",
			user:        "randomuser",
			eventAction: github.IssueActionUnlabeled,
			eventLabel:  "someotherlabel/release-v0.1",
		},
		{
			name:        "should ignore none release-blocker label",
			user:        "randomuser",
			eventAction: github.IssueActionLabeled,
			eventLabel:  "someotherlabel/release-v0.1",
		},
	}

	org := "someorg"

	for _, tc := range tests {

		t.Logf("test case %s", tc.name)
		fc := &testutils.FakeClient{
			Issues:              make(map[int]*github.Issue),
			IssueComments:       make(map[int][]github.IssueComment),
			IssueCommentsAdded:  []string{},
			IssueLabelsExisting: []string{},
			OrgMembers:          make(map[string][]string),
			RepoBranches: []github.Branch{
				{
					Name: "release-v0.1",
				},
			},
		}
		s := &Server{
			ghc:     fc,
			botName: "bot",
			log:     logrus.WithField("plugin", pluginName),
		}

		if tc.hasLabel {
			fc.IssueLabelsExisting = []string{tc.expectedLabel}
		}

		issue := issueEvent{
			IssueEvent: github.IssueEvent{
				Action: tc.eventAction,
				Repo: github.Repo{
					Owner: github.User{
						Login: org,
					},
					Name: org,
				},
				Issue: github.Issue{
					Number: 1,
				},
				Label: github.Label{
					Name: tc.eventLabel,
				},
			},
			Sender: github.User{
				Login: tc.user,
			},
		}

		err := s.handleIssue(logrus.WithField("testcase", tc.name), issue)
		if err != nil {
			t.Errorf("For case %s, didn't expect error from release-blocker: %v", tc.name, err)
			continue
		}
		if tc.shouldLabel {
			if fc.IssueLabelsAdded[0] != tc.expectedLabel {
				t.Errorf("For case %s, didn't add expected release-blocker label: %v", tc.name, fc.IssueLabelsAdded)
				continue
			}
		} else if len(fc.IssueLabelsAdded) != 0 {
			t.Errorf("For case %s, didn't add expected release-blocker label", tc.name)
			continue
		}

		if tc.shouldUnlabel {
			if fc.IssueLabelsRemoved[0] != tc.expectedLabel {
				t.Errorf("For case %s, didn't remove expected release-blocker label", tc.name)
				continue
			}

		} else if len(fc.IssueLabelsRemoved) != 0 {
			t.Errorf("For case %s, unexpected label removed", tc.name)
			continue
		}

		if tc.shouldComment {
			if len(fc.IssueCommentsAdded) == 0 {
				t.Errorf("For case %s, didn't add expected comment", tc.name)
				continue
			}
		} else if len(fc.IssueCommentsAdded) != 0 {
			t.Errorf("For case %s, unexpected comment", tc.name)
			continue
		}
	}

}

func TestHandleIssueComment(t *testing.T) {

	var tests = []struct {
		name          string
		userName      string
		body          string
		expectedLabel string
		hasLabel      bool
		shouldLabel   bool
		shouldUnlabel bool
		shouldComment bool
		isMember      bool
	}{
		{
			name:     "test random comment",
			userName: "random-user",
			body:     "random comment",
			isMember: true,
		},
		{
			name:     "test random comment not member",
			userName: "random-user",
			body:     "random comment",
			isMember: false,
		},
		{
			name:          "non org member tries to add blocker",
			userName:      "random-user",
			body:          "/release-block release-v0.1",
			isMember:      false,
			shouldComment: true,
		},
		{
			name:          "non org member tries to remove blocker",
			userName:      "random-user",
			body:          "/release-block cancel release-v0.1",
			isMember:      false,
			shouldComment: true,
		},
		{
			name:          "org member adds blocker",
			userName:      "random-user",
			body:          "/release-block release-v0.1",
			isMember:      true,
			shouldLabel:   true,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1",
		},
		{
			name:          "org member adds blocker for non-existent branch",
			userName:      "random-user",
			body:          "/release-block release-v0.1-does-not-exist",
			isMember:      true,
			shouldLabel:   false,
			shouldComment: true,
		},
		{
			name:          "org member adds blocker that already exists",
			userName:      "random-user",
			body:          "/release-block release-v0.1",
			isMember:      true,
			shouldLabel:   false,
			hasLabel:      true,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1",
		},
		{
			name:          "org member removes blocker",
			userName:      "random-user",
			body:          "/release-block cancel release-v0.1",
			isMember:      true,
			shouldUnlabel: true,
			hasLabel:      true,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1",
		},
		// it should be possible to remove a label even if the branch doesn't exist anymore
		{
			name:          "org member removes blocker for non-existent label",
			userName:      "random-user",
			body:          "/release-block cancel release-v0.1-does-not-exist",
			isMember:      true,
			shouldUnlabel: true,
			hasLabel:      true,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1-does-not-exist",
		},
		{
			name:          "org member removes blocker that's already removed",
			userName:      "random-user",
			body:          "/release-block cancel release-v0.1",
			isMember:      true,
			shouldUnlabel: false,
			hasLabel:      false,
			expectedLabel: "someorg/someorg#1:release-blocker/release-v0.1",
		},
	}

	org := "someorg"

	for _, tc := range tests {
		t.Logf("test case %s", tc.name)
		fc := &testutils.FakeClient{
			Issues:              make(map[int]*github.Issue),
			IssueComments:       make(map[int][]github.IssueComment),
			IssueCommentsAdded:  []string{},
			IssueLabelsExisting: []string{},
			OrgMembers:          make(map[string][]string),
			RepoBranches: []github.Branch{
				{
					Name: "release-v0.1",
				},
			},
		}

		foc := &testutils.FakeOwnersClient{
			ExistingTopLevelApprovers: sets.NewString(tc.userName),
		}

		froc := &testutils.FakeRepoownersClient{
			Foc: foc,
		}

		if tc.isMember {
			fc.OrgMembers[org] = []string{tc.userName}
		}

		if tc.hasLabel {
			fc.IssueLabelsExisting = []string{tc.expectedLabel}
		}

		branchExists := func(org string, repo string, targetBranch string) (bool, error) {

			if targetBranch == "release-v0.1" {
				return true, nil
			}
			return false, nil
		}

		s := &Server{
			ghc:          fc,
			log:          logrus.WithField("plugin", pluginName),
			branchExists: branchExists,
			ownersClient: froc,
		}

		ic := github.IssueCommentEvent{
			Action: github.IssueCommentActionCreated,
			Repo: github.Repo{
				Owner: github.User{
					Login: org,
				},
				Name: org,
			},
			Issue: github.Issue{
				Number: 1,
			},
			Comment: github.IssueComment{
				User: github.User{
					Login: tc.userName,
				},
				Body: tc.body,
			},
		}

		if err := s.handleIssueComment(logrus.WithField("testcase", tc.name), ic); err != nil {
			t.Errorf("For case %s, didn't expect error from release-blocker: %v", tc.name, err)
			continue
		}

		if tc.shouldLabel {
			if fc.IssueLabelsAdded[0] != tc.expectedLabel {
				t.Errorf("For case %s, didn't add expected release-blocker label: %v", tc.name, fc.IssueLabelsAdded)
				continue
			}
		} else if len(fc.IssueLabelsAdded) != 0 {
			t.Errorf("For case %s, didn't add expected release-blocker label", tc.name)
			continue
		}

		if tc.shouldUnlabel {
			if fc.IssueLabelsRemoved[0] != tc.expectedLabel {
				t.Errorf("For case %s, didn't remove expected release-blocker label", tc.name)
				continue
			}

		} else if len(fc.IssueLabelsRemoved) != 0 {
			t.Errorf("For case %s, unexpected label removed", tc.name)
			continue
		}

		if tc.shouldComment {
			if len(fc.IssueCommentsAdded) == 0 {
				t.Errorf("For case %s, didn't add expected comment", tc.name)
				continue
			}
		} else if len(fc.IssueCommentsAdded) != 0 {
			t.Errorf("For case %s, unexpected comment", tc.name)
			continue
		}
	}
}
