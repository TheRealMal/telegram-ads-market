package model

import "ads-mrkt/internal/market/domain/entity"

type WaitlistResponse struct {
	ChannelsConnected []*entity.Channel `json:"channels_connected"`
	Referrals         []*entity.User    `json:"referrals"`
}
