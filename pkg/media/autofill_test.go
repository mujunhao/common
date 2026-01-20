package media

import (
	"context"
	"testing"
)

// 模拟 Resolver
type autoFillMockResolver struct {
	data map[string]*ResourceInfo
}

func (m *autoFillMockResolver) Resolve(ctx context.Context, ids []string) (map[string]*ResourceInfo, error) {
	result := make(map[string]*ResourceInfo)
	for _, id := range ids {
		if info, ok := m.data[id]; ok {
			result[id] = info
		}
	}
	return result, nil
}

// ========== 源结构体（模拟 ent） ==========

type ProductLanguage struct {
	Name        string
	Cover       string   // 文件ID
	Gallery     []string // 文件ID列表
	Description string   // 富文本
}

type Product struct {
	ID        uint32
	Points    float64
	Status    int32
	Languages map[string]*ProductLanguage
}

// ========== 目标结构体（DTO）- 双字段模式 ==========

type ProductLangDTO struct {
	Name        string   `json:"name"`
	Cover       FileID   `json:"cover"`                       // ID 保持不变
	CoverURL    URL      `json:"cover_url" media:"Cover"`     // URL 从 Cover 获取
	Gallery     FileIDs  `json:"gallery"`                     // IDs 保持不变
	GalleryURL  URLs     `json:"gallery_url" media:"Gallery"` // URLs 从 Gallery 获取
	Description RichText `json:"description"`                 // 富文本
}

type ProductDTO struct {
	ID        uint32                     `json:"id"`
	Points    float64                    `json:"points"`
	Status    int32                      `json:"status"`
	Languages map[string]*ProductLangDTO `json:"languages"`
}

