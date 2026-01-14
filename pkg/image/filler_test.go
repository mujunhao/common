package image

import (
	"context"
	"testing"
)

// mockResolver 测试用的 mock 解析器
type mockResolver struct {
	data map[string]*ResourceInfo
}

func newMockResolver(data map[string]*ResourceInfo) *mockResolver {
	return &mockResolver{data: data}
}

func (m *mockResolver) Resolve(ctx context.Context, ids []string) (map[string]*ResourceInfo, error) {
	result := make(map[string]*ResourceInfo)
	for _, id := range ids {
		if info, ok := m.data[id]; ok {
			result[id] = info
		}
	}
	return result, nil
}

// 测试数据
var testData = map[string]*ResourceInfo{
	"file_1": {
		URL:      "https://cdn.example.com/file_1.jpg",
		Variants: map[string]string{"thumbnail": "https://cdn.example.com/file_1_thumb.jpg"},
		Success:  true,
	},
	"file_2": {
		URL:      "https://cdn.example.com/file_2.jpg",
		Variants: map[string]string{"thumbnail": "https://cdn.example.com/file_2_thumb.jpg"},
		Success:  true,
	},
	"file_3": {
		URL:      "https://cdn.example.com/file_3.jpg",
		Variants: map[string]string{"thumbnail": "https://cdn.example.com/file_3_thumb.jpg"},
		Success:  true,
	},
	"file_failed": {
		Success: false,
		Error:   "file not found",
	},
}

func TestSingle(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	id := "file_1"
	var url string

	err := filler.Fill(ctx, Single(&id, &url))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	if url != "https://cdn.example.com/file_1.jpg" {
		t.Errorf("expected url to be filled, got: %s", url)
	}
}

func TestSingleEmpty(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	id := ""
	var url string

	err := filler.Fill(ctx, Single(&id, &url))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	if url != "" {
		t.Errorf("expected url to be empty, got: %s", url)
	}
}

func TestSingleFailed(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	id := "file_failed"
	url := "original"

	err := filler.Fill(ctx, Single(&id, &url))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	// 失败时保持原值
	if url != "original" {
		t.Errorf("expected url to keep original value, got: %s", url)
	}
}

func TestSingleTo(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	type ImageData struct {
		URL       string
		Thumbnail string
	}

	id := "file_1"
	var data ImageData

	err := filler.Fill(ctx, SingleTo(&id, &data, func(r *ResourceInfo) ImageData {
		return ImageData{
			URL:       r.URL,
			Thumbnail: r.GetVariant("thumbnail"),
		}
	}))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	if data.URL != "https://cdn.example.com/file_1.jpg" {
		t.Errorf("expected data.URL, got: %s", data.URL)
	}
	if data.Thumbnail != "https://cdn.example.com/file_1_thumb.jpg" {
		t.Errorf("expected data.Thumbnail, got: %s", data.Thumbnail)
	}
}

func TestMulti(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	ids := []string{"file_1", "file_2", "file_3"}
	var urls []string

	err := filler.Fill(ctx, Multi(&ids, &urls))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	if len(urls) != 3 {
		t.Fatalf("expected 3 urls, got: %d", len(urls))
	}

	expected := []string{
		"https://cdn.example.com/file_1.jpg",
		"https://cdn.example.com/file_2.jpg",
		"https://cdn.example.com/file_3.jpg",
	}
	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("urls[%d] expected %s, got: %s", i, expected[i], url)
		}
	}
}

func TestMultiWithEmpty(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	ids := []string{"file_1", "", "file_3"}
	var urls []string

	err := filler.Fill(ctx, Multi(&ids, &urls))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	if len(urls) != 3 {
		t.Fatalf("expected 3 urls, got: %d", len(urls))
	}

	// 空ID对应的URL也应该是空
	if urls[1] != "" {
		t.Errorf("urls[1] expected empty, got: %s", urls[1])
	}
}

func TestMultiWithFailed(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	ids := []string{"file_1", "file_failed", "file_3"}
	var urls []string

	err := filler.Fill(ctx, Multi(&ids, &urls))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	if len(urls) != 3 {
		t.Fatalf("expected 3 urls, got: %d", len(urls))
	}

	// 失败的ID对应的URL应该是空
	if urls[1] != "" {
		t.Errorf("urls[1] expected empty for failed file, got: %s", urls[1])
	}
}

func TestRich(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	raw := `Hello <img data-href="file_1" src="old1.jpg"> World <img data-href="file_2" src="old2.jpg">`
	var rendered string

	err := filler.Fill(ctx, Rich(&raw, &rendered))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	expected := `Hello <img data-href="file_1" src="https://cdn.example.com/file_1.jpg"> World <img data-href="file_2" src="https://cdn.example.com/file_2.jpg">`
	if rendered != expected {
		t.Errorf("expected: %s\ngot: %s", expected, rendered)
	}
}

