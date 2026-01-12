package subscribe

import (
	"github.com/heyinLab/common/pkg/common"
)

const (
	// DefaultServiceName 默认的平台服务名称（用于服务发现）
	DefaultServiceName = "subscription-server"
)

// Config 平台服务客户端配置
type Config = common.ServiceConfig

// DefaultConfig 返回默认的平台服务客户端配置
//
// 默认配置:
//   - Endpoint: "discovery:///subscription-server"
//   - ServiceName: "subscription-server"
//   - Timeout: 10s
func DefaultConfig() *Config {
	return common.NewServiceConfig(DefaultServiceName)
}