func TestAutoFill(t *testing.T) {
	// 模拟文件URL映射
	resolver := &autoFillMockResolver{
		data: map[string]*ResourceInfo{
			"cover_id":  {URL: "https://cdn.example.com/cover.jpg", Success: true},
			"gallery_1": {URL: "https://cdn.example.com/g1.jpg", Success: true},
			"gallery_2": {URL: "https://cdn.example.com/g2.jpg", Success: true},
			"rich_img":  {URL: "https://cdn.example.com/rich.jpg", Success: true},
			"cover_en":  {URL: "https://cdn.example.com/cover_en.jpg", Success: true},
			"video_id":  {URL: "https://cdn.example.com/video.mp4", Success: true},
		},
	}
	filler := NewFiller(resolver)

	// 源数据（模拟从数据库查询）
	products := []*Product{
		{
			ID:     1,
			Points: 99.9,
			Status: 1,
			Languages: map[string]*ProductLanguage{
				"zh": {
					Name:        "商品A",
					Cover:       "cover_id",
					Gallery:     []string{"gallery_1", "gallery_2"},
					Description: `<p>介绍</p><img data-href="rich_img"><video data-href="video_id"></video>`,
				},
				"en": {
					Name:        "Product A",
					Cover:       "cover_en",
					Gallery:     []string{"gallery_1"},
					Description: `<p>Description</p>`,
				},
			},
		},
	}

	// 执行 AutoFill
	var result []*ProductDTO
	err := AutoFill(context.Background(), filler, products, &result)
	if err != nil {
		t.Fatalf("AutoFill error: %v", err)
	}

	// 验证结果
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	dto := result[0]

	// 验证基本字段
	if dto.ID != 1 {
		t.Errorf("ID: expected 1, got %d", dto.ID)
	}
	if dto.Points != 99.9 {
		t.Errorf("Points: expected 99.9, got %f", dto.Points)
	}
	if dto.Status != 1 {
		t.Errorf("Status: expected 1, got %d", dto.Status)
	}

	// 验证中文
	zh := dto.Languages["zh"]
	if zh == nil {
		t.Fatal("zh language is nil")
	}
	if zh.Name != "商品A" {
		t.Errorf("zh.Name: expected 商品A, got %s", zh.Name)
	}

	// 验证双字段模式 - ID保持不变
	if string(zh.Cover) != "cover_id" {
		t.Errorf("zh.Cover (ID): expected cover_id, got %s", zh.Cover)
	}
	// 验证双字段模式 - URL自动填充
	if string(zh.CoverURL) != "https://cdn.example.com/cover.jpg" {
		t.Errorf("zh.CoverURL: expected URL, got %s", zh.CoverURL)
	}

	// 验证多图 - IDs保持不变
	if len(zh.Gallery) != 2 {
		t.Errorf("zh.Gallery: expected 2 items, got %d", len(zh.Gallery))
	}
	if string(zh.Gallery[0]) != "gallery_1" {
		t.Errorf("zh.Gallery[0] (ID): expected gallery_1, got %s", zh.Gallery[0])
	}

	// 验证多图 - URLs自动填充
	if len(zh.GalleryURL) != 2 {
		t.Errorf("zh.GalleryURL: expected 2 items, got %d", len(zh.GalleryURL))
	}
	if zh.GalleryURL[0] != "https://cdn.example.com/g1.jpg" {
		t.Errorf("zh.GalleryURL[0]: expected URL, got %s", zh.GalleryURL[0])
	}
	if zh.GalleryURL[1] != "https://cdn.example.com/g2.jpg" {
		t.Errorf("zh.GalleryURL[1]: expected URL, got %s", zh.GalleryURL[1])
	}

	// 验证富文本替换
	expectedDesc := `<p>介绍</p><img src="https://cdn.example.com/rich.jpg"><video src="https://cdn.example.com/video.mp4"></video>`
	if string(zh.Description) != expectedDesc {
		t.Errorf("zh.Description:\nexpected: %s\ngot: %s", expectedDesc, zh.Description)
	}

	// 验证英文
	en := dto.Languages["en"]
	if en == nil {
		t.Fatal("en language is nil")
	}
	if string(en.Cover) != "cover_en" {
		t.Errorf("en.Cover (ID): expected cover_en, got %s", en.Cover)
	}
	if string(en.CoverURL) != "https://cdn.example.com/cover_en.jpg" {
		t.Errorf("en.CoverURL: expected URL, got %s", en.CoverURL)
	}

	t.Log("All tests passed!")
	t.Logf("zh.Cover (ID): %s", zh.Cover)
	t.Logf("zh.CoverURL: %s", zh.CoverURL)
	t.Logf("zh.Gallery (IDs): %v", zh.Gallery)
	t.Logf("zh.GalleryURL: %v", zh.GalleryURL)
	t.Logf("zh.Description: %s", zh.Description)
}

func TestAutoFillOne(t *testing.T) {
	resolver := &autoFillMockResolver{
		data: map[string]*ResourceInfo{
			"single_cover": {URL: "https://cdn.example.com/single.jpg", Success: true},
		},
	}
	filler := NewFiller(resolver)

	src := &Product{
		ID:     2,
		Points: 50.0,
		Languages: map[string]*ProductLanguage{
			"zh": {
				Name:  "单个商品",
				Cover: "single_cover",
			},
		},
	}

	var dst ProductDTO
	err := AutoFillOne(context.Background(), filler, src, &dst)
	if err != nil {
		t.Fatalf("AutoFillOne error: %v", err)
	}

	if dst.ID != 2 {
		t.Errorf("ID: expected 2, got %d", dst.ID)
	}
	if string(dst.Languages["zh"].Cover) != "single_cover" {
		t.Errorf("Cover (ID): expected single_cover, got %s", dst.Languages["zh"].Cover)
	}
	if string(dst.Languages["zh"].CoverURL) != "https://cdn.example.com/single.jpg" {
		t.Errorf("CoverURL: expected URL, got %s", dst.Languages["zh"].CoverURL)
	}

	t.Log("AutoFillOne test passed!")
}

// ========== 模拟实际 ent.Product 结构（I18n 是 map[string]interface{}）==========

type EntProduct struct {
	ID          uint32
	ProductCode string
	ProductName string
	Image       string                 // 文件ID (UUID)
	I18n        map[string]interface{} // 多语言内容，模拟 ent 的 JSON 字段
}

// ========== 目标结构体（模拟实际 ProductDTO）==========

