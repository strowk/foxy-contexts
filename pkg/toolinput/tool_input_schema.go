package toolinput

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/strowk/foxy-contexts/pkg/mcp"
)

var (
	ErrTypeMissingRequiredProperty = errors.New("missing required property")

	ErrMissingRequestedProperty = errors.New("missing requested property")
	ErrCouldNotParseProperty    = errors.New("could not parse property")

	ErrUnknownProperty     = errors.New("unknown property")
	ErrUnknownPropertyType = errors.New("unknown property type")
)

type ToolInput interface {
	Boolean(name string) (bool, error)
	BooleanOr(name string, defaultValue bool) bool
	String(name string) (string, error)
	StringOr(name string, defaultValue string) string
	Number(name string) (float64, error)
	NumberOr(name string, defaultValue float64) float64
	Object(name string) (map[string]interface{}, error)
	ObjectOr(name string, defaultValue map[string]interface{}) map[string]interface{}
	Array(name string) ([]interface{}, error)
	ArrayOr(name string, defaultValue []interface{}) []interface{}
}

type toolInput struct {
	args map[string]interface{}
}

func newToolInput(args map[string]interface{}) *toolInput {
	return &toolInput{
		args: args,
	}
}

func (t *toolInput) Boolean(name string) (bool, error) {
	if v, ok := t.args[name]; ok {
		if b, ok := v.(bool); ok {
			return b, nil
		}
		if s, ok := v.(string); ok {
			if s == "" {
				return false, nil
			}
			srtParsedBool, err := strconv.ParseBool(s)
			if err != nil {
				return false, fmt.Errorf("%w %s: %w", ErrCouldNotParseProperty, name, err)
			}
			return srtParsedBool, nil
		}

		return false, fmt.Errorf("%w %s: unknown type %T", ErrCouldNotParseProperty, name, v)
	}
	return false, ErrMissingRequestedProperty
}

func (t *toolInput) BooleanOr(name string, defaultValue bool) bool {
	resolvedValue, err := t.Boolean(name)
	if err == nil {
		return resolvedValue
	}
	return defaultValue
}

func (t *toolInput) String(name string) (string, error) {
	if v, ok := t.args[name]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
		return "", fmt.Errorf("%w %s: unknown type %T", ErrCouldNotParseProperty, name, v)
	}
	return "", ErrMissingRequestedProperty
}

func (t *toolInput) StringOr(name string, defaultValue string) string {
	resolvedValue, err := t.String(name)
	if err == nil {
		return resolvedValue
	}
	return defaultValue
}

func (t *toolInput) Number(name string) (float64, error) {
	if v, ok := t.args[name]; ok {
		if n, ok := v.(float64); ok {
			return n, nil
		}
		if s, ok := v.(string); ok {
			strParsedFloat, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return 0, fmt.Errorf("%w %s: %w", ErrCouldNotParseProperty, name, err)
			}
			return strParsedFloat, nil
		}
		return 0, fmt.Errorf("%w %s: unknown type %T", ErrCouldNotParseProperty, name, v)
	}
	return 0, ErrMissingRequestedProperty
}

func (t *toolInput) NumberOr(name string, defaultValue float64) float64 {
	resolvedValue, err := t.Number(name)
	if err == nil {
		return resolvedValue
	}
	return defaultValue
}

func (t *toolInput) Object(name string) (map[string]interface{}, error) {
	if v, ok := t.args[name]; ok {
		if o, ok := v.(map[string]interface{}); ok {
			return o, nil
		}
		return nil, fmt.Errorf("%w %s: unknown type %T", ErrCouldNotParseProperty, name, v)
	}
	return nil, ErrMissingRequestedProperty
}

func (t *toolInput) ObjectOr(name string, defaultValue map[string]interface{}) map[string]interface{} {
	resolvedValue, err := t.Object(name)
	if err == nil {
		return resolvedValue
	}
	return defaultValue
}

