package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/pluginhelp"

	"kubevirt.io/project-infra/external-plugins/checklist/plugin/handler"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"

	"kubevirt.io/project-infra/external-plugins/checklist/plugin/server"
)

type options struct {
	dryRun         bool
	hmacSecretFile string
	endpoint       string
	port           int
	jobsNs         string
	github         prowflagutil.GitHubOptions
}

func (o *options) validate() {
	var errs []error
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
	for _, group := range []flagutil.OptionGroup{&o.github} {
		group.AddFlags(fs)
	}
	fs.Parse(os.Args[1:])
	return o
}

func main() {
	opts := gatherOptions()
	opts.validate()

	logger := setupLogger()
	logger.Infoln("Setting up events server")

	if err := secret.Add(opts.github.TokenPath, opts.hmacSecretFile); err != nil {
		logrus.WithError(err).Fatalf("Failed to start secrets agent.")
	}

	githubClient, err := opts.github.GitHubClient(opts.dryRun)
	mustSucceed(err, "Could not instantiate github client.")

	eventsChan := make(chan *handler.GitHubEvent)

	eventsHandler := handler.NewGitHubEventsHandler(
		eventsChan,
		logger,
		githubClient)

	eventsServer := server.NewGitHubEventsServer(secret.GetTokenGenerator(opts.hmacSecretFile), eventsHandler)

	serverMux := http.NewServeMux()
	serverMux.Handle(opts.endpoint, eventsServer)
	srv := &http.Server{Addr: fmt.Sprintf(":%d", opts.port), Handler: serverMux}
	interrupts.ListenAndServe(srv, 5*time.Second)
	logger.Infoln("Events server is listening on port:", opts.port)
	externalplugins.ServeExternalPluginHelp(serverMux, logger.WithField("plugin-help", ""), helpProvider)
	interrupts.WaitForGracefulShutdown()
	logger.Println("Checklist plugin server was gracefully shut down")
}

func helpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The Checklist plugin is used to trigger phase 2 jobs when PR is ready for merging.`,
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
