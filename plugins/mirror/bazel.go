package mirror

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"

	"cloud.google.com/go/storage"
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

func WriteWorkspace(dryRun bool, workspace *build.File, path string) error {
	if dryRun {
		fmt.Println(build.FormatString(workspace))
		return nil
	}
	return ioutil.WriteFile(path, build.Format(workspace), 0666 )
}

func GetArtifacts(workspace *build.File) (artifacts []Artifact, err error) {
	for _, ruleName := range []string{"http_archive", "http_file", "rpm"} {
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
		if len(artifact.URLs()) == 0 {
			continue
		}
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

func WriteToBucket(dryRun bool, ctx context.Context, client *storage.Client, url string, bucket string, name string) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	reportObject := client.Bucket(bucket).Object(name)
	reader, err := reportObject.NewReader(ctx)
	if err != nil && err != storage.ErrObjectNotExist{
		return fmt.Errorf("error checking if object exists: %v", err)
	} else if err == nil {
		// object already exists
		reader.Close()
		log.Printf("File %s already exists, will not upload again\n", name)
		return nil
	}
	log.Printf("File will be written to gs://%s/%s", bucket, reportObject.ObjectName())

	if dryRun {
		return nil
	}
	reportOutputWriter := reportObject.NewWriter(ctx)
	defer reportOutputWriter.Close()
	_, err = io.Copy(reportOutputWriter, resp.Body)
	return err
}

func GenerateFilePath(bucket string, artifact *Artifact) string {
	u, _ := url.Parse("https://storage.googleapis.com")
	u.Path = path.Join(u.Path, bucket, artifact.SHA256())
	return u.String()
}