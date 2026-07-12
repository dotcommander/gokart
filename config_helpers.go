package gokart

// GetString returns the string value for key or defaultValue when the key is
// absent or has another type.
func GetString(config map[string]any, key, defaultValue string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return defaultValue
}

// GetInt returns the numeric value for key as an int. Values decoded from JSON
// maps commonly arrive as float64, so those and int64 values are accepted.
func GetInt(config map[string]any, key string, defaultValue int) int {
	switch value := config[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return defaultValue
	}
}

// GetFloat returns the numeric value for key as a float32.
func GetFloat(config map[string]any, key string, defaultValue float32) float32 {
	switch value := config[key].(type) {
	case float32:
		return value
	case float64:
		return float32(value)
	case int:
		return float32(value)
	default:
		return defaultValue
	}
}

// GetBool returns the bool value for key or defaultValue when the key is absent
// or has another type.
func GetBool(config map[string]any, key string, defaultValue bool) bool {
	if value, ok := config[key].(bool); ok {
		return value
	}
	return defaultValue
}
