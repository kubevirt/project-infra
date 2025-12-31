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

package flakestats

import (
	"fmt"
	"sort"
	"time"
)

type ReportData struct {
	OverallFailures *TopXTest
	TopXTests
	DaysInThePast   int
	Date            time.Time
	ShareCategories []ShareCategory
	Org             string
	Repo            string
}

type TopXTests []*TopXTest

type ShareCategory struct {
	CSSClassName       string
	MinPercentageValue float64
}

var shareCategories = []ShareCategory{
	{
		"lightyellow",
		0.25,
	},
	{
		"yellow",
		1.0,
	},
	{
		"orange",
		2.5,
	},
	{
		"orangered",
		5.0,
	},
	{
		"red",
		10.0,
	},
	{
		"darkred",
		25.0,
	},
}

func (t TopXTests) Len() int {
	return len(t)
}

func (t TopXTests) Less(i, j int) bool {

	// go through the FailuresPerDay from most recent to last
	// the one which has more recent failures is less than the other
	// where "more recent failures" means per i,j that
	// 1) mostRecentSetOfFailures := most recent complete set
	//    of directly adjacent failures per each day
	//    i.e. assuming today is Wed
	//         thus [Wed, Tue, Mon] is more recent than [Tue, Mon, Sun]
	// 2) sum(mostRecentSetOfFailures(i)) > sum(mostRecentSetOfFailures(j))
	tIFailuresPerDaySum, tJFailuresPerDaySum := 0, 0
	daysInThePast := t[i].daysInThePast
	if daysInThePast < t[j].daysInThePast {
		daysInThePast = t[j].daysInThePast
	}
	for day := 0; day < daysInThePast; day++ {
		dayForFailure := time.Now().Add(time.Duration(-1*day*24) * time.Hour)
		dateKeyForFailure := dayForFailure.Format(time.DateOnly) + "T00:00:00Z"
		tIFailuresPerDay, iExists := t[i].FailuresPerDay[dateKeyForFailure]
		tJFailuresPerDay, jExists := t[j].FailuresPerDay[dateKeyForFailure]
		if !iExists && !jExists {
			if tIFailuresPerDaySum > tJFailuresPerDaySum {
				return true
			}
			continue
		}
		if !jExists {
			return true
		}
		if !iExists {
			return false
		}
		tIFailuresPerDaySum += tIFailuresPerDay.Sum
		tJFailuresPerDaySum += tJFailuresPerDay.Sum
	}

	// continue comparing the remaining values
	iAllFailures := t[i].AllFailures
	jAllFailures := t[j].AllFailures
	return iAllFailures.Sum > jAllFailures.Sum ||
		(iAllFailures.Sum == jAllFailures.Sum && iAllFailures.Max > jAllFailures.Max) ||
		(iAllFailures.Sum == jAllFailures.Sum && iAllFailures.Max == jAllFailures.Max && iAllFailures.Avg > jAllFailures.Avg)
}

func (t TopXTests) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t TopXTests) CalculateShareFromTotalFailures() *TopXTest {
	overall := &TopXTest{
		Name:            "Test failures overall",
		AllFailures:     &FailureCounter{Name: "overall"},
		FailuresPerDay:  map[string]*FailureCounter{},
		FailuresPerLane: map[string]*FailureCounter{},
	}
	for _, test := range t {
		overall.AllFailures.add(test.AllFailures.Sum)

		// aggregate failures per test per day
		for day, failuresPerDay := range test.FailuresPerDay {
			_, failuresPerDayExists := overall.FailuresPerDay[day]
			if !failuresPerDayExists {
				overall.FailuresPerDay[day] = &FailureCounter{
					Name: failuresPerDay.Name,
					URL: fmt.Sprintf(
						"https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/kubevirt/kubevirt/flakefinder-%s-024h.html",
						formatFromRFC3339ToRFCDate(day),
					),
				}
			}
			overall.FailuresPerDay[day].add(failuresPerDay.Sum)
		}

		// aggregate failures per test per lane
		for lane, failuresPerLane := range test.FailuresPerLane {
			_, failuresPerLaneExists := overall.FailuresPerLane[lane]
			if !failuresPerLaneExists {
				overall.FailuresPerLane[lane] = &FailureCounter{
					Name: lane,
					URL:  generateTestGridURLForJob(lane),
				}
			}
			overall.FailuresPerLane[lane].add(failuresPerLane.Sum)
		}

	}
	for index := range t {
		t[index].CalculateShareFromTotalFailures(overall.AllFailures.Sum)
	}
	overall.CalculateShareFromTotalFailures(overall.AllFailures.Sum)
	return overall
}

func NewTopXTest(topXTestName string) *TopXTest {
	return &TopXTest{
		Name:            topXTestName,
		AllFailures:     &FailureCounter{Name: "All failures"},
		FailuresPerDay:  map[string]*FailureCounter{},
		FailuresPerLane: map[string]*FailureCounter{},
	}
}

type TopXTest struct {
	Name                   string
	AllFailures            *FailureCounter
	FailuresPerDay         map[string]*FailureCounter
	FailuresPerLane        map[string]*FailureCounter
	NoteHasBeenQuarantined bool
	daysInThePast          int
}

func (t *TopXTest) CalculateShareFromTotalFailures(totalFailures int) {
	t.AllFailures.setShare(totalFailures)
	for key := range t.FailuresPerLane {
		t.FailuresPerLane[key].setShare(totalFailures)
	}
	for key := range t.FailuresPerDay {
		t.FailuresPerDay[key].setShare(totalFailures)
	}
}

func (t *TopXTest) GetSortedFailuresPerLane() []*FailureCounter {
	lanes := make([]*FailureCounter, 0, len(t.FailuresPerLane))
	for _, fc := range t.FailuresPerLane {
		lanes = append(lanes, fc)
	}

	sort.Slice(lanes, func(i, j int) bool {
		if lanes[i].Sum == lanes[j].Sum {
			return lanes[i].Name < lanes[j].Name
		}
		return lanes[i].Sum > lanes[j].Sum
	})

	return lanes
}

type FailureCounter struct {
	Name          string
	Count         int
	Sum           int
	Avg           float64
	Max           int
	SharePercent  float64
	ShareCategory ShareCategory
	URL           string
	PrCount       int
}

func (f *FailureCounter) add(value int) {
	f.Sum += value
	if value > f.Max {
		f.Max = value
	}
	f.Avg = (float64(value) + float64(f.Count)*f.Avg) / float64(f.Count+1)
	f.Count++
}

func (f *FailureCounter) setShare(totalFailures int) {
	f.SharePercent = float64(f.Sum) / float64(totalFailures) * 100
	for _, shareCategory := range shareCategories {
		if shareCategory.MinPercentageValue <= f.SharePercent {
			f.ShareCategory = shareCategory
		}
	}
}
