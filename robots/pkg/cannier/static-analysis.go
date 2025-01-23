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

package cannier

import (
	"go/ast"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"regexp"
	"strings"
)

func getStaticAnalysisExtractors(test *ginkgo.TestDescriptor) ([]featureExtractor, error) {

	return []featureExtractor{
		func(featureSet *FeatureSet) error { featureSet.ASTDepth = calculateASTDepth(test.Test()); return nil },
		func(featureSet *FeatureSet) error {
			// TODO: use test node
			featureSet.Assertions = countAssertions(test.FileCode())
			return nil
		},
		func(featureSet *FeatureSet) error {
			featureSet.CyclomaticComplexity = calculateCyclomaticComplexity(test.Test())
			return nil
		},
		func(featureSet *FeatureSet) error {
			featureSet.ExternalModules = countExternalModules(test.File())
			return nil
		},
		func(featureSet *FeatureSet) error {
			// TODO: LOC for test func only
			featureSet.TestLinesOfCode = countLinesOfCode(test.FileCode())
			return nil
		},
		func(featureSet *FeatureSet) error {
			featureSet.HalsteadVolume = calculateHalsteadVolume(test.FileCode())
			return nil
		},
		calculateMaintainability,
	}, nil
}

// calculateMaintainability calculates FeatureSet.Maintainability
// TODO: stubbed
func calculateMaintainability(featureSet *FeatureSet) error {
	featureSet.Maintainability = 0.0
	return nil
}

type maxDepthCapture struct {
	maxDepth int
}

func (c *maxDepthCapture) updateIfGreater(depth int) {
	if depth > c.maxDepth {
		c.maxDepth = depth
	}
}

type astVisitor struct {
	depth int
	*maxDepthCapture
}

func (v *astVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	childDepth := v.depth + 1
	v.maxDepthCapture.updateIfGreater(childDepth)
	w := &astVisitor{depth: childDepth, maxDepthCapture: v.maxDepthCapture}
	return w
}

// calculateASTDepth calculates the maximum AST depth.
func calculateASTDepth(node ast.Node) int {
	visitor := &astVisitor{depth: 0, maxDepthCapture: &maxDepthCapture{maxDepth: 0}}
	ast.Walk(visitor, node)
	return visitor.maxDepthCapture.maxDepth
}

// countAssertions counts the number of assertion statements.
// TODO: add Ginkgo stuff
func countAssertions(code string) int {
	assertionRegex := regexp.MustCompile(`\b(require|assert)\.`)
	return len(assertionRegex.FindAllString(code, -1))
}

// calculateCyclomaticComplexity calculates cyclomatic complexity.
func calculateCyclomaticComplexity(node ast.Node) int {
	// TODO: replace with gocyclo call
	complexity := 1
	ast.Inspect(node, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.CaseClause:
			complexity++
		}
		return true
	})
	return complexity
}

// countExternalModules counts non-standard library imports.
func countExternalModules(node ast.Node) int {
	count := 0
	ast.Inspect(node, func(n ast.Node) bool {
		if imp, ok := n.(*ast.ImportSpec); ok {
			if !strings.Contains(imp.Path.Value, "golang.org") && !strings.Contains(imp.Path.Value, "github.com") {
				count++
			}
		}
		return true
	})
	return count
}

// calculateHalsteadVolume calculates Halstead volume
// TODO: stubbed with a simple formula
func calculateHalsteadVolume(code string) float64 {
	// Simple example using unique operators and operands
	operators := regexp.MustCompile(`[+\-*/%=&|<>!]`).FindAllString(code, -1)
	operands := regexp.MustCompile(`\b\w+\b`).FindAllString(code, -1)
	n1 := len(operators)
	n2 := len(operands)
	N := n1 + n2
	n := len(unique(operators)) + len(unique(operands))
	if n == 0 {
		return 0
	}
	return float64(N) * (float64(n) / float64(n1+n2))
}

// countLinesOfCode counts the total number of lines in the code.
// TODO: count lines of test func instead of file
func countLinesOfCode(code string) int {
	return len(strings.Split(code, "\n"))
}

// unique returns a unique slice of strings.
func unique(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
