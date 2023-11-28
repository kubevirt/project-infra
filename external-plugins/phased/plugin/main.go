package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/pluginhelp"

	"kubevirt.io/project-infra/external-plugins/phased/plugin/handler"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"

	"kubevirt.io/project-infra/external-plugins/phased/plugin/server"
)

type options struct {
	dryRun         bool
	hmacSecretFile string
	endpoint       string
	port           int
	prowConfigPath string
	jobsConfigBase string
	cacheDir       string
	prowLocation   string
	github         prowflagutil.GitHubOptions
}

func (o *options) validate() {
	var errs []error
	if o.prowConfigPath == "" {
		errs = append(errs, fmt.Errorf("prow-config-path can't be empty"))
	}
	if o.jobsConfigBase == "" {
		errs = append(errs, fmt.Errorf("jobs-config-path can't be empty"))
	}
	if o.cacheDir == "" {
		errs = append(errs, fmt.Errorf("cache-dir can't be empty"))
	}
	if o.prowLocation == "" {
		errs = append(errs, fmt.Errorf("prow-location can't be empty"))
	}
	err := o.github.Validate(o.dryRun)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			logrus.WithError(err).Error("entry validation failure")
		}
		logrus.Fatalf("Arguments validation failed!")
	}
}

func gatherOptions() *options {
	o := &options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.port,
		"port",
		9900,
		"Port to listen on.")
	fs.StringVar(&o.endpoint,
		"endpoint",
		"/",
		"Endpoint to listen on.")
	fs.BoolVar(&o.dryRun,
		"dry-run",
		false,
		"If set, dump the job config to stdout.")
	fs.StringVar(&o.hmacSecretFile,
		"hmac-secret-file",
		"/etc/webhook/hmac",
		"Path to the file containing the GitHub HMAC secret.")
	fs.StringVar(&o.prowConfigPath,
		"prow-config-path",
		"",
		"Path to Prow configuration (required).")
	fs.StringVar(&o.jobsConfigBase,
		"jobs-config-base",
		"",
		"Base path to a directory with Prow job configs (required).")
	fs.StringVar(&o.cacheDir,
		"cache-dir",
		"",
		"Directory to store git repos cache in.")
	fs.StringVar(&o.prowLocation,
		"prow-location",
		"",
		"Prow raw git location")
	for _, group := range []flagutil.OptionGroup{&o.github} {
		group.AddFlags(fs)
	}
	fs.Parse(os.Args[1:])
	return o
}

func clientFactoryCacheDirOpt(cacheDir string) func(opts *git.ClientFactoryOpts) {
	return func(cfo *git.ClientFactoryOpts) {
		cfo.CacheDirBase = &cacheDir
	}
}

func main() {
	opts := gatherOptions()
	opts.validate()

	logger := setupLogger()
	logger.Infoln("Setting up events server")

	err := secret.Add(opts.github.TokenPath, opts.hmacSecretFile)
	mustSucceed(err, "Failed to start secrets agent")

	githubClient, err := opts.github.GitHubClient(opts.dryRun)
	mustSucceed(err, "Could not instantiate github client")

	gitClientFactory, err := git.NewClientFactory(clientFactoryCacheDirOpt(opts.cacheDir))
	mustSucceed(err, "Could not instantiate git client factory")

	eventsChan := make(chan *handler.GitHubEvent)

	eventsHandler := handler.NewGitHubEventsHandler(
		eventsChan,
		logger,
		githubClient,
		opts.prowConfigPath,
		opts.jobsConfigBase,
		opts.prowLocation,
		gitClientFactory)

	eventsServer := server.NewGitHubEventsServer(secret.GetTokenGenerator(opts.hmacSecretFile), eventsHandler)

	serverMux := http.NewServeMux()
	serverMux.Handle(opts.endpoint, eventsServer)
	srv := &http.Server{Addr: fmt.Sprintf(":%d", opts.port), Handler: serverMux}
	interrupts.ListenAndServe(srv, 5*time.Second)
	logger.Infoln("Events server is listening on port:", opts.port)
	externalplugins.ServeExternalPluginHelp(serverMux, logger.WithField("plugin-help", ""), helpProvider)
	interrupts.WaitForGracefulShutdown()
	logger.Println("Phased server was gracefully shut down")
}

func helpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The Phased plugin is used to trigger phase 2 jobs when PR is ready for merging.`,
	}
	return pluginHelp, nil
}

func mustSucceed(err error, message string) {
	if err != nil {
		logrus.WithError(err).Fatal(message)
	}
}

func setupLogger() *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC1123Z})
	l.SetLevel(logrus.TraceLevel)
	l.SetOutput(os.Stdout)
	return l
}
