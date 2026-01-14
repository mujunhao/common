package image

// ResourceInfo 资源信息
type ResourceInfo struct {
	// URL 资源访问URL
	URL string
	// Variants 变体URL映射（如缩略图、裁剪图等）
	// key: 变体ID，如 "thumbnail_200x200"
	// value: 变体URL
	Variants map[string]string
	// Success 是否成功获取
	Success bool
	// Error 错误信息（Success=false时）
	Error string
}

// GetVariant 获取指定变体的URL
// 如果变体不存在，返回原图URL
func (r *ResourceInfo) GetVariant(name string) string {
	if r.Variants != nil {
		if url, ok := r.Variants[name]; ok {
			return url
		}
	}
	return r.URL
}
