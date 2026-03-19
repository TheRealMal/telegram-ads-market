package entity

type AdminRights struct {
	DeleteMessages bool `json:"delete_messages"`
	EditMessages   bool `json:"edit_messages"`
	PostMessages   bool `json:"post_messages"`
	DeleteStories  bool `json:"delete_stories"`
	PostStories    bool `json:"post_stories"`
	CanViewStats   bool `json:"can_view_stats"`
}

type BotMemberStatus string

const (
	BotMemberStatusAdministrator BotMemberStatus = "administrator"
	BotMemberStatusCreator       BotMemberStatus = "creator"
	BotMemberStatusMember        BotMemberStatus = "member"
	BotMemberStatusLeft          BotMemberStatus = "left"
	BotMemberStatusKicked        BotMemberStatus = "kicked"
)

type Channel struct {
	AdminRights     AdminRights     `json:"-"`
	ID              int64           `json:"id"`
	AccessHash      int64           `json:"-"`
	BotMemberStatus BotMemberStatus `json:"-"`
	Title           string          `json:"title"`
	Username        string          `json:"username"`
	Photo           string          `json:"photo"`
}

// CanPostMessages reports whether the bot is an administrator in this channel and can post messages.
func (ch *Channel) CanPostMessages() bool {
	return ch.BotMemberStatus == BotMemberStatusAdministrator
}

// HasEnoughRights reports whether the admin rights include the minimum required for posting ads.
func (ar *AdminRights) HasEnoughRights() bool {
	return ar.PostMessages
}
