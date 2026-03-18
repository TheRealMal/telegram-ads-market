package entity

import (
	"strconv"
)

type Event interface {
	ToMap() map[string]interface{}
	FromMap(m map[string]interface{})
	StreamKey() string
}

func mustParseInt64(v interface{}) int64 {
	vStr := v.(string)
	vInt, err := strconv.ParseInt(vStr, 10, 64)
	if err != nil {
		return -1
	}
	return vInt
}