type TestProductLangDTO struct {
	Name        string   `json:"name"`
	Description RichText `json:"description"` // 富文本，包含 data-href
}

type TestProductDTO struct {
	ID          uint32                         `json:"id"`
	ProductCode string                         `json:"product_code"`
	ProductName string                         `json:"product_name"`
	Image       FileID                         `json:"image"`                   // ID 保持不变
	ImageURL    URL                            `json:"image_url" media:"Image"` // URL 从 Image 获取
	I18n        map[string]*TestProductLangDTO `json:"i18n"`                    // 多语言
}

// TestAutoFillWithInterfaceMap 测试 map[string]interface{} 到 map[string]*Struct 的转换
// 这是实际 MerchantGetProduct API 的场景
func TestAutoFillWithInterfaceMap(t *testing.T) {
	// 模拟文件URL映射 - 使用实际的 ULID 格式
	resolver := &autoFillMockResolver{
		data: map[string]*ResourceInfo{
			"01KEVAE4NE69CFVG0XJ3K6R82Z": {URL: "https://cdn.example.com/new-image.jpg?sign=fresh123", Success: true},
			"01KEXGF5VGAMAH4TVMAG28CRMM": {URL: "https://cdn.example.com/new-rich.jpg?sign=fresh456", Success: true},
			"01KES4MXS651DAAXBDS1N1574T": {URL: "https://cdn.example.com/new-rich2.jpg?sign=fresh789", Success: true},
		},
	}
	filler := NewFiller(resolver)

	// 模拟实际 API 返回的数据格式
	// 注意：src 属性在前，data-href 在后（这是实际数据的格式）
	products := []*EntProduct{
		{
			ID:          46,
			ProductCode: "1768876056614-4f6e2566867641488f5ba6aa2f526d8e",
			ProductName: "ipone(测试详情页)",
			Image:       "01KEVAE4NE69CFVG0XJ3K6R82Z",
			I18n: map[string]interface{}{
				"zh-CN": map[string]interface{}{
					"name":        "ipone(测试详情页)",
					"description": `<p>测试<img src="https://old-url.com/old.jpg?old-sign" alt="" data-href="01KEXGF5VGAMAH4TVMAG28CRMM" style=""/></p>`,
				},
				"en-US": map[string]interface{}{
					"name":        "ipone(测试详情页)",
					"description": `<p>测试English</p><p><img src="https://old-url.com/old2.jpg?old-sign" alt="" data-href="01KEXGF5VGAMAH4TVMAG28CRMM" style=""/></p>`,
				},
				"ar-SA": map[string]interface{}{
					"name":        "ipone(测试详情页)",
					"description": `<p>测试<img src="https://old-url.com/old3.jpg?old-sign" alt="" data-href="01KES4MXS651DAAXBDS1N1574T" style=""/></p>`,
				},
			},
		},
	}

	// 执行 AutoFill
	var result []*TestProductDTO
	err := AutoFill(context.Background(), filler, products, &result)
	if err != nil {
		t.Fatalf("AutoFill error: %v", err)
	}

	// 验证结果
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	dto := result[0]

	// 验证基本字段
	if dto.ID != 46 {
		t.Errorf("ID: expected 46, got %d", dto.ID)
	}
	if dto.ProductCode != "1768876056614-4f6e2566867641488f5ba6aa2f526d8e" {
		t.Errorf("ProductCode mismatch")
	}

	// 验证 Image URL 转换
	if string(dto.Image) != "01KEVAE4NE69CFVG0XJ3K6R82Z" {
		t.Errorf("Image (ID): expected 01KEVAE4NE69CFVG0XJ3K6R82Z, got %s", dto.Image)
	}
	if string(dto.ImageURL) != "https://cdn.example.com/new-image.jpg?sign=fresh123" {
		t.Errorf("ImageURL: expected fresh URL, got %s", dto.ImageURL)
	}

	// 验证 I18n 不为空
	if dto.I18n == nil {
		t.Fatal("I18n is nil")
	}

	// 验证中文
	zhCN := dto.I18n["zh-CN"]
	if zhCN == nil {
		t.Fatal("zh-CN language is nil")
	}
	if zhCN.Name != "ipone(测试详情页)" {
		t.Errorf("zh-CN.Name: expected ipone(测试详情页), got %s", zhCN.Name)
	}

	// 验证富文本 URL 被替换（关键测试！）
	// 原始: src="https://old-url.com/old.jpg?old-sign" ... data-href="01KEXGF5VGAMAH4TVMAG28CRMM"
	// 期望: src="https://cdn.example.com/new-rich.jpg?sign=fresh456" ... data-href="01KEXGF5VGAMAH4TVMAG28CRMM"
	expectedZhDesc := `<p>测试<img src="https://cdn.example.com/new-rich.jpg?sign=fresh456" alt="" data-href="01KEXGF5VGAMAH4TVMAG28CRMM" style=""/></p>`
	if string(zhCN.Description) != expectedZhDesc {
		t.Errorf("zh-CN.Description URL not replaced!\nexpected: %s\ngot: %s", expectedZhDesc, zhCN.Description)
	}

	// 验证英文
	enUS := dto.I18n["en-US"]
	if enUS == nil {
		t.Fatal("en-US language is nil")
	}
	expectedEnDesc := `<p>测试English</p><p><img src="https://cdn.example.com/new-rich.jpg?sign=fresh456" alt="" data-href="01KEXGF5VGAMAH4TVMAG28CRMM" style=""/></p>`
	if string(enUS.Description) != expectedEnDesc {
		t.Errorf("en-US.Description URL not replaced!\nexpected: %s\ngot: %s", expectedEnDesc, enUS.Description)
	}

	// 验证阿拉伯语（使用不同的 data-href ID）
	arSA := dto.I18n["ar-SA"]
	if arSA == nil {
		t.Fatal("ar-SA language is nil")
	}
	expectedArDesc := `<p>测试<img src="https://cdn.example.com/new-rich2.jpg?sign=fresh789" alt="" data-href="01KES4MXS651DAAXBDS1N1574T" style=""/></p>`
	if string(arSA.Description) != expectedArDesc {
		t.Errorf("ar-SA.Description URL not replaced!\nexpected: %s\ngot: %s", expectedArDesc, arSA.Description)
	}

	t.Log("TestAutoFillWithInterfaceMap passed!")
	t.Logf("Image (ID): %s", dto.Image)
	t.Logf("ImageURL: %s", dto.ImageURL)
	t.Logf("zh-CN.Description: %s", zhCN.Description)
	t.Logf("en-US.Description: %s", enUS.Description)
	t.Logf("ar-SA.Description: %s", arSA.Description)
}

