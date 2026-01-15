package subscribe

import (
	"context"
	v1 "github.com/heyinLab/common/api/gen/go/subscribe/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	consulapi "github.com/hashicorp/consul/api"
)

var (
	testFreeOrder = &v1.InternalSubscriptionOrderInfo{
		OrderNo:          "1111",
		OrderType:        v1.InternalOrderType_INTERNAL_ORDER_TYPE_NEW,
		BillingCycle:     v1.InternalBillingCycle_INTERNAL_BILLING_CYCLE_MONTHLY,
		Currency:         "CNY",
		Status:           v1.InternalOrderStatus_INTERNAL_ORDER_STATUS_PAID,
		ServiceStartDate: timestamppb.New(time.Now()),
	}
)

func TestGetTenantSubscriptions(t *testing.T) {
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

	// 测试获取订阅列表
	ctx := context.Background()
	subscriptions, err := client.SubscribeClient().GetTenantSubscriptions(ctx, "1001", "cloud_server")
	if err != nil {
		t.Logf("获取订阅列表失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("成功获取订阅列表，总数: %d", len(subscriptions))
}

func TestCreateSubscription(t *testing.T) {
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

	// 测试商家创建订阅
	ctx := context.Background()
	subscription, err := client.SubscribeClient().CreateSubscription(ctx,
		"1766128805992-cc52eac1dbf24d9e811e3c1462118351",
		"1766128806730-5afa806357844e7195b760af195e4e8b",
		testFreeOrder,
		&CreateSubscriptionOptions{
			StartDate:        nil,
			EndDate:          nil,
			AutomaticRenewal: true,
			IsTrial:          true,
		},
	)
	if err != nil {
		t.Logf("商家创建订阅失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("商家创建订阅成功: %v", subscription)
}

func TestReNewSubscription(t *testing.T) {
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

	// 测试商家续订订阅
	ctx := context.Background()
	subscription, err := client.SubscribeClient().ReNewSubscription(ctx,
		"1766128805992-cc52eac1dbf24d9e811e3c1462118351",
		"1766128806730-5afa806357844e7195b760af195e4e8b",
		nil,
		testFreeOrder,
	)
	if err != nil {
		t.Logf("商家续订订阅失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("商家续订订阅成功: %v", subscription)
}

func TestUpgradeSubscription(t *testing.T) {
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

	// 测试商家升级订阅
	ctx := context.Background()
	subscription, err := client.SubscribeClient().UpgradeSubscription(ctx,
		"1766128805992-cc52eac1dbf24d9e811e3c1462118351",
		"1766128806730-5afa806357844e7195b760af195e4e8b",
		testFreeOrder,
		&UpgradeSubscriptionOptions{
			StartDate: timestamppb.New(time.Now()),
			EndDate:   nil,
		},
	)
	if err != nil {
		t.Logf("商家升级订阅失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("商家升级订阅成功: %v", subscription)
}

func TestInternalGetSubscriptionStats(t *testing.T) {
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

	// 测试获取商户订阅状态
	ctx := context.Background()
	subscription, err := client.SubscribeClient().InternalGetSubscriptionStats(ctx, "312")
	if err != nil {
		t.Logf("获取商户订阅状态失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("获取商户订阅状态成功: %v", subscription)
}
