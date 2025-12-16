package common

import (
	"fmt"
	"time"
)

const (
	// DefaultTimeout 默认超时时间
	DefaultTimeout = 10 * time.Second
)

// ServiceConfig 通用服务客户端配置
type ServiceConfig struct {
	// Endpoint 服务端点
	// 直连方式: "localhost:9000" 或 "192.168.1.100:9000"
	// 服务发现方式: "discovery:///service-name"
	Endpoint string

	// ServiceName 服务名称（用于服务发现）
	ServiceName string

	// Timeout 请求超时时间
	Timeout time.Duration
}

// NewServiceConfig 创建新的服务配置
//
// 参数:
//   - serviceName: 服务名称（用于服务发现）
//
// 返回:
//   - *ServiceConfig: 配置实例
func NewServiceConfig(serviceName string) *ServiceConfig {
	return &ServiceConfig{
		Endpoint:    fmt.Sprintf("discovery:///%s", serviceName),
		ServiceName: serviceName,
		Timeout:     DefaultTimeout,
	}
}

// Validate 验证配置
func (c *ServiceConfig) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("服务端点不能为空")
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return nil
}

// WithEndpoint 设置服务端点
//
// 参数:
//   - endpoint: 服务端点地址
//
// 示例:
//   - 直连: "localhost:9000"
//   - 服务发现: "discovery:///service-name"
func (c *ServiceConfig) WithEndpoint(endpoint string) *ServiceConfig {
	c.Endpoint = endpoint
	return c
}

// WithServiceName 设置服务名称
func (c *ServiceConfig) WithServiceName(name string) *ServiceConfig {
	c.ServiceName = name
	c.Endpoint = fmt.Sprintf("discovery:///%s", name)
	return c
}

// WithTimeout 设置请求超时时间
func (c *ServiceConfig) WithTimeout(timeout time.Duration) *ServiceConfig {
	c.Timeout = timeout
	return c
}

// Copy 创建配置的副本
func (c *ServiceConfig) Copy() *ServiceConfig {
	return &ServiceConfig{
		Endpoint:    c.Endpoint,
		ServiceName: c.ServiceName,
		Timeout:     c.Timeout,
	}
}