// TestDataHrefRegex 单独测试正则表达式
func TestDataHrefRegex(t *testing.T) {
	// 测试实际数据格式：src 在前，data-href 在后
	html := `<img src="https://old-url.com/old.jpg?q-sign=xxx" alt="" data-href="01KEXGF5VGAMAH4TVMAG28CRMM" style=""/>`

	// 测试提取 ID
	ids := extractDataHrefIDs(html)
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID, got %d", len(ids))
	}
	if ids[0] != "01KEXGF5VGAMAH4TVMAG28CRMM" {
		t.Errorf("expected 01KEXGF5VGAMAH4TVMAG28CRMM, got %s", ids[0])
	}
	t.Logf("Extracted IDs: %v", ids)

	// 测试替换 URL
	resources := map[string]*ResourceInfo{
		"01KEXGF5VGAMAH4TVMAG28CRMM": {URL: "https://new-url.com/fresh-signed-url.jpg", Success: true},
	}
	newHTML := replaceDataHrefURLs(html, resources)

	expectedHTML := `<img src="https://new-url.com/fresh-signed-url.jpg" alt="" data-href="01KEXGF5VGAMAH4TVMAG28CRMM" style=""/>`
	if newHTML != expectedHTML {
		t.Errorf("URL replacement failed!\nexpected: %s\ngot: %s", expectedHTML, newHTML)
	}

	t.Logf("Original HTML: %s", html[:80]+"...")
	t.Logf("New HTML: %s", newHTML[:80]+"...")
}
