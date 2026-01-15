package media

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

// dataHelfRegex 匹配任意标签的 data-helf="xxx" 属性中的文件ID
// 支持 <img>, <video>, <audio> 等任意标签
var dataHelfRegex = regexp.MustCompile(`data-helf=["']([^"']+)["']`)

// ==================== 类型缓存 ====================

// typeInfo 缓存的类型信息
type typeInfo struct {
	fields []fieldInfo
}

// fieldInfo 字段信息
type fieldInfo struct {
	srcIndex   int    // 源字段索引（用于普通字段映射）
	dstIndex   int    // 目标字段索引
	name       string // 字段名
	fieldType  fieldType
	idSrcIndex int // ID来源字段索引（用于URL/URLs类型，从对应的ID字段获取值）
	// 嵌套类型信息（slice/struct/map）
	elemInfo *typeInfo
	srcElem  reflect.Type
	dstElem  reflect.Type
	keyType  reflect.Type // map的key类型
}

// fieldType 字段类型
type fieldType int

const (
	fieldTypeBasic    fieldType = iota // 基本类型，直接复制
	fieldTypeURL                       // URL 类型（双字段模式）
	fieldTypeURLs                      // URLs 类型（双字段模式）
	fieldTypeRichText                  // RichText 类型
	fieldTypeSlice                     // 切片类型，需要递归
	fieldTypeStruct                    // 结构体类型，需要递归
	fieldTypeMap                       // Map类型，需要递归（如多语言 map[string]*Lang）
)

// typeCache 类型信息缓存
var typeCache sync.Map // map[typePair]*typeInfo

// typePair 类型对
type typePair struct {
	src reflect.Type
	dst reflect.Type
}

// ==================== AutoFill 入口 ====================

// AutoFill 自动映射并填充文件URL
//
// 将源切片自动映射到目标切片，并填充所有文件URL
//
// 支持的字段类型:
//   - URL: 单文件URL（双字段模式），CoverURL 从 Cover 获取ID
//   - URLs: 多文件URL（双字段模式），GalleryURL 从 Gallery 获取IDs
//   - RichText: 富文本，data-helf="file_id" → src="url"
//
// 参数:
//   - ctx: 上下文
//   - filler: 填充器
//   - src: 源数据切片（如 []*ent.Product）
//   - dst: 目标切片指针（如 *[]*ProductResponse）
//
// 示例:
//
//	var responses []*ProductResponse
//	media.AutoFill(ctx, filler, products, &responses)
func AutoFill[S, D any](ctx context.Context, filler *Filler, src []S, dst *[]D) error {
	if len(src) == 0 || dst == nil {
		return nil
	}

	// 1. 创建目标切片
	result := make([]D, len(src))

	// 2. 获取类型信息
	srcType := reflect.TypeOf(src).Elem()
	dstType := reflect.TypeOf(result).Elem()
	info := getTypeInfo(srcType, dstType)

	// 3. 收集所有文件ID
	collector := &idCollector{ids: make(map[string]struct{})}

	// 4. 映射并收集ID
	// 如果目标是指针类型，需要先创建实例
	dstIsPtr := dstType.Kind() == reflect.Ptr
	for i := range src {
		srcVal := reflect.ValueOf(&src[i]).Elem()
		if dstIsPtr {
			// 创建新实例并设置到result
			newElem := reflect.New(dstType.Elem())
			reflect.ValueOf(&result[i]).Elem().Set(newElem)
			mapAndCollect(srcVal, newElem.Elem(), info, collector)
		} else {
			dstVal := reflect.ValueOf(&result[i]).Elem()
			mapAndCollect(srcVal, dstVal, info, collector)
		}
	}

	// 5. 批量获取URL
	if len(collector.ids) > 0 {
		ids := make([]string, 0, len(collector.ids))
		for id := range collector.ids {
			ids = append(ids, id)
		}

		resources, err := filler.resolver.Resolve(ctx, ids)
		if err != nil {
			return err
		}

		// 6. 填充URL
		for i := range result {
			dstVal := reflect.ValueOf(&result[i]).Elem()
			fillURLs(dstVal, info, resources)
		}
	}

	*dst = result
	return nil
}

// AutoFillOne 自动映射并填充单个对象
//
// 参数:
//   - ctx: 上下文
//   - filler: 填充器
//   - src: 源对象指针
//   - dst: 目标对象指针
//
// 示例:
//
//	var response ProductResponse
//	media.AutoFillOne(ctx, filler, product, &response)
func AutoFillOne[S, D any](ctx context.Context, filler *Filler, src *S, dst *D) error {
	if src == nil || dst == nil {
		return nil
	}

	srcSlice := []S{*src}
	var dstSlice []D

	if err := AutoFill(ctx, filler, srcSlice, &dstSlice); err != nil {
		return err
	}

	if len(dstSlice) > 0 {
		*dst = dstSlice[0]
	}
	return nil
}

