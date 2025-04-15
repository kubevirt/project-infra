package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	v1 "sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/pluginhelp"

	"kubevirt.io/project-infra/external-plugins/test-subset/plugin/handler"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/config/secret"
	"sigs.k8s.io/prow/pkg/flagutil"
	gitv2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/interrupts"
	"sigs.k8s.io/prow/pkg/pluginhelp/externalplugins"

	"kubevirt.io/project-infra/external-plugins/test-subset/plugin/server"
)

type options struct {
	dryRun         bool
	hmacSecretFile string
	endpoint       string
	port           int
	prowConfigPath string
	jobsConfigBase string
	kubeconfig     string
	jobsNs         string
	cacheDir       string
	prowLocation   string
	github         flagutil.GitHubOptions
}

func (o *options) validate() {
	var errs []error
	if o.prowConfigPath == "" {
		errs = append(errs, fmt.Errorf("prow-config-path can't be empty"))
	}
	if o.jobsConfigBase == "" {
		errs = append(errs, fmt.Errorf("jobs-config-path can't be empty"))
	}
	if o.jobsNs == "" {
		errs = append(errs, fmt.Errorf("jobs-namespace can't be empty"))
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
	fs.StringVar(&o.kubeconfig,
		"kubeconfig",
		"",
		"Path to kubeconfig. If empty, will try to use K8s defaults.")
	fs.StringVar(&o.prowConfigPath,
		"prow-config-path",
		"",
		"Path to Prow configuration (required).")
	fs.StringVar(&o.jobsConfigBase,
		"jobs-config-base",
		"",
		"Base path to a directory with Prow job configs (required).")
	fs.StringVar(&o.jobsNs,
		"jobs-namespace",
		"",
		"The namespace in which Prow jobs should be created.")
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
	if err := fs.Parse(os.Args[1:]); err != nil {
		logrus.Fatalf("failed to parse: %v", err)
	}
	return o
}

func clientFactoryCacheDirOpt(cacheDir string) func(opts *gitv2.ClientFactoryOpts) {
	return func(cfo *gitv2.ClientFactoryOpts) {
		cfo.CacheDirBase = &cacheDir
	}
}

func main() {
	opts := gatherOptions()
	opts.validate()

	logger := setupLogger()
	logger.Infoln("Setting up events server")

	var config *rest.Config
	var err error
	if opts.kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
		mustSucceed(err, "Could not instantiate K8s config from the given kubeconfig.")
	} else {
		config, err = rest.InClusterConfig()
		mustSucceed(err, "Could not instantiate K8s config from the in cluster config.")
	}
	prowClient, err := v1.NewForConfig(config)
	mustSucceed(err, "Could not instantiate a Prow client from the given kubeconfig.")

	if err := secret.Add(opts.github.TokenPath, opts.hmacSecretFile); err != nil {
		logrus.WithError(err).Fatalf("Failed to start secrets agent.")
	}

	githubClient, err := opts.github.GitHubClient(opts.dryRun)
	mustSucceed(err, "Could not instantiate github client.")

	gitClientFactory, err := gitv2.NewClientFactory(clientFactoryCacheDirOpt(opts.cacheDir))
	mustSucceed(err, "Could not instantiate git client factory")

	eventsChan := make(chan *handler.GitHubEvent)

	eventsHandler := handler.NewGitHubEventsHandler(
		eventsChan,
		logger,
		prowClient.ProwJobs(opts.jobsNs),
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
	logger.Println("Test-subset plugin server was gracefully shut down")
}

func helpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The Test-subset plugin is used to trigger a job that runs custom subset of the tests.
		<br>A pull request is considered trusted if the author is a member of the 'trusted organization' for the repository.
        <br>The arguments are the job name and the requested filter.
		<br>This allows runnign subset of the tests without changing the code.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/test-subset <job_name> <filter>",
		Description: "Triggering job with the selected filter.",
		Featured:    true,
		WhoCanUse:   "Members of the trusted organization for the repo.",
		Examples:    []string{"/test-subset pull-kubevirt-e2e-k8s-1.30-sig-network (USB)"},
	})
	return pluginHelp, nil
}

func mustSucceed(err error, message string) {
	if err != nil {
		logrus.WithError(err).Fatal(message)
	}
}

func setupLogger() *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetLevel(logrus.TraceLevel)
	l.SetOutput(os.Stdout)
	return l
}
