package product

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	consulapi "github.com/hashicorp/consul/api"
)

func TestGetPlan(t *testing.T) {
	config := consulapi.DefaultConfig()
	config.Address = "192.168.3.6:8500"
	config.Token = ""
	config.Datacenter = "dc1"
	config.Scheme = "http"

	// 创建 Consul 客户端
	consulClient, err := consulapi.NewClient(config)
	if err != nil {
		t.Skipf("无法连接到 Consul: %v", err)
		return
	}

	// 创建 Consul 服务发现
	discovery := consul.New(consulClient)

	// 创建平台服务客户端
	client, err := NewClientWithDiscovery(DefaultConfig(), discovery)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试获取套餐信息
	ctx := context.Background()
	plan, err := client.ProductClient().GetPlan(ctx, "1766108752535-8bd10afcb6494395a52f3c5282a3907d",
		&GetPlanOption{IncludeParameters: func() *bool { tr := true; return &tr }()})
	if err != nil {
		t.Logf("获取套餐信息失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("成功获取套餐信息: %v", plan)
}

func TestMerchantGetPlan(t *testing.T) {
	config := consulapi.DefaultConfig()
	config.Address = "192.168.3.6:8500"
	config.Token = ""
	config.Datacenter = "dc1"
	config.Scheme = "http"

	// 创建 Consul 客户端
	consulClient, err := consulapi.NewClient(config)
	if err != nil {
		t.Skipf("无法连接到 Consul: %v", err)
		return
	}

	// 创建 Consul 服务发现
	discovery := consul.New(consulClient)

	// 创建平台服务客户端
	client, err := NewClientWithDiscovery(DefaultConfig(), discovery)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试商户获取套餐信息
	ctx := context.Background()
	plan, err := client.ProductClient().MerchantGetPlan(ctx, "1766108752535-8bd10afcb6494395a52f3c5282a3907d",
		&MerchantGetPlanOption{IncludeParameters: func() *bool { tr := true; return &tr }()})
	if err != nil {
		t.Logf("商户获取套餐信息失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("商户成功获取套餐信息: %v", plan)
}

func TestGetProduct(t *testing.T) {
	config := consulapi.DefaultConfig()
	config.Address = "192.168.3.6:8500"
	config.Token = ""
	config.Datacenter = "dc1"
	config.Scheme = "http"

	// 创建 Consul 客户端
	consulClient, err := consulapi.NewClient(config)
	if err != nil {
		t.Skipf("无法连接到 Consul: %v", err)
		return
	}

	// 创建 Consul 服务发现
	discovery := consul.New(consulClient)

	// 创建平台服务客户端
	client, err := NewClientWithDiscovery(DefaultConfig(), discovery)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试获取产品信息
	ctx := context.Background()
	product, err := client.ProductClient().GetProduct(ctx, "cloud_server",
		&GetProductOption{IncludePlans: func() *bool { tr := true; return &tr }()})
	if err != nil {
		t.Logf("获取产品信息失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("成功获取产品信息: %v", product)
}

func TestMerchantGetProduct(t *testing.T) {
	config := consulapi.DefaultConfig()
	config.Address = "192.168.3.6:8500"
	config.Token = ""
	config.Datacenter = "dc1"
	config.Scheme = "http"

	// 创建 Consul 客户端
	consulClient, err := consulapi.NewClient(config)
	if err != nil {
		t.Skipf("无法连接到 Consul: %v", err)
		return
	}

	// 创建 Consul 服务发现
	discovery := consul.New(consulClient)

	// 创建平台服务客户端
	client, err := NewClientWithDiscovery(DefaultConfig(), discovery)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试商户获取产品信息
	ctx := context.Background()
	product, err := client.ProductClient().MerchantGetProduct(ctx, "cloud_server",
		&GetMerchantGetProduct{IncludePlans: func() *bool { tr := true; return &tr }()})
	if err != nil {
		t.Logf("商户获取产品信息失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("商户成功获取产品信息: %v", product)
}

func TestListPricingRules(t *testing.T) {
	config := consulapi.DefaultConfig()
	config.Address = "192.168.3.6:8500"
	config.Token = ""
	config.Datacenter = "dc1"
	config.Scheme = "http"

	// 创建 Consul 客户端
	consulClient, err := consulapi.NewClient(config)
	if err != nil {
		t.Skipf("无法连接到 Consul: %v", err)
		return
	}

	// 创建 Consul 服务发现
	discovery := consul.New(consulClient)

	// 创建平台服务客户端
	client, err := NewClientWithDiscovery(DefaultConfig(), discovery)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 测试获取定价规则列表
	ctx := context.Background()
	resp, err := client.ProductClient().ListPricingRules(ctx, &ListPricingRulesOption{
		Page:      func() *int32 { tr := int32(1); return &tr }(),
		PageSize:  func() *int32 { tr := int32(5); return &tr }(),
		Search:    nil,
		RuleType:  nil,
		Status:    nil,
		IsVisible: nil,
	})
	if err != nil {
		t.Logf("获取定价规则列表失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("成功定价规则列表 数量:%v", len(resp.Rules))
}
