package subscribe

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	v1 "github.com/heyinLab/common/api/gen/go/subscribe/v1"
	middleware "github.com/heyinLab/common/pkg/middleware/grpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client 订阅服务连接管理
type Client struct {
	config          *Config
	conn            *grpc.ClientConn
	logger          *log.Helper
	subscribeClient *SubscribeClient
}

// SubscribeClient 订阅服务业务客户端
type SubscribeClient struct {
	client v1.SubscriptionInternalServiceClient
	logger *log.Helper
	config *Config
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
		config:          config,
		conn:            conn,
		logger:          logger,
		subscribeClient: newSubscribeClient(conn, logger, config),
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
		config:          config,
		conn:            conn,
		logger:          logger,
		subscribeClient: newSubscribeClient(conn, logger, config),
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
func (c *Client) SubscribeClient() *SubscribeClient {
	return c.subscribeClient
}

func newSubscribeClient(conn *grpc.ClientConn, logger *log.Helper, config *Config) *SubscribeClient {
	return &SubscribeClient{
		client: v1.NewSubscriptionInternalServiceClient(conn),
		logger: logger,
		config: config,
	}
}

// GetTenantSubscriptions 获取商家指定产品订阅列表
func (c *SubscribeClient) GetTenantSubscriptions(ctx context.Context, tenantCode string, productCode string) ([]*v1.InternalSubscriptionInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalListSubscriptions(ctx, &v1.InternalListSubscriptionsRequest{
		TenantCode:  &tenantCode,
		ProductCode: &productCode,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取订阅列表失败:tenant_code=%s, product_code=%s,error=%v", tenantCode, productCode, err)
		return nil, err
	}

	return resp.Subscriptions, nil
}

type CreateSubscriptionOptions struct {
	// 订阅开始时间
	StartDate *timestamppb.Timestamp
	// 订阅结束时间
	EndDate *timestamppb.Timestamp
	// 是否自动续费
	AutomaticRenewal bool
	// 是否试用
	IsTrial bool
}

// CreateSubscription 商家创建订阅
func (c *SubscribeClient) CreateSubscription(ctx context.Context, productCode string, planCode string, order *v1.InternalSubscriptionOrderInfo, opts *CreateSubscriptionOptions) (*v1.InternalSubscriptionInfo, error) {
	req := &v1.InternalCreateSubscriptionRequest{
		ProductCode:      productCode,
		PlanCode:         planCode,
		AutomaticRenewal: false,
		StartDate:        nil,
		EndDate:          nil,
		IsTrial:          false,
		Order:            order,
	}
	if opts != nil {
		if opts.StartDate != nil {
			req.StartDate = opts.StartDate
		}
		if opts.EndDate != nil {
			req.EndDate = opts.EndDate
		}
		req.IsTrial = opts.IsTrial
		req.AutomaticRenewal = opts.AutomaticRenewal
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalCreateSubscription(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("创建订阅失败:product_code=%s plan_code=:%s err=%v", productCode, planCode, err)
		return nil, err
	}
	return resp.Subscription, nil
}

// ReNewSubscription 续订订阅
func (c *SubscribeClient) ReNewSubscription(ctx context.Context, productCode string, planCode string, reNewTime *durationpb.Duration, order *v1.InternalSubscriptionOrderInfo) (*v1.InternalSubscriptionInfo, error) {
	req := &v1.InternalReNewSubscriptionRequest{
		ProductCode: productCode,
		PlanCode:    planCode,
		ReNewTime:   reNewTime,
		Order:       order,
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalReNewSubscription(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("续订订阅失败:product_code=%s plan_code=:%s renew_time=:%s err=%v", productCode, planCode, reNewTime.String(), err)
		return nil, err
	}

	return resp.Subscription, nil
}

type UpgradeSubscriptionOptions struct {
	// 订阅开始时间
	StartDate *timestamppb.Timestamp
	// 订阅结束时间
	EndDate *timestamppb.Timestamp
}

// UpgradeSubscription 升级订阅
func (c *SubscribeClient) UpgradeSubscription(ctx context.Context, productCode string, planCode string, order *v1.InternalSubscriptionOrderInfo, opts *UpgradeSubscriptionOptions) (*v1.InternalSubscriptionInfo, error) {
	req := &v1.InternalUpgradeSubscriptionRequest{
		ProductCode: productCode,
		PlanCode:    planCode,
		StartDate:   nil,
		EndDate:     nil,
		Order:       order,
	}
	if opts != nil {
		if opts.StartDate != nil {
			req.StartDate = opts.StartDate
		}
		if opts.EndDate != nil {
			req.EndDate = opts.EndDate
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalUpgradeSubscription(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("升级订阅失败:product_code=%s plan_code=:%s err=%v", productCode, planCode, err)
		return nil, err
	}

	return resp.Subscription, nil
}

// 获取商户订阅状态
func (c *SubscribeClient) InternalGetSubscriptionStats(ctx context.Context, tenantCode string) (*v1.InternalGetSubscriptionStatsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalGetSubscriptionStats(ctx, &v1.InternalGetSubscriptionStatsRequest{TenantCode: tenantCode})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取商户订阅状态失败:tenant_code=%serr=%v", tenantCode, err)
		return nil, err
	}

	return resp, nil
}

func (c *SubscribeClient) InternalGetSubscriptionStatsByProductCode(ctx context.Context, productCode string) (
	*v1.InternalGetSubscriptionStatsByProductCodeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalGetSubscriptionStatsByProductCode(ctx,
		&v1.InternalGetSubscriptionStatsByProductCodeRequest{ProductCode: productCode})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取产品订阅状态失败:productCode=%serr=%v", productCode, err)
		return nil, err
	}

	return resp, nil
}

// QuotaResult 配额操作
type QuotaResult struct {
	Success         bool                      // 操作是否成功
	DimensionKey    string                    // 维度标识
	QuotaLimit      int32                     // 配额上限
	QuotaUsed       int32                     // 当前已使用量
	QuotaUsedBefore int32                     // 操作前已使用量
	QuotaRemaining  int32                     // 剩余配
	IsUnlimited     bool                      // 是否无限制
	UsagePercentage float64                   // 使用百分比
	ErrorMessage    string                    // 错误信息
	ErrorCode       v1.InternalQuotaErrorCode // 错误码
}

// Use 使用配额
func (c *SubscribeClient) Use(ctx context.Context, tenantCode, productCode, dimensionKey string, amount int32) (*QuotaResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalCheckAndUseQuota(ctx, &v1.InternalCheckAndUseQuotaRequest{
		TenantCode:   tenantCode,
		ProductCode:  productCode,
		DimensionKey: dimensionKey,
		Amount:       amount,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("配额使用失败: tenant=%s, product=%s, dimension=%s, err=%v",
			tenantCode, productCode, dimensionKey, err)
		return nil, err
	}

	return &QuotaResult{
		Success:         resp.Success,
		DimensionKey:    resp.DimensionKey,
		QuotaLimit:      resp.QuotaLimit,
		QuotaUsed:       resp.QuotaUsedAfter,
		QuotaUsedBefore: resp.QuotaUsedBefore,
		QuotaRemaining:  resp.QuotaRemaining,
		IsUnlimited:     resp.IsUnlimited,
		ErrorMessage:    resp.ErrorMessage,
		ErrorCode:       resp.ErrorCode,
	}, nil
}

// MustUse 使用配额
func (c *SubscribeClient) MustUse(ctx context.Context, tenantCode, productCode, dimensionKey string, amount int32) error {
	result, err := c.Use(ctx, tenantCode, productCode, dimensionKey, amount)
	if err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("配额不足: %s", result.ErrorMessage)
	}
	return nil
}

// Release 释放配额
func (c *SubscribeClient) Release(ctx context.Context, tenantCode, productCode, dimensionKey string, amount int32) (*QuotaResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalReleaseQuota(ctx, &v1.InternalReleaseQuotaRequest{
		TenantCode:   tenantCode,
		ProductCode:  productCode,
		DimensionKey: dimensionKey,
		Amount:       amount,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("配额释放失败: tenant=%s, product=%s, dimension=%s, err=%v",
			tenantCode, productCode, dimensionKey, err)
		return nil, err
	}

	return &QuotaResult{
		Success:         resp.Success,
		DimensionKey:    resp.DimensionKey,
		QuotaUsed:       resp.QuotaUsedAfter,
		QuotaUsedBefore: resp.QuotaUsedBefore,
		ErrorMessage:    resp.ErrorMessage,
	}, nil
}

// GetUsage 查询配额使用情况
func (c *SubscribeClient) GetUsage(ctx context.Context, tenantCode, productCode string, dimensionKey *string) ([]*QuotaResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalGetQuotaUsage(ctx, &v1.InternalGetQuotaUsageRequest{
		TenantCode:   tenantCode,
		ProductCode:  productCode,
		DimensionKey: dimensionKey,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("查询配额使用情况失败: tenant=%s, product=%s, err=%v",
			tenantCode, productCode, err)
		return nil, err
	}

	results := make([]*QuotaResult, 0, len(resp.Usages))
	for _, u := range resp.Usages {
		results = append(results, &QuotaResult{
			Success:         true,
			DimensionKey:    u.DimensionKey,
			QuotaLimit:      u.QuotaLimit,
			QuotaUsed:       u.QuotaUsed,
			QuotaRemaining:  u.QuotaRemaining,
			IsUnlimited:     u.IsUnlimited,
			UsagePercentage: u.UsagePercentage,
		})
	}
	return results, nil
}
