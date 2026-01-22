package subscribe

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	v1 "github.com/heyinLab/common/api/gen/go/subscribe/v1"
	middleware "github.com/heyinLab/common/pkg/middleware/grpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client 订阅服务客户端
// 统一封装订阅管理和配额管理功能
type Client struct {
	config     *Config
	conn       *grpc.ClientConn
	logger     *log.Helper
	grpcClient v1.SubscriptionInternalServiceClient
}

// NewClient 创建订阅服务客户端
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	logger := log.NewHelper(log.With(
		log.GetLogger(),
		"module", "subscribe-client",
	))

	conn, err := middleware.CreateGRPCConn(config, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	return &Client{
		config:     config,
		conn:       conn,
		logger:     logger,
		grpcClient: v1.NewSubscriptionInternalServiceClient(conn),
	}, nil
}

// NewClientWithDiscovery 使用服务发现创建订阅服务客户端
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
		"module", "subscribe-client",
	))

	conn, err := middleware.CreateGRPCConn(config, discovery, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	logger.Infof("订阅服务客户端连接成功: endpoint=%s", config.Endpoint)

	return &Client{
		config:     config,
		conn:       conn,
		logger:     logger,
		grpcClient: v1.NewSubscriptionInternalServiceClient(conn),
	}, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ============================================================================
// 订阅管理方法
// ============================================================================

// GetTenantSubscriptions 获取商家指定产品订阅列表
func (c *Client) GetTenantSubscriptions(ctx context.Context, tenantCode, productCode string) ([]*v1.InternalSubscriptionInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalListSubscriptions(ctx, &v1.InternalListSubscriptionsRequest{
		TenantCode:  &tenantCode,
		ProductCode: &productCode,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取订阅列表失败: tenant=%s, product=%s, err=%v", tenantCode, productCode, err)
		return nil, err
	}

	return resp.Subscriptions, nil
}

// CreateSubscriptionOptions 创建订阅选项
type CreateSubscriptionOptions struct {
	StartDate        *timestamppb.Timestamp // 订阅开始时间
	EndDate          *timestamppb.Timestamp // 订阅结束时间
	AutomaticRenewal bool                   // 是否自动续费
	IsTrial          bool                   // 是否试用
}

// CreateSubscription 创建订阅
func (c *Client) CreateSubscription(ctx context.Context, productCode, planCode string, order *v1.InternalSubscriptionOrderInfo, opts *CreateSubscriptionOptions) (*v1.InternalSubscriptionInfo, error) {
	req := &v1.InternalCreateSubscriptionRequest{
		ProductCode:      productCode,
		PlanCode:         planCode,
		AutomaticRenewal: false,
		IsTrial:          false,
		Order:            order,
	}
	if opts != nil {
		req.StartDate = opts.StartDate
		req.EndDate = opts.EndDate
		req.IsTrial = opts.IsTrial
		req.AutomaticRenewal = opts.AutomaticRenewal
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalCreateSubscription(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("创建订阅失败: product=%s, plan=%s, err=%v", productCode, planCode, err)
		return nil, err
	}
	return resp.Subscription, nil
}

// ReNewSubscription 续订订阅
func (c *Client) ReNewSubscription(ctx context.Context, productCode, planCode string, reNewTime *durationpb.Duration, order *v1.InternalSubscriptionOrderInfo) (*v1.InternalSubscriptionInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalReNewSubscription(ctx, &v1.InternalReNewSubscriptionRequest{
		ProductCode: productCode,
		PlanCode:    planCode,
		ReNewTime:   reNewTime,
		Order:       order,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("续订订阅失败: product=%s, plan=%s, err=%v", productCode, planCode, err)
		return nil, err
	}
	return resp.Subscription, nil
}

// UpgradeSubscriptionOptions 升级订阅选项
type UpgradeSubscriptionOptions struct {
	StartDate *timestamppb.Timestamp // 订阅开始时间
	EndDate   *timestamppb.Timestamp // 订阅结束时间
}

// UpgradeSubscription 升级订阅
func (c *Client) UpgradeSubscription(ctx context.Context, productCode, planCode string, order *v1.InternalSubscriptionOrderInfo, opts *UpgradeSubscriptionOptions) (*v1.InternalSubscriptionInfo, error) {
	req := &v1.InternalUpgradeSubscriptionRequest{
		ProductCode: productCode,
		PlanCode:    planCode,
		Order:       order,
	}
	if opts != nil {
		req.StartDate = opts.StartDate
		req.EndDate = opts.EndDate
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalUpgradeSubscription(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("升级订阅失败: product=%s, plan=%s, err=%v", productCode, planCode, err)
		return nil, err
	}
	return resp.Subscription, nil
}

// GetSubscriptionStats 获取商户订阅统计
func (c *Client) GetSubscriptionStats(ctx context.Context, tenantCode string) (*v1.InternalGetSubscriptionStatsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalGetSubscriptionStats(ctx, &v1.InternalGetSubscriptionStatsRequest{
		TenantCode: tenantCode,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取商户订阅状态失败: tenant=%s, err=%v", tenantCode, err)
		return nil, err
	}
	return resp, nil
}

// GetSubscriptionStatsByProduct 获取产品订阅统计
func (c *Client) GetSubscriptionStatsByProduct(ctx context.Context, productCode string) (*v1.InternalGetSubscriptionStatsByProductCodeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalGetSubscriptionStatsByProductCode(ctx, &v1.InternalGetSubscriptionStatsByProductCodeRequest{
		ProductCode: productCode,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取产品订阅状态失败: product=%s, err=%v", productCode, err)
		return nil, err
	}
	return resp, nil
}

// UseResult 使用配额的结果
type UseResult struct {
	Success         bool
	DimensionKey    string
	QuotaLimit      int32
	QuotaUsedBefore int32
	QuotaUsedAfter  int32
	QuotaRemaining  int32
	IsUnlimited     bool
}

// Use 使用配额
func (c *Client) Use(ctx context.Context, tenantCode, productCode, dimensionKey string, amount int32) (*UseResult, error) {
	if amount <= 0 {
		amount = 1
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalCheckAndUseQuota(ctx, &v1.InternalCheckAndUseQuotaRequest{
		TenantCode:   tenantCode,
		ProductCode:  productCode,
		DimensionKey: dimensionKey,
		Amount:       amount,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("使用配额失败: tenant=%s, product=%s, dimension=%s, err=%v",
			tenantCode, productCode, dimensionKey, err)
		return nil, err
	}

	return &UseResult{
		Success:         resp.Success,
		DimensionKey:    resp.DimensionKey,
		QuotaLimit:      resp.QuotaLimit,
		QuotaUsedBefore: resp.QuotaUsedBefore,
		QuotaUsedAfter:  resp.QuotaUsedAfter,
		QuotaRemaining:  resp.QuotaRemaining,
		IsUnlimited:     resp.IsUnlimited,
	}, nil
}

// MustUse 使用配额，失败时直接返回错误
func (c *Client) MustUse(ctx context.Context, tenantCode, productCode, dimensionKey string, amount int32) error {
	result, err := c.Use(ctx, tenantCode, productCode, dimensionKey, amount)
	if err != nil {
		return err
	}
	if !result.Success {
		return errors.New(429, "QUOTA_EXCEEDED", "配额不足: "+dimensionKey)
	}
	return nil
}

// ReleaseResult 释放配额的结果
type ReleaseResult struct {
	Success         bool
	DimensionKey    string
	QuotaUsedBefore int32
	QuotaUsedAfter  int32
}

// Release 释放配额（-N）
func (c *Client) Release(ctx context.Context, tenantCode, productCode, dimensionKey string, amount int32) (*ReleaseResult, error) {
	if amount <= 0 {
		amount = 1
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalReleaseQuota(ctx, &v1.InternalReleaseQuotaRequest{
		TenantCode:   tenantCode,
		ProductCode:  productCode,
		DimensionKey: dimensionKey,
		Amount:       amount,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("释放配额失败: tenant=%s, product=%s, dimension=%s, err=%v",
			tenantCode, productCode, dimensionKey, err)
		return nil, err
	}

	return &ReleaseResult{
		Success:         resp.Success,
		DimensionKey:    resp.DimensionKey,
		QuotaUsedBefore: resp.QuotaUsedBefore,
		QuotaUsedAfter:  resp.QuotaUsedAfter,
	}, nil
}

// QuotaUsage 配额使用情况
type QuotaUsage struct {
	DimensionKey    string
	QuotaLimit      int32
	QuotaUsed       int32
	QuotaRemaining  int32
	IsUnlimited     bool
	UsagePercentage float64
	Unit            *string
}

// GetUsage 查询配额使用情况
func (c *Client) GetUsage(ctx context.Context, tenantCode, productCode string, dimensionKey *string) ([]*QuotaUsage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.grpcClient.InternalGetQuotaUsage(ctx, &v1.InternalGetQuotaUsageRequest{
		TenantCode:   tenantCode,
		ProductCode:  productCode,
		DimensionKey: dimensionKey,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("查询配额失败: tenant=%s, product=%s, err=%v",
			tenantCode, productCode, err)
		return nil, err
	}

	usages := make([]*QuotaUsage, 0, len(resp.Usages))
	for _, u := range resp.Usages {
		usages = append(usages, &QuotaUsage{
			DimensionKey:    u.DimensionKey,
			QuotaLimit:      u.QuotaLimit,
			QuotaUsed:       u.QuotaUsed,
			QuotaRemaining:  u.QuotaRemaining,
			IsUnlimited:     u.IsUnlimited,
			UsagePercentage: u.UsagePercentage,
			Unit:            u.Unit,
		})
	}

	return usages, nil
}
