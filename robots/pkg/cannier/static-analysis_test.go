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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestStaticAnalysis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "metrics suite")
}

var _ = Describe("static analysis", func() {
	When("ast depth is calculated for a simple test file", func() {

		var file *ast.File

		BeforeEach(func() {

			const testFilename = "testdata/simple_test.go"

			fs := token.NewFileSet()
			var err error
			file, err = parser.ParseFile(fs, testFilename, nil, parser.ParseComments)
			if err != nil {
				panic(err)
			}
		})

		It("computes a value greater zero", func() {
			astDepth := calculateASTDepth(file)
			Expect(astDepth).To(BeNumerically(">", 0))
		})

	})
})
