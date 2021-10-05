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

package release

import (
	"reflect"
	"testing"

	"kubevirt.io/project-infra/robots/pkg/querier"
)

func Test_getLatestMinorReleases(t *testing.T) {
	type args struct {
		releases []*querier.SemVer
	}
	tests := []struct {
		name                    string
		args                    args
		wantLatestMinorReleases []*querier.SemVer
	}{
		{
			name: "empty list",
			args: args{
				releases: nil,
			},
			wantLatestMinorReleases: nil,
		},
		{
			name: "two elements, same minor",
			args: args{
				releases: []*querier.SemVer{
					{"1", "17", "42"},
					{"1", "17", "37"},
				},
			},
			wantLatestMinorReleases: []*querier.SemVer{
				{"1", "17", "42"},
			},
		},
		{
			name: "five elements, three minors",
			args: args{
				releases: []*querier.SemVer{
					{"1", "17", "42"},
					{"1", "17", "37"},
					{"1", "16", "4"},
					{"1", "16", "3"},
					{"1", "15", "1"},
				},
			},
			wantLatestMinorReleases: []*querier.SemVer{
				{"1", "17", "42"},
				{"1", "16", "4"},
				{"1", "15", "1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotLatestMinorReleases := GetLatestMinorReleases(tt.args.releases); !reflect.DeepEqual(gotLatestMinorReleases, tt.wantLatestMinorReleases) {
				t.Errorf("GetLatestMinorReleases() = %v, want %v", gotLatestMinorReleases, tt.wantLatestMinorReleases)
			}
		})
	}
}
