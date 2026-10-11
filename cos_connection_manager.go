package cos_storage

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/storage-lock/go-storage"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// COSConnectionManager 管理 COS 客户端连接
type COSConnectionManager struct {
	BucketURL    string
	SecretID     string
	SecretKey    string

	presetClient *cos.Client

	initMu      sync.Mutex
	initialized bool
	client      *cos.Client
}

const COSConnectionManagerName = "cos-connection-manager"

var _ storage.ConnectionManager[*cos.Client] = &COSConnectionManager{}

// NewCOSConnectionManagerFromConfig 从配置创建连接管理器
func NewCOSConnectionManagerFromConfig(bucketURL, secretID, secretKey string) *COSConnectionManager {
	return &COSConnectionManager{
		BucketURL: bucketURL,
		SecretID:  secretID,
		SecretKey: secretKey,
	}
}

// NewCOSConnectionManagerFromClient 从已有客户端创建连接管理器
func NewCOSConnectionManagerFromClient(client *cos.Client) *COSConnectionManager {
	return &COSConnectionManager{
		presetClient: client,
		initialized:  true,
		client:       client,
	}
}

func (x *COSConnectionManager) Name() string {
	return COSConnectionManagerName
}

func (x *COSConnectionManager) Take(ctx context.Context) (*cos.Client, error) {
	if x.presetClient != nil {
		return x.presetClient, nil
	}
	x.initMu.Lock()
	defer x.initMu.Unlock()
	if x.client != nil {
		return x.client, nil
	}
	u, _ := url.Parse(x.BucketURL)
	x.client = cos.NewClient(&cos.BaseURL{BucketURL: u}, http.DefaultClient)
	return x.client, nil
}

func (x *COSConnectionManager) Return(ctx context.Context, connection *cos.Client) error {
	return nil
}

func (x *COSConnectionManager) Shutdown(ctx context.Context) error {
	return nil
}