// ==================== 内部实现 ====================

// idCollector ID收集器
type idCollector struct {
	ids map[string]struct{}
}

func (c *idCollector) add(id string) {
	if id != "" {
		c.ids[id] = struct{}{}
	}
}

func (c *idCollector) addAll(ids []string) {
	for _, id := range ids {
		c.add(id)
	}
}

// getTypeInfo 获取类型信息（带缓存）
func getTypeInfo(srcType, dstType reflect.Type) *typeInfo {
	// 解引用指针
	srcType = deref(srcType)
	dstType = deref(dstType)

	pair := typePair{src: srcType, dst: dstType}
	if cached, ok := typeCache.Load(pair); ok {
		return cached.(*typeInfo)
	}

	info := buildTypeInfo(srcType, dstType)
	typeCache.Store(pair, info)
	return info
}

// deref 解引用指针类型
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// buildTypeInfo 构建类型信息
func buildTypeInfo(srcType, dstType reflect.Type) *typeInfo {
	if srcType.Kind() != reflect.Struct || dstType.Kind() != reflect.Struct {
		return &typeInfo{}
	}

	// 构建源字段索引映射
	srcFields := make(map[string]int)
	for i := 0; i < srcType.NumField(); i++ {
		f := srcType.Field(i)
		if f.IsExported() {
			srcFields[f.Name] = i
		}
	}

	var fields []fieldInfo
	for i := 0; i < dstType.NumField(); i++ {
		dstField := dstType.Field(i)
		if !dstField.IsExported() {
			continue
		}

		dstFieldType := dstField.Type

		// 检查是否为 URL 类型（双字段模式）
		if dstFieldType == reflect.TypeOf(URL("")) {
			// 通过 tag 指定源字段名，如 `media:"Cover"`
			idFieldName := dstField.Tag.Get("media")
			if idFieldName == "" {
				// 兼容：如果没有 tag，尝试去掉 URL 后缀
				idFieldName = strings.TrimSuffix(dstField.Name, "URL")
			}
			if idSrcIdx, ok := srcFields[idFieldName]; ok {
				fields = append(fields, fieldInfo{
					srcIndex:   -1, // 不直接从同名字段复制
					dstIndex:   i,
					name:       dstField.Name,
					fieldType:  fieldTypeURL,
					idSrcIndex: idSrcIdx,
				})
			}
			continue
		}

		// 检查是否为 URLs 类型（双字段模式）
		if dstFieldType == reflect.TypeOf(URLs{}) {
			// 通过 tag 指定源字段名，如 `media:"Gallery"`
			idFieldName := dstField.Tag.Get("media")
			if idFieldName == "" {
				// 兼容：如果没有 tag，尝试去掉 URL 后缀
				idFieldName = strings.TrimSuffix(dstField.Name, "URL")
			}
			if idSrcIdx, ok := srcFields[idFieldName]; ok {
				fields = append(fields, fieldInfo{
					srcIndex:   -1,
					dstIndex:   i,
					name:       dstField.Name,
					fieldType:  fieldTypeURLs,
					idSrcIndex: idSrcIdx,
				})
			}
			continue
		}

		// 其他类型需要同名字段
		srcIdx, ok := srcFields[dstField.Name]
		if !ok {
			continue
		}

		srcField := srcType.Field(srcIdx)
		fi := fieldInfo{
			srcIndex: srcIdx,
			dstIndex: i,
			name:     dstField.Name,
		}

		// 判断字段类型
		switch {
		case dstFieldType == reflect.TypeOf(FileID("")):
			// FileID 类型直接复制（ID保持不变）
			fi.fieldType = fieldTypeBasic
		case dstFieldType == reflect.TypeOf(FileIDs{}):
			// FileIDs 类型直接复制（IDs保持不变）
			fi.fieldType = fieldTypeBasic
		case dstFieldType == reflect.TypeOf(RichText("")):
			fi.fieldType = fieldTypeRichText
		case dstFieldType.Kind() == reflect.Slice:
			fi.srcElem = srcField.Type.Elem()
			fi.dstElem = dstFieldType.Elem()
			// 基础类型切片（如 []string）直接复制
			if isBasicType(fi.dstElem) {
				fi.fieldType = fieldTypeBasic
			} else {
				fi.fieldType = fieldTypeSlice
				fi.elemInfo = getTypeInfo(fi.srcElem, fi.dstElem)
			}
		case dstFieldType.Kind() == reflect.Map:
			fi.fieldType = fieldTypeMap
			fi.keyType = dstFieldType.Key()
			fi.srcElem = srcField.Type.Elem()
			fi.dstElem = dstFieldType.Elem()
			fi.elemInfo = getTypeInfo(fi.srcElem, fi.dstElem)
		case deref(dstFieldType).Kind() == reflect.Struct && !isBasicType(dstFieldType):
			fi.fieldType = fieldTypeStruct
			fi.srcElem = srcField.Type
			fi.dstElem = dstFieldType
			fi.elemInfo = getTypeInfo(fi.srcElem, fi.dstElem)
		default:
			fi.fieldType = fieldTypeBasic
		}

		fields = append(fields, fi)
	}

	return &typeInfo{fields: fields}
}

