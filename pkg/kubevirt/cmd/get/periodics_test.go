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

package get

import (
	"testing"

	"sigs.k8s.io/prow/pkg/config"
)

func Test_periodicsData_Less(t *testing.T) {
	tests := []struct {
		name   string
		fields [][]string
		want   bool
	}{
		{
			name: "5 4 * * * >= 5 4 * * *",
			fields: [][]string{
				{"a", "5 4 * * *"},
				{"a", "5 4 * * *"},
			},
			want: false,
		},
		{
			name: "5 4 * * * < 5 4 * 1 *",
			fields: [][]string{
				{"a", "5 4 * * *"},
				{"a", "5 4 * 1 *"},
			},
			want: true,
		},
		{
			name: "5 4 * 1 * < 5 4 1 1 *",
			fields: [][]string{
				{"a", "5 4 * 1 *"},
				{"a", "5 4 1 1 *"},
			},
			want: true,
		},
		{
			name: "5 4 1 2 * >= 5 4 1 1,3,5 *",
			fields: [][]string{
				{"a", "5 4 1 2 *"},
				{"a", "5 4 1 1,3,5 *"},
			},
			want: false,
		},
		{
			name: "5 4 1 1 * < 5 4 1 1,3,5 *",
			fields: [][]string{
				{"a", "5 4 1 1 *"},
				{"a", "5 4 1 1,3,5 *"},
			},
			want: false,
		},
		{
			name: "5 4 1 1 * < 5 4 1 2,3,5 *",
			fields: [][]string{
				{"a", "5 4 1 1 *"},
				{"a", "5 4 1 2,3,5 *"},
			},
			want: true,
		},
		{
			name: "5 4 1 1 * < 5 4 2,3,5 1 *",
			fields: [][]string{
				{"a", "5 4 1 1 *"},
				{"a", "5 4 2,3,5 1 *"},
			},
			want: true,
		},
		{
			name: "5 4 1 1 * < 5 4 2-3,5 1 *",
			fields: [][]string{
				{"a", "5 4 1 1 *"},
				{"a", "5 4 2-3,5 1 *"},
			},
			want: true,
		},
		{
			name: "5 4 3-4 1 * < 5 4 2-3,5 1 *",
			fields: [][]string{
				{"a", "5 4 3-4 1 *"},
				{"a", "5 4 2-3,5 1 *"},
			},
			want: false,
		},
		{
			name: "5 4 * * 3,1 < 5 4 * * 4,2",
			fields: [][]string{
				{"a", "5 4 * * 3,1"},
				{"a", "5 4 * * 4,2"},
			},
			want: true,
		},
		{
			name: "errors: (empty) >= 5 4 * * 4,2",
			fields: [][]string{
				{"a", ""},
				{"a", "5 4 * * 4,2"},
			},
			want: false,
		},
		{
			name: "errors: 5 4 * * 2, >= 5 4 * * 4,3",
			fields: [][]string{
				{"a", "5 4 * * 2,"},
				{"a", "5 4 * * 4,3"},
			},
			want: true,
		},
		{
			name: "errors: 30 1,9,17 * * * < 0 6,22 * * *",
			fields: [][]string{
				{"a", "30 1,9,17 * * *"},
				{"a", "0 6,22 * * *"},
			},
			want: true,
		},
		{
			name: "errors: 0 6,22 * * * >= 30 1,9,17 * * *",
			fields: [][]string{
				{"a", "0 6,22 * * *"},
				{"a", "30 1,9,17 * * *"},
			},
			want: false,
		},
		{
			name: "30 23 * * * > 30 21 * * *",
			fields: [][]string{
				{"a", "30 21 * * *"},
				{"a", "30 23 * * *"},
			},
			want: true,
		},
		{
			name: "30 21 * * * <= 30 23 * * *",
			fields: [][]string{
				{"a", "30 23 * * *"},
				{"a", "30 21 * * *"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := PeriodicsData{
				Periodics: []config.Periodic{
					{
						Cron: tt.fields[0][1],
					},
					{
						Cron: tt.fields[1][1],
					},
				},
			}
			if got := d.Less(0, 1); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}
