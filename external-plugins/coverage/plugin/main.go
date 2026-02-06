package server

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	v1 "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/acme/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/test-infra/pkg/flagutil"
	"kubevirt.io/project-infra/external-plugins/phased/plugin/handler"
	"kubevirt.io/project-infra/external-plugins/phased/plugin/server"
	"sigs.k8s.io/prow/pkg/config/secret"
	"sigs.k8s.io/prow/pkg/interrupts"
	"sigs.k8s.io/prow/pkg/pluginhelp"
	"sigs.k8s.io/prow/pkg/pluginhelp/externalplugins"
)

type options struct {
	dryRun         bool
	hmacSecretFile string
	endpoint       string
	port           int
	kubeconfig     string
	jobsNs         string
	github         prowflagutil.GitHubOptions
}

func (o *options) validate() {
	var errs []error
	if o.jobsNs == "" {
		errs = append(errs, fmt.Errorf("jobs-namespace can't be empty"))
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

	fs.StringVar(&o.jobsNs,
		"jobs-namespace",
		"",
		"The namespace in which Prow jobs should be created.")

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

	//creating k8s config
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

	if err := secret.Add(opts.github.TokenPath, opts.hmacSecretFile); err != nil {
		logrus.WithError(err).Fatalf("Failed to laod secrets.")
	}

	githubClient, err := opts.github.GitHubClient(opts.dryRun)
	mustSucceed(err, "Could not create GitHub client.")

	eventsChan := make(chan *handler.GitHubEvent)

	eventsHandler := handler.NewGitHubEventsHandler(
		eventsChan,
		logger,
		prowClient.ProwJobs(opts.jobsNs),
		githubClient,
		opts.dryRun,
	)

	//Webhook server
	eventsServer := server.NewGitHubEventsServer(
		secret.GetTokenGenerator(opts.hmacSecretFile),
		eventsChan,
	)

	//HTTP routing
	serverMux := http.NewServeMux()
	serverMux.Handle(opts.endpoint, eventsServer)

	//HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", opts.port),
		Handler: serverMux,
	}

	//Starting server
	interrupts.ListenAndServe(srv, 5*time.Second)
	logger.Infoln("Coverage server us listening on port: ", opts.port)

	//Serve plugin help endpoint
	externalplugins.ServeExternalPluginHelp(serverMux,
		logger.WithField("plugin-help", ""),
		helpProvider)

	//Waiting for server shutdown
	interrupts.WaitForGracefulShutdown()
	logger.Println("Coverage plugin server was gracefully shut down")
}

// Help provider return plugin help info
func helpProvider(_ []prowconfig.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The coverage plugin automatically runs Go unit test coverage
			 on pull requests containing Go code changes.`,
	}
	return pluginHelp, nil
}
