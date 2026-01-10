package common

// 常用 Header
const (
	USERCODE   string = "X-User-Code"
	TENANTCODE string = "X-Tenant-Code"
	REGIONNAME string = "X-Region-Name"
)

// OpenAPI 认证相关的 context key
type openapiContextKey string

const (
	KeyAuthType    openapiContextKey = "auth_type"
	KeyAPIKeyID    openapiContextKey = "api_key_id"
	KeyProductCode openapiContextKey = "product_code"
)

// AuthType 认证类型
type AuthType string

const (
	AuthTypeToken   AuthType = "token"   // JWT Token 认证
	AuthTypeOpenAPI AuthType = "openapi" // OpenAPI 签名认证
)
