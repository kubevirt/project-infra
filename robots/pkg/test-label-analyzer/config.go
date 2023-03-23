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
 * Copyright 2023 Red Hat, Inc.
 */

package test_label_analyzer

import "regexp"

// A LabelCategory defines a category of tests that share a common label either in their test name or as a Ginkgo label
type LabelCategory struct {
	Name            string         `yaml:"name"`
	TestNameLabelRE *regexp.Regexp `yaml:"testNameLabelRE"`
	GinkgoLabelRE   *regexp.Regexp `yaml:"ginkgoLabelRE"`
}

// Config defines the configuration file structure that is required to map tests to categories.
type Config struct {
	Categories []LabelCategory `yaml:"categories"`
}

func NewQuarantineDefaultConfig() Config {
	return Config{
		Categories: []LabelCategory{
			{
				Name:            "Quarantine",
				TestNameLabelRE: regexp.MustCompile("\\[QUARANTINE\\]"),
				GinkgoLabelRE:   regexp.MustCompile("Quarantine"),
			},
		},
	}
}
