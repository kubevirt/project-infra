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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"encoding/json"
	"errors"
	"k8s.io/test-infra/prow/kube"
	"net/http"
	"os/signal"

	"flag"
	"os"

	log "github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/github"

	"kubevirt.io/project-infra/robots/prowjob-experiment/handler"
)

type options struct {
	ProwConfigPath         string
	JobConfigPathsPatterns string
	Addr                   string
	Namespace              string
	// dryRun bool
}

// Validate the options
func (opts *options) Validate() {
	var validationErrors []error
	if opts.ProwConfigPath == "" {
		validationErrors = append(validationErrors, errors.New("prow config path was not specified"))
	}
	if opts.JobConfigPathsPatterns == "" {
		validationErrors = append(validationErrors, errors.New("job config path patterns were not specified"))
	}
	if opts.Namespace == "" {
		validationErrors = append(validationErrors, errors.New("namespace was not specified"))
	}

	if len(validationErrors) == 0 {
		return
	}
	for _, err := range validationErrors {
		log.Errorln(err.Error())
	}
	os.Exit(1)
}

func gatherOptions() options {
	opts := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&opts.ProwConfigPath, "config-path", "", "Path to config.yaml")
	fs.StringVar(&opts.JobConfigPathsPatterns,
		"job-config-pattern", "",
		"Shell filename pattern for prowjob configs.",
	)
	fs.StringVar(&opts.Addr, "address", ":8720", "ip:port to listen on.")
	fs.StringVar(&opts.Namespace, "namespace", "", "Namespace to deploy prowjobs in")
	err := fs.Parse(os.Args[1:])
	if err != nil {
		log.Errorln(err.Error())
		os.Exit(1)
	}
	return opts
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func eventsHandler(cl *kube.Client, opts options) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var event github.PullRequestEvent
		err := json.NewDecoder(req.Body).Decode(&event)
		if err != nil {
			log.Errorf("Error handling event: %s", err.Error())
			return
		}
		go handler.HandlePullRequestEvent(cl, &event, opts.ProwConfigPath, opts.JobConfigPathsPatterns)
	})
}

func shutdown(c <-chan os.Signal, s *http.Server) {
	select {
	case <-c:
		log.Infoln("Shutting down...")
		s.Shutdown(context.Background())
	}

}

func main() {
	opts := gatherOptions()
	opts.Validate()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	cl, err := kube.NewClientInCluster(opts.Namespace)
	if err != nil {
		log.WithError(err).Fatal("failed to init kube client")
	}

	router := http.NewServeMux()
	router.Handle("/", eventsHandler(cl, opts))

	server := &http.Server{
		Addr:    opts.Addr,
		Handler: router,
	}

	log.Infof("Listening for events on: %s", opts.Addr)
	go shutdown(c, server)
	server.ListenAndServe()
}
