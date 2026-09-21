package limiter

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/iam"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"google.golang.org/api/option/internaloption"
)

const (
	bucketName      = "kubevirtest"
	credentialsPath = "testaccount.json"
)

func Test(t *testing.T) {
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		t.Skipf("credentials file %s not found", credentialsPath)
	}

	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		CredentialsFile: credentialsPath,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
	})
	if err != nil {
		t.Fatalf("failed to load default credentials: %v", err)
	}

	ctx := context.Background()

	options := []option.ClientOption{
		option.WithAuthCredentials(creds),
		internaloption.EnableNewAuthLibrary(),
	}

	client, err := storage.NewClient(ctx, options...)
	if err != nil {
		t.Fatalf("failed to obtain a gce client: %v", err)
	}

	// add users
	if err := addTestPermissions(client, bucketName); err != nil {
		t.Fatalf("failed to add test users: %v", err)
	}

	// remove users
	if err := cutBucketConnections(client, []string{bucketName}); err != nil {
		t.Fatalf("error: %v", err)
	}

	// check if they are removed
	policy, err := getPolicy(client, bucketName)
	if err != nil {
		t.Fatalf("failed to fetch users for bucket: %v", err)
	}

	for _, role := range policy.Roles() {
		for _, user := range policy.Members(role) {
			if strings.HasPrefix(user, "serviceAccount") {
				t.Fatalf("service account should have been removed: %v", user)
			}

			if user == "allUsers" {
				t.Fatalf("allUsers should have been removed: %v", user)
			}
		}
	}
}

func addTestPermissions(c *storage.Client, bucketName string) error {
	ctx := context.Background()

	bucket := c.Bucket(bucketName)
	policy, err := bucket.IAM().Policy(ctx)
	if err != nil {
		return err
	}
	policy.Add("allUsers", "roles/storage.objectViewer")
	if err := bucket.IAM().SetPolicy(ctx, policy); err != nil {
		return err
	}
	return nil
}

func getPolicy(c *storage.Client, bucketName string) (*iam.Policy, error) {
	ctx := context.Background()

	policy, err := c.Bucket(bucketName).IAM().Policy(ctx)
	if err != nil {
		return nil, err
	}
	return policy, nil
}
