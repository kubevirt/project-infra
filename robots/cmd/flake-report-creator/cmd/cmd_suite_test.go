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

package cmd

import (
	"strings"
	"testing"
)

type expectation struct {
	expectedToBeContained string
}

func (e *expectation) contains(content string, t *testing.T) {
	if !strings.Contains(content, e.expectedToBeContained) {
		t.Errorf("Expected contained: %q, actual %q", e.expectedToBeContained, content)
	}
}

func expectContains(expectedToBeContained string) expectation {
	return expectation{expectedToBeContained: expectedToBeContained}
}
