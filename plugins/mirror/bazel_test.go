package mirror

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/bazelbuild/buildtools/build"
)

var data = []byte(`
http_archive(
    name = "io_bazel_rules_container_rpm",
    sha256 = "151261f1b81649de6e36f027c945722bff31176f1340682679cade2839e4b1e1",
    strip_prefix = "rules_container_rpm-0.0.5",
    urls = ["https://github.com/rmohr/rules_container_rpm/archive/v0.0.5.tar.gz"],
)

rpm(
    name = "io_bazel_rules_container_rpm1",
    sha256 = "151261f1b81649de6e36f027c945722bff31176f1340682679cade2839e4b1e1",
    strip_prefix = "rules_container_rpm-0.0.5",
    urls = ["https://github.com/rmohr/rules_container_rpm/archive/v0.0.5.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_container_rpm1",
    sha256 = "151261f1b81649de6e36f027c945722bff31176f1340682679cade2839e4b1e1",
    strip_prefix = "rules_container_rpm-0.0.5",
    urls = ["https://github.com/rmohr/rules_container_rpm/archive/v0.0.5.tar.gz", "https://kubevirt.storage.googleapis.com/xx"],
)

http_file(
    name = "qemu-img",
    sha256 = "eadbd81fe25827a9d7712d0d96b134ab834bfab9e7332a8e9cf54dedd3c02581",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/updates/28/Everything/x86_64/Packages/q/qemu-img-2.11.2-5.fc28.x86_64.rpm",
    ],
)

http_file(
    name = "qemu-img1",
    sha256 = "eadbd81fe25827a9d7712d0d96b134ab834bfab9e7332a8e9cf54dedd3c02581",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/updates/28/Everything/x86_64/Packages/q/qemu-img-2.11.2-5.fc28.x86_64.rpm",
        "https://kubevirt.storage.googleapis.com/xx",
    ],
)
`)

func Test(t *testing.T) {
	file, err := build.ParseWorkspace("workspace", data)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := GetArtifacts(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 5 {
		t.Fatalf("expect 5 artifacts, found %v", len(artifacts))
	}

	invalid := FilterArtifactsWithoutMirror(artifacts, regexp.MustCompile(`^https://kubevirt.storage.googleapis.com/.+`))
	if len(invalid) != 3 {
		t.Fatalf("expect 3 invalid artifacts, found %v", len(invalid))
	}
}

type MockHTTPClient struct {
	responses MockResponses
}

func (m MockHTTPClient) Get(uri string) (resp *http.Response, err error) {
	response, exists := m.responses[uri]
	if !exists {
		return nil, fmt.Errorf("Unexpected url call for Get: %s", uri)
	}
	return response.resp, response.err
}
func (m MockHTTPClient) Head(uri string) (resp *http.Response, err error) {
	response, exists := m.responses[uri]
	if !exists {
		return nil, fmt.Errorf("Unexpected url call for Head: %s", uri)
	}
	return response.resp, response.err
}

type MockResponse struct {
	resp *http.Response
	err  error
}

type MockResponses map[string]MockResponse

type testCaseDataForTestRemoveStaleDownloadURLS struct {
	data           []byte
	responses      MockResponses
	expectedLength int
}

var testCasesForTestRemoveStaleDownloadURLS = []testCaseDataForTestRemoveStaleDownloadURLS{
	{
		data: []byte(`
rpm(
    name = "vim-minimal-2__8.2.2146-2.fc32.x86_64",
    sha256 = "1cf36a5d4a96954167ebd75ca34a21b0b6fd00a7935820528b515ab936ee6393",
    urls = [
        "https://mirror.ette.biz/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://kubevirt.storage.googleapis.com/builddeps/1cf36a5d4a96954167ebd75ca34a21b0b6fd00a7935820528b515ab936ee6393",
    ],
)
`),
		responses: MockResponses{
			"https://mirror.ette.biz/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm":
			MockResponse{
				resp: &http.Response{
					StatusCode: 404,
					Body:       http.NoBody,
				},
			},
		},
		expectedLength: 1,
	},
	{
		data: []byte(`
rpm(
    name = "findutils-1__4.7.0-4.fc32.x86_64",
    sha256 = "c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://kubevirt.storage.googleapis.com/builddeps/c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
    ],
)
`),
		responses: MockResponses{
			"https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm":
			MockResponse{
				resp: &http.Response{
					StatusCode: 200,
					Body:       http.NoBody,
				},
			},
		},
		expectedLength: 2,
	},
	{
		data: []byte(`
rpm(
    name = "findutils-1__4.7.0-4.fc32.x86_64",
    sha256 = "c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
)
`),
		responses: MockResponses{
			"https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm":
			MockResponse{
				resp: &http.Response{
					StatusCode: 200,
					Body:       http.NoBody,
				},
			},
		},
		expectedLength: 0,
	},
}

func TestRemoveStaleDownloadURLS(t *testing.T) {
	for _, workspaceData := range testCasesForTestRemoveStaleDownloadURLS {
		mockHTTPClient := MockHTTPClient{
			responses: workspaceData.responses,
		}
		file, err := build.ParseWorkspace("workspace", workspaceData.data)
		if err != nil {
			t.Fatal(err)
		}
		artifacts, err := GetArtifacts(file)
		if err != nil {
			t.Fatal(err)
		}

		RemoveStaleDownloadURLS(artifacts, regexp.MustCompile("^https://kubevirt.storage.googleapis.com/.+"), mockHTTPClient)
		if len(artifacts[0].URLs()) != workspaceData.expectedLength {
			t.Fatalf("expected length was %d, actual was %d, URLS: %v", workspaceData.expectedLength, len(artifacts[0].URLs()), artifacts[0].URLs())
		}
	}
}
