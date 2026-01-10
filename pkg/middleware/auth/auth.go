package auth

import (
	"context"
	"strconv"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	businessErrors "github.com/heyinLab/common/pkg/errors"
	"github.com/heyinLab/common/pkg/middleware/common"
)

// GetAuthType 获取认证类型
func GetAuthType(ctx context.Context) common.AuthType {
	if v, ok := ctx.Value(common.KeyAuthType).(common.AuthType); ok {
		return v
	}
	return common.AuthTypeToken
}

// GetAPIKeyID 获取 API Key ID（仅 OpenAPI 认证有值）
func GetAPIKeyID(ctx context.Context) uint64 {
	if v, ok := ctx.Value(common.KeyAPIKeyID).(uint64); ok {
		return v
	}
	return 0
}

// GetProductCode 获取产品编码（仅 OpenAPI 认证有值）
func GetProductCode(ctx context.Context) string {
	if v, ok := ctx.Value(common.KeyProductCode).(string); ok {
		return v
	}
	return ""
}

// IsOpenAPIRequest 判断是否为 OpenAPI 请求
func IsOpenAPIRequest(ctx context.Context) bool {
	return GetAuthType(ctx) == common.AuthTypeOpenAPI
}

// Operator 操作者信息（用于审计日志）
type Operator struct {
	Type string // "user" 或 "api_key"
	ID   uint64
}

// GetOperator 获取操作者信息
func GetOperator(ctx context.Context) Operator {
	if IsOpenAPIRequest(ctx) {
		return Operator{Type: "api_key", ID: GetAPIKeyID(ctx)}
	}
	claims, ok := FromContext(ctx)
	if ok && claims.UserCode != "" {
		return Operator{Type: "user", ID: 0} // ID 不再使用，返回0
	}
	return Operator{Type: "unknown", ID: 0}
}

// Server 统一认证中间件，支持 JWT Token 和 OpenAPI 两种认证方式
func Server(needTenant bool) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// 从 context 中获取 transport 信息 (HTTP/gRPC)
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, errors.New(
					int(businessErrors.ErrSystemError.HttpCode),
					businessErrors.ErrSystemError.Type,
					businessErrors.ErrSystemError.Message,
				)
			}

			header := tr.RequestHeader()

			// 1. 先检查认证类型
			authType := header.Get("X-Auth-Type")
			isOpenAPI := authType == "openapi"

			// 2. 读取公共 headers (现在使用 code 字符串)
			userCode := header.Get(common.USERCODE)
			regionName := header.Get(common.REGIONNAME)

			if !isOpenAPI {
				// JWT Token 认证：X-User-Code 必须存在且有效
				if userCode == "" {
					return nil, errors.New(
						int(businessErrors.ErrAuthHeaderMissing.HttpCode),
						businessErrors.ErrAuthHeaderMissing.Type,
						"X-User-Code header is missing",
					)
				}
			}

			// 3. 处理租户 Code
			var tenantCode string

			if needTenant {
				tenantCode = header.Get(common.TENANTCODE)
				if tenantCode == "" {
					return nil, errors.New(
						int(businessErrors.ErrTenantMissing.HttpCode),
						businessErrors.ErrTenantMissing.Type,
						businessErrors.ErrTenantMissing.Message,
					)
				}
			}

			// 4. 创建 Claims 并注入 context
			claims := &Claims{
				UserCode:   userCode,
				TenantCode: tenantCode,
				RegionName: regionName,
			}
			newCtx := NewContext(ctx, claims)

			// 5. 如果是 OpenAPI 请求，设置额外的 context 值
			if isOpenAPI {
				newCtx = context.WithValue(newCtx, common.KeyAuthType, common.AuthTypeOpenAPI)

				// 读取 API Key ID
				if apiKeyIDStr := header.Get("X-API-Key-ID"); apiKeyIDStr != "" {
					if id, err := strconv.ParseUint(apiKeyIDStr, 10, 64); err == nil {
						newCtx = context.WithValue(newCtx, common.KeyAPIKeyID, id)
					}
				}

				// 读取 Product Code
				if productCode := header.Get("X-Product-Code"); productCode != "" {
					newCtx = context.WithValue(newCtx, common.KeyProductCode, productCode)
				}
			}

			return handler(newCtx, req)
		}
	}
}
