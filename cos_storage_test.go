package cos_storage

import (
	"context"
	"os"
	"testing"

	storage_test_helper "github.com/storage-lock/go-storage-test-helper"
	"github.com/stretchr/testify/assert"
)

func TestCOSStorage(t *testing.T) {
	envName := "STORAGE_LOCK_COS_BUCKET_URL"
	bucketURL := os.Getenv(envName)
	if bucketURL == "" {
		t.Skip("STORAGE_LOCK_COS_BUCKET_URL not set, skipping COS test")
	}

	secretID := os.Getenv("STORAGE_LOCK_COS_SECRET_ID")
	secretKey := os.Getenv("STORAGE_LOCK_COS_SECRET_KEY")

	ctx := context.Background()
	cm := NewCOSConnectionManagerFromConfig(bucketURL, secretID, secretKey)
	s, err := NewCOSStorage(ctx, &COSStorageOptions{
		ConnectionManager: cm,
		KeyPrefix:         storage_test_helper.TestTableName,
		TimeProvider:      &LocalTimeProvider{},
	})
	assert.Nil(t, err)
	storage_test_helper.TestStorage(t, s)
}