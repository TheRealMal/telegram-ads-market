package entity

// BotAdminRights represents the bot's admin rights in a channel (Bot API format).
type BotAdminRights struct {
	CanPostMessages   bool `json:"can_post_messages"`
	CanEditMessages   bool `json:"can_edit_messages"`
	CanDeleteMessages bool `json:"can_delete_messages"`
	CanPostStories    bool `json:"can_post_stories"`
	CanEditStories    bool `json:"can_edit_stories"`
	CanDeleteStories  bool `json:"can_delete_stories"`
	CanPromoteMembers bool `json:"can_promote_members"`
}

// UserbotAdminRights represents the userbot's admin rights in a channel (MTProto format).
type UserbotAdminRights struct {
	DeleteMessages bool `json:"delete_messages"`
	EditMessages   bool `json:"edit_messages"`
	PostMessages   bool `json:"post_messages"`
	DeleteStories  bool `json:"delete_stories"`
	PostStories    bool `json:"post_stories"`
	CanViewStats   bool `json:"can_view_stats"`
}

// AdminRights is the top-level structure stored in the admin_rights JSONB column.
type AdminRights struct {
	Bot     BotAdminRights     `json:"bot"`
	Userbot UserbotAdminRights `json:"userbot"`
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
	AdminRights    AdminRights    `json:"-"`
	BotAdminRights BotAdminRights `json:"bot_admin_rights"`
	ID             int64          `json:"id"`
	AccessHash     int64          `json:"-"`
	BotMemberStatus BotMemberStatus `json:"-"`
	Title          string         `json:"title"`
	Username       string         `json:"username"`
	Photo          string         `json:"photo"`
}

// CanPostMessages reports whether the bot is an administrator in this channel and has post_messages right.
func (ch *Channel) CanPostMessages() bool {
	return ch.BotMemberStatus == BotMemberStatusAdministrator && ch.AdminRights.Bot.CanPostMessages
}

// CanPostStories reports whether the bot is an administrator in this channel and has post_stories right.
func (ch *Channel) CanPostStories() bool {
	return ch.BotMemberStatus == BotMemberStatusAdministrator && ch.AdminRights.Bot.CanPostStories
}

// CanPromoteMembers reports whether the bot is an administrator in this channel and can promote other users.
func (ch *Channel) CanPromoteMembers() bool {
	return ch.BotMemberStatus == BotMemberStatusAdministrator && ch.AdminRights.Bot.CanPromoteMembers
}

// HasEnoughRights reports whether the userbot admin rights include the minimum required for posting ads.
func (ar *UserbotAdminRights) HasEnoughRights() bool {
	return ar.PostMessages
}
