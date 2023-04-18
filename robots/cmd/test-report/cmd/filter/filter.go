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
 *
 */

package filter

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	test_report "kubevirt.io/project-infra/robots/pkg/test-report"
	"os"
	"sort"
	"strings"
)

var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: "filter takes the first argument as a json in format map[string]map[string]int and returns the test names of tests that have not run",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("no file as argument given")
		}
		_, err := os.Stat(args[0])
		if os.IsNotExist(err) {
			return err
		}
		file, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		var input *map[string]map[string]int
		err = json.Unmarshal(file, &input)
		if err != nil {
			return err
		}
		filtered := runFilter(input)
		fmt.Printf(strings.Join(filtered, "\n"))
		return nil
	},
}

func FilterCmd() *cobra.Command {
	return filterCmd
}

func runFilter(input *map[string]map[string]int) []string {
	var result []string
	for testName, testLanesToExecutions := range *input {
		found := false
		for _, testExecution := range testLanesToExecutions {
			if testExecution == test_report.TestExecution_Run {
				found = true
				break
			}
		}
		if !found {
			result = append(result, testName)
		}
	}
	sort.Strings(result)
	return result
}
