package limiter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/storage"
)

type BillingAlert struct {
	Data []byte `json:"data"`
}

// https://cloud.google.com/billing/docs/how-to/budgets#manage-notifications
type BillingInfo struct {
	AlertThresholdExceeded float32 `json:"alertThresholdExceeded"`
}

// CutBucketConnections consumes a Pub/Sub message.
func CutBucketConnections(ctx context.Context, alert BillingAlert) error {
	rawBuckets := os.Getenv("GOOGLE_CLOUD_BUCKETS")
	if rawBuckets == "" {
		return fmt.Errorf("GOOGLE_CLOUD_BUCKETS environment variable must be set")
	}
	buckets := strings.Split(rawBuckets, ";")
	for i, bucket := range buckets {
		buckets[i] = strings.TrimSpace(bucket)
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain a gce client: %v", err)
	}

	info := &BillingInfo{}
	if err := json.Unmarshal(alert.Data, info); err != nil {
		return fmt.Errorf("failed to unmarshal billing info: %v", err)
	}

	if info.AlertThresholdExceeded < 1 {
		fmt.Printf("Alert threshold is not yet exceeded: %v\n", info.AlertThresholdExceeded)
		return nil
	} else {
		return cutBucketConnections(client, buckets)
	}
}

func cutBucketConnections(client *storage.Client, buckets []string) error {

	for  _, bucket := range buckets {
		err := removeUsers(client, bucket)
		if err != nil {
			return fmt.Errorf("failed to remove users from bucket %v: %v", bucket, err)
		}
	}
	return nil
}

func removeUsers(c *storage.Client, bucketName string) error {
	ctx := context.Background()

	bucket := c.Bucket(bucketName)
	policy, err := bucket.IAM().Policy(ctx)
	if err != nil {
		return err
	}

	for _, role := range policy.Roles() {
		users := policy.Members(role)
		for _, user := range users {
			if strings.HasPrefix(user, "serviceAccount") || user == "allUsers" {
				log.Printf("Removing role %s from user %s", role, user)
				policy.Remove(user, role)
			}
		}
	}
	if err := bucket.IAM().SetPolicy(ctx, policy); err != nil {
		return err
	}
	return nil
}
