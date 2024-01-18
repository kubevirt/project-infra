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
 */

package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/config/secret"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type options struct {
	dryRun          bool
	logLevel        int
	githubTokenFile string
	org             string
	repo            string
	repositoryPath  string
	outputFile      string
}

//go:embed "upcoming-changes.gomd"
var upcomingChangesTemplate string

func (o *options) Validate() error {
	if o.org == "" {
		return fmt.Errorf("org is required")
	}
	if o.repo == "" {
		return fmt.Errorf("repo is required")
	}
	if o.githubTokenFile == "" {
		return fmt.Errorf("github-token-file is required")
	}
	if o.repositoryPath == "" {
		return fmt.Errorf("path-to-repository is required")
	}
	if o.outputFile == "" {
		temp, err := os.CreateTemp("", "upcoming-changes-*.md")
		if err != nil {
			return err
		}
		o.outputFile = temp.Name()
	}
	return nil
}

func gatherOptions() options {
	o := options{}
	flag.IntVar(&o.logLevel, "log-level", int(log.WarnLevel), "log level from logrus, see https://pkg.go.dev/github.com/sirupsen/logrus@v1.8.1#WarnLevel")
	flag.StringVar(&o.org, "org", "", "The project org")
	flag.StringVar(&o.repo, "repo", "", "The project repo")
	flag.BoolVar(&o.dryRun, "dry-run", true, "Should this be a dry run")
	flag.StringVar(&o.githubTokenFile, "github-token-file", "", "file containing the github token.")
	flag.StringVar(&o.repositoryPath, "path-to-repository", "", "path to git repository")
	flag.StringVar(&o.outputFile, "output-file", "", "path to output file, which will be overwritten if it exists")
	flag.Parse()
	return o
}

func main() {
	o := gatherOptions()
	if err := o.Validate(); err != nil {
		log.Fatalf("Invalid options: %v", err)
	}

	logger := log.New()
	logger.SetFormatter(&log.JSONFormatter{})
	logger.SetLevel(log.Level(o.logLevel))

	logEntry := logger.WithField("app", "feature-announce")

	if err := secret.Add(o.githubTokenFile); err != nil {
		logEntry.WithError(err).Fatal("error starting secrets agent")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(secret.GetSecret(o.githubTokenFile))},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)

	gitCli := gitCommandLine{
		directory: o.repositoryPath,
		logger:    logEntry,
	}
	announcer := featureAnnouncer{
		githubClient: githubClient,
		gitCli:       gitCli,
		options:      o,
		logger:       logEntry,
	}

	err := announcer.generateUpcomingChangesAnnouncement()
	if err != nil {
		logEntry.WithError(err).Fatalf("failed to create announcement")
	}
}

type UpcomingChangesAnnouncementData struct {
	UpcomingChanges []*ReleaseNote
}

type gitCommandLine struct {
	directory string
	logger    *log.Entry
}

func (g *gitCommandLine) execute(args ...string) (string, error) {
	args = append([]string{"-C", g.directory}, args...)
	g.logger.Debugf("executing 'git %v", args)
	cmd := exec.Command("git", args...)
	bytes, err := cmd.CombinedOutput()
	output := string(bytes)
	if err != nil {
		return "", fmt.Errorf("git command %q failed: %w (%s)", args, err, output)
	}
	return output, nil
}

type featureAnnouncer struct {
	options      options
	gitCli       gitCommandLine
	githubClient *github.Client
	logger       *log.Entry
}

func (f *featureAnnouncer) generateUpcomingChangesAnnouncement() error {
	latestReleaseTag, err := f.fetchLatestTag()
	if err != nil {
		return fmt.Errorf("error fetching latest tag: %w", err)
	}
	features, err := f.fetchCodeChanges(latestReleaseTag)
	if err != nil {
		return fmt.Errorf("error fetching code changes for tag %s: %w", latestReleaseTag, err)
	}

	return f.writeUpcomingChangesAnnouncement(features)
}

func (f *featureAnnouncer) fetchLatestTag() (string, error) {
	// fetch latest tag from branch
	//     a)										  b)						 c)
	// 	  `git tag --list --sort="-version:refname" | grep -vE 'alpha|beta|rc' | head -1`
	gitTagCmdOutput, err := f.gitCli.execute("tag", "--list", fmt.Sprintf("--sort=%s", "-version:refname"))
	if err != nil {
		return "", fmt.Errorf("error fetching tag: %w", err)
	}
	if gitTagCmdOutput == "" {
		return "", fmt.Errorf("error fetching tag: no output")
	}
	gitTags := strings.Split(gitTagCmdOutput, "\n")
	versionRegex := regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	for _, tag := range gitTags {
		if !versionRegex.MatchString(tag) {
			continue
		}
		return tag, nil
	}
	return "", fmt.Errorf("error fetching latest tag: no tag found")
}

