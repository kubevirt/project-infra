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
	"os"
	"strings"

	"github.com/onsi/ginkgo/v2/types"
	"golang.org/x/tools/imports"
)

const decoratorsImport = "kubevirt.io/kubevirt/tests/decorators"

func QuarantineTest(report *types.SpecReport) error {
	content, err := os.ReadFile(report.LeafNodeLocation.FileName)
	if err != nil {
		return fmt.Errorf("could not read file for quarantine test %q: %w", report.FullText(), err)
	}
	code, err := quarantine(string(content), report.LeafNodeText)
	if err != nil {
		return fmt.Errorf("could not quarantine test %q: %w", report.FullText(), err)
	}
	code = ensureImport(code, decoratorsImport)
	formatted, err := imports.Process(report.LeafNodeLocation.FileName, []byte(code), &imports.Options{FormatOnly: true})
	if err != nil {
		return fmt.Errorf("could not format file for quarantined test %q: %w", report.FullText(), err)
	}
	err = os.WriteFile(report.LeafNodeLocation.FileName, formatted, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not write file for quarantined test %q: %w", report.FullText(), err)
	}
	return nil
}

func modify(input string, substring string, replacement string) (string, error) {
	if substring == "" {
		return "", fmt.Errorf("substring must not be empty")
	}
	return strings.Replace(input, substring, replacement, 1), nil
}

func quarantine(input string, nodeText string) (string, error) {
	replacement := fmt.Sprintf(`"[QUARANTINE]%s", decorators.Quarantine`, nodeText)
	return modify(input, fmt.Sprintf(`%q`, nodeText), replacement)
}

func ensureImport(code string, importPath string) string {
	quoted := fmt.Sprintf("%q", importPath)
	if strings.Contains(code, quoted) {
		return code
	}
	idx := strings.Index(code, "import (")
	if idx < 0 {
		return code
	}
	insertPos := idx + len("import (")
	return code[:insertPos] + "\n\t" + quoted + code[insertPos:]
}
