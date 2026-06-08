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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

func NewTestDescriptorForName(name string, filename string) (*SourceTestDescriptor, error) {
	if name == "" || filename == "" {
		return nil, fmt.Errorf("name and filename are required")
	}
	t := &SourceTestDescriptor{testName: name, filename: filename}
	err := t.initForName()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func NewTestDescriptorForID(testNameWithID string, filename string) (*SourceTestDescriptor, error) {
	testId, err := GetTestId(testNameWithID)
	if err != nil {
		return nil, fmt.Errorf("testID is required and needs to match format")
	}
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	t := &SourceTestDescriptor{testID: testId, filename: filename}
	err = t.initForID()
	if err != nil {
		return nil, err
	}
	return t, nil
}

type SourceTestDescriptor struct {
	testName    string
	testID      string
	filename    string
	fileCode    string
	file        *ast.File
	outlineNode *Node
	testNode    *ast.CallExpr
}

func (t *SourceTestDescriptor) TestName() string {
	return t.testName
}

func (t *SourceTestDescriptor) Filename() string {
	return t.filename
}

func (t *SourceTestDescriptor) FileCode() string {
	return t.fileCode
}

func (t *SourceTestDescriptor) File() *ast.File {
	return t.file
}

func (t *SourceTestDescriptor) OutlineNode() *Node {
	return t.outlineNode
}

func (t *SourceTestDescriptor) Test() *ast.CallExpr {
	return t.testNode
}

type initFunc func() error

func (t *SourceTestDescriptor) initForName() error {
	return t.init([]initFunc{
		t.initContent,
		t.initFile,
		t.initOutlineNodeForName,
		t.initTestNode,
	})
}

func (t *SourceTestDescriptor) initForID() error {
	return t.init([]initFunc{
		t.initContent,
		t.initFile,
		t.initOutlineNodeForID,
		t.initTestNode,
	})
}

func (t *SourceTestDescriptor) init(inits []initFunc) error {
	for _, init := range inits {
		err := init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *SourceTestDescriptor) initContent() (err error) {
	content, err := os.ReadFile(t.Filename())
	t.fileCode = string(content)
	return
}

func (t *SourceTestDescriptor) initFile() (err error) {
	fs := token.NewFileSet()
	t.file, err = parser.ParseFile(fs, t.Filename(), nil, parser.ParseComments)
	return
}

func (t *SourceTestDescriptor) initOutlineNodeForName() error {
	outlineFromFile, err := OutlineFromFile(t.filename)
	if err != nil {
		return err
	}

	var traverseOutline func(text string, nodes []*Node)
	traverseOutline = func(text string, nodes []*Node) {
		if t.outlineNode != nil {
			return
		}
		for _, n := range nodes {
			nodeText := strings.TrimSpace(strings.Join([]string{text, n.Text}, " "))
			if nodeText == t.testName {
				t.outlineNode = n
				return
			}
			traverseOutline(nodeText, n.Nodes)
		}
	}
	traverseOutline("", outlineFromFile)
	if t.outlineNode == nil {
		return fmt.Errorf("could not find ginkgo outline node with name %q in file %q", t.testName, t.filename)
	}
	return nil
}

func (t *SourceTestDescriptor) initOutlineNodeForID() error {
	outlineFromFile, err := OutlineFromFile(t.filename)
	if err != nil {
		return err
	}

	var traverseOutline func(text string, nodes []*Node)
	traverseOutline = func(text string, nodes []*Node) {
		if t.outlineNode != nil {
			return
		}
		for _, n := range nodes {
			nodeText := strings.TrimSpace(strings.Join([]string{text, n.Text}, " "))
			if strings.Contains(nodeText, t.testID) {
				// TODO: we need to make sure that the test node has already been reached (should usually be the case,
				// or there would be test_id duplicates, but you never know)
				t.outlineNode = n
				t.testName = nodeText
				return
			}
			traverseOutline(nodeText, n.Nodes)
		}
	}
	traverseOutline("", outlineFromFile)
	if t.outlineNode == nil {
		return fmt.Errorf("could not find ginkgo outline node with name %q in file %q", t.testName, t.filename)
	}
	return nil
}

func (t *SourceTestDescriptor) initTestNode() error {
	ast.Inspect(t.file, func(node ast.Node) bool {
		if t.testNode != nil {
			return true
		}
		fn, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if int(fn.Fun.Pos())-1 == t.outlineNode.Start && int(node.End())-1 == t.outlineNode.End {
			t.testNode = fn
		}
		return true
	})
	if t.testNode == nil {
		return fmt.Errorf("could not find ginkgo call node with name %q in file %q", t.testName, t.filename)
	}
	return nil
}
