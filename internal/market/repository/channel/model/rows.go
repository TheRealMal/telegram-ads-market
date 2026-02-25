package model

import (
	"encoding/json"

	"ads-mrkt/internal/market/domain/entity"
)

type ChannelRow struct {
	ID         int64           `db:"id"`
	AccessHash int64           `db:"access_hash"`
	AdminRights json.RawMessage `db:"admin_rights"`
	Title      string          `db:"title"`
	Username   string          `db:"username"`
	Photo      string          `db:"photo"`
}

type ChannelStatsRow struct {
	Stats json.RawMessage `db:"stats"`
}

func ChannelRowToEntity(row ChannelRow) (*entity.Channel, error) {
	var rights entity.AdminRights
	if len(row.AdminRights) > 0 {
		if err := json.Unmarshal(row.AdminRights, &rights); err != nil {
			return nil, err
		}
	}
	return &entity.Channel{
		ID:          row.ID,
		AccessHash:  row.AccessHash,
		AdminRights: rights,
		Title:       row.Title,
		Username:    row.Username,
		Photo:       row.Photo,
	}, nil
}
