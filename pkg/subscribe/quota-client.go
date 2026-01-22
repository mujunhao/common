package subscribe

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/heyinLab/common/api/gen/go/subscribe/v1"
	"google.golang.org/grpc"
)

// QuotaClient 配额服务客户端
// 封装 SubscriptionInternalService 的配额相关接口
type QuotaClient struct {
	productCode        string // 产品编码，初始化时绑定
	config             *Config
	conn               *grpc.ClientConn
	logger             *log.Helper
	subscriptionClient v1.SubscriptionInternalServiceClient
}

// NewQuotaClient 创建配额客户端
// productCode: 产品编码，如 "points-mall"，初始化时绑定，后续调用不需要再传
func NewQuotaClient(conn *grpc.ClientConn, productCode string, config *Config, logger *log.Helper) *QuotaClient {
	return &QuotaClient{
		productCode:        productCode,
		config:             config,
		conn:               conn,
		logger:             logger,
		subscriptionClient: v1.NewSubscriptionInternalServiceClient(conn),
	}
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

// Use 使用配额（+N）
// tenantCode: 租户编码，从 context 获取
// dimensionKey: 维度键，如 "goods_count"
// amount: 使用数量，默认为 1
func (c *QuotaClient) Use(ctx context.Context, tenantCode, dimensionKey string, amount int32) (*UseResult, error) {
	if amount <= 0 {
		amount = 1
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.subscriptionClient.InternalCheckAndUseQuota(ctx, &v1.InternalCheckAndUseQuotaRequest{
		TenantCode:   tenantCode,
		ProductCode:  c.productCode,
		DimensionKey: dimensionKey,
		Amount:       amount,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("使用配额失败: tenant=%s, product=%s, dimension=%s, err=%v",
			tenantCode, c.productCode, dimensionKey, err)
		return nil, err
	}

	if !resp.Success {
		return &UseResult{
			Success:         false,
			DimensionKey:    resp.DimensionKey,
			QuotaLimit:      resp.QuotaLimit,
			QuotaUsedBefore: resp.QuotaUsedBefore,
			QuotaRemaining:  resp.QuotaRemaining,
			IsUnlimited:     resp.IsUnlimited,
		}, nil
	}

	return &UseResult{
		Success:         true,
		DimensionKey:    resp.DimensionKey,
		QuotaLimit:      resp.QuotaLimit,
		QuotaUsedBefore: resp.QuotaUsedBefore,
		QuotaUsedAfter:  resp.QuotaUsedAfter,
		QuotaRemaining:  resp.QuotaRemaining,
		IsUnlimited:     resp.IsUnlimited,
	}, nil
}

// MustUse 使用配额，失败时直接返回错误
// 适用于不需要处理详细结果的场景
func (c *QuotaClient) MustUse(ctx context.Context, tenantCode, dimensionKey string, amount int32) error {
	result, err := c.Use(ctx, tenantCode, dimensionKey, amount)
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

func (c *QuotaClient) Release(ctx context.Context, tenantCode, dimensionKey string, amount int32) (*ReleaseResult, error) {
	if amount <= 0 {
		amount = 1
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.subscriptionClient.InternalReleaseQuota(ctx, &v1.InternalReleaseQuotaRequest{
		TenantCode:   tenantCode,
		ProductCode:  c.productCode,
		DimensionKey: dimensionKey,
		Amount:       amount,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("释放配额失败: tenant=%s, product=%s, dimension=%s, err=%v",
			tenantCode, c.productCode, dimensionKey, err)
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

func (c *QuotaClient) GetUsage(ctx context.Context, tenantCode string, dimensionKey *string) ([]*QuotaUsage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.subscriptionClient.InternalGetQuotaUsage(ctx, &v1.InternalGetQuotaUsageRequest{
		TenantCode:   tenantCode,
		ProductCode:  c.productCode,
		DimensionKey: dimensionKey,
	})
	if err != nil {
		c.logger.WithContext(ctx).Errorf("查询配额失败: tenant=%s, product=%s, err=%v",
			tenantCode, c.productCode, err)
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

// GetProductCode 获取绑定的产品编码
func (c *QuotaClient) GetProductCode() string {
	return c.productCode
}
