package gokart

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

// ParseConfig converts a config map to a typed struct. The config tag names
// fields, default supplies missing scalar values, and required marks fields
// that must be present after defaults are applied. Anonymous structs are
// flattened.
func ParseConfig[T any](config map[string]any) (T, error) {
	var result T
	resultType := reflect.TypeOf(result)
	if resultType == nil || resultType.Kind() != reflect.Struct {
		return result, fmt.Errorf("ParseConfig: T must be a struct, got %T", result)
	}

	input := make(map[string]any, len(config))
	for key, value := range config {
		input[key] = value
	}
	if err := applyConfigDefaults(resultType, input, config); err != nil {
		return result, err
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &result,
		TagName:          "config",
		Squash:           true,
		WeaklyTypedInput: false,
		DecodeHook:       configConversionHook,
		MatchName: func(mapKey, fieldName string) bool {
			return mapKey == fieldName || mapKey == strings.ToLower(fieldName)
		},
	})
	if err != nil {
		return result, fmt.Errorf("create config decoder: %w", err)
	}
	if err := decoder.Decode(input); err != nil {
		return result, fmt.Errorf("cannot convert config: %w", err)
	}
	return result, nil
}

func applyConfigDefaults(resultType reflect.Type, input, original map[string]any) error {
	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := applyConfigDefaults(field.Type, input, original); err != nil {
				return err
			}
			continue
		}

		key := configFieldKey(field)
		if key == "-" {
			continue
		}
		value, exists := original[key]
		if (!exists || value == nil) && field.Tag.Get("default") != "" {
			parsed, err := parseConfigDefault(field.Type, field.Tag.Get("default"))
			if err != nil {
				return fmt.Errorf("field %s: invalid default %q: %w", field.Name, field.Tag.Get("default"), err)
			}
			input[key] = parsed
			exists = true
		}
		if field.Tag.Get("required") == "true" && !exists {
			return fmt.Errorf("field %s (config key %q): required but not provided", field.Name, key)
		}
	}
	return nil
}

func configFieldKey(field reflect.StructField) string {
	key := field.Tag.Get("config")
	if comma := strings.IndexByte(key, ','); comma >= 0 {
		key = key[:comma]
	}
	if key == "" {
		key = strings.ToLower(field.Name)
	}
	return key
}

func parseConfigDefault(fieldType reflect.Type, value string) (any, error) {
	switch fieldType.Kind() {
	case reflect.String:
		return reflect.ValueOf(value).Convert(fieldType).Interface(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return reflect.ValueOf(parsed).Convert(fieldType).Interface(), nil
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return reflect.ValueOf(parsed).Convert(fieldType).Interface(), nil
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		return reflect.ValueOf(parsed).Convert(fieldType).Interface(), nil
	default:
		return nil, fmt.Errorf("unsupported field type %v for string default", fieldType.Kind())
	}
}

func configConversionHook(from, to reflect.Type, data any) (any, error) {
	if data == nil || from.AssignableTo(to) {
		return data, nil
	}
	if configNumericKind(from.Kind()) && configNumericKind(to.Kind()) && from.ConvertibleTo(to) {
		return reflect.ValueOf(data).Convert(to).Interface(), nil
	}
	if from.ConvertibleTo(to) && !forbiddenConfigConversion(from.Kind(), to.Kind()) {
		return reflect.ValueOf(data).Convert(to).Interface(), nil
	}
	return data, nil
}

func configNumericKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func forbiddenConfigConversion(from, to reflect.Kind) bool {
	if from == reflect.String && (configNumericKind(to) || to == reflect.Bool) {
		return true
	}
	if to == reflect.Bool && from != reflect.Bool || from == reflect.Bool && to != reflect.Bool {
		return true
	}
	return to == reflect.String && from != reflect.String
}

// MustParseConfig is ParseConfig for initialization paths where invalid
// configuration is a programmer error.
func MustParseConfig[T any](config map[string]any) T {
	result, err := ParseConfig[T](config)
	if err != nil {
		panic(fmt.Sprintf("MustParseConfig: %v", err))
	}
	return result
}
