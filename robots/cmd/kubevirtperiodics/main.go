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
	"flag"
	"fmt"
	"github.com/bndr/gotabulate"
	"github.com/lnquy/cron"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"os"
	"sort"
	"strings"
)

type checkPeriodicjobruntimesOptions struct {
	jobConfigPathKubevirtPeriodics string
}

func (o checkPeriodicjobruntimesOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPeriodics); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPeriodics is required: %v", err)
	}
	return nil
}

var checkPeriodicjobruntimesOpts = checkPeriodicjobruntimesOptions{}

type data struct {
	rows [][]string
}

func (d data) Len() int {
	return len(d.rows)
}
func (d data) Less(i, j int) bool {
	return d.rows[i][0] < d.rows[j][0]
}
func (d data) Swap(i, j int) {
	d.rows[i], d.rows[j] = d.rows[j], d.rows[i]
}

func main() {
	log := logrus.New().WithField("robot", "kubevirtperiodics")

	flag.StringVar(&checkPeriodicjobruntimesOpts.jobConfigPathKubevirtPeriodics, "job-config-path-kubevirt-periodics", "", "The path to the kubevirt periodic job definitions")
	flag.Parse()

	err := checkPeriodicjobruntimesOpts.Validate()
	if err != nil {
		log.Fatalf("validation failed: %v", err)
	}

	descriptor, err := cron.NewDescriptor()
	if err != nil {
		log.Fatalf("creating descriptor failed: %v", err)
	}

	data := data{}

	periodicsJobConfig, err := config.ReadJobConfig(checkPeriodicjobruntimesOpts.jobConfigPathKubevirtPeriodics)
	for _, periodic := range periodicsJobConfig.Periodics {
		if !strings.Contains(periodic.Name, "e2e") {
			continue
		}
		description, err := descriptor.ToDescription(periodic.Cron, cron.Locale_en)
		if err != nil {
			data.rows = append(data.rows, []string{periodic.Name, periodic.Interval})
			continue
		}
		data.rows = append(data.rows, []string{periodic.Name, description})
	}

	sort.Sort(data)

	t := gotabulate.Create(data.rows)
	t.SetHeaders([]string{"job_name", "runs"})
	t.SetEmptyString("-")
	t.SetAlign("left")
	fmt.Println(t.Render("grid"))
}
