package introspection

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// OpenAPISchema represents an OpenAPI 3.x schema object
type OpenAPISchema struct {
	Type                 string                       `json:"type,omitempty"`
	Format               string                       `json:"format,omitempty"`
	Title                string                       `json:"title,omitempty"`
	Description          string                       `json:"description,omitempty"`
	Default              interface{}                  `json:"default,omitempty"`
	Enum                 []interface{}                `json:"enum,omitempty"`
	ReadOnly             bool                         `json:"readOnly,omitempty"`
	WriteOnly            bool                         `json:"writeOnly,omitempty"`
	Deprecated           bool                         `json:"deprecated,omitempty"`
	Example              interface{}                  `json:"example,omitempty"`
	Properties           map[string]*OpenAPISchema    `json:"properties,omitempty"`
	Required             []string                     `json:"required,omitempty"`
	Items                *OpenAPISchema               `json:"items,omitempty"`
	AdditionalProperties interface{}                  `json:"additionalProperties,omitempty"`
	AllOf                []*OpenAPISchema             `json:"allOf,omitempty"`
	OneOf                []*OpenAPISchema             `json:"oneOf,omitempty"`
	AnyOf                []*OpenAPISchema             `json:"anyOf,omitempty"`
	Not                  *OpenAPISchema               `json:"not,omitempty"`
	MinLength            *int                         `json:"minLength,omitempty"`
	MaxLength            *int                         `json:"maxLength,omitempty"`
	Pattern              string                       `json:"pattern,omitempty"`
	Minimum              *float64                     `json:"minimum,omitempty"`
	Maximum              *float64                     `json:"maximum,omitempty"`
	ExclusiveMinimum     bool                         `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     bool                         `json:"exclusiveMaximum,omitempty"`
	MultipleOf           *float64                     `json:"multipleOf,omitempty"`
	MinItems             *int                         `json:"minItems,omitempty"`
	MaxItems             *int                         `json:"maxItems,omitempty"`
	UniqueItems          bool                         `json:"uniqueItems,omitempty"`
	MinProperties        *int                         `json:"minProperties,omitempty"`
	MaxProperties        *int                         `json:"maxProperties,omitempty"`
}

// GenerateSchemaFromType generates an OpenAPI schema from a Go type using reflection
func GenerateSchemaFromType(t reflect.Type) *OpenAPISchema {
	return generateSchemaFromTypeWithSeen(t, make(map[reflect.Type]bool))
}

// GenerateSchemaFromValue generates an OpenAPI schema from a Go value using reflection
func GenerateSchemaFromValue(v interface{}) *OpenAPISchema {
	if v == nil {
		return &OpenAPISchema{Type: "null"}
	}
	return GenerateSchemaFromType(reflect.TypeOf(v))
}

func generateSchemaFromTypeWithSeen(t reflect.Type, seen map[reflect.Type]bool) *OpenAPISchema {
	// Handle pointers
	if t.Kind() == reflect.Ptr {
		return generateSchemaFromTypeWithSeen(t.Elem(), seen)
	}

	// Check for cycles
	if seen[t] {
		return &OpenAPISchema{
			Type:        "object",
			Description: fmt.Sprintf("Circular reference to %s", t.Name()),
		}
	}

	switch t.Kind() {
	case reflect.Bool:
		return &OpenAPISchema{Type: "boolean"}
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema := &OpenAPISchema{Type: "integer"}
		if t.Kind() == reflect.Int64 || t.Kind() == reflect.Uint64 {
			schema.Format = "int64"
		} else if t.Kind() == reflect.Int32 || t.Kind() == reflect.Uint32 {
			schema.Format = "int32"
		}
		return schema
		
	case reflect.Float32:
		return &OpenAPISchema{Type: "number", Format: "float"}
		
	case reflect.Float64:
		return &OpenAPISchema{Type: "number", Format: "double"}
		
	case reflect.String:
		return &OpenAPISchema{Type: "string"}
		
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			// []byte is typically base64 encoded
			return &OpenAPISchema{Type: "string", Format: "byte"}
		}
		return &OpenAPISchema{
			Type:  "array",
			Items: generateSchemaFromTypeWithSeen(t.Elem(), seen),
		}
		
	case reflect.Map:
		return &OpenAPISchema{
			Type: "object",
			AdditionalProperties: generateSchemaFromTypeWithSeen(t.Elem(), seen),
		}
		
	case reflect.Struct:
		// Special handling for time.Time
		if t == reflect.TypeOf(time.Time{}) {
			return &OpenAPISchema{Type: "string", Format: "date-time"}
		}
		
		// Mark as seen to prevent cycles
		seen[t] = true
		
		schema := &OpenAPISchema{
			Type:       "object",
			Properties: make(map[string]*OpenAPISchema),
			Required:   []string{},
		}
		
		// If the struct has a name, use it as title
		if t.Name() != "" {
			schema.Title = t.Name()
		}
		
		// Process struct fields
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			
			// Skip unexported fields
			if !field.IsExported() {
				continue
			}
			
			// Get JSON tag
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue // Skip this field
			}
			
			fieldName := field.Name
			omitempty := false
			
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					fieldName = parts[0]
				}
				for _, part := range parts[1:] {
					if part == "omitempty" {
						omitempty = true
					}
				}
			}
			
			// Generate schema for field
			fieldSchema := generateSchemaFromTypeWithSeen(field.Type, seen)
			
			// Add validation from validate tag
			if validateTag := field.Tag.Get("validate"); validateTag != "" {
				applyValidationRules(fieldSchema, validateTag)
				if strings.Contains(validateTag, "required") && !omitempty {
					schema.Required = append(schema.Required, fieldName)
				}
			}
			
			// Add binding validation
			if bindingTag := field.Tag.Get("binding"); bindingTag != "" {
				if strings.Contains(bindingTag, "required") && !omitempty {
					schema.Required = append(schema.Required, fieldName)
				}
			}
			
			// Add description from doc tag if present
			if docTag := field.Tag.Get("doc"); docTag != "" {
				fieldSchema.Description = docTag
			}
			
			// Add example from example tag if present
			if exampleTag := field.Tag.Get("example"); exampleTag != "" {
				fieldSchema.Example = exampleTag
			}
			
			// Add enum from enum tag if present
			if enumTag := field.Tag.Get("enum"); enumTag != "" {
				enums := strings.Split(enumTag, ",")
				fieldSchema.Enum = make([]interface{}, len(enums))
				for i, e := range enums {
					fieldSchema.Enum[i] = strings.TrimSpace(e)
				}
			}
			
			schema.Properties[fieldName] = fieldSchema
		}
		
		return schema
		
	case reflect.Interface:
		// For interfaces, we can't determine the exact type
		return &OpenAPISchema{
			Type:        "object",
			Description: "Any type",
		}
		
	default:
		return &OpenAPISchema{
			Type:        "string",
			Description: fmt.Sprintf("Unknown type: %v", t.Kind()),
		}
	}
}

// applyValidationRules applies validation rules from validate tags to the schema
func applyValidationRules(schema *OpenAPISchema, validateTag string) {
	rules := strings.Split(validateTag, ",")
	
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		
		// Handle rules with parameters
		if strings.Contains(rule, "=") {
			parts := strings.SplitN(rule, "=", 2)
			ruleName := parts[0]
			ruleValue := parts[1]
			
			switch ruleName {
			case "min":
				if schema.Type == "string" {
					val := parseIntOrZero(ruleValue)
					schema.MinLength = &val
				} else if schema.Type == "integer" || schema.Type == "number" {
					val := parseFloatOrZero(ruleValue)
					schema.Minimum = &val
				} else if schema.Type == "array" {
					val := parseIntOrZero(ruleValue)
					schema.MinItems = &val
				}
				
			case "max":
				if schema.Type == "string" {
					val := parseIntOrZero(ruleValue)
					schema.MaxLength = &val
				} else if schema.Type == "integer" || schema.Type == "number" {
					val := parseFloatOrZero(ruleValue)
					schema.Maximum = &val
				} else if schema.Type == "array" {
					val := parseIntOrZero(ruleValue)
					schema.MaxItems = &val
				}
				
			case "len":
				val := parseIntOrZero(ruleValue)
				if schema.Type == "string" {
					schema.MinLength = &val
					schema.MaxLength = &val
				} else if schema.Type == "array" {
					schema.MinItems = &val
					schema.MaxItems = &val
				}
				
			case "gt":
				val := parseFloatOrZero(ruleValue)
				schema.Minimum = &val
				schema.ExclusiveMinimum = true
				
			case "gte":
				val := parseFloatOrZero(ruleValue)
				schema.Minimum = &val
				
			case "lt":
				val := parseFloatOrZero(ruleValue)
				schema.Maximum = &val
				schema.ExclusiveMaximum = true
				
			case "lte":
				val := parseFloatOrZero(ruleValue)
				schema.Maximum = &val
				
			case "oneof":
				values := strings.Fields(ruleValue)
				schema.Enum = make([]interface{}, len(values))
				for i, v := range values {
					schema.Enum[i] = v
				}
				
			case "pattern", "regexp":
				schema.Pattern = ruleValue
			}
			
		} else {
			// Handle rules without parameters
			switch rule {
			case "email":
				schema.Format = "email"
				
			case "url", "uri":
				schema.Format = "uri"
				
			case "uuid":
				schema.Format = "uuid"
				
			case "ipv4":
				schema.Format = "ipv4"
				
			case "ipv6":
				schema.Format = "ipv6"
				
			case "ip":
				schema.Format = "ip"
				
			case "hostname":
				schema.Format = "hostname"
				
			case "datetime":
				schema.Format = "date-time"
				
			case "date":
				schema.Format = "date"
				
			case "time":
				schema.Format = "time"
				
			case "alpha":
				schema.Pattern = "^[a-zA-Z]+$"
				
			case "alphanum":
				schema.Pattern = "^[a-zA-Z0-9]+$"
				
			case "numeric":
				schema.Pattern = "^[0-9]+$"
				
			case "hexadecimal":
				schema.Pattern = "^[0-9a-fA-F]+$"
				
			case "base64":
				schema.Format = "byte"
			}
		}
	}
}

func parseIntOrZero(s string) int {
	var val int
	fmt.Sscanf(s, "%d", &val)
	return val
}

func parseFloatOrZero(s string) float64 {
	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}