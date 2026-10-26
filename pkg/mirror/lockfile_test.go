package mirror

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const (
	testBucket       = "builddeps"
	testRPMHexSHA256 = "dd11bab2ea0abdfa310362eace871422a003340bf223135626500f8f5a985f6b"
	testRPMIntegrity = "sha256-3RG6suoKvfoxA2LqzocUIqADNAvyIxNWJlAPj1qYX2s="
	testStaleRPMURL  = "http://mirror.stream.centos.org/stale-acl.rpm"
	testLiveRPMURL   = "http://mirror.stream.centos.org/acl.rpm"
)

func TestDecodeSHA256(t *testing.T) {
	t.Parallel()

	bytes31 := bytes.Repeat([]byte{0xab}, 31)
	bytes32 := bytes.Repeat([]byte{0xab}, 32)
	bytes33 := bytes.Repeat([]byte{0xab}, 33)

	tests := []struct {
		name      string
		hexDigest string
		integrity string
		want      string
		wantErr   bool
	}{
		{
			name:      "valid_sri_integrity",
			integrity: testRPMIntegrity,
			want:      testRPMHexSHA256,
		},
		{
			name:      "valid_hex_sha256",
			hexDigest: testRPMHexSHA256,
			want:      testRPMHexSHA256,
		},
		{
			name:    "missing_both_digests",
			wantErr: true,
		},
		{
			name:      "unsupported_integrity_algorithm",
			integrity: "sha512-" + base64.StdEncoding.EncodeToString(bytes32),
			wantErr:   true,
		},
		{
			name:      "invalid_integrity_base64",
			integrity: "sha256-%%%not-base64%%%",
			wantErr:   true,
		},
		{
			name:      "integrity_31_bytes_short_boundary",
			integrity: "sha256-" + base64.StdEncoding.EncodeToString(bytes31),
			wantErr:   true,
		},
		{
			name:      "integrity_33_bytes_long_boundary",
			integrity: "sha256-" + base64.StdEncoding.EncodeToString(bytes33),
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := decodeSHA256(tc.hexDigest, tc.integrity)
			if (err != nil) != tc.wantErr {
				t.Fatalf("decodeSHA256(%q, %q) error = %v, wantErr %v", tc.hexDigest, tc.integrity, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("decodeSHA256(%q, %q) = %q, want %q", tc.hexDigest, tc.integrity, got, tc.want)
			}
		})
	}
}

func TestLoadLockFileErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		missing bool
		content string
	}{
		{
			name:    "missing_file",
			missing: true,
		},
		{
			name:    "malformed_json",
			content: `{"rpms": [`,
		},
		{
			name:    "invalid_rpm_digest",
			content: `{"rpms": [{"name": "acl", "integrity": "sha512-bad", "urls": ["` + testLiveRPMURL + `"]}]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lockPath := filepath.Join(t.TempDir(), "rpms.json")
			if !tc.missing {
				if err := os.WriteFile(lockPath, []byte(tc.content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			if _, _, err := LoadLockFile(lockPath); err == nil {
				t.Fatalf("LoadLockFile(%q) = nil error, want non-nil error", lockPath)
			}
		})
	}
}

func TestLockFileFormatting(t *testing.T) {
	t.Parallel()

	mirrorPattern := regexp.MustCompile(`^https://storage\.googleapis\.com/.+`)
	mockClient := MockHTTPClient{
		responses: MockResponses{
			testStaleRPMURL: {resp: &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}},
			testLiveRPMURL:  {resp: &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}},
		},
	}

	tests := []struct {
		name        string
		unformatted string
		formatted   string
	}{
		{
			name:        "full_config_with_integrity_and_stale_url",
			unformatted: "testdata/rpms_unformatted.json",
			formatted:   "testdata/rpms_formatted.json",
		},
		{
			name:        "minimal_config_with_hex_sha256",
			unformatted: "testdata/rpms_sha256_unformatted.json",
			formatted:   "testdata/rpms_sha256_formatted.json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(tc.unformatted)
			if err != nil {
				t.Fatal(err)
			}
			lockPath := filepath.Join(t.TempDir(), "rpms.json")
			if err := os.WriteFile(lockPath, input, 0644); err != nil {
				t.Fatal(err)
			}

			lf, artifacts, err := LoadLockFile(lockPath)
			if err != nil {
				t.Fatalf("LoadLockFile(%q) unexpected error: %v", lockPath, err)
			}

			for _, a := range FilterArtifactsWithoutMirror(artifacts, mirrorPattern) {
				a.AppendURL(GenerateFilePath(testBucket, "", a))
				a.SetAuthPatterns(nil)
			}
			RemoveStaleDownloadURLS(artifacts, mirrorPattern, mockClient)
			if err := CheckArtifactsHaveURLS(artifacts); err != nil {
				t.Fatalf("CheckArtifactsHaveURLS() unexpected error: %v", err)
			}

			if err := lf.Write(true); err != nil {
				t.Fatalf("lf.Write(true) unexpected error: %v", err)
			}
			if err := lf.Write(false); err != nil {
				t.Fatalf("lf.Write(false) unexpected error: %v", err)
			}

			got, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(tc.formatted)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Errorf("lf.Write(false) =\n%s\nwant:\n%s", string(got), string(want))
			}
		})
	}
}
