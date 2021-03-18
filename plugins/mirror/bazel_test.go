package mirror

import (
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

type testDataForTestRemoveStaleDownloadURLS struct {
	Data           []byte
	ExpectedLength int
}

// FIXME: workspaceDataFoRPMWithStaleDownloadURLs below can get stale as links might get outdated, which will cause tests to fail. Use proper mocking instead
var workspaceDataFoRPMWithStaleDownloadURLs = []testDataForTestRemoveStaleDownloadURLS{
	{
		[]byte(`
rpm(
    name = "vim-minimal-2__8.2.2146-2.fc32.x86_64",
    sha256 = "1cf36a5d4a96954167ebd75ca34a21b0b6fd00a7935820528b515ab936ee6393",
    urls = [
        "https://mirror.ette.biz/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://sjc.edge.kernel.org/fedora-buffet/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://mirror.genesisadaptive.com/fedora/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://mirror.umd.edu/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://kubevirt.storage.googleapis.com/builddeps/1cf36a5d4a96954167ebd75ca34a21b0b6fd00a7935820528b515ab936ee6393",
    ],
)
`),
		1,
	},
	{
		[]byte(`
rpm(
    name = "findutils-1__4.7.0-4.fc32.x86_64",
    sha256 = "c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://kubevirt.storage.googleapis.com/builddeps/c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
    ],
)
`),
		5,
	},
}

func TestRemoveStaleDownloadURLS(t *testing.T) {
	for _, workspaceData := range workspaceDataFoRPMWithStaleDownloadURLs {
		file, err := build.ParseWorkspace("workspace", workspaceData.Data)
		if err != nil {
			t.Fatal(err)
		}
		artifacts, err := GetArtifacts(file)
		if err != nil {
			t.Fatal(err)
		}

		RemoveStaleDownloadURLS(artifacts, regexp.MustCompile("^https://kubevirt.storage.googleapis.com/.+"))
		if len(artifacts[0].URLs()) != workspaceData.ExpectedLength {
			t.Fatalf("expected length was %d, actual was %d, URLS: %v", workspaceData.ExpectedLength, len(artifacts[0].URLs()), artifacts[0].URLs())
		}
	}
}
