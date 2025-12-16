package resource

import (
	"github.com/heyinLab/common/pkg/common"
)

const (
	// DefaultServiceName 默认的资源服务名称（用于服务发现）
	DefaultServiceName = "resourceServer"

	// DefaultURLExpiresIn 默认URL过期时间（秒）
	DefaultURLExpiresIn = 3600
)

// InternalConfig 资源内部服务客户端配置
type InternalConfig = common.ServiceConfig

// DefaultInternalConfig 返回默认的内部服务客户端配置
//
// 默认配置:
//   - Endpoint: "discovery:///resourceServer"
//   - ServiceName: "resourceServer"
//   - Timeout: 10s
func DefaultInternalConfig() *InternalConfig {
	return common.NewServiceConfig(DefaultServiceName)
}
