package platform

import (
	"context"
	"fmt"
	middleware "github.com/heyinLab/common/pkg/middleware/grpc"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	v1 "github.com/heyinLab/common/api/gen/go/merchant/v1"
	"google.golang.org/grpc"
)

// Client 平台服务客户端
//
// 聚合了所有平台相关的服务客户端，提供统一的访问入口
//
// 当前支持的服务：
// - IAM: 身份认证和权限管理服务
//
// 使用示例:
//
//	client, err := platform.NewClientWithDiscovery(
//	    platform.DefaultConfig(),
//	    consulDiscovery,
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// 使用 IAM 服务
//	tree, err := client.IAM().GetTenantPermissionsTree(ctx, &platform.GetTenantPermissionsTreeOptions{
//	    Status: "GA",
//	})
type Client struct {
	config *Config
	conn   *grpc.ClientConn
	logger *log.Helper

	// 子服务客户端
	iamClient *IAMClient
}

// NewClient 创建平台服务客户端（直连方式）
//
// 参数:
//   - config: 客户端配置，可以使用 DefaultConfig() 获取默认配置
//
// 返回:
//   - *Client: 客户端实例
//   - error: 创建失败时的错误信息
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	logger := log.NewHelper(log.With(
		log.GetLogger(),
		"module", "platform-client",
	))

	conn, err := middleware.CreateGRPCConn(config, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	return &Client{
		config:    config,
		conn:      conn,
		logger:    logger,
		iamClient: newIAMClient(conn, logger),
	}, nil
}

// NewClientWithDiscovery 创建带服务发现的平台服务客户端
//
// 参数:
//   - config: 客户端配置
//   - discovery: 服务发现实例（如 Consul）
//
// 返回:
//   - *Client: 客户端实例
//   - error: 创建失败时的错误信息
func NewClientWithDiscovery(config *Config, discovery registry.Discovery) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if discovery == nil {
		return nil, fmt.Errorf("服务发现实例不能为空")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	logger := log.NewHelper(log.With(
		log.GetLogger(),
		"module", "platform-client",
	))

	conn, err := middleware.CreateGRPCConn(config, discovery, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	logger.Infof("平台服务客户端连接成功 (服务发现): endpoint=%s, timeout=%v", config.Endpoint, config.Timeout)

	return &Client{
		config:    config,
		conn:      conn,
		logger:    logger,
		iamClient: newIAMClient(conn, logger),
	}, nil
}

// Close 关闭客户端连接
//
// 释放 gRPC 连接资源，应该在程序退出前调用
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ========== 服务访问器 ==========

// IAM 返回 IAM 服务客户端
//
// 用于访问身份认证和权限管理相关功能
//
// 使用示例:
//
//	tree, err := client.IAM().GetTenantPermissionsTree(ctx, &GetTenantPermissionsTreeOptions{
//	    Status: "GA",
//	})
func (c *Client) IAM() *IAMClient {
	return c.iamClient
}

// ========== IAM 客户端 ==========

// IAMClient IAM 服务客户端
//
// 提供身份认证和权限管理相关功能
type IAMClient struct {
	client v1.MerchantIamServiceClient
	logger *log.Helper
}

// newIAMClient 创建 IAM 客户端
func newIAMClient(conn *grpc.ClientConn, logger *log.Helper) *IAMClient {
	return &IAMClient{
		client: v1.NewMerchantIamServiceClient(conn),
		logger: logger,
	}
}

// SetTenantPermissions 将权限代码列表下发到租户
//
// 参数:
//   - ctx: 上下文
//   - codes: 权限代码列表
//
// 返回:
//   - *v1.SetTenantPermissionsResponse: 返回结果，包含是否成功及总数
//   - error: 调用失败的错误
func (c *IAMClient) SetTenantPermissions(ctx context.Context, tenantCode string, codes []string) (*v1.SetTenantPermissionsResponse, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("权限代码列表不能为空")
	}

	resp, err := c.client.SetTenantPermissions(ctx, &v1.SetTenantPermissionsRequest{Codes: codes, TenantCode: &tenantCode})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("设置租户权限失败, codes=%v, err=%v", codes, err)
		return nil, err
	}

	return resp, nil
}

type ListTenantOptions struct {
	Name        *string          // 名称
	Status      *v1.TenantStatus // 状态
	Country     *string          // 国家
	Type        *v1.TenantType   // 类型
	AccessLevel *v1.AccessLevel  // 访问等级
}

// 获取租户列表
func (c *IAMClient) ListTenant(ctx context.Context, page, limit int32, opt *ListTenantOptions) (*v1.InternalListTenantResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 20 {
		limit = 20
	}
	req := &v1.InternalListTenantRequest{
		Page:  page,
		Limit: limit,
	}
	if opt != nil {
		req.Name = opt.Name
		req.Status = opt.Status
		req.Country = opt.Country
		req.Type = opt.Type
	}
	resp, err := c.client.InternalListTenant(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取租户列表失败, opt=%v, err=%v", opt, err)
		return nil, err
	}

	return resp, nil
}

// ========== 辅助函数 ==========

// getStringValue 获取指针字符串的值
func getStringValue(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}
