package entity

type AdminRights struct {
	DeleteMessages bool `json:"delete_messages"`
	EditMessages   bool `json:"edit_messages"`
	PostMessages   bool `json:"post_messages"`
	DeleteStories  bool `json:"delete_stories"`
	PostStories    bool `json:"post_stories"`
	CanViewStats   bool `json:"can_view_stats"`
}

const (
	BotMemberStatusAdministrator string = "administrator"
	BotMemberStatusCreator       string = "creator"
	BotMemberStatusMember        string = "member"
	BotMemberStatusLeft          string = "left"
	BotMemberStatusKicked        string = "kicked"
)

type Channel struct {
	AdminRights     AdminRights `json:"-"`
	ID              int64       `json:"id"`
	AccessHash      int64       `json:"-"`
	BotMemberStatus string      `json:"-"`
	Title           string      `json:"title"`
	Username        string      `json:"username"`
	Photo           string      `json:"photo"`
}
