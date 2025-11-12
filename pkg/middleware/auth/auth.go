package auth

import (
	businessErrors "codeup.aliyun.com/68ce48b215dfc6c8604f8fb0/go-heyin-common/pkg/errors"
	"context"
	"strconv"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// 从 context 中获取 transport 信息 (HTTP/gRPC)
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, errors.New(int(businessErrors.ErrSystemError.HttpCode), businessErrors.ErrSystemError.Type, businessErrors.ErrSystemError.Message)
			}

			// 信任上游传递来的header X-User-ID X-User-Type  X-Tenant-ID X-Region-Name
			userId := tr.RequestHeader().Get("X-User-ID")
			userType := tr.RequestHeader().Get("X-User-Type")
			tenantId := tr.RequestHeader().Get("X-Tenant-ID")
			regionName := tr.RequestHeader().Get("X-Region-Name")
			// 检查必需的 header
			if userId == "" {
				return nil, errors.New(
					int(businessErrors.ErrAuthHeaderMissing.HttpCode),
					businessErrors.ErrAuthHeaderMissing.Type,
					"X-User-ID header is missing",
				)
			}

			if tenantId == "" {
				return nil, errors.New(
					int(businessErrors.ErrTenantMissing.HttpCode),
					businessErrors.ErrTenantMissing.Type,
					businessErrors.ErrTenantMissing.Message,
				)
			}

			// 解析用户ID
			userIdUint, err := strconv.ParseUint(userId, 10, 32)
			if err != nil {
				return nil, errors.New(
					int(businessErrors.ErrAuthHeaderInvalid.HttpCode),
					businessErrors.ErrAuthHeaderInvalid.Type,
					"Invalid X-User-ID format",
				)
			}

			// 解析租户ID
			tenantIdUint, err := strconv.ParseUint(tenantId, 10, 32)
			if err != nil {
				return nil, errors.New(
					int(businessErrors.ErrTenantInvalid.HttpCode),
					businessErrors.ErrTenantInvalid.Type,
					businessErrors.ErrTenantInvalid.Message,
				)
			}

			claims := &Claims{
				UserID:     uint32(userIdUint),
				UserType:   userType,
				TenantID:   uint32(tenantIdUint),
				RegionName: regionName,
			}
			newCtx := NewContext(ctx, claims)

			return handler(newCtx, req)
		}
	}
}
