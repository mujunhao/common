package system

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	consulapi "github.com/hashicorp/consul/api"
)

func TestGetCountryInfo(t *testing.T) {
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
	country, err := client.systemClient.GetCountryInfo(ctx, "CN")
	if err != nil {
		t.Logf("获取国家失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}
	t.Logf("成功获取国家 %+v", country)
}
