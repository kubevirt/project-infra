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
 * Copyright The KubeVirt Authors.
 */

package git

import (
	"fmt"
	"os/exec"
)

func execGit(sourceFilepath string, args []string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = sourceFilepath
	output, err := handleOutput(command)
	return output, err
}

func handleOutput(command *exec.Cmd) ([]byte, error) {
	output, err := command.Output()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			return nil, fmt.Errorf("exec %s failed: %s", command, e.Stderr)
		case *exec.Error:
			return nil, fmt.Errorf("exec %s failed: %s", command, e)
		default:
			return nil, fmt.Errorf("exec %s failed: %s", command, err)
		}
	}
	return output, nil
}
