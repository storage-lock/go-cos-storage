package cos_storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	"github.com/stretchr/testify/assert"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type takeErrorManager struct{}

func (takeErrorManager) Name() string { return "take-error" }
func (takeErrorManager) Take(context.Context) (*cos.Client, error) {
	return nil, errors.New("take failed")
}
func (takeErrorManager) Return(context.Context, *cos.Client) error { return nil }
func (takeErrorManager) Shutdown(context.Context) error            { return nil }

func TestCOSConnectionManager(t *testing.T) {
	preset := &cos.Client{}
	fromClient := NewCOSConnectionManagerFromClient(preset)
	assert.Equal(t, COSConnectionManagerName, fromClient.Name())
	got, err := fromClient.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, preset, got)
	assert.NoError(t, fromClient.Return(context.Background(), got))
	assert.NoError(t, fromClient.Shutdown(context.Background()))

	fromConfig := NewCOSConnectionManagerFromConfig("https://bucket.cos.ap-guangzhou.myqcloud.com", "id", "key")
	got, err = fromConfig.Take(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, got)
	again, err := fromConfig.Take(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, got, again)
}

func TestCOSStorage_Branches(t *testing.T) {
	assert.Panics(t, func() { _, _ = NewCOSStorage(context.Background(), nil) })
	_, err := NewCOSStorage(context.Background(), NewCOSStorageOptions())
	assert.ErrorIs(t, err, ErrConnectionManagerNil)
	opts := NewCOSStorageOptions().SetConnectionManager(takeErrorManager{})
	opts.KeyPrefix = ""
	assert.ErrorIs(t, opts.Check(), ErrKeyPrefixEmpty)

	_, err = NewCOSStorage(context.Background(), &COSStorageOptions{ConnectionManager: takeErrorManager{}, KeyPrefix: "locks"})
	assert.EqualError(t, err, "take failed")

	s, err := NewCOSStorage(context.Background(), NewCOSStorageOptions().
		SetConnectionManager(NewCOSConnectionManagerFromClient(&cos.Client{})).
		SetTimeProvider(&LocalTimeProvider{}))
	assert.NoError(t, err)
	assert.Equal(t, COSStorageName, s.GetName())
	assert.Len(t, s.Capabilities(), 2)
	assert.ErrorIs(t, s.CreateWithVersion(context.Background(), "lock", 1, &storage.LockInformation{}), storage_lock.ErrStorageCapabilityMissing)
	assert.ErrorIs(t, s.UpdateWithVersion(context.Background(), "lock", 1, 2, &storage.LockInformation{}), storage_lock.ErrStorageCapabilityMissing)
	assert.ErrorIs(t, s.DeleteWithVersion(context.Background(), "lock", 1, &storage.LockInformation{}), storage_lock.ErrStorageCapabilityMissing)
	_, err = s.Get(context.Background(), "lock")
	assert.ErrorIs(t, err, storage_lock.ErrStorageCapabilityMissing)
	now, err := s.GetTime(context.Background())
	assert.NoError(t, err)
	assert.False(t, now.IsZero())
	listed, err := s.List(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, listed)
	assert.NoError(t, s.Close(context.Background()))

	plain, err := NewCOSStorage(context.Background(), &COSStorageOptions{
		ConnectionManager: NewCOSConnectionManagerFromClient(&cos.Client{}),
		KeyPrefix:         "locks",
	})
	assert.NoError(t, err)
	now, err = plain.GetTime(context.Background())
	assert.NoError(t, err)
	assert.WithinDuration(t, time.Now(), now, time.Second)
	_, err = (&COSStorage{}).Watch(context.Background(), "")
	assert.Error(t, err)
	localNow, err := (&LocalTimeProvider{}).GetTime(context.Background())
	assert.NoError(t, err)
	assert.False(t, localNow.IsZero())
	plain.timeProvider = nil
	now, err = plain.GetTime(context.Background())
	assert.NoError(t, err)
	assert.False(t, now.IsZero())
}
