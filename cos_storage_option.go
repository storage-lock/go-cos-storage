package cos_storage

import (
	"github.com/storage-lock/go-storage"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// COSStorageOptions 腾讯云 COS 存储的选项
type COSStorageOptions struct {
	ConnectionManager storage.ConnectionManager[*cos.Client]
	KeyPrefix         string
	TimeProvider      storage.TimeProvider
}

func NewCOSStorageOptions() *COSStorageOptions {
	return &COSStorageOptions{
		KeyPrefix: storage.DefaultStorageTableName,
	}
}

func (x *COSStorageOptions) SetConnectionManager(cm storage.ConnectionManager[*cos.Client]) *COSStorageOptions {
	x.ConnectionManager = cm
	return x
}

func (x *COSStorageOptions) SetKeyPrefix(keyPrefix string) *COSStorageOptions {
	x.KeyPrefix = keyPrefix
	return x
}

func (x *COSStorageOptions) SetTimeProvider(tp storage.TimeProvider) *COSStorageOptions {
	x.TimeProvider = tp
	return x
}

func (x *COSStorageOptions) Check() error {
	if x.KeyPrefix == "" {
		return ErrKeyPrefixEmpty
	}
	if x.ConnectionManager == nil {
		return ErrConnectionManagerNil
	}
	return nil
}