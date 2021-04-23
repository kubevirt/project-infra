package mirror

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/bazelbuild/buildtools/build"
)

type Artifact struct {
	rule *build.Rule
}

func (a *Artifact) URLs() []string {
	if a.rule.AttrString("url") != "" {
		return []string{a.rule.AttrString("url")}
	}
	return a.rule.AttrStrings("urls")
}

func (a *Artifact) Name() string {
	return a.rule.AttrString("name")
}

func (a *Artifact) SHA256() string {
	return a.rule.AttrString("sha256")
}

func (a *Artifact) AppendURL(url string) {
	if a.rule.Attr("urls") == nil {
		a.rule.SetAttr("urls", &build.ListExpr{})
	}
	if a.rule.AttrString("url") != "" {
		a.rule.Attr("urls").(*build.ListExpr).List = append(a.rule.Attr("urls").(*build.ListExpr).List, &build.StringExpr{Value: a.rule.AttrString("url")})
		a.rule.DelAttr("url")
	}
	a.rule.Attr("urls").(*build.ListExpr).List = append(a.rule.Attr("urls").(*build.ListExpr).List, &build.StringExpr{Value: url})
}

func (a *Artifact) RemoveURLs(notFoundUrls []string) {
	var urlsToRemove = map[string]struct{}{}
	for _, urlToRemove := range notFoundUrls {
		urlsToRemove[urlToRemove] = struct{}{}
	}

	var remainingArtifactURLs []build.Expr

	urls := a.rule.Attr("urls")
	if urls == nil {
		urlAttr := a.rule.AttrString("url")
		if urlAttr != "" {
			for _, notFoundUrl := range notFoundUrls {
				if urlAttr == notFoundUrl {
					a.rule.DelAttr("url")
					break
				}
			}
		}
		return
	}

	list := urls.(*build.ListExpr).List
	for _, urlValue := range list {
		if _, exists := urlsToRemove[urlValue.(*build.StringExpr).Value]; !exists {
			remainingArtifactURLs = append(remainingArtifactURLs, urlValue)
		}
	}

	urls.(*build.ListExpr).List = remainingArtifactURLs
}

type HTTPClient interface {
	Head(uri string) (resp *http.Response, err error)
	Get(uri string)  (resp *http.Response, err error)
}

var Client HTTPClient

func init() {
	Client = http.DefaultClient
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
	return ioutil.WriteFile(path, build.Format(workspace), 0666)
}

func GetArtifacts(workspace *build.File) (artifacts []Artifact, err error) {
	for _, ruleName := range []string{"http_archive", "http_file", "rpm"} {
		rules := workspace.Rules(ruleName)
		for _, rule := range rules {
			artifacts = append(artifacts, Artifact{
				rule: rule,
			})
		}
	}
	return artifacts, err
}

func FilterArtifactsWithoutMirror(artifacts []Artifact, regexp *regexp.Regexp) (noMirror []Artifact) {
	for _, artifact := range artifacts {
		if len(artifact.URLs()) == 0 {
			continue
		}
		if mirror := getMirror(artifact, regexp); mirror == "" {
			noMirror = append(noMirror, artifact)
		}
	}
	return noMirror
}

func RemoveStaleDownloadURLS(artifacts []Artifact, ignoreURLSMatching *regexp.Regexp, client HTTPClient) {
	for _, artifact := range artifacts {
		var notFoundUrls []string

		for _, notFoundUrl := range artifact.URLs() {
			if ignoreURLSMatching.MatchString(notFoundUrl) {
				continue
			}
			resp, err := client.Head(notFoundUrl)
			if err != nil {
				log.Printf("Could not connect to source URL: %v", err)
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				log.Printf("Could not find artifact %s, will remove URL: Status Code: %v", notFoundUrl, resp.StatusCode)
				notFoundUrls = append(notFoundUrls, notFoundUrl)
			}
		}

		artifact.RemoveURLs(notFoundUrls)
	}

}

