package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	authWare "github.com/heyinLab/common/pkg/middleware/auth"
	"github.com/heyinLab/common/pkg/middleware/common"
	"google.golang.org/grpc/metadata"
)

func ExtractClaims() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// 1. 获取 gRPC 传入的 metadata
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				// 准备一个空的 claims 对象
				claims := &authWare.Claims{}
				hasData := false

				// 2. 提取 UserCode
				if vals := md.Get(common.USERCODE); len(vals) > 0 {
					claims.UserCode = vals[0]
					hasData = true
				}

				// 3. 提取 TenantCode
				if vals := md.Get(common.TENANTCODE); len(vals) > 0 {
					claims.TenantCode = vals[0]
				}

				// 4. 提取 RegionName
				if vals := md.Get(common.REGIONNAME); len(vals) > 0 {
					claims.RegionName = vals[0]
				}

				// 5. 如果成功提取到了数据，将其注入到 Context 中
				// 这样后续的业务逻辑（Service层）就可以通过 authWare.FromContext(ctx) 拿到了
				if hasData {
					ctx = authWare.NewContext(ctx, claims)
				}
			}

			return handler(ctx, req)
		}
	}
}
