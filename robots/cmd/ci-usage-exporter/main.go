package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	configflagutil "k8s.io/test-infra/prow/flagutil/config"
	"k8s.io/test-infra/prow/pjutil"

	"kubevirt.io/project-infra/robots/pkg/ci-usage-exporter/metrics"
)

const (
	metricsPath = "/metrics"
	healthPath  = "/healthz"
)

type options struct {
	metricsPort   int
	healthPort    int
	jobConfigPath string
	configPath    string
}

func flagOptions() options {
	o := options{}

	flag.IntVar(&o.metricsPort, "metrics-port", 9836, "Port to serve metrics")
	flag.IntVar(&o.healthPort, "health-port", 8081, "Port to serve health endpoint")
	flag.StringVar(&o.jobConfigPath, "job-config-path", "", "Path to Prow jobs configuration directory")
	flag.StringVar(&o.configPath, "config-path", "", "Path to Prow configuration file")
	flag.Parse()
	return o
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()

	health := pjutil.NewHealthOnPort(o.healthPort)
	health.ServeReady()

	cfg := configflagutil.ConfigOptions{
		ConfigPath:    o.configPath,
		JobConfigPath: o.jobConfigPath,
	}
	configAgent, err := cfg.ConfigAgent()
	if err != nil {
		log.Fatalf("Could not initialize config agent: %v", err)
	}
	config := metrics.Config{
		ProwConfig: configAgent.Config(),
	}

	metricsHandler := metrics.NewHandler()
	if err := metricsHandler.Start(config); err != nil {
		log.Fatalf("Could not start metrics collection: %v", err)
	}
	defer func() {
		if err := metricsHandler.Stop(); err != nil {
			log.Printf("Could not stop metrics collection: %v", err)
		}
	}()

	addr := fmt.Sprintf(":%d", o.metricsPort)
	sm := http.NewServeMux()
	sm.Handle(metricsPath, promhttp.Handler())
	srv := &http.Server{
		Addr:    addr,
		Handler: sm,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("Start serving metrics on %s%s", addr, metricsPath)

	<-done
	log.Print("Server Stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed: %+v", err)
	}
	log.Print("Server Exited Properly")
}
