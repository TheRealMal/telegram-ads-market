package model

type StateRow struct {
	Pts  int `db:"pts"`
	Qts  int `db:"qts"`
	Date int `db:"date"`
	Seq  int `db:"seq"`
}

type ChannelPtsRow struct {
	ChannelID int64 `db:"channel_id"`
	Pts       int   `db:"pts"`
}

type ChannelPtsOnlyRow struct {
	Pts int `db:"pts"`
}