func (f *featureAnnouncer) fetchCodeChanges(latestReleaseTag string) ([]*ReleaseNote, error) {
	// 2. fetch pull request references from merge commits
	//    `git logger --oneline --merges v1.1.0..`
	logOutput, err := f.gitCli.execute("log", "--merges", "--oneline", fmt.Sprintf("%s..", latestReleaseTag))
	if err != nil {
		return nil, fmt.Errorf("error fetching merge commit logs: %w", err)
	}
	mergeCommitLogLines := strings.Split(logOutput, "\n")
	mergePRRegex := regexp.MustCompile(`Merge pull request #([0-9]+)`)

	var features []*ReleaseNote
	for _, line := range mergeCommitLogLines {
		if !mergePRRegex.MatchString(line) {
			continue
		}
		prRegexMatches := mergePRRegex.FindStringSubmatch(line)
		prNumber, err := strconv.Atoi(prRegexMatches[1])
		if err != nil {
			return nil, fmt.Errorf("error fetching pull request number from %q: %w", line, err)
		}
		releaseNote, err := f.getReleaseNote(prNumber)
		if err != nil {
			return nil, fmt.Errorf("error fetching pull request data from %q: %w", line, err)
		}
		if releaseNote == nil {
			continue
		}
		features = append(features, releaseNote)
	}
	return features, nil
}

type ReleaseNote struct {
	PullRequestNumber int
	GitHubHandle      string
	ReleaseNote       string
}

func (f *featureAnnouncer) getReleaseNote(prNumber int) (*ReleaseNote, error) {

	f.logger.Debugf("Searching for release note for PR #%d", prNumber)
	pr, _, err := f.githubClient.PullRequests.Get(context.Background(), f.options.org, f.options.repo, prNumber)
	if err != nil {
		return nil, err
	}

	for _, label := range pr.Labels {
		if label.Name != nil && *label.Name == "release-note-none" {
			return nil, nil
		}
	}

	if pr.Body == nil || *pr.Body == "" {
		return nil, err
	}

	body := strings.Split(*pr.Body, "\n")

	return f.extractReleaseNoteContent(prNumber, body, *pr.User.Login)
}

func (f *featureAnnouncer) extractReleaseNoteContent(number int, body []string, gitHubHandle string) (*ReleaseNote, error) {
	var releaseNoteLines []string
	releaseNoteStarted := false
	for _, line := range body {
		switch releaseNoteStarted {
		case false:
			if strings.Contains(line, "```release-note") {
				releaseNoteStarted = true
			}
		case true:
			if !strings.Contains(line, "```") {
				line = strings.TrimSpace(line)
				line = strings.ReplaceAll(line, "\r", "")
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				releaseNoteLines = append(releaseNoteLines, line)
				continue
			}
			note := strings.Join(releaseNoteLines, "\n")
			if strings.Contains(strings.ToLower(note), "none") {
				return nil, nil
			}
			return &ReleaseNote{
				PullRequestNumber: number,
				GitHubHandle:      gitHubHandle,
				ReleaseNote:       note,
			}, nil
		}
	}
	return nil, nil
}

func (f *featureAnnouncer) writeUpcomingChangesAnnouncement(features []*ReleaseNote) error {

	var sanitizedFeatures []*ReleaseNote
	for _, feature := range features {
		sanitizedFeatures = append(sanitizedFeatures, &ReleaseNote{
			PullRequestNumber: feature.PullRequestNumber,
			GitHubHandle:      feature.GitHubHandle,
			ReleaseNote:       f.sanitizeForMarkdown(feature.ReleaseNote),
		})
	}

	writer, err := os.OpenFile(f.options.outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", f.options.outputFile, err)
	}

	upcomingChangesTemplateInstance, err := template.New("upcoming-changes").Parse(upcomingChangesTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse go template: %w", err)
	}

	err = upcomingChangesTemplateInstance.Execute(writer, UpcomingChangesAnnouncementData{
		UpcomingChanges: sanitizedFeatures,
	})
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", f.options.outputFile, err)
	}
	f.logger.Infof("output file written to %q", f.options.outputFile)
	return nil
}

func (f *featureAnnouncer) sanitizeForMarkdown(input string) string {
	input = strings.ReplaceAll(input, "\n", "<br>")
	return input
}
