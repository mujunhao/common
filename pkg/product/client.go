package product

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	v1 "github.com/heyinLab/common/api/gen/go/product/v1"
	middleware "github.com/heyinLab/common/pkg/middleware/grpc"
	"google.golang.org/grpc"
)

type Client struct {
	config        *Config
	conn          *grpc.ClientConn
	logger        *log.Helper
	productClient *ProductClient
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
		"module", "product-client",
	))

	conn, err := middleware.CreateGRPCConn(config, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}
	return &Client{
		config:        config,
		conn:          conn,
		logger:        logger,
		productClient: newProductClient(conn, logger, config),
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
		"module", "product-client",
	))

	conn, err := middleware.CreateGRPCConn(config, discovery, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	logger.Infof("平台服务客户端连接成功 (服务发现): endpoint=%s, timeout=%v", config.Endpoint, config.Timeout)

	return &Client{
		config:        config,
		conn:          conn,
		logger:        logger,
		productClient: newProductClient(conn, logger, config),
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) ProductClient() *ProductClient {
	return c.productClient
}

type ProductClient struct {
	client v1.ProductInternalServiceClient
	logger *log.Helper
	config *Config
}

func newProductClient(conn *grpc.ClientConn, logger *log.Helper, config *Config) *ProductClient {
	return &ProductClient{
		client: v1.NewProductInternalServiceClient(conn),
		logger: logger,
		config: config,
	}
}

type GetPlanOption struct {
	IncludeParameters *bool // 是否包含规则
}

// GetPlan 获取套餐信息
func (c *ProductClient) GetPlan(ctx context.Context, planCode string, opt *GetPlanOption) (*v1.InternalProductPlanInfo, error) {
	req := &v1.InternalGetPlanRequest{
		PlanCode:          planCode,
		IncludeParameters: nil,
	}
	if opt != nil {
		if opt.IncludeParameters != nil {
			req.IncludeParameters = opt.IncludeParameters
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalGetPlan(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取套餐信息失败:plan_ode=%s,error=%v", planCode, err)
		return nil, err
	}

	return resp.Plan, nil
}

type MerchantGetPlanOption struct {
	IncludeParameters *bool // 是否包含规则
}

// MerchantGetPlan 商户获取套餐详情
func (c *ProductClient) MerchantGetPlan(ctx context.Context, planCode string, opt *MerchantGetPlanOption) (*v1.InternalProductPlanInfo, error) {
	req := &v1.InternalMerchantGetPlanRequest{
		PlanCode:          planCode,
		IncludeParameters: nil,
	}
	if opt != nil {
		if opt.IncludeParameters != nil {
			req.IncludeParameters = opt.IncludeParameters
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalMerchantGetPlan(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("商户获取套餐信息失败:plan_ode=%s,error=%v", planCode, err)
		return nil, err
	}

	return resp.Plan, nil
}

type GetProductOption struct {
	IncludePlans *bool // 是否包含套餐列表
}

// GetProduct 获取产品信息
func (c *ProductClient) GetProduct(ctx context.Context, productCode string, opt *GetProductOption) (*v1.InternalProductInfo, error) {
	req := &v1.InternalGetProductRequest{
		ProductCode:  productCode,
		IncludePlans: nil,
	}
	if opt != nil {
		if opt.IncludePlans != nil {
			req.IncludePlans = opt.IncludePlans
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalGetProduct(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取产品信息失败:product_code=%s,error=%v", productCode, err)
		return nil, err
	}

	return resp.Product, nil
}

type GetMerchantGetProduct struct {
	IncludePlans *bool // 是否包含套餐列表
}

// MerchantGetProduct 商户获取产品
func (c *ProductClient) MerchantGetProduct(ctx context.Context, productCode string, opt *GetMerchantGetProduct) (*v1.InternalProductInfo, error) {
	req := &v1.InternalMerchantGetProductRequest{
		ProductCode:  productCode,
		IncludePlans: nil,
	}
	if opt != nil {
		if opt.IncludePlans != nil {
			req.IncludePlans = opt.IncludePlans
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalMerchantGetProduct(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("商户获取产品信息失败:product_code=%s,error=%v", productCode, err)
		return nil, err
	}

	return resp.Product, nil
}

type ListPricingRulesOption struct {
	Page      *int32                 // 页码
	PageSize  *int32                 // 每页数量
	Search    *string                // 关键词搜索
	RuleType  *v1.InternalRuleType   // 规则类型筛选
	Status    *v1.InternalRuleStatus // 状态筛选
	IsVisible *bool                  // 是否可见筛选
}

// 获取定价规则列表
func (c *ProductClient) ListPricingRules(ctx context.Context, opt *ListPricingRulesOption) (*v1.InternalListPricingRulesResponse, error) {
	req := &v1.InternalListPricingRulesRequest{
		Page:      nil,
		PageSize:  nil,
		Search:    nil,
		RuleType:  nil,
		Status:    nil,
		IsVisible: nil,
	}

	if opt != nil {
		if opt.Page != nil {
			req.Page = opt.Page
		}
		if opt.PageSize != nil {
			req.PageSize = opt.PageSize
		}
		if opt.Search != nil {
			req.Search = opt.Search
		}
		if opt.RuleType != nil {
			req.RuleType = opt.RuleType
		}
		if opt.Status != nil {
			req.Status = opt.Status
		}
		if opt.IsVisible != nil {
			req.IsVisible = opt.IsVisible
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.InternalListPricingRules(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Errorf("获取定价规则列表失败:error=%v", err)
		return nil, err
	}

	return resp, nil
}