func CheckArtifactsHaveURLS(artifacts []Artifact) error {
	artifactsWithoutURLs := []string{}
	for _, artifact := range artifacts {
		if len(artifact.URLs()) == 0 {
			artifactsWithoutURLs = append(artifactsWithoutURLs, artifact.Name())
		}
	}
	if len(artifactsWithoutURLs) == 0 {
		return nil
	}
	return fmt.Errorf("artifacts without urls found: '%s'", strings.Join(artifactsWithoutURLs, "', '"))
}

func getMirror(artifact Artifact, regexp *regexp.Regexp) string {
	for _, urlStr := range artifact.URLs() {
		if regexp.MatchString(urlStr) {
			return urlStr
		}
	}
	return ""
}

func WriteToBucket(dryRun bool, ctx context.Context, client *storage.Client, artifact Artifact, bucket string, httpClient HTTPClient) (err error) {
	reportObject := client.Bucket(bucket).Object(artifact.SHA256())
	reader, err := reportObject.NewReader(ctx)
	if err != nil && err != storage.ErrObjectNotExist {
		return fmt.Errorf("error checking if object exists: %v", err)
	} else if err == nil {
		// object already exists
		reader.Close()
		log.Printf("File %s already exists, will not upload again\n", artifact.SHA256())
		return nil
	}
	for _, uri := range artifact.URLs() {
		resp, err := httpClient.Get(uri)
		if err != nil {
			log.Printf("Could not connect to source, continuing with next URL: %v", err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			log.Printf("Could not upload artifact from %s, continuing with next URL: Status Code: %v", uri, resp.StatusCode)
			continue
		}

		log.Printf("File will be written to gs://%s/%s", bucket, reportObject.ObjectName())

		var reportOutputWriter io.WriteCloser
		if dryRun {
			reportOutputWriter, err = os.OpenFile("/dev/null", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("Failed to open /dev/null: %v", err)
			}
		} else {
			reportOutputWriter = reportObject.NewWriter(ctx)
		}
		defer reportOutputWriter.Close()

		sha := sha256.New()
		body := io.TeeReader(resp.Body, sha)
		_, err = io.Copy(reportOutputWriter, body)
		if err != nil {
			log.Printf("Could not upload artifact from %s, continuing with next URL: %v", uri, err)
			continue
		}
		if toHex(sha) != artifact.SHA256() {
			log.Printf("Could not upload artifact from %s, continuing with next URL: Expected shasum %v, got %v", uri, artifact.SHA256(), toHex(sha))
			continue
		}
		return nil
	}
	return fmt.Errorf("artifact download urls exhausted, failed to upload %s", artifact.Name())
}

func VerifyArtifact(ctx context.Context, client *storage.Client, artifact Artifact, bucket string) (err error) {
	reportObject := client.Bucket(bucket).Object(artifact.SHA256())
	reader, err := reportObject.NewReader(ctx)
	if err == storage.ErrObjectNotExist {
		return fmt.Errorf("artifact %v: object is not cached: %v", artifact.Name(), err)
	} else if err != nil {
		return fmt.Errorf("artifact %v: error checking if object exists: %v", artifact.Name(), err)
	}
	defer reader.Close()
	sha := sha256.New()
	if _, err := io.Copy(sha, reader); err != nil {
		return fmt.Errorf("artifact %v: failed to download the object: %v", artifact.Name(), err)

	}
	if toHex(sha) != artifact.SHA256() {
		return fmt.Errorf("artifact %v: expected shasum %v, got %v", artifact.Name(), artifact.SHA256(), toHex(sha))
	}
	return nil
}

func GenerateFilePath(bucket string, artifact *Artifact) string {
	u, _ := url.Parse("https://storage.googleapis.com")
	u.Path = path.Join(u.Path, bucket, artifact.SHA256())
	return u.String()
}

func toHex(hasher hash.Hash) string {
	return hex.EncodeToString(hasher.Sum(nil))
}
