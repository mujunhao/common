package image

import (
	"regexp"
)

// Binding 字段绑定接口
type Binding interface {
	collectIDs() []string
	fill(resources map[string]*ResourceInfo)
}

// ==================== Single 单图绑定 ====================

type singleBinding[T any] struct {
	id     *string
	target *T
	fillFn func(*ResourceInfo) T
}

// Single 创建单图绑定
//
// 将文件ID对应的URL填充到目标字段
//
// 参数:
//   - id: 文件ID字段指针
//   - url: 目标URL字段指针
//
// 使用示例:
//
//	image.Single(&p.CoverID, &p.CoverURL)
func Single(id *string, url *string) Binding {
	return SingleTo(id, url, func(r *ResourceInfo) string {
		return r.URL
	})
}

// SingleTo 创建单图绑定（泛型版本）
//
// 将文件ID对应的资源信息转换后填充到目标字段
//
// 参数:
//   - id: 文件ID字段指针
//   - target: 目标字段指针（任意类型）
//   - fillFn: 转换函数，将 ResourceInfo 转换为目标类型
//
// 使用示例:
//
//	type ImageData struct {
//	    URL       string
//	    Thumbnail string
//	}
//
//	image.SingleTo(&p.CoverID, &p.CoverData, func(r *image.ResourceInfo) ImageData {
//	    return ImageData{
//	        URL:       r.URL,
//	        Thumbnail: r.GetVariant("thumbnail"),
//	    }
//	})
func SingleTo[T any](id *string, target *T, fillFn func(*ResourceInfo) T) Binding {
	return &singleBinding[T]{
		id:     id,
		target: target,
		fillFn: fillFn,
	}
}

func (b *singleBinding[T]) collectIDs() []string {
	if b.id == nil || *b.id == "" {
		return nil
	}
	return []string{*b.id}
}

func (b *singleBinding[T]) fill(resources map[string]*ResourceInfo) {
	if b.id == nil || *b.id == "" || b.target == nil {
		return
	}
	if info, ok := resources[*b.id]; ok && info.Success {
		*b.target = b.fillFn(info)
	}
}

// ==================== Multi 多图绑定 ====================

type multiBinding[T any] struct {
	ids     *[]string
	targets *[]T
	fillFn  func(*ResourceInfo) T
}

// Multi 创建多图绑定
//
// 将文件ID列表对应的URL列表填充到目标字段
// 保持ID和URL的顺序对应
//
// 参数:
//   - ids: 文件ID列表字段指针
//   - urls: 目标URL列表字段指针
//
// 使用示例:
//
//	image.Multi(&p.GalleryIDs, &p.GalleryURLs)
func Multi(ids *[]string, urls *[]string) Binding {
	return MultiTo(ids, urls, func(r *ResourceInfo) string {
		return r.URL
	})
}

// MultiTo 创建多图绑定（泛型版本）
//
// 将文件ID列表对应的资源信息转换后填充到目标字段
//
// 参数:
//   - ids: 文件ID列表字段指针
//   - targets: 目标列表字段指针（任意类型）
//   - fillFn: 转换函数
//
// 使用示例:
//
//	image.MultiTo(&p.GalleryIDs, &p.GalleryData, func(r *image.ResourceInfo) ImageData {
//	    return ImageData{URL: r.URL, Thumbnail: r.GetVariant("thumb")}
//	})
func MultiTo[T any](ids *[]string, targets *[]T, fillFn func(*ResourceInfo) T) Binding {
	return &multiBinding[T]{
		ids:     ids,
		targets: targets,
		fillFn:  fillFn,
	}
}

func (b *multiBinding[T]) collectIDs() []string {
	if b.ids == nil || len(*b.ids) == 0 {
		return nil
	}
	result := make([]string, 0, len(*b.ids))
	for _, id := range *b.ids {
		if id != "" {
			result = append(result, id)
		}
	}
	return result
}

func (b *multiBinding[T]) fill(resources map[string]*ResourceInfo) {
	if b.ids == nil || len(*b.ids) == 0 || b.targets == nil {
		return
	}
	ids := *b.ids
	results := make([]T, len(ids))
	for i, id := range ids {
		if id == "" {
			continue
		}
		if info, ok := resources[id]; ok && info.Success {
			results[i] = b.fillFn(info)
		}
	}
	*b.targets = results
}

// ==================== Rich 富文本绑定 ====================

// 默认图片占位符正则：data-href="file_id" src="..."
// 匹配 data-href="fileID" src="任意内容" 格式，替换后保留 data-href，更新 src 为新URL
var defaultPattern = regexp.MustCompile(`data-href="([a-zA-Z0-9_-]+)" src="[^"]*"`)

type richBinding struct {
	raw      *string
	rendered *string
	pattern  *regexp.Regexp
	variant  string
}

// Rich 创建富文本绑定
//
// 替换富文本中的图片占位符为实际URL
// 占位符格式：{{img:file_id}}
//
// 参数:
//   - raw: 原始富文本字段指针
//   - rendered: 渲染后的富文本字段指针
//
// 使用示例:
//
//	image.Rich(&p.Description, &p.DescriptionHTML)
func Rich(raw *string, rendered *string) *richBinding {
	return &richBinding{
		raw:      raw,
		rendered: rendered,
		pattern:  defaultPattern,
	}
}

// Pattern 设置自定义匹配模式
//
// 正则必须包含一个捕获组用于提取文件ID
//
// 使用示例:
//
//	image.Rich(&p.Content, &p.ContentHTML).Pattern(regexp.MustCompile(`\[img:(\w+)\]`))
func (b *richBinding) Pattern(p *regexp.Regexp) *richBinding {
	b.pattern = p
	return b
}

// UseVariant 使用指定变体URL替换
//
// 使用示例:
//
//	image.Rich(&p.Content, &p.ContentHTML).UseVariant("thumbnail_800x800")
func (b *richBinding) UseVariant(name string) *richBinding {
	b.variant = name
	return b
}

func (b *richBinding) collectIDs() []string {
	if b.raw == nil || *b.raw == "" {
		return nil
	}
	matches := b.pattern.FindAllStringSubmatch(*b.raw, -1)
	if len(matches) == 0 {
		return nil
	}
	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 && m[1] != "" {
			ids = append(ids, m[1])
		}
	}
	return ids
}

func (b *richBinding) fill(resources map[string]*ResourceInfo) {
	if b.raw == nil || *b.raw == "" || b.rendered == nil {
		return
	}
	*b.rendered = b.pattern.ReplaceAllStringFunc(*b.raw, func(match string) string {
		subs := b.pattern.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		fileID := subs[1]
		info, ok := resources[fileID]
		if !ok || !info.Success {
			return match // 保持原占位符
		}
		var url string
		if b.variant != "" {
			url = info.GetVariant(b.variant)
		} else {
			url = info.URL
		}
		// 保留 data-href，更新 src
		return `data-href="` + fileID + `" src="` + url + `"`
	})
}
