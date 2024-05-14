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

package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config/org"
	kv_labels "kubevirt.io/project-infra/robots/pkg/labels"
	"kubevirt.io/project-infra/robots/pkg/sig"
	"os"
	"regexp"
	"strings"
)

var (
	debug             bool
	orgsYamlPath      string
	labelsYamlPath    string
	targetOrgName     string
	ownersAliasesPath string
	sigHandleMatcher  = regexp.MustCompile(`^sig-(.*)(-(approvers|reviewers))$`)
)

func main() {
	flag.StringVar(&ownersAliasesPath, "owners-aliases-path", "", "path to the OWNERS_ALIASES file")
	flag.StringVar(&orgsYamlPath, "orgs-yaml-path", "", "path to the orgs.yaml file that is the input configuration file for peribolos")
	flag.StringVar(&labelsYamlPath, "labels-yaml-path", "", "path to the labels.yaml file that is the input configuration file for label_sync")
	flag.StringVar(&targetOrgName, "target-org-name", "", "name of the org inside orgs.yaml file that should be synced")
	flag.BoolVar(&debug, "debug", false, "whether debug output should be printed")
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}
	if ownersAliasesPath == "" {
		log.Fatal("owners-aliases-path is required")
	}
	if orgsYamlPath == "" {
		log.Fatal("orgs-yaml-path is required")
	}
	if labelsYamlPath == "" {
		log.Fatal("labels-yaml-path is required")
	}

	// 1) deserialize input file

	raw, err := os.ReadFile(ownersAliasesPath)
	if err != nil {
		log.Fatalf("failed to read file %q: %v", ownersAliasesPath, err)
	}
	var ownersAliases sig.OwnersAliases
	err = yaml.Unmarshal(raw, &ownersAliases)
	if err != nil {
		log.Fatalf("failed to unmarshall file %q: %v", ownersAliasesPath, err)
	}

	// 2) condense sigs and labels out of OWNERS_ALIASES
	//    shortcut for sig label - we assume that it's always
	//    `sig/%s`

	sigs := make(map[string]sets.Set[string])
	labels := sets.Set[string]{}
	for aliasName, gitHubHandles := range ownersAliases.Aliases {
		if !sigHandleMatcher.MatchString(aliasName) {
			continue
		}
		bareSigName := sigHandleMatcher.FindStringSubmatch(aliasName)[1]
		if _, ok := sigs[bareSigName]; !ok {
			sigs[bareSigName] = sets.Set[string]{}
		}
		sigs[bareSigName].Insert(gitHubHandles...)
		labels.Insert(fmt.Sprintf("sig/%s", bareSigName))
	}
	log.Debugf(`OWNERS_ALIASES:
	sigs: %v
	labels: %v`, sigs, labels)

	// 3) update peribolos config file
	//    either generate new teams for sigs that don't exist
	//    or update team members by merging current members
	//    with sig members from OWNERS_ALIASES

	raw, err = os.ReadFile(orgsYamlPath)
	if err != nil {
		log.Fatalf("failed to read file %q: %v", ownersAliasesPath, err)
	}
	var fullConfig *org.FullConfig
	err = yaml.Unmarshal(raw, &fullConfig)
	if err != nil {
		log.Fatalf("failed to unmarshall file %q: %v", ownersAliasesPath, err)
	}
	if _, ok := fullConfig.Orgs[targetOrgName]; !ok {
		log.Fatalf("org %q not found in config %q", targetOrgName, orgsYamlPath)
	}

	for bareSigName, gitHubHandles := range sigs {
		fullSigName := fmt.Sprintf("sig-%s", bareSigName)
		if _, ok := fullConfig.Orgs[targetOrgName].Teams[fullSigName]; !ok {
			fullConfig.Orgs[targetOrgName].Teams[fullSigName] = org.Team{
				TeamMetadata: org.TeamMetadata{
					Description: ptr(fmt.Sprintf("auto-generated team for sig-%s, based on OWNERS_ALIASES", bareSigName)),
					Privacy:     ptr(org.Closed),
				},
				Members: gitHubHandles.UnsortedList(),
			}
			log.Infof("GitHub team %s added for org %s", fullSigName, targetOrgName)
		} else {
			// update existing team by merging GitHub handles with existing team member handles
			list := gitHubHandles.UnsortedList()
			merged := sets.Set[string]{}
			merged.Insert(list...)
			team := fullConfig.Orgs[targetOrgName].Teams[fullSigName]
			merged.Insert(team.Members...)
			team.Members = merged.UnsortedList()
			fullConfig.Orgs[targetOrgName].Teams[fullSigName] = team
			log.Infof("GitHub team %s updated for org %s", fullSigName, targetOrgName)
		}
	}

	newConfig, err := yaml.Marshal(&fullConfig)
	if err != nil {
		log.Fatalf("failed to marshall config: %v", err)
	}
	err = os.WriteFile(orgsYamlPath, newConfig, 0666)
	if err != nil {
		log.Fatalf("failed to write file %q: %v", orgsYamlPath, err)
	}

	// 4) update labels config file
	//    sig labels are put into the default section, as they are global for the org
	raw, err = os.ReadFile(labelsYamlPath)
	if err != nil {
		log.Fatalf("failed to read file %q: %v", labelsYamlPath, err)
	}
	var labelConfig *kv_labels.Configuration
	err = yaml.Unmarshal(raw, &labelConfig)
	if err != nil {
		log.Fatalf("failed to unmarshall file %q: %v", labelsYamlPath, err)
	}

	// create map to more easily find the labels for the sigs
	defaultLabels := make(map[string]*kv_labels.Label)
	for _, label := range labelConfig.Default.Labels {
		defaultLabels[label.Name] = &label
	}
	for _, label := range labels.UnsortedList() {
		if _, ok := defaultLabels[label]; !ok {
			newLabel := kv_labels.Label{
				Name:             label,
				Color:            "c5def5",
				Description:      fmt.Sprintf("Label to mark a PR relevant for the SIG %s", strings.Split(label, "/")[1]),
				Target:           "prs",
				ProwPlugin:       "label",
				IsExternalPlugin: false,
				AddedBy:          "anyone",
				Previously:       nil,
				DeleteAfter:      nil,
			}
			labelConfig.Default.Labels = append(labelConfig.Default.Labels, newLabel)
			log.Infof("GitHub label %s created", label)
		}
	}
	newLabelConfig, err := yaml.Marshal(&labelConfig)
	if err != nil {
		log.Fatalf("failed to marshall label config: %v", err)
	}
	err = os.WriteFile(labelsYamlPath, newLabelConfig, 0666)
	if err != nil {
		log.Fatalf("failed to write file %q: %v", labelsYamlPath, err)
	}

}

func ptr[T any](e T) *T {
	return &e
}
