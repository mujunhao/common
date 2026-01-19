package system

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	v1 "github.com/heyinLab/common/api/gen/go/system/v1"
	middleware "github.com/heyinLab/common/pkg/middleware/grpc"
	"google.golang.org/grpc"
)

type Client struct {
	config       *Config
	conn         *grpc.ClientConn
	logger       *log.Helper
	systemClient *SystemClient
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
		"module", "system-client",
	))

	conn, err := middleware.CreateGRPCConn(config, nil, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}
	return &Client{
		config:       config,
		conn:         conn,
		logger:       logger,
		systemClient: newSystemClient(conn, logger, config),
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
		"module", "system-client",
	))

	conn, err := middleware.CreateGRPCConn(config, discovery, logger)
	if err != nil {
		return nil, fmt.Errorf("创建 gRPC 连接失败: %w", err)
	}

	logger.Infof("平台服务客户端连接成功 (服务发现): endpoint=%s, timeout=%v", config.Endpoint, config.Timeout)

	return &Client{
		config:       config,
		conn:         conn,
		logger:       logger,
		systemClient: newSystemClient(conn, logger, config),
	}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) SystemClient() *SystemClient {
	return c.systemClient
}

type SystemClient struct {
	client v1.SystemInternalServiceClient
	logger *log.Helper
	config *Config
}

func newSystemClient(conn *grpc.ClientConn, logger *log.Helper, config *Config) *SystemClient {
	return &SystemClient{
		client: v1.NewSystemInternalServiceClient(conn),
		logger: logger,
		config: config,
	}
}

func (s *SystemClient) GetCountryInfo(ctx context.Context, countryCode string) (*v1.InternalCountry, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	resp, err := s.client.InternalGetCountryInfo(ctx, &v1.InternalGetCountryInfoRequest{
		CountryCode: &countryCode,
	})

	if err != nil {
		s.logger.WithContext(ctx).Errorf("获取国家列表失败:code=%s,error=%v", countryCode, err)
		return nil, err
	}

	return resp.Country, nil
}
