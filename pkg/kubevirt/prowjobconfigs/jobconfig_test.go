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

import "testing"

func Test_AdvanceCronExpression(t *testing.T) {
	type args struct {
		sourceCronExpr string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "zero one seven thirteen nineteen",
			args: args{
				sourceCronExpr: "0 1,7,13,19 * * *",
			},
			want: "10 2,8,14,20 * * *",
		},
		{
			name: "fifty one seven thirteen nineteen",
			args: args{
				sourceCronExpr: "50 1,7,13,19 * * *",
			},
			want: "0 2,8,14,20 * * *",
		},
		{
			name: "zero five eleven seventeen twentythree",
			args: args{
				sourceCronExpr: "0 5,11,17,23 * * *",
			},
			want: "10 0,6,12,18 * * *",
		},
		{
			name: "zero zero six twelve eighteen",
			args: args{
				sourceCronExpr: "0 0,6,12,18 * * *",
			},
			want: "10 1,7,13,19 * * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AdvanceCronExpression(tt.args.sourceCronExpr); got != tt.want {
				t.Errorf("AdvanceCronExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}
