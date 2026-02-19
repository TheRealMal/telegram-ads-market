package entity

import "strconv"

const streamKeyChannelUpdateStats = "events:channel_update_stats"

type EventChannelUpdateStats struct {
	ID        string `json:"-"`
	ChannelID int64  `json:"channel_id"`
}

var _ Event = (*EventChannelUpdateStats)(nil)

func (e *EventChannelUpdateStats) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"channel_id": strconv.FormatInt(e.ChannelID, 10),
	}
}

func (e *EventChannelUpdateStats) FromMap(m map[string]interface{}) {
	e.ChannelID = int64FromMap(m, "channel_id")
}

func (e *EventChannelUpdateStats) StreamKey() string {
	return streamKeyChannelUpdateStats
}