func TestRichWithVariant(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	raw := `Thumbnail: <img data-href="file_1" src="old.jpg">`
	var rendered string

	err := filler.Fill(ctx, Rich(&raw, &rendered).UseVariant("thumbnail"))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	expected := `Thumbnail: <img data-href="file_1" src="https://cdn.example.com/file_1_thumb.jpg">`
	if rendered != expected {
		t.Errorf("expected: %s\ngot: %s", expected, rendered)
	}
}

func TestRichWithFailed(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	raw := `Image: <img data-href="file_failed" src="old.jpg">`
	var rendered string

	err := filler.Fill(ctx, Rich(&raw, &rendered))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	// 失败的保持原占位符
	expected := `Image: <img data-href="file_failed" src="old.jpg">`
	if rendered != expected {
		t.Errorf("expected: %s\ngot: %s", expected, rendered)
	}
}

func TestFillOne(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	type Product struct {
		CoverID  string
		CoverURL string
	}

	productBindings := func(p *Product) []Binding {
		return []Binding{
			Single(&p.CoverID, &p.CoverURL),
		}
	}

	product := &Product{CoverID: "file_1"}

	err := FillOne(ctx, filler, product, productBindings)
	if err != nil {
		t.Fatalf("FillOne failed: %v", err)
	}

	if product.CoverURL != "https://cdn.example.com/file_1.jpg" {
		t.Errorf("expected CoverURL to be filled, got: %s", product.CoverURL)
	}
}

func TestFillSlice(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	type Product struct {
		CoverID  string
		CoverURL string
	}

	productBindings := func(p *Product) []Binding {
		return []Binding{
			Single(&p.CoverID, &p.CoverURL),
		}
	}

	products := []*Product{
		{CoverID: "file_1"},
		{CoverID: "file_2"},
		{CoverID: "file_3"},
	}

	err := FillSlice(ctx, filler, products, productBindings)
	if err != nil {
		t.Fatalf("FillSlice failed: %v", err)
	}

	expected := []string{
		"https://cdn.example.com/file_1.jpg",
		"https://cdn.example.com/file_2.jpg",
		"https://cdn.example.com/file_3.jpg",
	}

	for i, p := range products {
		if p.CoverURL != expected[i] {
			t.Errorf("products[%d].CoverURL expected %s, got: %s", i, expected[i], p.CoverURL)
		}
	}
}

func TestNestedStruct(t *testing.T) {
	filler := NewFiller(newMockResolver(testData))
	ctx := context.Background()

	type LangContent struct {
		BannerID  string
		BannerURL string
	}

	type I18n struct {
		ZH *LangContent
		EN *LangContent
	}

	type Product struct {
		CoverID  string
		CoverURL string
		I18n     I18n
	}

	langBindings := func(l *LangContent) []Binding {
		if l == nil {
			return nil
		}
		return []Binding{
			Single(&l.BannerID, &l.BannerURL),
		}
	}

	productBindings := func(p *Product) []Binding {
		bindings := []Binding{
			Single(&p.CoverID, &p.CoverURL),
		}
		bindings = append(bindings, langBindings(p.I18n.ZH)...)
		bindings = append(bindings, langBindings(p.I18n.EN)...)
		return bindings
	}

	product := &Product{
		CoverID: "file_1",
		I18n: I18n{
			ZH: &LangContent{BannerID: "file_2"},
			EN: &LangContent{BannerID: "file_3"},
		},
	}

	err := FillOne(ctx, filler, product, productBindings)
	if err != nil {
		t.Fatalf("FillOne failed: %v", err)
	}

	if product.CoverURL != "https://cdn.example.com/file_1.jpg" {
		t.Errorf("CoverURL expected, got: %s", product.CoverURL)
	}
	if product.I18n.ZH.BannerURL != "https://cdn.example.com/file_2.jpg" {
		t.Errorf("I18n.ZH.BannerURL expected, got: %s", product.I18n.ZH.BannerURL)
	}
	if product.I18n.EN.BannerURL != "https://cdn.example.com/file_3.jpg" {
		t.Errorf("I18n.EN.BannerURL expected, got: %s", product.I18n.EN.BannerURL)
	}
}

func TestDeduplication(t *testing.T) {
	// 验证相同ID只查询一次
	callCount := 0
	resolver := &countingResolver{
		data: testData,
		onResolve: func(ids []string) {
			callCount++
			// 应该只有一个ID
			if len(ids) != 1 {
				t.Errorf("expected 1 unique ID, got: %d", len(ids))
			}
		},
	}

	filler := NewFiller(resolver)
	ctx := context.Background()

	// 同一个ID出现多次
	id1 := "file_1"
	id2 := "file_1"
	var url1, url2 string

	err := filler.Fill(ctx, Single(&id1, &url1), Single(&id2, &url2))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 resolve call, got: %d", callCount)
	}

	if url1 != url2 {
		t.Errorf("same ID should produce same URL")
	}
}

