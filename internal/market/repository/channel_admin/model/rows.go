package model

type ChannelAdminExistsRow struct {
	One int `db:"one"`
}

type CountRow struct {
	Count int64 `db:"count"`
}
