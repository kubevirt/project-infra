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
 * Copyright the KubeVirt Authors.
 *
 */

package options

import (
	"fmt"
	"os"
)

func NewOutputFileOptions(tempFilePattern string) *OutputFileOptions {
	return &OutputFileOptions{
		tempFilePattern: tempFilePattern,
	}
}

type OutputFileOptions struct {
	OutputFile          string
	OverwriteOutputFile bool
	tempFilePattern     string
}

func (o *OutputFileOptions) Validate() error {
	if o.OutputFile == "" {
		file, err := os.CreateTemp("", o.tempFilePattern)
		if err != nil {
			return fmt.Errorf("failed to generate temp file: %w", err)
		}
		o.OutputFile = file.Name()
	} else {
		if !o.OverwriteOutputFile {
			stats, err := os.Stat(o.OutputFile)
			if stats != nil || !os.IsNotExist(err) {
				return fmt.Errorf("file %q exists or error occurred: %w", o.OutputFile, err)
			}
		}
	}
	return nil
}