// TestRealFileIDs 测试真实文件ID场景
func TestRealFileIDs(t *testing.T) {
	// 模拟真实文件ID的数据
	realTestData := map[string]*ResourceInfo{
		"01K9Y2ZCET0DGEV8FQZV8MM3EJ": {
			URL:      "https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ_thumb.jpg"},
			Success:  true,
		},
		"01KEMKBGE856M5CHA29YC2E7RD": {
			URL:      "https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD_thumb.jpg"},
			Success:  true,
		},
		"01KER5ED3B9F4ZRGZ9X1MMTTMX": {
			URL:      "https://cdn.example.com/01KER5ED3B9F4ZRGZ9X1MMTTMX.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01KER5ED3B9F4ZRGZ9X1MMTTMX_thumb.jpg"},
			Success:  true,
		},
		"01KER6SQYDZ15JD47ZCQSKYS31": {
			URL:      "https://cdn.example.com/01KER6SQYDZ15JD47ZCQSKYS31.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01KER6SQYDZ15JD47ZCQSKYS31_thumb.jpg"},
			Success:  true,
		},
	}

	filler := NewFiller(newMockResolver(realTestData))
	ctx := context.Background()

	type Product struct {
		CoverID     string
		CoverURL    string
		GalleryIDs  []string
		GalleryURLs []string
	}

	product := &Product{
		CoverID: "01K9Y2ZCET0DGEV8FQZV8MM3EJ",
		GalleryIDs: []string{
			"01K9Y2ZCET0DGEV8FQZV8MM3EJ",
			"01KEMKBGE856M5CHA29YC2E7RD",
			"01KER5ED3B9F4ZRGZ9X1MMTTMX",
			"01KER6SQYDZ15JD47ZCQSKYS31",
			"01K9Y2ZCET0DGEV8FQZV8MM3EJ", // 重复ID
		},
	}

	productBindings := func(p *Product) []Binding {
		return []Binding{
			Single(&p.CoverID, &p.CoverURL),
			Multi(&p.GalleryIDs, &p.GalleryURLs),
		}
	}

	err := FillOne(ctx, filler, product, productBindings)
	if err != nil {
		t.Fatalf("FillOne failed: %v", err)
	}

	// 验证封面
	if product.CoverURL != "https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ.jpg" {
		t.Errorf("CoverURL expected, got: %s", product.CoverURL)
	}

	// 验证画廊（包含重复ID）
	if len(product.GalleryURLs) != 5 {
		t.Fatalf("expected 5 gallery URLs, got: %d", len(product.GalleryURLs))
	}

	expectedURLs := []string{
		"https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ.jpg",
		"https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD.jpg",
		"https://cdn.example.com/01KER5ED3B9F4ZRGZ9X1MMTTMX.jpg",
		"https://cdn.example.com/01KER6SQYDZ15JD47ZCQSKYS31.jpg",
		"https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ.jpg", // 重复ID应该得到相同URL
	}

	for i, url := range product.GalleryURLs {
		if url != expectedURLs[i] {
			t.Errorf("GalleryURLs[%d] expected %s, got: %s", i, expectedURLs[i], url)
		}
	}

	t.Logf("封面URL: %s", product.CoverURL)
	t.Logf("画廊URLs: %v", product.GalleryURLs)
}

