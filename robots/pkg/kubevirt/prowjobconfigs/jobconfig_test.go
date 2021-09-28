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
			name: "zero one nine seventeen",
			args: args{
				sourceCronExpr: "0 1,9,17 * * *",
			},
			want: "10 2,10,18 * * *",
		},
		{
			name: "fifty one nine seventeen",
			args: args{
				sourceCronExpr: "50 1,9,17 * * *",
			},
			want: "0 2,10,18 * * *",
		},
		{
			name: "zero eight sixteen twentyfour",
			args: args{
				sourceCronExpr: "0 8,16,24 * * *",
			},
			want: "10 1,9,17 * * *",
		},
		{
			name: "zero seven fifteen twentythree",
			args: args{
				sourceCronExpr: "0 7,15,23 * * *",
			},
			want: "10 0,8,16 * * *",
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
