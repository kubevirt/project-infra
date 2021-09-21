package flags

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)


const (
	FlagDryRun = "dry-run"
	FlagGitHubTokenPath = "github-token-path"
	FlagGitHubEndpoint = "github-endpoint"
)

type GlobalOptions struct {
	DryRun bool
	GitHubTokenPath string
	GitHubEndPoint string
}

var Options = GlobalOptions{}

func (o *GlobalOptions) Validate() error {
	logrus.StandardLogger().WithField("robot", "kubevirt").Infof("Options: %+v", Options)

	if o.GitHubTokenPath != "" {
		if fileInfo, err := os.Stat(o.GitHubTokenPath); !os.IsNotExist(err) {
			if fileInfo.IsDir() {
				return fmt.Errorf("File %s is a directory!", o.GitHubTokenPath)
			}
		} else {
			return err
		}
	}
	return nil
}

func AddPersistentFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&Options.DryRun, FlagDryRun, true, "Whether the file should get modified or just modifications printed to stdout.")
	cmd.PersistentFlags().StringVar(&Options.GitHubTokenPath, FlagGitHubTokenPath, "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	cmd.PersistentFlags().StringVar(&Options.GitHubEndPoint, FlagGitHubEndpoint, "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
}