// TestRealFileIDsRich 测试真实文件ID的富文本场景
func TestRealFileIDsRich(t *testing.T) {
	// 模拟真实文件ID的数据
	realTestData := map[string]*ResourceInfo{
		"01K9Y2ZCET0DGEV8FQZV8MM3EJ": {
			URL:      "https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ_thumb.jpg"},
			Success:  true,
		},
		"01KEMKBGE856M5CHA29YC2E7RD": {
			URL:      "https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD_thumb.jpg"},
			Success:  true,
		},
		"01KER5ED3B9F4ZRGZ9X1MMTTMX": {
			URL:      "https://cdn.example.com/01KER5ED3B9F4ZRGZ9X1MMTTMX.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01KER5ED3B9F4ZRGZ9X1MMTTMX_thumb.jpg"},
			Success:  true,
		},
		"01KER6SQYDZ15JD47ZCQSKYS31": {
			URL:      "https://cdn.example.com/01KER6SQYDZ15JD47ZCQSKYS31.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01KER6SQYDZ15JD47ZCQSKYS31_thumb.jpg"},
			Success:  true,
		},
	}

	filler := NewFiller(newMockResolver(realTestData))
	ctx := context.Background()

	// 模拟富文本内容，包含多个图片占位符
	rawContent := `
<h1>产品介绍</h1>
<p>这是一款优质产品，以下是产品图片：</p>
<img data-href="01K9Y2ZCET0DGEV8FQZV8MM3EJ" src="old1.jpg" alt="主图">
<p>更多细节展示：</p>
<div class="gallery">
  <img data-href="01KEMKBGE856M5CHA29YC2E7RD" src="old2.jpg" alt="细节1">
  <img data-href="01KER5ED3B9F4ZRGZ9X1MMTTMX" src="old3.jpg" alt="细节2">
  <img data-href="01KER6SQYDZ15JD47ZCQSKYS31" src="old4.jpg" alt="细节3">
</div>
<p>再次展示主图：<img data-href="01K9Y2ZCET0DGEV8FQZV8MM3EJ" src="old5.jpg"></p>
`

	var renderedContent string

	err := filler.Fill(ctx, Rich(&rawContent, &renderedContent))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	// 验证所有占位符都被替换
	expectedContent := `
<h1>产品介绍</h1>
<p>这是一款优质产品，以下是产品图片：</p>
<img data-href="01K9Y2ZCET0DGEV8FQZV8MM3EJ" src="https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ.jpg" alt="主图">
<p>更多细节展示：</p>
<div class="gallery">
  <img data-href="01KEMKBGE856M5CHA29YC2E7RD" src="https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD.jpg" alt="细节1">
  <img data-href="01KER5ED3B9F4ZRGZ9X1MMTTMX" src="https://cdn.example.com/01KER5ED3B9F4ZRGZ9X1MMTTMX.jpg" alt="细节2">
  <img data-href="01KER6SQYDZ15JD47ZCQSKYS31" src="https://cdn.example.com/01KER6SQYDZ15JD47ZCQSKYS31.jpg" alt="细节3">
</div>
<p>再次展示主图：<img data-href="01K9Y2ZCET0DGEV8FQZV8MM3EJ" src="https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ.jpg"></p>
`

	if renderedContent != expectedContent {
		t.Errorf("渲染结果不匹配\n期望:\n%s\n实际:\n%s", expectedContent, renderedContent)
	}

	t.Logf("原始内容:\n%s", rawContent)
	t.Logf("渲染后内容:\n%s", renderedContent)
}

// TestRealFileIDsRichWithVariant 测试富文本使用缩略图变体
func TestRealFileIDsRichWithVariant(t *testing.T) {
	realTestData := map[string]*ResourceInfo{
		"01K9Y2ZCET0DGEV8FQZV8MM3EJ": {
			URL:      "https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ_thumb.jpg"},
			Success:  true,
		},
		"01KEMKBGE856M5CHA29YC2E7RD": {
			URL:      "https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD.jpg",
			Variants: map[string]string{"thumbnail": "https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD_thumb.jpg"},
			Success:  true,
		},
	}

	filler := NewFiller(newMockResolver(realTestData))
	ctx := context.Background()

	rawContent := `缩略图列表: <img data-href="01K9Y2ZCET0DGEV8FQZV8MM3EJ" src="old1.jpg"> 和 <img data-href="01KEMKBGE856M5CHA29YC2E7RD" src="old2.jpg">`
	var renderedContent string

	// 使用 thumbnail 变体
	err := filler.Fill(ctx, Rich(&rawContent, &renderedContent).UseVariant("thumbnail"))
	if err != nil {
		t.Fatalf("Fill failed: %v", err)
	}

	expectedContent := `缩略图列表: <img data-href="01K9Y2ZCET0DGEV8FQZV8MM3EJ" src="https://cdn.example.com/01K9Y2ZCET0DGEV8FQZV8MM3EJ_thumb.jpg"> 和 <img data-href="01KEMKBGE856M5CHA29YC2E7RD" src="https://cdn.example.com/01KEMKBGE856M5CHA29YC2E7RD_thumb.jpg">`

	if renderedContent != expectedContent {
		t.Errorf("渲染结果不匹配\n期望: %s\n实际: %s", expectedContent, renderedContent)
	}

	t.Logf("原始: %s", rawContent)
	t.Logf("渲染(thumbnail): %s", renderedContent)
}

type countingResolver struct {
	data      map[string]*ResourceInfo
	onResolve func(ids []string)
}

func (c *countingResolver) Resolve(ctx context.Context, ids []string) (map[string]*ResourceInfo, error) {
	if c.onResolve != nil {
		c.onResolve(ids)
	}
	result := make(map[string]*ResourceInfo)
	for _, id := range ids {
		if info, ok := c.data[id]; ok {
			result[id] = info
		}
	}
	return result, nil
}
