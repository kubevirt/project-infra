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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package validation

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type ContentValidator interface {
	IsValid(content []byte) error
	GetTargetFileName(filename string) string
}

type JSONValidator struct{}

func (j JSONValidator) IsValid(content []byte) error {
	if json.Valid(content) {
		return nil
	}
	return fmt.Errorf("json invalid:\n%s", string(content))
}

func (j JSONValidator) GetTargetFileName(filename string) string {
	return strings.TrimSuffix(filename, ".html") + ".json"
}

type HTMLValidator struct{}

func (j HTMLValidator) IsValid(content []byte) error {
	stringContent := string(content)
	r := strings.NewReader(stringContent)
	d := xml.NewDecoder(r)

	d.Strict = true
	d.Entity = xml.HTMLEntity
	for {
		_, err := d.Token()
		switch err {
		case io.EOF:
			return nil
		case nil:
		default:
			return fmt.Errorf("Report:\n%s\n\nerror %v when trying to validate", err, stringContent)
		}
	}
}

func (j HTMLValidator) GetTargetFileName(filename string) string {
	return filename
}

type CSVValidator struct{}

func (j CSVValidator) IsValid(content []byte) error {
	reader := csv.NewReader(strings.NewReader(string(content)))
	_, err := reader.ReadAll()
	if err == nil {
		return nil
	}
	return fmt.Errorf("csv invalid: \n %v \n %s", err, string(content))
}

func (j CSVValidator) GetTargetFileName(filename string) string {
	return strings.TrimSuffix(filename, ".html") + ".json"
}
