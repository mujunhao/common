package system

import (
	"github.com/heyinLab/common/pkg/common"
)

const (
	// DefaultServiceName 默认的平台服务名称（用于服务发现）
	DefaultServiceName = "system-server"
)

// Config 平台服务客户端配置
type Config = common.ServiceConfig

// DefaultConfig 返回默认的平台服务客户端配置
//
// 默认配置:
//   - Endpoint: "discovery:///system-server"
//   - ServiceName: "system-server"
//   - Timeout: 10s
func DefaultConfig() *Config {
	return common.NewServiceConfig(DefaultServiceName)
}
