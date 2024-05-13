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

package sig

type Person struct {
	github  string `yaml:"github"`
	name    string `yaml:"name"`
	company string `yaml:"company"`
}

type Leadership struct {
	Chairs    []Person `yaml:"chairs"`
	TechLeads []Person `yaml:"tech_leads"`
}

type SpecialInterestGroups struct {
	Dir              string     `yaml:"dir,omitempty"`
	Name             string     `yaml:"name,omitempty"`
	MissionStatement string     `yaml:"mission_statement,omitempty"`
	Leadership       Leadership `yaml:"leadership"`
}

type OwnersAliases struct {
	Aliases map[string][]string
}
