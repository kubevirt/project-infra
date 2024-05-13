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
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config/org"
	"kubevirt.io/project-infra/robots/pkg/sig"
	"os"
	"regexp"
	"sigs.k8s.io/yaml"
)

var (
	debug             bool
	orgsYamlPath      string
	targetOrgName     string
	ownersAliasesPath string
	sigHandleMatcher  = regexp.MustCompile(`^sig-(.*)(-(approvers|reviewers))$`)
)

func main() {
	flag.StringVar(&ownersAliasesPath, "owners-aliases-path", "", "path to the OWNERS_ALIASES file")
	flag.StringVar(&orgsYamlPath, "orgs-yaml-path", "", "path to the orgs.yaml file that is the input configuration file for peribolos")
	flag.StringVar(&targetOrgName, "target-org-name", "", "name of the org inside orgs.yaml file that should be synced")
	flag.BoolVar(&debug, "debug", false, "whether debug output should be printed")
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if ownersAliasesPath == "" {
		log.Fatal("owners-aliases-path is required")
	}
	raw, err := os.ReadFile(ownersAliasesPath)
	if err != nil {
		log.Fatalf("failed to read file %q: %v", ownersAliasesPath, err)
	}
	var ownersAliases sig.OwnersAliases
	err = yaml.Unmarshal(raw, &ownersAliases)
	if err != nil {
		log.Fatalf("failed to unmarshall file %q: %v", ownersAliasesPath, err)
	}
	if orgsYamlPath == "" {
		log.Fatal("orgs-yaml-path is required")
	}
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
	log.Debugf("sigs: %v", sigs)
	log.Debugf("labels: %v", labels)
}
