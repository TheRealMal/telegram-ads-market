package entity

import "strconv"

type Event interface {
	ToMap() map[string]interface{}
	FromMap(m map[string]interface{})
	StreamKey() string
}

func stringFromMap(m map[string]interface{}, k string) string {
	if v, ok := m[k]; ok && v != nil {
		switch x := v.(type) {
		case string:
			return x
		case []byte:
			return string(x)
		}
	}
	return ""
}

func int64FromMap(m map[string]interface{}, k string) int64 {
	if v, ok := m[k]; ok && v != nil {
		switch x := v.(type) {
		case string:
			n, _ := strconv.ParseInt(x, 10, 64)
			return n
		case int64:
			return x
		case float64:
			return int64(x)
		}
	}
	return 0
}
