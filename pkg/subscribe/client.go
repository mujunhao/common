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

type Client struct {
	config          *Config
	conn            *grpc.ClientConn
	logger          *log.Helper
	subscribeClient *SubscribeClient
}

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

	logger.Infof("平台服务客户端连接成功 (服务发现): endpoint=%s, timeout=%v", config.Endpoint, config.Timeout)

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

type SubscribeClient struct {
	client v1.SubscriptionInternalServiceClient
	logger *log.Helper
	config *Config
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
		c.logger.WithContext(ctx).Errorf("获取订阅列表失败:tenant_code=%d, product_code=%s,error=%v", tenantCode, productCode, err)
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
		c.logger.WithContext(ctx).Errorf("获取商户订阅状态失败:tenant_code=%serr=%v", err)
		return nil, err
	}

	return resp, nil
}
