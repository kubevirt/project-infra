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

package ginkgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/onsi/ginkgo/v2/ginkgo/command"
	"github.com/onsi/ginkgo/v2/ginkgo/run"
	"github.com/onsi/ginkgo/v2/types"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
)

func DryRun(path string) (report types.Report, output []byte, err error) {

	tempfile, err := os.CreateTemp("", "ginkgo-report-*.json")
	if err != nil {
		return report, output, err
	}
	defer os.Remove(tempfile.Name())

	// since there's no output catchable from the command, we need to use pipe
	// and redirect the output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buildRunCommand := run.BuildRunCommand()

	outC := make(chan string)

	// since we are using the outline command version that panics on any error
	// we need to handle the panic, returning an error only if the command.AbortDetails
	// indicate that case
	defer func() {
		if r := recover(); r != nil {
			errClose := w.Close()
			if errClose != nil {
				log.Warnf("err on close: %v", errClose)
			}
			os.Stdout = old
			out := <-outC
			output = []byte(out)
			switch x := r.(type) {
			case error:
				err = x
			case command.AbortDetails:
				d := r.(command.AbortDetails)
				if d.Error != nil {
					if strings.Contains(d.Error.Error(), "file does not import \"github.com/onsi/ginkgo/v2\"") {
						err = nil
						return
					}
					err = d.Error
				}
			default:
				err = fmt.Errorf("unknown panic: %v", r)
			}
		}
	}()

	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			panic(err)
		}
		outC <- buf.String()
	}()

	buildRunCommand.Run([]string{"-v", "--dry-run", "--json-report", tempfile.Name(), path+"/..."}, nil)

	// restore the output to normal
	err = w.Close()
	os.Stdout = old
	out := <-outC
	output = []byte(out)

	reportContent, err := io.ReadAll(tempfile)
	if err != nil {
		return report, output, err
	}
	err = json.Unmarshal(reportContent, &report)
	return report, output, err
}
