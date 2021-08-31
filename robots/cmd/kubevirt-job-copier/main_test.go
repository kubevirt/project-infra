package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"kubevirt.io/project-infra/robots/pkg/querier"
	"reflect"
	"testing"
)

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

func Test_getSourceAndTargetRelease(t *testing.T) {
	type args struct {
		releases []*github.RepositoryRelease
	}
	tests := []struct {
		name  string
		args  args
		wantTargetRelease  *querier.SemVer
		wantSourceRelease *querier.SemVer
		wantErr error
	}{
		{
			name: "has one patch release for latest",
			args: args{
				releases: []*github.RepositoryRelease{
					release("v1.22.0"),
					release("v1.21.3"),
				},
			},
			wantTargetRelease: &querier.SemVer{
				Major: "1",
				Minor: "22",
				Patch: "0",
			},
			wantSourceRelease: &querier.SemVer{
				Major: "1",
				Minor: "21",
				Patch: "3",
			},
			wantErr: nil,
		},
		{
			name: "has two patch releases for latest",
			args: args{
				releases: []*github.RepositoryRelease{
					release("v1.22.1"),
					release("v1.22.0"),
					release("v1.21.3"),
				},
			},
			wantTargetRelease: &querier.SemVer{
				Major: "1",
				Minor: "22",
				Patch: "1",
			},
			wantSourceRelease: &querier.SemVer{
				Major: "1",
				Minor: "21",
				Patch: "3",
			},
			wantErr: nil,
		},
		{
			name: "has one release only, should err",
			args: args{
				releases: []*github.RepositoryRelease{
					release("v1.22.1"),
				},
			},
			wantTargetRelease: nil,
			wantSourceRelease: nil,
			wantErr: fmt.Errorf("less than two releases"),
		},
		{
			name: "has two major same releases",
			args: args{
				releases: []*github.RepositoryRelease{
					release("v1.22.1"),
					release("v1.22.0"),
				},
			},
			wantTargetRelease: &querier.SemVer{
				Major: "1",
				Minor: "22",
				Patch: "1",
			},
			wantSourceRelease: nil,
			wantErr: fmt.Errorf("no source release found"),
		},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTargetRelease, gotSourceRelease, gotErr := getSourceAndTargetRelease(tt.args.releases)
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("getSourceAndTargetRelease() got = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotTargetRelease, tt.wantTargetRelease) {
				t.Errorf("getSourceAndTargetRelease() got = %v, want %v", gotTargetRelease, tt.wantTargetRelease)
			}
			if !reflect.DeepEqual(gotSourceRelease, tt.wantSourceRelease) {
				t.Errorf("getSourceAndTargetRelease() got1 = %v, want %v", gotSourceRelease, tt.wantSourceRelease)
			}
		})
	}
}

func release(version string) *github.RepositoryRelease {
	result := github.RepositoryRelease{}
	result.TagName = &version
	return &result
}
