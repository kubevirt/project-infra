package fakegithub

import (
	"fmt"
	"sync"

	"sigs.k8s.io/prow/pkg/github"
)

// FakeClient is a fake GitHub client.
type FakeClient struct {
	mu            sync.Mutex
	labels        map[string][]string
	addedLabels   map[string][]string
	removedLabels map[string][]string
	pullRequests  map[int]*github.PullRequest
	issueComments map[int][]github.IssueComment
	comments      map[string][]string
}

// NewFakeClient creates a new FakeClient.
func NewFakeClient() *FakeClient {
	return &FakeClient{
		labels:        make(map[string][]string),
		addedLabels:   make(map[string][]string),
		removedLabels: make(map[string][]string),
		pullRequests:  make(map[int]*github.PullRequest),
		issueComments: make(map[int][]github.IssueComment),
		comments:      make(map[string][]string),
	}
}

// AddLabel adds a label to an issue.
func (f *FakeClient) AddLabel(org, repo string, number int, label string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", org, repo, number)
	f.addedLabels[key] = append(f.addedLabels[key], label)
	return nil
}

// RemoveLabel removes a label from an issue.
func (f *FakeClient) RemoveLabel(org, repo string, number int, label string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", org, repo, number)
	f.removedLabels[key] = append(f.removedLabels[key], label)
	return nil
}

// GetIssueLabels gets the labels on an issue.
func (f *FakeClient) GetIssueLabels(org, repo string, number int) ([]github.Label, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", org, repo, number)
	var labels []github.Label
	for _, labelName := range f.labels[key] {
		labels = append(labels, github.Label{Name: labelName})
	}
	return labels, nil
}

// GetPullRequest gets a pull request.
func (f *FakeClient) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if pr, ok := f.pullRequests[number]; ok {
		return pr, nil
	}
	return nil, fmt.Errorf("pull request %d not found", number)
}

// EditPullRequest edits a pull request.
func (f *FakeClient) EditPullRequest(org, repo string, number int, pr *github.PullRequest) (*github.PullRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.pullRequests[number]; !ok {
		return nil, fmt.Errorf("pull request %d not found", number)
	}
	f.pullRequests[number].Body = pr.Body
	return f.pullRequests[number], nil
}

// CreateComment creates a comment on an issue.
func (f *FakeClient) CreateComment(org, repo string, number int, comment string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", org, repo, number)
	f.comments[key] = append(f.comments[key], comment)
	return nil
}

// HasLabel returns true if the given label was added.
func (f *FakeClient) HasLabel(org, repo string, number int, label string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", org, repo, number)
	for _, l := range f.addedLabels[key] {
		if l == label {
			return true
		}
	}
	return false
}

// HasRemovedLabel returns true if the given label was removed.
func (f *FakeClient) HasRemovedLabel(org, repo string, number int, label string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", org, repo, number)
	for _, l := range f.removedLabels[key] {
		if l == label {
			return true
		}
	}
	return false
}

// SetLabels sets the initial labels for a PR.
func (f *FakeClient) SetLabels(org, repo string, number int, labels []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", org, repo, number)
	f.labels[key] = labels
}

// SetPullRequest sets a pull request for the fake client to return.
func (f *FakeClient) SetPullRequest(number int, pr *github.PullRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pullRequests[number] = pr
}

// GetComments returns the comments for a given PR.
func (f *FakeClient) GetComments(org, repo string, number int) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%s/%s#%d", org, repo, number)
	return f.comments[key]
}
