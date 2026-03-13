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
	"io"
	"os"
	"strings"

	"github.com/onsi/ginkgo/v2/ginkgo/command"
	"github.com/onsi/ginkgo/v2/ginkgo/outline"
	log "github.com/sirupsen/logrus"
)

// CloneWithoutNodes creates a copy of this node excluding its children
func (n Node) CloneWithoutNodes() *Node {
	return &Node{
		Name:    n.Name,
		Text:    n.Text,
		Start:   n.Start,
		End:     n.End,
		Spec:    n.Spec,
		Focused: n.Focused,
		Pending: n.Pending,
		Labels:  n.Labels,
	}
}

// CloneWithNodes creates a copy of this node including clones of its children
func (n Node) CloneWithNodes(filters ...NodeFilter) *Node {
	clone := &Node{
		Name:    n.Name,
		Text:    n.Text,
		Start:   n.Start,
		End:     n.End,
		Spec:    n.Spec,
		Focused: n.Focused,
		Pending: n.Pending,
		Labels:  n.Labels,
	}
nextChild:
	for _, child := range n.Nodes {
		for _, f := range filters {
			if !f(child) {
				continue nextChild
			}
		}
		clone.Nodes = append(clone.Nodes, child.CloneWithNodes())
	}
	return clone
}

func OutlineFromFile(path string) (testOutline []*Node, err error) {

	// since there's no output catchable from the command, we need to use pipe
	// and redirect the output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buildOutlineCommand := outline.BuildOutlineCommand()

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
			switch x := r.(type) {
			case error:
				err = x
			case command.AbortDetails:
				d := r.(command.AbortDetails)
				if strings.Contains(d.Error.Error(), "file does not import \"github.com/onsi/ginkgo/v2\"") {
					err = nil
					return
				}
				err = d.Error
			default:
				err = fmt.Errorf("unknown panic: %v", r)
			}
		}
	}()

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			panic(err)
		}
		outC <- buf.String()
	}()

	buildOutlineCommand.Run([]string{"--format", "json", path}, nil)

	// restore the output to normal
	err = w.Close()
	os.Stdout = old
	out := <-outC
	output := []byte(out)

	testOutline, err = ToOutline(output)
	if err != nil {
		return nil, fmt.Errorf("ToOutline failed on %s: %w", path, err)
	}
	return testOutline, nil
}

func ToOutline(fileData []byte) (testOutline []*Node, err error) {
	err = json.Unmarshal(fileData, &testOutline)
	return testOutline, err
}
