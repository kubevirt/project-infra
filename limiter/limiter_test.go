package limiter

import (
	"context"
	"strings"
	"testing"

	"cloud.google.com/go/iam"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

const (
	bucketName = "kubevirtest"
	credentialsPath = "testaccount.json"
)

func Test(t *testing.T) {

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsPath))
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

			if "allUsers" == user {
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