func (t *toolInput) Array(name string) ([]interface{}, error) {
	if v, ok := t.args[name]; ok {
		if a, ok := v.([]interface{}); ok {
			return a, nil
		}
		return nil, fmt.Errorf("%w %s: unknown type %T", ErrCouldNotParseProperty, name, v)
	}
	return nil, ErrMissingRequestedProperty
}

func (t *toolInput) ArrayOr(name string, defaultValue []interface{}) []interface{} {
	resolvedValue, err := t.Array(name)
	if err == nil {
		return resolvedValue
	}
	return defaultValue
}

type ToolInputSchema interface {
	GetMcpToolInputSchema() mcp.ToolInputSchema
	Validate(args map[string]interface{}) (ToolInput, error)
}

type toolInputSchema struct {
	properties map[string]map[string]interface{}
	required   []string
}

func (t *toolInputSchema) Validate(args map[string]interface{}) (ToolInput, error) {
	for _, r := range t.required {
		if _, ok := args[r]; !ok {
			return nil, ErrTypeMissingRequiredProperty
		}
	}

	input := newToolInput(args)

	for argumentKey := range args {
		if propertySchema, ok := t.properties[argumentKey]; ok {
			switch propertySchema["type"] {
			case "string":
				_, err := input.String(argumentKey)
				if err != nil {
					return nil, err
				}
			case "boolean":
				_, err := input.Boolean(argumentKey)
				if err != nil {
					return nil, err
				}
			case "number":
				_, err := input.Number(argumentKey)
				if err != nil {
					return nil, err
				}

			case "object":
				_, err := input.Object(argumentKey)
				if err != nil {
					return nil, err
				}
			case "array":
				_, err := input.Array(argumentKey)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("%w %s: %T", ErrUnknownPropertyType, argumentKey, propertySchema["type"])
			}
		} else {
			return nil, fmt.Errorf("%w %s", ErrUnknownProperty, argumentKey)
		}
	}

	return input, nil
}

type ToolInputSchemaOption func(*toolInputSchema) error

func (t *toolInputSchema) GetMcpToolInputSchema() mcp.ToolInputSchema {
	return mcp.ToolInputSchema{
		Type:       "object",
		Properties: t.properties,
		Required:   t.required,
	}
}

func NewToolInputSchema(opts ...ToolInputSchemaOption) ToolInputSchema {
	t := &toolInputSchema{
		properties: make(map[string]map[string]interface{}),
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

func WithString(name string, description string) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		tis.properties[name] = map[string]interface{}{
			"type":        "string",
			"description": description,
		}
		return nil
	}
}

func WithRequiredString(name string, description string) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		WithString(name, description)(tis)
		tis.required = append(tis.required, name)
		return nil
	}
}

func WithBoolean(name string, description string) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		tis.properties[name] = map[string]interface{}{
			"type":        "boolean",
			"description": description,
		}

		return nil
	}
}

func WithRequiredBoolean(name string, description string) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		WithBoolean(name, description)(tis)
		tis.required = append(tis.required, name)
		return nil
	}
}

func WithNumber(name string, description string) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		tis.properties[name] = map[string]interface{}{
			"type":        "number",
			"description": description,
		}
		return nil
	}
}

func WithRequiredNumber(name string, description string) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		WithNumber(name, description)(tis)
		tis.required = append(tis.required, name)
		return nil
	}
}

func WithObject(name string, description string, properties map[string]map[string]interface{}) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		tis.properties[name] = map[string]interface{}{
			"type":        "object",
			"description": description,
			"properties":  properties,
		}
		return nil
	}
}

func WithRequiredObject(name string, description string, properties map[string]map[string]interface{}) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		WithObject(name, description, properties)(tis)
		tis.required = append(tis.required, name)
		return nil
	}
}

func WithArray(name string, description string, items map[string]interface{}) ToolInputSchemaOption {
	return func(tis *toolInputSchema) error {
		tis.properties[name] = map[string]interface{}{
			"type":        "array",
			"description": description,
			"items":       items,
		}
		return nil
	}
}
