package mongoschema

import "go.mongodb.org/mongo-driver/v2/bson"

// ConvertBsonToMap recursively converts bson.M to map[string]any.
func ConvertBsonToMap(v any) any {
	switch val := v.(type) {
	case bson.M:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = ConvertBsonToMap(v)
		}
		return result
	case bson.A:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = ConvertBsonToMap(v)
		}
		return result
	default:
		return v
	}
}
