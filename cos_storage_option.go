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

// SetTimeProvider 设置时间源。COS 本身没有服务端时钟，默认靠调用方注入
//（生产用 NTP，测试用 LocalTimeProvider）；与 go-oss-storage/go-s3-storage 的
// 同名方法对齐，go-cos-locks 两处调用此方法，缺失会导致下游构建失败。
func (x *COSStorageOptions) SetTimeProvider(tp storage.TimeProvider) *COSStorageOptions {
	x.TimeProvider = tp
	return x
}

func (x *COSStorageOptions) Check() error {
	if x.ConnectionManager == nil {
		return ErrConnectionManagerNil
	}
	if x.KeyPrefix == "" {
		return ErrKeyPrefixEmpty
	}
	return nil
}

