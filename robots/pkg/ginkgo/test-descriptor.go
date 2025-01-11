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

type TestDescriptor struct {
	testName    string
	filename    string
	fileCode    string
	file        *ast.File
	outlineNode *Node
	testNode    *ast.CallExpr
}

func (t *TestDescriptor) TestName() string {
	return t.testName
}

func (t *TestDescriptor) Filename() string {
	return t.filename
}

func (t *TestDescriptor) FileCode() string {
	return t.fileCode
}

func (t *TestDescriptor) File() *ast.File {
	return t.file
}

func (t *TestDescriptor) OutlineNode() *Node {
	return t.outlineNode
}

func (t *TestDescriptor) Test() *ast.CallExpr {
	return t.testNode
}

type initFunc func() error

func (t *TestDescriptor) init() error {
	for _, init := range []initFunc{
		t.initContent,
		t.initFile,
		t.initOutlineNode,
		t.initTestNode,
	} {
		err := init()
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TestDescriptor) initContent() (err error) {
	content, err := os.ReadFile(t.Filename())
	t.fileCode = string(content)
	return
}

func (t *TestDescriptor) initFile() (err error) {
	fs := token.NewFileSet()
	t.file, err = parser.ParseFile(fs, t.Filename(), nil, parser.ParseComments)
	return
}

func (t *TestDescriptor) initOutlineNode() error {
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

func (t *TestDescriptor) initTestNode() error {
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

func NewTestDescriptorForName(name string, filename string) (*TestDescriptor, error) {
	if name == "" || filename == "" {
		return nil, fmt.Errorf("name and filename are required")
	}
	t := &TestDescriptor{testName: name, filename: filename}
	err := t.init()
	if err != nil {
		return nil, err
	}
	return t, nil
}
