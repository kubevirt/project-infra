package main

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/bazelbuild/buildtools/build"
)

type Artifacts struct {
	Name string
	URLs []string
}

func LoadWorkspace(path string) (*build.File, error) {
	workspaceData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WORSPACE file: %v", err)
	}
	workspace, err := build.ParseWorkspace(path, workspaceData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WORSPACE file: %v", err)
	}
	return workspace, nil
}

func GetArtifacts(workspace *build.File) (artifacts []Artifacts, err error) {
	for _, ruleName := range []string{"http_archive", "http_file"} {
		rules := workspace.Rules(ruleName)
		for _, rule := range rules {
			artifacts = append(artifacts, Artifacts{
				Name: rule.Name(),
				URLs: rule.AttrStrings("urls"),
			} )
		}
	}
	return artifacts, err
}

func FilterArtifactsWithoutMirror(artifacts []Artifacts, regexp *regexp.Regexp) (noMirror []Artifacts)  {
	for _, artifact := range artifacts {
		var mirror string
		for _, url := range artifact.URLs {
			if regexp.MatchString(url) {
				mirror = url
				break
			}
		}
		if mirror == "" {
			noMirror = append(noMirror, artifact)
		}
	}
	return noMirror
}
