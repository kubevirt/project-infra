/*
 * Copyright 2021 The KubeVirt Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package prowjobconfigs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

const OrgAndRepoForJobConfig = "kubevirt/kubevirt"

var SigNames = []string{
	"sig-network",
	"sig-storage",
	"sig-compute",
	"operator",
}

var cronRegex *regexp.Regexp

func init() {
	var err error
	cronRegex, err = regexp.Compile("[0-9] [0-9]+,[0-9]+,[0-9]+ \\* \\* \\*")
	if err != nil {
		panic(err)
	}
}

func CreatePresubmitJobName(latestReleaseSemver *querier.SemVer, sigName string) string {
	return fmt.Sprintf("pull-kubevirt-e2e-k8s-%s.%s-%s", latestReleaseSemver.Major, latestReleaseSemver.Minor, sigName)
}

func CreatePeriodicJobName(latestReleaseSemver *querier.SemVer, sigName string) string {
	return fmt.Sprintf("periodic-kubevirt-e2e-k8s-%s.%s-%s", latestReleaseSemver.Major, latestReleaseSemver.Minor, sigName)
}

func CreateTargetValue(latestReleaseSemver *querier.SemVer, sigName string) string {
	return fmt.Sprintf("k8s-%s.%s-%s", latestReleaseSemver.Major, latestReleaseSemver.Minor, sigName)
}

// AdvanceCronExpression advances source cron expression to +1h10m
// cron expression must have format of i.e. "0 1,9,17 * * *" or it will panic
func AdvanceCronExpression(sourceCronExpr string) string {
	if !cronRegex.MatchString(sourceCronExpr) {
		log.Log().WithField("cronRegex", cronRegex).WithField("sourceCronExpr", sourceCronExpr).Fatal("cronRegex doesn't match")
	}
	parts := strings.Split(sourceCronExpr, " ")
	mins, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		panic(err)
	}
	mins = (mins + 10) % 60
	firstHour, err := strconv.ParseInt(strings.Split(parts[1], ",")[0], 10, 64)
	firstHour = (firstHour + 1) % 8
	return fmt.Sprintf("%d %d,%d,%d * * *", mins, firstHour, firstHour+8, firstHour+16)
}
