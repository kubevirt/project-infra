package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/bazelbuild/buildtools/build"
)

type Artifact struct {
	rule *build.Rule
}

func (a *Artifact) URLs() []string {
	return a.rule.AttrStrings("urls")
}

func (a *Artifact) SHA256() string {
	return a.rule.AttrString("sha256")
}

func (a *Artifact) AppendURL(url string) {
	list := a.rule.Attr("urls").(*build.ListExpr).List
	a.rule.Attr("urls").(*build.ListExpr).List = append(list, &build.StringExpr{Value: url})
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

func WriteWorkspace(workspace *build.File, path string) error {
	fmt.Println(build.FormatString(workspace))
	return nil
}

func GetArtifacts(workspace *build.File) (artifacts []Artifact, err error) {
	for _, ruleName := range []string{"http_archive", "http_file"} {
		rules := workspace.Rules(ruleName)
		for _, rule := range rules {
			artifacts = append(artifacts, Artifact{
				rule: rule,
			} )
		}
	}
	return artifacts, err
}

func FilterArtifactsWithoutMirror(artifacts []Artifact, regexp *regexp.Regexp) (noMirror []Artifact)  {
	for _, artifact := range artifacts {
		var mirror string
		for _, url := range artifact.URLs() {
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

func UploadFile(url string) error {
		defer out.Close()

		// Get the data
		resp, err := http.Get(url)
		if err != nil {
		return err
	}
		defer resp.Body.Close()

		// Write the body to GCS
		_, err = io.Copy(out, resp.Body)
		if err != nil {
		return err
	}

		return nil
}