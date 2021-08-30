package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"os"
	"strings"
)

type options struct {
	tokenPath           string
	endpoint            string
	org                 string
	repo                string
	author              string
	branchName          string
	ensureLabelsMissing string
}

func (o *options) Validate() error {
	if o.org == "" {
		return fmt.Errorf("org is required")
	}
	if o.repo == "" {
		return fmt.Errorf("repo is required")
	}
	if o.author == "" {
		return fmt.Errorf("author is required")
	}
	if o.branchName == "" {
		return fmt.Errorf("branch-name is required")
	}
	return nil
}

func (o *options) GetEnsureLabelsMissing() []string {
	return strings.Split(o.ensureLabelsMissing, ",")
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&o.tokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	fs.StringVar(&o.endpoint, "github-endpoint", "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
	fs.StringVar(&o.org, "org", "", "The org for the PR.")
	fs.StringVar(&o.repo, "repo", "", "The repo for the PR.")
	fs.StringVar(&o.author, "author", "", "The author for the PR.")
	fs.StringVar(&o.branchName, "branch-name", "", "The branch name for the PR.")
	fs.StringVar(&o.ensureLabelsMissing, "ensure-labels-missing", "lgtm", "What labels have to be missing on the PR (list of comma separated labels).")
	fs.Parse(os.Args[1:])
	return o
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	// TODO: Use global option from the prow config.
	logrus.SetLevel(logrus.DebugLevel)
	o := gatherOptions()
	if err := o.Validate(); err != nil {
		log().WithError(err).Fatal("Invalid arguments provided.")
	}

	ctx := context.Background()
	var client *github.Client
	if o.tokenPath == "" {
		var err error
		client, err = github.NewEnterpriseClient(o.endpoint, o.endpoint, nil)
		if err != nil {
			log().Panicln(err)
		}
	} else {
		tokenBytes, err := os.ReadFile(o.tokenPath)
		if err != nil {
			log().Panicln(err)
		}
		token := string(tokenBytes)
		token = strings.TrimSuffix(token, "\n")
		if err != nil {
			log().Panicln(err)
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		client, err = github.NewEnterpriseClient(o.endpoint, o.endpoint, oauth2.NewClient(ctx, ts))
		if err != nil {
			log().Panicln(err)
		}
	}

	prs, _, err := client.PullRequests.List(ctx, o.org, o.repo, &github.PullRequestListOptions{
		State:       "open",
		Head:        fmt.Sprintf("%s:%s", o.author, o.branchName),
		ListOptions: github.ListOptions{},
	})
	if err != nil {
		log().WithError(err).Fatal("failed to find PR")
	} else if len(prs) == 0 {
		logrus.Info("No PR found")
		os.Exit(0)
	} else if len(prs) > 1 {
		logrus.Fatalf("More than one PR found: %+v", prs)
	}

	if checkAnyLabelExists(prs[0], o.GetEnsureLabelsMissing()) {
		log().Fatalf("ensureLabelsMissing failed")
	}

}

func checkAnyLabelExists(prToCheck *github.PullRequest, labelsToCheck []string) bool {
	labels := map[string]struct{}{}
	for _, label := range prToCheck.Labels {
		name := *label.Name
		labels[name] = struct{}{}
	}
	labelsExist := false
	for _, label := range labelsToCheck {
		if _, exists := labels[label]; exists {
			log().Infof("label %s exists on PR %+v", label, prToCheck)
			labelsExist = true
		}
	}
	return labelsExist
}

func log() *logrus.Entry {
	return logrus.StandardLogger().WithField("robot", "labels-checker")
}
