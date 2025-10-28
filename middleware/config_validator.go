package middleware

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	// Validate 验证配置
	Validate(config map[string]interface{}) error
}

// ConfigRule 配置规则
type ConfigRule struct {
	// Required 是否必填
	Required bool
	// Type 期望的类型
	Type string
	// Default 默认值
	Default interface{}
	// Pattern 正则表达式模式（用于字符串验证）
	Pattern string
	// Min 最小值（用于数字、字符串长度、数组长度）
	Min interface{}
	// Max 最大值（用于数字、字符串长度、数组长度）
	Max interface{}
	// Enum 枚举值列表
	Enum []interface{}
	// CustomValidator 自定义验证函数
	CustomValidator func(interface{}) error
}

// ConfigSchema 配置模式
type ConfigSchema struct {
	Rules map[string]ConfigRule
}

// NewConfigSchema 创建新的配置模式
func NewConfigSchema() *ConfigSchema {
	return &ConfigSchema{
		Rules: make(map[string]ConfigRule),
	}
}

// AddRule 添加配置规则
func (cs *ConfigSchema) AddRule(key string, rule ConfigRule) {
	cs.Rules[key] = rule
}

// Validate 验证配置
func (cs *ConfigSchema) Validate(config map[string]interface{}) error {
	// 检查必填字段
	for key, rule := range cs.Rules {
		value, exists := config[key]

		// 检查必填字段
		if rule.Required && !exists {
			if rule.Default != nil {
				// 使用默认值
				config[key] = rule.Default
			} else {
				return fmt.Errorf("required field '%s' is missing", key)
			}
		}

		// 如果字段不存在，跳过验证
		if !exists {
			continue
		}

		// 验证字段值
		if err := cs.validateField(key, value, rule); err != nil {
			return err
		}
	}

	return nil
}

// validateField 验证单个字段
func (cs *ConfigSchema) validateField(key string, value interface{}, rule ConfigRule) error {
	// 类型验证
	if rule.Type != "" {
		if err := cs.validateType(key, value, rule.Type); err != nil {
			return err
		}
	}

	// 枚举值验证
	if len(rule.Enum) > 0 {
		if err := cs.validateEnum(key, value, rule.Enum); err != nil {
			return err
		}
	}

	// 正则表达式验证
	if rule.Pattern != "" {
		if err := cs.validatePattern(key, value, rule.Pattern); err != nil {
			return err
		}
	}

	// 最小值验证
	if rule.Min != nil {
		if err := cs.validateMin(key, value, rule.Min); err != nil {
			return err
		}
	}

	// 最大值验证
	if rule.Max != nil {
		if err := cs.validateMax(key, value, rule.Max); err != nil {
			return err
		}
	}

	// 自定义验证
	if rule.CustomValidator != nil {
		if err := rule.CustomValidator(value); err != nil {
			return fmt.Errorf("field '%s' failed custom validation: %v", key, err)
		}
	}

	return nil
}

// validateType 验证类型
func (cs *ConfigSchema) validateType(key string, value interface{}, expectedType string) error {
	// 处理JSON数字类型，它们会被解析为float64
	if expectedType == "int" || expectedType == "float" {
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("field '%s' must be a number, got %T", key, value)
		}
		return nil
	}

	// 处理其他类型
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string, got %T", key, value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean, got %T", key, value)
		}
	case "array":
		if !isArray(value) {
			return fmt.Errorf("field '%s' must be an array, got %T", key, value)
		}
	case "object":
		if !isObject(value) {
			return fmt.Errorf("field '%s' must be an object, got %T", key, value)
		}
	default:
		return fmt.Errorf("unknown type '%s' for field '%s'", expectedType, key)
	}

	return nil
}

// validateEnum 验证枚举值
func (cs *ConfigSchema) validateEnum(key string, value interface{}, enum []interface{}) error {
	for _, e := range enum {
		if reflect.DeepEqual(value, e) {
			return nil
		}
	}

	return fmt.Errorf("field '%s' must be one of %v, got %v", key, enum, value)
}

// validatePattern 验证正则表达式
func (cs *ConfigSchema) validatePattern(key string, value interface{}, pattern string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("field '%s' must be a string for pattern validation", key)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern for field '%s': %v", key, err)
	}

	if !re.MatchString(str) {
		return fmt.Errorf("field '%s' does not match pattern '%s'", key, pattern)
	}

	return nil
}

// validateMin 验证最小值
func (cs *ConfigSchema) validateMin(key string, value interface{}, min interface{}) error {
	switch v := value.(type) {
	case float64:
		if minFloat, ok := min.(float64); ok {
			if v < minFloat {
				return fmt.Errorf("field '%s' must be at least %f, got %f", key, minFloat, v)
			}
		}
	case string:
		if minInt, ok := min.(int); ok {
			if len(v) < minInt {
				return fmt.Errorf("field '%s' length must be at least %d, got %d", key, minInt, len(v))
			}
		}
	default:
		if isArray(value) {
			if minInt, ok := min.(int); ok {
				if len(value.([]interface{})) < minInt {
					return fmt.Errorf("field '%s' length must be at least %d", key, minInt)
				}
			}
		}
	}

	return nil
}

