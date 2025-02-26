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

package git

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"

	"testing"
)

func TestGit(t *testing.T) {
	RegisterFailHandler(Fail)

	stat, err := os.Stat("testdata/repo")
	if os.IsNotExist(err) || !stat.IsDir() {
		var output []byte
		output, err = execGit("testdata", []string{"clone", "repo.gitbundle", "repo"})
		if err != nil {
			panic(err)
		}
		fmt.Printf(string(output))
	}

	RunSpecs(t, "Git Main Suite")
}
