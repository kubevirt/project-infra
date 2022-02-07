package cmd

import (
	"github.com/bndr/gojenkins"
	log "github.com/sirupsen/logrus"
	"reflect"
	"testing"
)

func Test_fetchJunitFilesFromArtifacts(t *testing.T) {
	type args struct {
		completedBuilds []*gojenkins.Build
		fLog            *log.Entry
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "fetches all relevant artifacts",
			args: args{
				completedBuilds: []*gojenkins.Build{
					{
						Raw: &gojenkins.BuildResponse{
							Artifacts: []struct {
								DisplayPath  string `json:"displayPath"`
								FileName     string `json:"fileName"`
								RelativePath string `json:"relativePath"`
							}{
								{
									FileName: "footest.xml",
								},
								{
									FileName: "junit.functest.xml",
								},
								{
									FileName: "partial.junit.functest.1.xml",
								},
								{
									FileName: "partial.junit.functest.2.xml",
								},
								{
									FileName: "partial.junit.functest.3.xml",
								},
								{
									FileName: "bartest.xml",
								},
								{
									FileName: "merged.junit.functest.xml",
								},
								{
									FileName: "foobar.junit.functest.xml",
								},
							},
						},
						Job:     nil,
						Jenkins: nil,
						Base:    "",
						Depth:   0,
					},
				},
				fLog: log.StandardLogger().WithField("test", "test"),
			},
			want: []string{
				"junit.functest.xml",
				"partial.junit.functest.1.xml",
				"partial.junit.functest.2.xml",
				"partial.junit.functest.3.xml",
				"merged.junit.functest.xml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fetchJunitFilesFromArtifacts(tt.args.completedBuilds, tt.args.fLog)
			actualFileNames := []string{}
			for _, artifact := range got {
				actualFileNames = append(actualFileNames, artifact.FileName)
			}
			if !reflect.DeepEqual(tt.want, actualFileNames) {
				t.Errorf("fetchJunitFilesFromArtifacts() = %v, want %v", actualFileNames, tt.want)
			}
		})
	}
}
