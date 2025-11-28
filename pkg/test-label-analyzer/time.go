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

package test_label_analyzer

import (
	"fmt"
	"time"
)

// Since returns a simple textual description of an approximate time that has passed since the input time and now
func Since(date time.Time) string {
	var age string
	since := time.Since(date)
	hours := int(since.Hours())
	switch {
	case hours < 48:
		age = fmt.Sprintf("%d hours", hours)
	case hours < 24*14:
		age = fmt.Sprintf("%d days", int(hours/24))
	case hours < 24*60:
		age = fmt.Sprintf("%d weeks", int(hours/24/7))
	case hours < 24*365*2:
		age = fmt.Sprintf("%d months", int(hours/24/30))
	default:
		age = fmt.Sprintf("%d years", int(hours/24/365))
	}
	return age
}
