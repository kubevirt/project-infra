package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"golang.org/x/oauth2"
	"kubevirt.io/project-infra/robots/pkg/dependabot"
	"kubevirt.io/project-infra/robots/pkg/dependabot/api"
)

type options struct {
	tokenPath string
	endpoint  string
	org       string
	repo      string
	repoDir   string
}

func (o *options) validate() error {
	if o.org == "" {
		return fmt.Errorf("org is required")
	}
	if o.repo == "" {
		return fmt.Errorf("repo is required")
	}
	if o.tokenPath == "" {
		return fmt.Errorf("token file required")
	}
	return nil
}

var o = options{}

func init() {
	flag.StringVar(&o.tokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	flag.StringVar(&o.endpoint, "github-endpoint", "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
	flag.StringVar(&o.org, "org", "kubevirt", "The org for the PR.")
	flag.StringVar(&o.repo, "repo", "kubevirt", "The repo for the PR.")
	flag.StringVar(&o.repoDir, "repo-dir", "", "The directory where the git repository is checked out")
}

func main() {
	if path := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); path != "" {
		if err := os.Chdir(path); err != nil {
			panic(err)
		}
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)

	flag.Parse()
	if err := o.validate(); err != nil {
		log().WithError(err).Fatal("Invalid arguments provided.")
	}

	rawToken, err := os.ReadFile(o.tokenPath)
	if err != nil {
		log().Panicln(err)
	}
	token := strings.TrimSpace(string(rawToken))
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	client, err := github.NewEnterpriseClient(o.endpoint, o.endpoint, oauth2.NewClient(ctx, ts))
	if err != nil {
		log().Panicln(err)
	}
	// The dependabot url does not have a `v3` suffix
	client.BaseURL, err = url.Parse(o.endpoint)
	if err != nil {
		log().Panicln(err)
	}
	req, err := client.NewRequest("GET", fmt.Sprintf("repos/%s/%s/dependabot/alerts", o.org, o.repo), nil)
	if err != nil {
		log().Panicln(err)
	}
	alerts := []api.Alert{}

	_, err = client.Do(context.Background(), req, &alerts)
	if err != nil {
		log().Panicln(err)
	}
	msgs := []string{}
	cves := api.GetOpenGolangCVEs(alerts)
	filteredAlerts := dependabot.FilterAlerts(alerts)
	for _, cve := range cves {

		if cve.FixedPackageVersion == "" {
			msgs = append(msgs, fmt.Sprintf("%v: skip %q in %q, no fix available yet", cve.CVE, cve.PackageName, cve.GoMod))
			continue
		}

		latestVersion := filteredAlerts[cve.PackageName]
		msg := fmt.Sprintf("%v: bump %q to version %q in %q", cve.CVE, cve.PackageName, latestVersion.LatestVersion, cve.GoMod)
		logrus.Debug(msg)
		rawFile, err := os.ReadFile(filepath.Join(o.repoDir, cve.GoMod))
		if err != nil {
			log().Panicln(err)
		}
		modFile, err := modfile.Parse("go.mod", rawFile, nil)
		if err != nil {
			log().Panicln(err)
		}
		if err := modFile.AddRequire(cve.PackageName, latestVersion.LatestVersion); err != nil {
			log().Panicln(err)
		}
		rawFile, err = modFile.Format()
		if err != nil {
			log().Panicln(err)
		}
		if err := os.WriteFile(filepath.Join(o.repoDir, cve.GoMod), rawFile, 0666); err != nil {
			log().Panicln(err)
		}
		msgs = append(msgs, msg)
	}

	if len(msgs) != 0 {
		fmt.Println(strings.Join(msgs, "\n"))
	}

}

func log() *logrus.Entry {
	return logrus.StandardLogger().WithField("robot", "dependabot")
}
