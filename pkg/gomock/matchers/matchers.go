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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package matchers

import (
	"fmt"
	"reflect"
	"strings"
)

type ContainsStringsMatcher struct {
	strings []string
}

func (m ContainsStringsMatcher) Matches(x interface{}) bool {
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.String:
		for _, stringToContain := range m.strings {
			if !strings.Contains(v.String(), stringToContain) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
func (m ContainsStringsMatcher) String() string {
	return fmt.Sprintf("contains all strings: %v", m.strings)
}

func ContainsStrings(s ...string) ContainsStringsMatcher {
	return ContainsStringsMatcher{strings: s}
}