// validateMax 验证最大值
func (cs *ConfigSchema) validateMax(key string, value interface{}, max interface{}) error {
	switch v := value.(type) {
	case float64:
		if maxFloat, ok := max.(float64); ok {
			if v > maxFloat {
				return fmt.Errorf("field '%s' must be at most %f, got %f", key, maxFloat, v)
			}
		}
	case string:
		if maxInt, ok := max.(int); ok {
			if len(v) > maxInt {
				return fmt.Errorf("field '%s' length must be at most %d, got %d", key, maxInt, len(v))
			}
		}
	default:
		if isArray(value) {
			if maxInt, ok := max.(int); ok {
				if len(value.([]interface{})) > maxInt {
					return fmt.Errorf("field '%s' length must be at most %d", key, maxInt)
				}
			}
		}
	}

	return nil
}

// isArray 检查是否是数组
func isArray(value interface{}) bool {
	_, ok := value.([]interface{})
	return ok
}

// isObject 检查是否是对象
func isObject(value interface{}) bool {
	_, ok := value.(map[string]interface{})
	return ok
}

// ValidatePluginConfig 验证插件配置
func ValidatePluginConfig(config map[string]interface{}, schema *ConfigSchema) error {
	if schema == nil {
		return nil // 没有模式则跳过验证
	}

	return schema.Validate(config)
}

// ParseJSONSchema 从JSON字符串解析配置模式
func ParseJSONSchema(jsonStr string) (*ConfigSchema, error) {
	var schemaData map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %v", err)
	}

	schema := NewConfigSchema()
	for key, ruleData := range schemaData {
		rule := ConfigRule{}

		// 解析Required
		if required, ok := ruleData["required"].(bool); ok {
			rule.Required = required
		}

		// 解析Type
		if ruleType, ok := ruleData["type"].(string); ok {
			rule.Type = ruleType
		}

		// 解析Default
		if defaultValue, ok := ruleData["default"]; ok {
			rule.Default = defaultValue
		}

		// 解析Pattern
		if pattern, ok := ruleData["pattern"].(string); ok {
			rule.Pattern = pattern
		}

		// 解析Min
		if min, ok := ruleData["min"]; ok {
			switch v := min.(type) {
			case float64:
				rule.Min = v
			case string:
				if intVal, err := strconv.Atoi(v); err == nil {
					rule.Min = intVal
				}
			}
		}

		// 解析Max
		if max, ok := ruleData["max"]; ok {
			switch v := max.(type) {
			case float64:
				rule.Max = v
			case string:
				if intVal, err := strconv.Atoi(v); err == nil {
					rule.Max = intVal
				}
			}
		}

		// 解析Enum
		if enum, ok := ruleData["enum"].([]interface{}); ok {
			rule.Enum = enum
		}

		schema.AddRule(key, rule)
	}

	return schema, nil
}

// GetPluginSchema 获取插件配置模式
func GetPluginSchema(pluginType string) *ConfigSchema {
	switch pluginType {
	case "cors":
		return getCORSSchema()
	case "logging":
		return getLoggingSchema()
	case "rate_limit":
		return getRateLimitSchema()
	default:
		return nil
	}
}

// getCORSSchema 获取CORS插件配置模式
func getCORSSchema() *ConfigSchema {
	schema := NewConfigSchema()

	schema.AddRule("allowed_origins", ConfigRule{
		Required: true,
		Type:     "array",
		Default:  []interface{}{"*"},
	})

	schema.AddRule("allowed_methods", ConfigRule{
		Required: true,
		Type:     "array",
		Default:  []interface{}{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	})

	schema.AddRule("allowed_headers", ConfigRule{
		Required: true,
		Type:     "array",
		Default:  []interface{}{"*"},
	})

	return schema
}

// getLoggingSchema 获取日志插件配置模式
func getLoggingSchema() *ConfigSchema {
	schema := NewConfigSchema()

	schema.AddRule("level", ConfigRule{
		Required: true,
		Type:     "string",
		Default:  "info",
		Enum:     []interface{}{"debug", "info", "warn", "error"},
	})

	return schema
}

// getRateLimitSchema 获取限流插件配置模式
func getRateLimitSchema() *ConfigSchema {
	schema := NewConfigSchema()

	schema.AddRule("requests_per_minute", ConfigRule{
		Required: true,
		Type:     "int",
		Default:  60.0,
		Min:      1.0,
	})

	schema.AddRule("burst_size", ConfigRule{
		Required: true,
		Type:     "int",
		Default:  10.0,
		Min:      1.0,
	})

	return schema
}
