package main

import "testing"

func Test_advanceCronExpression(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := advanceCronExpression(tt.args.sourceCronExpr); got != tt.want {
				t.Errorf("advanceCronExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}
