package entity

import (
	"strconv"

	"github.com/google/uuid"
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

func mustParseUUID(v interface{}) uuid.UUID {
	vStr := v.(string)
	vUUID, err := uuid.Parse(vStr)
	if err != nil {
		return uuid.Nil
	}
	return vUUID
}
