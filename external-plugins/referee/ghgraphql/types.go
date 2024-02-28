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

package ghgraphql

import (
	"time"
)

// PRTimelineForLastCommit represents the specific events a PR has received
type PRTimelineForLastCommit struct {

	// NumberOfRetestComments is the number of `/(re)test` comments that triggered a testing on the PR
	NumberOfRetestComments int

	// WasHeld determines whether the PR did receive a `/hold` comment
	WasHeld bool

	// WasHoldCanceled determines whether the PR did receive an `/unhold` or `/hold cancel` comment
	WasHoldCanceled bool

	// PRTimeLineItems holds all specific events and their data in order of appearance
	PRTimeLineItems []PRTimeLineItem
}

type PRTimeLineItemType string

const (
	RetestComment    PRTimeLineItemType = "retest_comment"
	HoldComment      PRTimeLineItemType = "hold_comment"
	UnholdComment    PRTimeLineItemType = "unhold_comment"
	HoldLabelAdded   PRTimeLineItemType = "hold_label_added"
	HoldLabelRemoved PRTimeLineItemType = "hold_label_removed"
)

type PRTimeLineItem struct {
	ItemType PRTimeLineItemType
	Item     TimelineItem
}

type PRLabels struct {
	IsHoldPresent bool
	Labels        []Label
}

type PullRequests struct {
	PRs []PullRequest
}

type PullRequest struct {
	Number int
	Title  string
}

type Author struct {
	Login string
}
type IssueCommentFragment struct {
	CreatedAt time.Time
	BodyText  string
	Author    Author
}

type Commit struct {
	CommittedDate time.Time
}

type PullRequestCommitFragment struct {
	Commit Commit
}

type Actor struct {
	Login string
}

type BaseRefForcePushFragment struct {
	Actor     Actor
	CreatedAt time.Time
}

type HeadRefForcePushFragment struct {
	Actor     Actor
	CreatedAt time.Time
}

type Label struct {
	Name string
}

type Labels struct {
	Nodes []Label
}

type TimelineItem struct {
	IssueCommentFragment      `graphql:"... on IssueComment"`
	PullRequestCommitFragment `graphql:"... on PullRequestCommit"`
	BaseRefForcePushFragment  `graphql:"... on BaseRefForcePushedEvent"`
	HeadRefForcePushFragment  `graphql:"... on HeadRefForcePushedEvent"`
}

type TimelineItems struct {
	Nodes []TimelineItem
}
