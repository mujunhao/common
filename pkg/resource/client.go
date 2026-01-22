package resource

import (
	"context"
	"fmt"

	middleware "github.com/heyinLab/common/pkg/middleware/grpc"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	v1 "github.com/heyinLab/common/api/gen/go/resource/v1"
	"google.golang.org/grpc"
)

// ResourceClient 资源服务内部客户端
//
// 封装了 ResourceInternalService 的 gRPC 调用，供内部微服务使用
//
// 使用示例:
//
//	client, err := resource.NewResourceClientWithDiscovery(
//	    resource.DefaultInternalConfig(),
//	    consulDiscovery,
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// 获取文件信息
//	file, err := client.GetFile(ctx, tenantCode, fileID)
type ResourceClient struct {
	config *InternalConfig
	conn   *grpc.ClientConn
	client v1.ResourceInternalServiceClient
	logger *log.Helper
}

// NewResourceClient 创建资源服务内部客户端（直连方式）
//
// 参数:
//   - config: 客户端配置，可以使用 DefaultInternalConfig() 获取默认配置
//
// 返回:
//   - *ResourceClient: 客户端实例
//   - error: 创建失败时的错误信息
//
// 使用示例:
//
//	config := resource.DefaultInternalConfig().
//	    WithEndpoint("localhost:9000")
//	client, err := resource.NewResourceClient(config)
func NewResourceClient(config *InternalConfig) (*ResourceClient, error) {
	if config == nil {
		config = DefaultInternalConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	logger := log.NewHelper(log.With(
		log.GetLogger(),
		"module", "resource-internal-client",
	))

	conn, err := middleware.CreateGRPCConn(config, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	return &ResourceClient{
		config: config,
		conn:   conn,
		client: v1.NewResourceInternalServiceClient(conn),
		logger: logger,
	}, nil
}

// NewResourceClientWithDiscovery 创建带服务发现的资源服务内部客户端
//
// 参数:
//   - config: 客户端配置
//   - discovery: 服务发现实例（如 Consul）
//
// 返回:
//   - *ResourceClient: 客户端实例
//   - error: 创建失败时的错误信息
//
// 使用示例:
//
//	// 创建 Consul 服务发现
//	consulClient, _ := consul.New(consulAPI)
//
//	config := resource.DefaultInternalConfig()
//	client, err := resource.NewResourceClientWithDiscovery(config, consulClient)
func NewResourceClientWithDiscovery(config *InternalConfig, discovery registry.Discovery) (*ResourceClient, error) {
	if config == nil {
		config = DefaultInternalConfig()
	}

	if discovery == nil {
		return nil, fmt.Errorf("服务发现实例不能为空")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	logger := log.NewHelper(log.With(
		log.GetLogger(),
		"module", "resource-internal-client",
	))

	conn, err := middleware.CreateGRPCConn(config, discovery, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	logger.Infof("资源内部服务客户端连接成功 (服务发现): endpoint=%s, timeout=%v", config.Endpoint, config.Timeout)

	return &ResourceClient{
		config: config,
		conn:   conn,
		client: v1.NewResourceInternalServiceClient(conn),
		logger: logger,
	}, nil
}

// Close 关闭客户端连接
func (c *ResourceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ========== 文件相关接口 ==========

// GetFile 获取单个文件信息
//
// 参数:
//   - ctx: 上下文
//   - TenantCode: 租户ID
//   - fileID: 文件ID
//
// 返回:
//   - *v1.InternalFileInfo: 文件信息
//   - error: 错误信息
func (c *ResourceClient) GetFile(ctx context.Context, tenantCode string, fileID string) (*v1.InternalFileInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalGetFile(ctx, &v1.InternalGetFileRequest{
		TenantCode: tenantCode,
		FileId:     fileID,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取文件信息失败: tenant_id=%d, file_id=%s, error=%v", tenantCode, fileID, err)
		return nil, err
	}

	return resp.File, nil
}

// GetFiles 批量获取文件信息
//
// 参数:
//   - ctx: 上下文
//   - TenantCode: 租户ID
//   - fileIDs: 文件ID列表（最多100个）
//
// 返回:
//   - map[string]*v1.InternalFileInfo: 文件ID到文件信息的映射
//   - []string: 获取失败的文件ID列表
//   - error: 错误信息
func (c *ResourceClient) GetFiles(ctx context.Context, tenantCode string, fileIDs []string) (map[string]*v1.InternalFileInfo, []string, error) {
	if len(fileIDs) == 0 {
		return make(map[string]*v1.InternalFileInfo), nil, nil
	}

	if len(fileIDs) > 100 {
		return nil, nil, fmt.Errorf("文件ID数量不能超过100个，当前: %d", len(fileIDs))
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalGetFiles(ctx, &v1.InternalGetFilesRequest{
		TenantCode: tenantCode,
		FileIds:    fileIDs,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("批量获取文件信息失败: tenant_id=%d, count=%d, error=%v", tenantCode, len(fileIDs), err)
		return nil, nil, err
	}

	return resp.Files, resp.FailedIds, nil
}

// GetFileUrlsOptions 获取文件URL的选项
type GetFileUrlsOptions struct {
	// 是否包含变体URL（如缩略图）
	IncludeVariants bool
	// URL有效期（秒），默认3600
	ExpiresIn int64
}

// GetFileUrls 批量获取文件URL
//
// 参数:
//   - ctx: 上下文
//   - fileIDs: 文件ID列表（最多100个）
//   - opts: 可选参数
//
// 返回:
//   - map[string]*v1.InternalFileUrlInfo: 文件ID到URL信息的映射
//   - error: 错误信息
//
// 说明:
//   - URL查询不需要租户隔离，支持平台级资源与租户资源混合使用
//   - 租户隔离在下载时由其他接口处理
func (c *ResourceClient) GetFileUrls(ctx context.Context, fileIDs []string, opts *GetFileUrlsOptions) (map[string]*v1.InternalFileUrlInfo, error) {
	if len(fileIDs) == 0 {
		return make(map[string]*v1.InternalFileUrlInfo), nil
	}

	if len(fileIDs) > 100 {
		return nil, fmt.Errorf("文件ID数量不能超过100个，当前: %d", len(fileIDs))
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	req := &v1.InternalGetFileUrlsRequest{
		FileIds: fileIDs,
	}

	if opts != nil {
		req.IncludeVariants = opts.IncludeVariants
		req.ExpiresIn = opts.ExpiresIn
	}

	resp, err := c.client.InternalGetFileUrls(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("批量获取文件URL失败: count=%d, error=%v", len(fileIDs), err)
		return nil, err
	}

	return resp.Results, nil
}

// GetFileUrl 获取单个文件URL（便捷方法）
//
// 参数:
//   - ctx: 上下文
//   - fileID: 文件ID
//
// 返回:
//   - string: 文件URL
//   - error: 错误信息
func (c *ResourceClient) GetFileUrl(ctx context.Context, fileID string) (string, error) {
	results, err := c.GetFileUrls(ctx, []string{fileID}, nil)
	if err != nil {
		return "", err
	}

	info, ok := results[fileID]
	if !ok || !info.Success {
		errMsg := "文件不存在"
		if ok && info.Error != "" {
			errMsg = info.Error
		}
		return "", fmt.Errorf("获取文件URL失败: %s", errMsg)
	}

	return info.Url, nil
}

// DownloadFileRequest 下载文件请求
type DownloadFileRequest struct {
	// 文件ID（必填）
	FileID string
	// 自定义下载文件名（可选）
	DownloadFilename string
	// 要下载的变体ID（可选）
	VariantID string
}

// GetDownloadUrls 批量获取下载URL
//
// 参数:
//   - ctx: 上下文
//   - TenantCode: 租户ID
//   - files: 下载文件请求列表（最多50个）
//   - expiresIn: URL有效期（秒），默认3600
//
// 返回:
//   - map[string]*v1.InternalFileDownloadInfo: 文件ID到下载信息的映射
//   - error: 错误信息
func (c *ResourceClient) GetDownloadUrls(ctx context.Context, tenantCode string, files []DownloadFileRequest, expiresIn int64) (map[string]*v1.InternalFileDownloadInfo, error) {
	if len(files) == 0 {
		return make(map[string]*v1.InternalFileDownloadInfo), nil
	}

	if len(files) > 50 {
		return nil, fmt.Errorf("文件数量不能超过50个，当前: %d", len(files))
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// 转换请求
	protoFiles := make([]*v1.InternalFileDownloadRequest, len(files))
	for i, f := range files {
		protoFiles[i] = &v1.InternalFileDownloadRequest{
			FileId:           f.FileID,
			DownloadFilename: f.DownloadFilename,
			VariantId:        f.VariantID,
		}
	}

	resp, err := c.client.InternalGetDownloadUrls(ctx, &v1.InternalGetDownloadUrlsRequest{
		TenantCode: tenantCode,
		Files:      protoFiles,
		ExpiresIn:  expiresIn,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("批量获取下载URL失败: tenant_id=%d, count=%d, error=%v", tenantCode, len(files), err)
		return nil, err
	}

	return resp.Results, nil
}

// GetDownloadUrl 获取单个文件下载URL（便捷方法）
//
// 参数:
//   - ctx: 上下文
//   - TenantCode: 租户ID
//   - fileID: 文件ID
//
// 返回:
//   - string: 下载URL
//   - error: 错误信息
func (c *ResourceClient) GetDownloadUrl(ctx context.Context, tenantCode string, fileID string) (string, error) {
	results, err := c.GetDownloadUrls(ctx, tenantCode, []DownloadFileRequest{{FileID: fileID}}, 3600)
	if err != nil {
		return "", err
	}

	info, ok := results[fileID]
	if !ok || !info.Success {
		errMsg := "文件不存在"
		if ok && info.Error != "" {
			errMsg = info.Error
		}
		return "", fmt.Errorf("获取下载URL失败: %s", errMsg)
	}

	return info.DownloadUrl, nil
}

// CheckFileExists 检查文件是否存在（秒传检查）
//
// 参数:
//   - ctx: 上下文
//   - TenantCode: 租户ID
//   - checksumSHA256: 文件的SHA256校验和
//   - size: 文件大小（字节，可选但推荐）
//
// 返回:
//   - bool: 文件是否存在
//   - *v1.InternalFileInfo: 已存在的文件信息（如果存在）
//   - error: 错误信息
func (c *ResourceClient) CheckFileExists(ctx context.Context, tenantCode string, checksumSHA256 string, size int64) (bool, *v1.InternalFileInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalCheckFileExists(ctx, &v1.InternalCheckFileExistsRequest{
		TenantCode:     tenantCode,
		ChecksumSha256: checksumSHA256,
		Size:           size,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("检查文件是否存在失败: tenant_id=%d, checksum=%s, error=%v", tenantCode, checksumSHA256, err)
		return false, nil, err
	}

	return resp.Exists, resp.File, nil
}

// ========== 配额相关接口 ==========

// GetQuota 获取租户配额信息
//
// 参数:
//   - ctx: 上下文
//   - TenantCode: 租户ID
//
// 返回:
//   - *v1.InternalQuotaInfo: 配额信息
//   - error: 错误信息
func (c *ResourceClient) GetQuota(ctx context.Context, tenantCode string) (*v1.InternalQuotaInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalGetQuota(ctx, &v1.InternalGetQuotaRequest{
		TenantCode: tenantCode,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取配额信息失败: tenant_id=%d, error=%v", tenantCode, err)
		return nil, err
	}

	return resp.Quota, nil
}

// CheckQuotaType 配额检查类型
type CheckQuotaType string

const (
	CheckQuotaTypeUpload   CheckQuotaType = "upload"   // 上传检查
	CheckQuotaTypeDownload CheckQuotaType = "download" // 下载检查
	CheckQuotaTypeStorage  CheckQuotaType = "storage"  // 存储检查
)

// CheckQuotaResult 配额检查结果
type CheckQuotaResult struct {
	// 是否允许操作
	Allowed bool
	// 不允许的原因
	Reason string
	// 当前配额信息
	Quota *v1.InternalQuotaInfo
}

// CheckQuota 检查配额是否允许操作
//
// 参数:
//   - ctx: 上下文
//   - TenantCode: 租户ID
//   - checkType: 检查类型（upload, download, storage）
//   - size: 预计使用量（字节）
//
// 返回:
//   - *CheckQuotaResult: 检查结果
//   - error: 错误信息
func (c *ResourceClient) CheckQuota(ctx context.Context, tenantCode string, checkType CheckQuotaType, size int64) (*CheckQuotaResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalCheckQuota(ctx, &v1.InternalCheckQuotaRequest{
		TenantCode: tenantCode,
		CheckType:  string(checkType),
		Size:       size,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("检查配额失败: tenant_id=%d, check_type=%s, size=%d, error=%v", tenantCode, checkType, size, err)
		return nil, err
	}

	return &CheckQuotaResult{
		Allowed: resp.Allowed,
		Reason:  resp.Reason,
		Quota:   resp.Quota,
	}, nil
}

// ========== 租户初始化接口 ==========

// InitTenantResult 初始化租户结果
type InitTenantResult struct {
	// 是否成功
	Success bool
	// 创建的存储桶ID
	BucketID string
	// 创建的存储桶名称
	BucketName string
	// 存储配额（字节）
	StorageQuota int64
	// 文件数配额
	FileCountQuota int64
	// 提示信息
	Message string
	// 错误信息（失败时）
	Error string
}

// InitTenant 初始化租户资源
//
// 为新注册的租户创建默认存储桶和配额
//
// 参数:
//   - ctx: 上下文
//   - TenantCode: 租户ID（必填，大于0）
//   - region: 存储区域（可选，默认"sea"）
//     可选值: cn|sea|us|eu
//
// 返回:
//   - *InitTenantResult: 初始化结果
//   - error: 错误信息
//
// 使用场景:
//   - IAM服务在创建租户时调用
//   - 租户首次开通存储服务
//
// 注意:
//   - 一个租户只能初始化一次
//   - 重复调用会返回错误
func (c *ResourceClient) InitTenant(ctx context.Context, tenantCode string, region string) (*InitTenantResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalInitTenant(ctx, &v1.InternalInitTenantRequest{
		TenantCode: tenantCode,
		Region:     region,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("初始化租户失败: tenant_id=%d, region=%s, error=%v", tenantCode, region, err)
		return nil, err
	}

	return &InitTenantResult{
		Success:        resp.Success,
		BucketID:       resp.BucketId,
		BucketName:     resp.BucketName,
		StorageQuota:   resp.StorageQuota,
		FileCountQuota: resp.FileCountQuota,
		Message:        resp.Message,
		Error:          resp.Error,
	}, nil
}
