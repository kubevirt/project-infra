package main

import (
	"testing"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

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
			expectedLabel: "someorg/someorg#1:release-block/release-v0.1",
		},
		{
			name:          "org member adds blocker that already exists",
			userName:      "random-user",
			body:          "/release-block release-v0.1",
			isMember:      true,
			shouldLabel:   false,
			hasLabel:      true,
			expectedLabel: "someorg/someorg#1:release-block/release-v0.1",
		},
		{
			name:          "org member removes blocker",
			userName:      "random-user",
			body:          "/release-block cancel release-v0.1",
			isMember:      true,
			shouldUnlabel: true,
			hasLabel:      true,
			expectedLabel: "someorg/someorg#1:release-block/release-v0.1",
		},
		{
			name:          "org member removes blocker that's already removed",
			userName:      "random-user",
			body:          "/release-block cancel release-v0.1",
			isMember:      true,
			shouldUnlabel: false,
			hasLabel:      false,
			expectedLabel: "someorg/someorg#1:release-block/release-v0.1",
		},
	}

	org := "someorg"

	for _, tc := range tests {
		t.Logf("test case %s", tc.name)
		fc := &fakegithub.FakeClient{
			Issues:              make(map[int]*github.Issue),
			IssueComments:       make(map[int][]github.IssueComment),
			IssueCommentsAdded:  []string{},
			IssueLabelsExisting: []string{},
			OrgMembers:          make(map[string][]string),
		}

		if tc.isMember {
			fc.OrgMembers[org] = []string{tc.userName}
		}

		if tc.hasLabel {
			fc.IssueLabelsExisting = []string{tc.expectedLabel}
		}

		s := &Server{
			GHC: fc,
			Log: logrus.WithField("plugin", pluginName),
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