// isBasicType 判断是否为基础类型（不需要递归）
func isBasicType(t reflect.Type) bool {
	t = deref(t)
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	}
	// time.Time 等也视为基础类型
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return true
	}
	return false
}

// mapAndCollect 映射字段并收集ID
func mapAndCollect(srcVal, dstVal reflect.Value, info *typeInfo, collector *idCollector) {
	// 解引用指针
	srcVal = derefValue(srcVal)
	dstVal = derefValue(dstVal)

	if !srcVal.IsValid() || !dstVal.IsValid() {
		return
	}

	for _, fi := range info.fields {
		dstField := dstVal.Field(fi.dstIndex)

		switch fi.fieldType {
		case fieldTypeBasic:
			srcField := srcVal.Field(fi.srcIndex)
			if srcField.Type().AssignableTo(dstField.Type()) {
				dstField.Set(srcField)
			} else if srcField.Type().ConvertibleTo(dstField.Type()) {
				dstField.Set(srcField.Convert(dstField.Type()))
			}

		case fieldTypeURL:
			// 从对应的ID字段获取值
			idField := srcVal.Field(fi.idSrcIndex)
			id := getStringValue(idField)
			// 先存储ID，后面fillURLs会替换成URL
			dstField.SetString(id)
			collector.add(id)

		case fieldTypeURLs:
			// 从对应的IDs字段获取值
			idsField := srcVal.Field(fi.idSrcIndex)
			ids := getStringSliceValue(idsField)
			if len(ids) > 0 {
				slice := reflect.MakeSlice(dstField.Type(), len(ids), len(ids))
				for i, id := range ids {
					slice.Index(i).SetString(id)
				}
				dstField.Set(slice)
				collector.addAll(ids)
			}

		case fieldTypeRichText:
			srcField := srcVal.Field(fi.srcIndex)
			// 复制值并提取ID
			text := getStringValue(srcField)
			dstField.SetString(text)
			matches := dataHelfRegex.FindAllStringSubmatch(text, -1)
			for _, m := range matches {
				if len(m) > 1 {
					collector.add(m[1])
				}
			}

		case fieldTypeSlice:
			srcField := srcVal.Field(fi.srcIndex)
			mapSliceAndCollect(srcField, dstField, fi, collector)

		case fieldTypeMap:
			srcField := srcVal.Field(fi.srcIndex)
			mapMapAndCollect(srcField, dstField, fi, collector)

		case fieldTypeStruct:
			srcField := srcVal.Field(fi.srcIndex)
			mapStructAndCollect(srcField, dstField, fi, collector)
		}
	}
}

// mapSliceAndCollect 映射切片并收集ID
func mapSliceAndCollect(srcField, dstField reflect.Value, fi fieldInfo, collector *idCollector) {
	srcField = derefValue(srcField)
	if !srcField.IsValid() || srcField.IsNil() || srcField.Len() == 0 {
		return
	}

	length := srcField.Len()
	slice := reflect.MakeSlice(dstField.Type(), length, length)

	for i := 0; i < length; i++ {
		srcElem := srcField.Index(i)
		dstElem := slice.Index(i)

		// 如果目标是指针类型，需要创建新实例
		if fi.dstElem.Kind() == reflect.Ptr {
			newElem := reflect.New(fi.dstElem.Elem())
			dstElem.Set(newElem)
			mapAndCollect(srcElem, newElem.Elem(), fi.elemInfo, collector)
		} else {
			mapAndCollect(srcElem, dstElem, fi.elemInfo, collector)
		}
	}

	dstField.Set(slice)
}

