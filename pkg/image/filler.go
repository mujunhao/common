package image

import (
	"context"
)

// Filler 图片URL填充器
//
// 负责收集绑定的文件ID，批量查询URL，然后分发填充
type Filler struct {
	resolver Resolver
}

// NewFiller 创建填充器
//
// 参数:
//   - resolver: URL解析器
//
// 使用示例:
//
//	resolver := image.NewResolver(resourceClient, getTenantCode)
//	filler := image.NewFiller(resolver)
func NewFiller(resolver Resolver) *Filler {
	return &Filler{resolver: resolver}
}

// Fill 填充资源URL
//
// 收集所有绑定的文件ID，去重后批量查询，然后分发填充
//
// 参数:
//   - ctx: 上下文
//   - bindings: 字段绑定列表
//
// 使用示例:
//
//	filler.Fill(ctx,
//	    image.Single(&p.CoverID, &p.CoverURL),
//	    image.Multi(&p.GalleryIDs, &p.GalleryURLs),
//	    image.Rich(&p.Detail, &p.DetailHTML),
//	)
func (f *Filler) Fill(ctx context.Context, bindings ...Binding) error {
	if len(bindings) == 0 {
		return nil
	}

	// 1. 收集所有ID并去重
	idSet := make(map[string]struct{})
	for _, b := range bindings {
		if b == nil {
			continue
		}
		for _, id := range b.collectIDs() {
			idSet[id] = struct{}{}
		}
	}

	if len(idSet) == 0 {
		return nil
	}

	// 2. 转换为切片
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	// 3. 批量查询
	resources, err := f.resolver.Resolve(ctx, ids)
	if err != nil {
		return err
	}

	// 4. 填充所有绑定
	for _, b := range bindings {
		if b != nil {
			b.fill(resources)
		}
	}

	return nil
}

// ==================== 泛型辅助函数 ====================

// BindingFunc 绑定函数类型
//
// 用于定义结构体的字段绑定关系
type BindingFunc[T any] func(*T) []Binding

// FillOne 填充单个对象
//
// 参数:
//   - ctx: 上下文
//   - f: 填充器
//   - item: 要填充的对象指针
//   - bindFn: 绑定函数
//
// 使用示例:
//
//	func ProductBindings(p *Product) []image.Binding {
//	    return []image.Binding{
//	        image.Single(&p.CoverID, &p.CoverURL),
//	        image.Multi(&p.GalleryIDs, &p.GalleryURLs),
//	    }
//	}
//
//	image.FillOne(ctx, filler, product, ProductBindings)
func FillOne[T any](ctx context.Context, f *Filler, item *T, bindFn BindingFunc[T]) error {
	if item == nil {
		return nil
	}
	return f.Fill(ctx, bindFn(item)...)
}

// FillSlice 批量填充对象切片
//
// 所有对象的文件ID会合并去重后一次性查询，然后分发填充
// 这是最高效的批量填充方式
//
// 参数:
//   - ctx: 上下文
//   - f: 填充器
//   - items: 要填充的对象切片
//   - bindFn: 绑定函数
//
// 使用示例:
//
//	products, _ := repo.ListProducts(ctx)
//	image.FillSlice(ctx, filler, products, ProductBindings)
func FillSlice[T any](ctx context.Context, f *Filler, items []*T, bindFn BindingFunc[T]) error {
	if len(items) == 0 {
		return nil
	}

	var bindings []Binding
	for _, item := range items {
		if item != nil {
			bindings = append(bindings, bindFn(item)...)
		}
	}

	return f.Fill(ctx, bindings...)
}

// FillMap 填充 map 中的对象
//
// 参数:
//   - ctx: 上下文
//   - f: 填充器
//   - items: 要填充的对象 map
//   - bindFn: 绑定函数
//
// 使用示例:
//
//	productsMap := map[string]*Product{...}
//	image.FillMap(ctx, filler, productsMap, ProductBindings)
func FillMap[K comparable, V any](ctx context.Context, f *Filler, items map[K]*V, bindFn BindingFunc[V]) error {
	if len(items) == 0 {
		return nil
	}

	var bindings []Binding
	for _, item := range items {
		if item != nil {
			bindings = append(bindings, bindFn(item)...)
		}
	}

	return f.Fill(ctx, bindings...)
}
