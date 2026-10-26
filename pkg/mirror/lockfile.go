package mirror

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/bazelbuild/buildtools/build"
)

type lockRPM struct {
	ID           string   `json:"id,omitempty"`
	PkgName      string   `json:"name,omitempty"`
	HexSHA256    string   `json:"sha256,omitempty"`
	Integrity    string   `json:"integrity,omitempty"`
	DownloadURLs []string `json:"urls"`
	Repository   string   `json:"repository,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	sha256       string
}

func (r *lockRPM) Name() string                            { return r.PkgName }
func (r *lockRPM) SHA256() string                          { return r.sha256 }
func (r *lockRPM) URLs() []string                          { return r.DownloadURLs }
func (r *lockRPM) SetAuthPatterns(_ []*build.KeyValueExpr) {}
func (r *lockRPM) AppendURL(url string)                    { r.DownloadURLs = append(r.DownloadURLs, url) }
func (r *lockRPM) RemoveURLs(notFoundURLs []string) {
	r.DownloadURLs = slices.DeleteFunc(r.DownloadURLs, func(u string) bool {
		return slices.Contains(notFoundURLs, u)
	})
}

type LockFile struct {
	path                 string
	CommandLineArguments []string            `json:"cli-arguments,omitempty"`
	Name                 string              `json:"name,omitempty"`
	Repositories         map[string][]string `json:"repositories,omitempty"`
	RPMs                 []*lockRPM          `json:"rpms"`
	Targets              []string            `json:"targets,omitempty"`
	ForceIgnored         []string            `json:"ignored,omitempty"`
}

func LoadLockFile(filePath string) (*LockFile, []Artifact, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading lockfile: %w", err)
	}
	lf := &LockFile{path: filePath}
	if err := json.Unmarshal(data, lf); err != nil {
		return nil, nil, fmt.Errorf("parsing lockfile %q: %w", filePath, err)
	}
	artifacts := make([]Artifact, 0, len(lf.RPMs))
	for _, r := range lf.RPMs {
		r.sha256, err = decodeSHA256(r.HexSHA256, r.Integrity)
		if err != nil {
			return nil, nil, fmt.Errorf("lockfile %q rpm %q: %w", filePath, r.Name(), err)
		}
		artifacts = append(artifacts, r)
	}
	return lf, artifacts, nil
}

func (lf *LockFile) Write(dryRun bool) error {
	out, err := json.MarshalIndent(lf, "", "\t")
	if err != nil {
		return err
	}
	if dryRun {
		fmt.Println(string(out))
		return nil
	}
	return os.WriteFile(lf.path, append(out, '\n'), 0644)
}

func decodeSHA256(hexDigest, integrity string) (string, error) {
	if hexDigest != "" {
		return hexDigest, nil
	}
	b64, ok := strings.CutPrefix(integrity, "sha256-")
	if !ok {
		return "", fmt.Errorf("missing or unsupported integrity %q", integrity)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) != sha256.Size {
		return "", fmt.Errorf("invalid sha256 integrity %q", integrity)
	}
	return hex.EncodeToString(raw), nil
}