// mapStructAndCollect 映射结构体并收集ID
func mapStructAndCollect(srcField, dstField reflect.Value, fi fieldInfo, collector *idCollector) {
	srcField = derefValue(srcField)
	if !srcField.IsValid() {
		return
	}

	// 如果目标是指针类型，需要创建新实例
	if fi.dstElem.Kind() == reflect.Ptr {
		newElem := reflect.New(fi.dstElem.Elem())
		dstField.Set(newElem)
		mapAndCollect(srcField, newElem.Elem(), fi.elemInfo, collector)
	} else {
		mapAndCollect(srcField, dstField, fi.elemInfo, collector)
	}
}

// mapMapAndCollect 映射map并收集ID（如多语言 map[string]*Lang）
func mapMapAndCollect(srcField, dstField reflect.Value, fi fieldInfo, collector *idCollector) {
	srcField = derefValue(srcField)
	if !srcField.IsValid() || srcField.IsNil() || srcField.Len() == 0 {
		return
	}

	// 创建目标map
	dstMap := reflect.MakeMap(dstField.Type())

	// 检查源是否为 map[string]interface{} 类型
	srcElemKind := deref(fi.srcElem).Kind()
	isInterfaceSrc := srcElemKind == reflect.Interface

	for _, key := range srcField.MapKeys() {
		srcElem := srcField.MapIndex(key)

		// 如果目标value是指针类型，需要创建新实例
		if fi.dstElem.Kind() == reflect.Ptr {
			newElem := reflect.New(fi.dstElem.Elem())
			if isInterfaceSrc {
				// 源是 interface{} 类型，特殊处理
				mapInterfaceToStruct(srcElem, newElem.Elem(), collector)
			} else {
				mapAndCollect(srcElem, newElem.Elem(), fi.elemInfo, collector)
			}
			dstMap.SetMapIndex(key, newElem)
		} else {
			newElem := reflect.New(fi.dstElem).Elem()
			if isInterfaceSrc {
				mapInterfaceToStruct(srcElem, newElem, collector)
			} else {
				mapAndCollect(srcElem, newElem, fi.elemInfo, collector)
			}
			dstMap.SetMapIndex(key, newElem)
		}
	}

	dstField.Set(dstMap)
}

func mapInterfaceToStruct(srcVal, dstVal reflect.Value, collector *idCollector) {
	srcVal = derefValue(srcVal)
	dstVal = derefValue(dstVal)

	if !srcVal.IsValid() || !dstVal.IsValid() {
		return
	}

	if srcVal.Kind() != reflect.Map {
		return
	}

	dstType := dstVal.Type()
	if dstType.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < dstType.NumField(); i++ {
		dstField := dstType.Field(i)
		if !dstField.IsExported() {
			continue
		}

		jsonTag := dstField.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			jsonTag = dstField.Name
		} else if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		srcMapVal := srcVal.MapIndex(reflect.ValueOf(jsonTag))
		if !srcMapVal.IsValid() {
			continue
		}

		actualVal := derefValue(srcMapVal)
		if !actualVal.IsValid() {
			continue
		}

		dstFieldVal := dstVal.Field(i)
		dstFieldType := dstField.Type

		switch {
		case dstFieldType.Kind() == reflect.String:
			if actualVal.Kind() == reflect.String {
				dstFieldVal.SetString(actualVal.String())
			}
		case dstFieldType == reflect.TypeOf(RichText("")):
			if actualVal.Kind() == reflect.String {
				text := actualVal.String()
				dstFieldVal.SetString(text)
				matches := dataHelfRegex.FindAllStringSubmatch(text, -1)
				for _, m := range matches {
					if len(m) > 1 {
						collector.add(m[1])
					}
				}
			}
		case dstFieldType == reflect.TypeOf(FileID("")):
			if actualVal.Kind() == reflect.String {
				dstFieldVal.SetString(actualVal.String())
				collector.add(actualVal.String())
			}
		case dstFieldType.Kind() == reflect.Int, dstFieldType.Kind() == reflect.Int64:
			switch actualVal.Kind() {
			case reflect.Float64:
				dstFieldVal.SetInt(int64(actualVal.Float()))
			case reflect.Int, reflect.Int64:
				dstFieldVal.SetInt(actualVal.Int())
			}
		case dstFieldType.Kind() == reflect.Float64:
			if actualVal.Kind() == reflect.Float64 {
				dstFieldVal.SetFloat(actualVal.Float())
			}
		case dstFieldType.Kind() == reflect.Bool:
			if actualVal.Kind() == reflect.Bool {
				dstFieldVal.SetBool(actualVal.Bool())
			}
		}
	}
}

