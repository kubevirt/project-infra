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
	"encoding/csv"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"sigs.k8s.io/prow/pkg/config"
)

func main() {
	jobDirName := "github/ci/prow-deploy/files/jobs/kubevirt/kubevirt"
	dirs, err := os.ReadDir(jobDirName)
	if err != nil {
		log.Fatalf("error reading kubevirt job dir: %v", err)
	}
	kubevirtJobFileNameRegexp := regexp.MustCompile(`kubevirt-(presubmits|periodics)\\.yaml`)
	kubevirtJobNameRegexp := regexp.MustCompile("(pull|periodic)-kubevirt-e2e-.*")
	kubevirtJobNamesToEnvVars := map[string]map[string]string{}
	kubevirtJobNames := []string{}
	envVarsMap := map[string]struct{}{}
	for _, file := range dirs {
		if file.IsDir() {
			continue
		}
		if !kubevirtJobFileNameRegexp.MatchString(file.Name()) {
			continue
		}
		fileName := filepath.Join(jobDirName, file.Name())
		log.Printf("reading file %q", fileName)
		jobConfig, err := config.ReadJobConfig(fileName)
		if err != nil {
			log.Fatalf("error parsing kubevirt job file: %v", err)
		}
		for _, periodic := range jobConfig.Periodics {
			if !kubevirtJobNameRegexp.MatchString(periodic.Name) {
				log.Printf("skipping periodic %q", periodic.Name)
				continue
			}
			kubevirtJobNames = append(kubevirtJobNames, periodic.Name)
			kubevirtJobNamesToEnvVars[periodic.Name] = map[string]string{}
			for _, envVar := range periodic.Spec.Containers[0].Env {
				envVarsMap[envVar.Name] = struct{}{}
				kubevirtJobNamesToEnvVars[periodic.Name][envVar.Name] = envVar.Value
			}
		}
		for _, presubmit := range jobConfig.PresubmitsStatic["kubevirt/kubevirt"] {
			if !kubevirtJobNameRegexp.MatchString(presubmit.Name) {
				log.Printf("skipping presubmit %q", presubmit.Name)
				continue
			}
			kubevirtJobNames = append(kubevirtJobNames, presubmit.Name)
			kubevirtJobNamesToEnvVars[presubmit.Name] = map[string]string{}
			for _, envVar := range presubmit.Spec.Containers[0].Env {
				envVarsMap[envVar.Name] = struct{}{}
				kubevirtJobNamesToEnvVars[presubmit.Name][envVar.Name] = envVar.Value
			}
		}
	}
	sort.Strings(kubevirtJobNames)
	envVars := []string{}
	for envVarName := range envVarsMap {
		envVars = append(envVars, envVarName)
	}
	sort.Strings(envVars)
	rows := [][]string{}
	for _, kubevirtJobName := range kubevirtJobNames {
		values := []string{kubevirtJobName}
		for _, envVar := range envVars {
			value, exists := kubevirtJobNamesToEnvVars[kubevirtJobName][envVar]
			if !exists {
				values = append(values, "")
			} else {
				values = append(values, value)
			}
		}
		rows = append(rows, values)
	}

	temp, err := os.CreateTemp("", "kubevirt-lanes-overview-*.csv")
	if err != nil {
		log.Fatalf("error opening file for writing: %v", err)
	}
	writer := csv.NewWriter(temp)
	headers := []string{"job_name"}
	headers = append(headers, envVars...)
	writer.Write(headers)
	writer.WriteAll(rows)
	log.Printf("Output written to %q", temp.Name())
}
