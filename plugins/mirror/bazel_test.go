package main

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
	if len(artifacts) != 4 {
		t.Fatalf("expect 4 artifacts, found %v", len(artifacts))
	}

	invalid := FilterArtifactsWithoutMirror(artifacts, regexp.MustCompile(`^https://kubevirt.storage.googleapis.com/.+`))
	if len(invalid) != 2 {
		t.Fatalf("expect 2 invalid artifacts, found %v", len(artifacts))
	}
	invalid[0].AppendURL("test")
}