// fillURLs 填充URL
func fillURLs(dstVal reflect.Value, info *typeInfo, resources map[string]*ResourceInfo) {
	dstVal = derefValue(dstVal)
	if !dstVal.IsValid() {
		return
	}

	for _, fi := range info.fields {
		dstField := dstVal.Field(fi.dstIndex)

		switch fi.fieldType {
		case fieldTypeURL:
			id := dstField.String()
			if res, ok := resources[id]; ok && res.Success {
				dstField.SetString(res.URL)
			}

		case fieldTypeURLs:
			if dstField.Len() > 0 {
				for i := 0; i < dstField.Len(); i++ {
					id := dstField.Index(i).String()
					if res, ok := resources[id]; ok && res.Success {
						dstField.Index(i).SetString(res.URL)
					}
				}
			}

		case fieldTypeRichText:
			text := dstField.String()
			newText := dataHelfRegex.ReplaceAllStringFunc(text, func(match string) string {
				m := dataHelfRegex.FindStringSubmatch(match)
				if len(m) > 1 {
					if res, ok := resources[m[1]]; ok && res.Success {
						// 将 data-helf="file_id" 替换为 src="url"
						return `src="` + res.URL + `"`
					}
				}
				return match
			})
			dstField.SetString(newText)

		case fieldTypeSlice:
			fillSliceURLs(dstField, fi, resources)

		case fieldTypeMap:
			fillMapURLs(dstField, fi, resources)

		case fieldTypeStruct:
			fillStructURLs(dstField, fi, resources)
		}
	}
}

// fillSliceURLs 填充切片中的URL
func fillSliceURLs(dstField reflect.Value, fi fieldInfo, resources map[string]*ResourceInfo) {
	dstField = derefValue(dstField)
	if !dstField.IsValid() || dstField.IsNil() {
		return
	}

	for i := 0; i < dstField.Len(); i++ {
		elem := dstField.Index(i)
		fillURLs(elem, fi.elemInfo, resources)
	}
}

// fillStructURLs 填充结构体中的URL
func fillStructURLs(dstField reflect.Value, fi fieldInfo, resources map[string]*ResourceInfo) {
	dstField = derefValue(dstField)
	if !dstField.IsValid() {
		return
	}
	fillURLs(dstField, fi.elemInfo, resources)
}

// fillMapURLs 填充map中的URL
func fillMapURLs(dstField reflect.Value, fi fieldInfo, resources map[string]*ResourceInfo) {
	dstField = derefValue(dstField)
	if !dstField.IsValid() || dstField.IsNil() {
		return
	}

	// 检查源是否为 interface{} 类型
	srcElemKind := deref(fi.srcElem).Kind()
	isInterfaceSrc := srcElemKind == reflect.Interface

	for _, key := range dstField.MapKeys() {
		elem := dstField.MapIndex(key)
		if elem.Kind() == reflect.Ptr && !elem.IsNil() {
			if isInterfaceSrc {
				fillInterfaceStructURLs(elem.Elem(), resources)
			} else {
				fillURLs(elem.Elem(), fi.elemInfo, resources)
			}
		}
	}
}

// fillInterfaceStructURLs 填充从 interface{} 转换来的结构体中的URL
func fillInterfaceStructURLs(dstVal reflect.Value, resources map[string]*ResourceInfo) {
	dstVal = derefValue(dstVal)
	if !dstVal.IsValid() || dstVal.Kind() != reflect.Struct {
		return
	}

	dstType := dstVal.Type()
	for i := 0; i < dstType.NumField(); i++ {
		dstField := dstType.Field(i)
		if !dstField.IsExported() {
			continue
		}

		fieldVal := dstVal.Field(i)
		fieldType := dstField.Type

		switch {
		case fieldType == reflect.TypeOf(RichText("")):
			text := fieldVal.String()
			newText := dataHelfRegex.ReplaceAllStringFunc(text, func(match string) string {
				m := dataHelfRegex.FindStringSubmatch(match)
				if len(m) > 1 {
					if res, ok := resources[m[1]]; ok && res.Success {
						return `src="` + res.URL + `"`
					}
				}
				return match
			})
			fieldVal.SetString(newText)
		}
	}
}

// derefValue 解引用Value
func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// getStringValue 获取字符串值
func getStringValue(v reflect.Value) string {
	v = derefValue(v)
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.String {
		return v.String()
	}
	return ""
}

// getStringSliceValue 获取字符串切片值
func getStringSliceValue(v reflect.Value) []string {
	v = derefValue(v)
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return nil
	}

	result := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = getStringValue(v.Index(i))
	}
	return result
}
