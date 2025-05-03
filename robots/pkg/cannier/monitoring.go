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
	"runtime"
	"time"
)

func getMonitoringExtractors(startTime time.Time) []featureExtractor {
	monitoringExtractors := []featureExtractor{
		func(featureSet *FeatureSet) error {
			featureSet.ReadCount, featureSet.WriteCount = getMonitoredFileSystemActivity()
			return nil
		},
		func(featureSet *FeatureSet) error {
			featureSet.MaxMemory = getPeakMemoryUsage()
			return nil
		},
		func(featureSet *FeatureSet) error {
			featureSet.ContextSwitches = monitorContextSwitches()
			return nil
		},
		func(featureSet *FeatureSet) error {
			featureSet.RunTime = time.Since(startTime).Seconds()
			return nil
		},
		func(featureSet *FeatureSet) error {
			monitorIOWaitTime(featureSet)
			return nil
		},
	}
	return monitoringExtractors
}

// TODO: stubbed
func monitorIOWaitTime(featureSet *FeatureSet) {
	featureSet.WaitTime = 0
}

// TODO: stubbed
func getMonitoredFileSystemActivity() (int, int) {
	// Use tools like `strace` for real monitoring
	return 0, 0
}

// TODO: stubbed
func getPeakMemoryUsage() int {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return int(memStats.Alloc / 1024) // In KB
}

// TODO: stubbed
func monitorContextSwitches() int {
	// Use tools like `/proc/[pid]/status` for real monitoring
	return 0
}
