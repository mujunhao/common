package image

import (
	"context"

	"github.com/heyinLab/common/pkg/resource"
)

// Resolver URL解析器接口
type Resolver interface {
	// Resolve 批量解析文件ID为资源信息
	//
	// 参数:
	//   - ctx: 上下文
	//   - ids: 文件ID列表（已去重）
	//
	// 返回:
	//   - map[string]*ResourceInfo: 文件ID到资源信息的映射
	//   - error: 解析失败时的错误
	Resolve(ctx context.Context, ids []string) (map[string]*ResourceInfo, error)
}

// ResolverOptions 解析器选项
type ResolverOptions struct {
	// IncludeVariants 是否包含变体URL（如缩略图）
	IncludeVariants bool
	// ExpiresIn URL有效期（秒），默认3600
	ExpiresIn int64
}

// resourceResolver 基于 resource.ResourceClient 的解析器实现
type resourceResolver struct {
	client *resource.ResourceClient
	opts   *ResolverOptions
}

// NewResolver 创建基于 ResourceClient 的解析器
//
// 参数:
//   - client: 资源服务客户端
//
// 使用示例:
//
//	resolver := image.NewResolver(resourceClient)
//	filler := image.NewFiller(resolver)
//
// 说明:
//   - URL查询不需要租户隔离，支持平台级资源与租户资源混合使用
//   - 租户隔离在下载时由其他接口处理
func NewResolver(client *resource.ResourceClient) Resolver {
	return &resourceResolver{
		client: client,
		opts: &ResolverOptions{
			IncludeVariants: true,
			ExpiresIn:       3600,
		},
	}
}

// NewResolverWithOptions 创建带选项的解析器
//
// 参数:
//   - client: 资源服务客户端
//   - opts: 解析器选项
//
// 使用示例:
//
//	resolver := image.NewResolverWithOptions(resourceClient, &image.ResolverOptions{
//	    IncludeVariants: true,
//	    ExpiresIn:       7200,
//	})
func NewResolverWithOptions(client *resource.ResourceClient, opts *ResolverOptions) Resolver {
	if opts == nil {
		opts = &ResolverOptions{
			IncludeVariants: true,
			ExpiresIn:       3600,
		}
	}
	return &resourceResolver{
		client: client,
		opts:   opts,
	}
}

// Resolve 实现 Resolver 接口
func (r *resourceResolver) Resolve(ctx context.Context, ids []string) (map[string]*ResourceInfo, error) {
	if len(ids) == 0 {
		return make(map[string]*ResourceInfo), nil
	}

	results, err := r.client.GetFileUrls(ctx, ids, &resource.GetFileUrlsOptions{
		IncludeVariants: r.opts.IncludeVariants,
		ExpiresIn:       r.opts.ExpiresIn,
	})
	if err != nil {
		return nil, err
	}

	resources := make(map[string]*ResourceInfo, len(results))
	for id, info := range results {
		resources[id] = &ResourceInfo{
			URL:      info.Url,
			Variants: info.VariantUrls,
			Success:  info.Success,
			Error:    info.Error,
		}
	}

	return resources, nil
}
