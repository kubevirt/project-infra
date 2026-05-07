package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/test-infra/pkg/flagutil"
	v1 "sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1"
	prowconfig "sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/config/secret"
	prowflagutil "sigs.k8s.io/prow/pkg/flagutil"
	"sigs.k8s.io/prow/pkg/interrupts"
	"sigs.k8s.io/prow/pkg/pluginhelp"
	"sigs.k8s.io/prow/pkg/pluginhelp/externalplugins"

	"kubevirt.io/project-infra/external-plugins/coverage/plugin/handler"
	"kubevirt.io/project-infra/external-plugins/coverage/plugin/server"
)

type options struct {
	dryRun         bool
	hmacSecretFile string
	endpoint       string
	port           int
	kubeconfig     string
	configPath     string
	github         prowflagutil.GitHubOptions
}

func (o *options) validate() {
	var errs []error
	if o.configPath == "" {
		errs = append(errs, fmt.Errorf("config can't be empty"))
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
		9901,
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

	fs.StringVar(&o.configPath,
		"config",
		"/etc/coverage/config.yaml",
		"Path to the job configuration file.")

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

	var config *rest.Config
	var err error
	if opts.kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", opts.kubeconfig)
		mustSucceed(err, "Could not create K8s config from the given kubeconfig.")
	} else {
		config, err = rest.InClusterConfig()
		mustSucceed(err, "Could not create K8s in cluster k8s config")
	}

	prowClient, err := v1.NewForConfig(config)
	mustSucceed(err, "Could not create Prow client.")

	cfg, err := handler.LoadConfig(opts.configPath)
	mustSucceed(err, "Could not load job configuration.")

	if err := secret.Add(opts.github.TokenPath, opts.hmacSecretFile); err != nil {
		logrus.WithError(err).Fatalf("Failed to load secrets.")
	}

	githubClient, err := opts.github.GitHubClient(opts.dryRun)
	mustSucceed(err, "Could not create GitHub client.")

	eventsHandler := handler.NewGitHubEventsHandler(
		logger,
		prowClient.ProwJobs(cfg.Defaults.Namespace),
		githubClient,
		cfg,
		opts.dryRun,
	)

	eventsServer := server.NewGitHubEventsServer(
		secret.GetTokenGenerator(opts.hmacSecretFile),
		eventsHandler,
	)

	serverMux := http.NewServeMux()
	serverMux.Handle(opts.endpoint, eventsServer)

	//HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", opts.port),
		Handler: serverMux,
	}
	//Starting server
	interrupts.ListenAndServe(srv, 5*time.Second)
	logger.Infoln("Coverage server is listening on port: ", opts.port)

	//Serve plugin help endpoint
	externalplugins.ServeExternalPluginHelp(serverMux,
		logger.WithField("plugin-help", ""),
		helpProvider)

	//Waiting for server shutdown
	interrupts.WaitForGracefulShutdown()
	logger.Println("Coverage plugin server was gracefully shut down")
}

// helpProvider returns the plugin help information for the coverage plugin.
func helpProvider(_ []prowconfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The coverage plugin automatically runs Go unit test coverage
			 on pull requests containing Go code changes.`,
	}
	return pluginHelp, nil
}

// mustSucceed exits if the error is not nil
func mustSucceed(err error, message string) {
	if err != nil {
		logrus.WithError(err).Fatal(message)
	}
}

// setupLogger creates and configures the logger
func setupLogger() *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123Z})
	l.SetLevel(logrus.TraceLevel)
	l.SetOutput(os.Stdout)
	return l
}
