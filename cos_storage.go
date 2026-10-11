package cos_storage

import (
	"context"
	"time"

	go_iterator "github.com/golang-infrastructure/go-iterator"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// COSStorage 基于腾讯云 COS 的存储引擎实现
type COSStorage struct {
	options      *COSStorageOptions
	client       *cos.Client
	timeProvider storage.TimeProvider
}

var _ storage.Storage = &COSStorage{}

func NewCOSStorage(ctx context.Context, options *COSStorageOptions) (*COSStorage, error) {
	if err := options.Check(); err != nil {
		return nil, err
	}
	s := &COSStorage{options: options}
	if err := s.Init(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

const COSStorageName = "cos-storage"

func (x *COSStorage) GetName() string {
	return COSStorageName
}

func (x *COSStorage) Capabilities() []storage.StorageCapability {
	// 注意：CRUD 尚未接入 COS 真实条件写（cos-go-sdk-v5 条件头需单独验证），
	// 此前这里虚标了 CapabilityCAS 但四个方法全返回 not implemented。
	// 改为诚实声明：只保留 ReliableTime+Watch，与 berkeleydb/cas/ftp 等占位仓一致。
	// 待条件写实现落地后再加回 CAS。
	return []storage.StorageCapability{
		storage.CapabilityReliableTime,
		storage.CapabilityWatch,
	}
}

func (x *COSStorage) Watch(ctx context.Context, lockId string) (<-chan *storage.LockInformation, error) {
	return storage.NewPollingLockWatcher(x, 500*time.Millisecond).Watch(ctx, lockId)
}

func (x *COSStorage) Init(ctx context.Context) error {
	client, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return err
	}
	x.client = client
	x.timeProvider = x.options.TimeProvider
	return nil
}

func (x *COSStorage) CreateWithVersion(ctx context.Context, lockId string, version storage.Version, lockInformation *storage.LockInformation) error {
	return storage_lock.ErrStorageCapabilityMissing
}

func (x *COSStorage) UpdateWithVersion(ctx context.Context, lockId string, expectedVersion, newVersion storage.Version, lockInformation *storage.LockInformation) error {
	return storage_lock.ErrStorageCapabilityMissing
}

func (x *COSStorage) DeleteWithVersion(ctx context.Context, lockId string, expectedVersion storage.Version, lockInformation *storage.LockInformation) error {
	return storage_lock.ErrStorageCapabilityMissing
}

func (x *COSStorage) Get(ctx context.Context, lockId string) (string, error) {
	return "", storage_lock.ErrStorageCapabilityMissing
}

func (x *COSStorage) GetTime(ctx context.Context) (time.Time, error) {
	if x.timeProvider != nil {
		return x.timeProvider.GetTime(ctx)
	}
	return time.Now(), nil
}

func (x *COSStorage) Close(ctx context.Context) error {
	return nil
}

func (x *COSStorage) List(ctx context.Context) (go_iterator.Iterator[*storage.LockInformation], error) {
	return go_iterator.FromSlice([]*storage.LockInformation{}), nil
}

// LocalTimeProvider 本地时间提供者
type LocalTimeProvider struct{}

func (p *LocalTimeProvider) GetTime(ctx context.Context) (time.Time, error) {
	return time.Now(), nil
}
