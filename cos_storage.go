package cos_storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	go_iterator "github.com/golang-infrastructure/go-iterator"
	"github.com/storage-lock/go-storage"
	storage_lock "github.com/storage-lock/go-storage-lock"
	"github.com/tencentyun/cos-go-sdk-v5"
)

const cosValueSeparator = "\x00"

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
	return []storage.StorageCapability{
		storage.CapabilityCAS,
		storage.CapabilityReliableTime,
	}
}

func (x *COSStorage) Init(ctx context.Context) error {
	client, err := x.options.ConnectionManager.Take(ctx)
	if err != nil {
		return err
	}
	x.client = client

	if x.options.TimeProvider != nil {
		x.timeProvider = x.options.TimeProvider
	} else {
		x.timeProvider = &LocalTimeProvider{}
	}
	return nil
}

type LocalTimeProvider struct{}

func (p *LocalTimeProvider) GetTime(ctx context.Context) (time.Time, error) {
	return time.Now(), nil
}

func (x *COSStorage) objectKey(lockId string) string {
	return fmt.Sprintf("%s/lock/%s", x.options.KeyPrefix, lockId)
}

func cosEncodeValue(version storage.Version, jsonStr string) []byte {
	return []byte(strconv.FormatUint(uint64(version), 10) + cosValueSeparator + jsonStr)
}

func cosDecodeValue(data []byte) (storage.Version, string, error) {
	parts := strings.SplitN(string(data), cosValueSeparator, 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid cos value format")
	}
	version, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", err
	}
	return storage.Version(version), parts[1], nil
}

// CreateWithVersion COS 条件 PUT
func (x *COSStorage) CreateWithVersion(ctx context.Context, lockId string, version storage.Version, lockInformation *storage.LockInformation) error {
	key := x.objectKey(lockId)
	value := cosEncodeValue(version, lockInformation.ToJsonString())

	// 使用 XOptionHeader 添加条件头部
	header := http.Header{}
	header.Set("x-cos-forbid-overwrite", "true")
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			XOptionHeader: &header,
		},
	}
	_, err := x.client.Object.Put(ctx, key, bytes.NewReader(value), opt)
	if err != nil {
		if cosIsPreconditionError(err) {
			return storage_lock.ErrVersionMiss
		}
		return err
	}
	return nil
}

// UpdateWithVersion 条件更新
func (x *COSStorage) UpdateWithVersion(ctx context.Context, lockId string, expectedVersion, newVersion storage.Version, lockInformation *storage.LockInformation) error {
	key := x.objectKey(lockId)

	// 获取当前对象和 ETag
	resp, err := x.client.Object.Head(ctx, key, nil)
	if err != nil {
		if cosIsNotFoundError(err) {
			return storage_lock.ErrLockNotFound
		}
		return err
	}
	etag := resp.Header.Get("ETag")

	// 获取内容
	resp2, err := x.client.Object.Get(ctx, key, nil)
	if err != nil {
		if cosIsNotFoundError(err) {
			return storage_lock.ErrLockNotFound
		}
		return err
	}
	data, err := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if err != nil {
		return err
	}

	// 检查版本
	currentVersion, _, err := cosDecodeValue(data)
	if err != nil {
		return err
	}
	if currentVersion != expectedVersion {
		return storage_lock.ErrVersionMiss
	}

	// 条件更新
	newValue := cosEncodeValue(newVersion, lockInformation.ToJsonString())
	header := http.Header{}
	header.Set("If-Match", etag)
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			XOptionHeader: &header,
		},
	}
	_, err = x.client.Object.Put(ctx, key, bytes.NewReader(newValue), opt)
	if err != nil {
		if cosIsPreconditionError(err) {
			return storage_lock.ErrVersionMiss
		}
		return err
	}
	return nil
}

// DeleteWithVersion 降级删除
func (x *COSStorage) DeleteWithVersion(ctx context.Context, lockId string, expectedVersion storage.Version, lockInformation *storage.LockInformation) error {
	key := x.objectKey(lockId)

	resp, err := x.client.Object.Get(ctx, key, nil)
	if err != nil {
		if cosIsNotFoundError(err) {
			return storage_lock.ErrLockNotFound
		}
		return err
	}
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	currentVersion, _, err := cosDecodeValue(data)
	if err != nil {
		return err
	}
	if currentVersion != expectedVersion {
		return storage_lock.ErrVersionMiss
	}

	_, err = x.client.Object.Delete(ctx, key)
	return err
}

func (x *COSStorage) Get(ctx context.Context, lockId string) (string, error) {
	key := x.objectKey(lockId)
	resp, err := x.client.Object.Get(ctx, key, nil)
	if err != nil {
		if cosIsNotFoundError(err) {
			return "", storage_lock.ErrLockNotFound
		}
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", storage_lock.ErrLockNotFound
	}
	_, jsonStr, err := cosDecodeValue(data)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func (x *COSStorage) GetTime(ctx context.Context) (time.Time, error) {
	return x.timeProvider.GetTime(ctx)
}

func (x *COSStorage) Close(ctx context.Context) error {
	x.client = nil
	return nil
}

func (x *COSStorage) List(ctx context.Context) (go_iterator.Iterator[*storage.LockInformation], error) {
	prefix := fmt.Sprintf("%s/lock/", x.options.KeyPrefix)
	slice := make([]*storage.LockInformation, 0)

	marker := ""
	for {
		opt := &cos.BucketGetOptions{
			Prefix:  prefix,
			Marker:  marker,
			MaxKeys: 1000,
		}
		result, _, err := x.client.Bucket.Get(ctx, opt)
		if err != nil {
			return nil, err
		}

		for _, obj := range result.Contents {
			resp, err := x.client.Object.Get(ctx, obj.Key, nil)
			if err != nil {
				continue
			}
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || len(data) == 0 {
				continue
			}
			_, jsonStr, err := cosDecodeValue(data)
			if err != nil {
				continue
			}
			lockInfo, err := storage.LockInformationFromJsonString(jsonStr)
			if err != nil {
				continue
			}
			slice = append(slice, lockInfo)
		}

		if !result.IsTruncated {
			break
		}
		marker = result.NextMarker
	}

	return go_iterator.FromSlice(slice), nil
}

func cosIsPreconditionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "PreconditionFailed") ||
		strings.Contains(errStr, "412") ||
		strings.Contains(errStr, "condition") ||
		strings.Contains(errStr, "Conflict")
}

func cosIsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "NoSuchKey") ||
		strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "does not exist")
}