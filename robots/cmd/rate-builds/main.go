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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bndr/gojenkins"
	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/robots/pkg/flakefinder/build"
	flakejenkins "kubevirt.io/project-infra/robots/pkg/jenkins"
	"net/http"
	"time"
)

const (
	defaultJenkinsBaseUrl = "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/"
)

type options struct {
	endpoint  string
	startFrom time.Duration
	jobName   string
}

func flagOptions() options {
	o := options{}
	flag.StringVar(&o.endpoint, "endpoint", defaultJenkinsBaseUrl, "jenkins base url")
	flag.DurationVar(&o.startFrom, "start-from", 14*24*time.Hour, "jenkins job name")
	flag.StringVar(&o.jobName, "job-name", "", "jenkins job name")
	flag.Parse()
	return o
}

func (o *options) validate() error {
	if o.jobName == "" {
		return fmt.Errorf("job-name is missing")
	}
	return nil
}

func main() {
	opts := flagOptions()

	log.StandardLogger().SetFormatter(&log.JSONFormatter{})
	jLog := log.StandardLogger().WithField("robot", "badbuilds")

	if err := opts.validate(); err != nil {
		jLog.Fatalf("validating command line options failed: %v", err)
	}

	ctx := context.Background()

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: 5,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	jLog.Printf("Creating client for %s", opts.endpoint)
	jenkins := gojenkins.CreateJenkins(client, opts.endpoint)
	_, err := jenkins.Init(ctx)
	if err != nil {
		jLog.Fatalf("failed to contact jenkins %s: %v", opts.endpoint, err)
	}

	job, err := jenkins.GetJob(ctx, opts.jobName)
	if err != nil {
		jLog.Fatalf("failed to get job %s: %v", opts.jobName, err)
	}

	startOfReport := time.Now().Add(-1 * opts.startFrom)
	buildNumbersToFailures := flakejenkins.GetBuildNumbersToFailuresForJob(startOfReport, job, ctx, jLog)

	result := build.NewRating(opts.jobName, opts.endpoint, opts.startFrom, buildNumbersToFailures)

	bytes, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		jLog.Fatalf("failed to get job %s: %v", opts.jobName, err)
	}
	fmt.Println(string(bytes))
}
