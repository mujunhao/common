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
	if ok {
		return Operator{Type: "user", ID: uint64(claims.UserID)}
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

			// 2. 读取公共 headers
			userId := header.Get(common.USERID)
			regionName := header.Get(common.REGIONNAME)

			var userIdUint uint64 = 0

			if isOpenAPI {
				// OpenAPI 认证：X-User-ID 可以为空或为 "0"
				if userId != "" {
					userIdUint, _ = strconv.ParseUint(userId, 10, 32)
				}
				// OpenAPI 请求的 UserID 固定为 0，不报错
			} else {
				// JWT Token 认证：X-User-ID 必须存在且有效
				if userId == "" {
					return nil, errors.New(
						int(businessErrors.ErrAuthHeaderMissing.HttpCode),
						businessErrors.ErrAuthHeaderMissing.Type,
						"X-User-ID header is missing",
					)
				}

				userIdUint, err = strconv.ParseUint(userId, 10, 32)
				if err != nil {
					return nil, errors.New(
						int(businessErrors.ErrAuthHeaderInvalid.HttpCode),
						businessErrors.ErrAuthHeaderInvalid.Type,
						"Invalid X-User-ID format",
					)
				}
			}

			// 3. 处理租户ID
			var tenantIdUint uint64 = 0

			if needTenant {
				tenantId := header.Get(common.TENANTID)
				if tenantId == "" {
					return nil, errors.New(
						int(businessErrors.ErrTenantMissing.HttpCode),
						businessErrors.ErrTenantMissing.Type,
						businessErrors.ErrTenantMissing.Message,
					)
				}
				t, err := strconv.ParseUint(tenantId, 10, 32)
				if err != nil {
					return nil, errors.New(
						int(businessErrors.ErrTenantInvalid.HttpCode),
						businessErrors.ErrTenantInvalid.Type,
						businessErrors.ErrTenantInvalid.Message,
					)
				}
				tenantIdUint = t
			}

			// 4. 创建 Claims 并注入 context
			claims := &Claims{
				UserID:     uint32(userIdUint),
				TenantID:   uint32(tenantIdUint),
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
