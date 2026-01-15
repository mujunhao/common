package platform

import (
	"context"
	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	consulapi "github.com/hashicorp/consul/api"
	"testing"
)

func TestListTenant(t *testing.T) {
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

	ctx := context.Background()
	rsp, err := client.IAM().ListTenant(ctx, 0, 0, nil)
	if err != nil {
		t.Logf("获取租户列表失败（可能服务未启动）: %v", err)
		t.Skip("跳过测试，服务可能未启动")
		return
	}

	t.Logf("成功获取租户列表，总数: %d", len(rsp.Items))
	for i, tenant := range rsp.Items {
		if i < 3 { // 只打印前3个节点避免输出过多
			t.Logf("租户: Name=%s, Code=%s", tenant.Name, tenant.Code)
		}
	}
}
