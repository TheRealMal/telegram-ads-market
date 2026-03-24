package updates

import (
	"ads-mrkt/internal/helpers/telegram"
	marketentity "ads-mrkt/internal/market/domain/entity"
)

// mapBotAdminRights converts a Bot API ChatMember into BotAdminRights.
// Creators implicitly have all rights.
func mapBotAdminRights(cm *telegram.ChatMember) marketentity.BotAdminRights {
	if cm.Status == "creator" {
		return marketentity.BotAdminRights{
			CanPostMessages:   true,
			CanEditMessages:   true,
			CanDeleteMessages: true,
			CanPostStories:    true,
			CanEditStories:    true,
			CanDeleteStories:  true,
			CanPromoteMembers: true,
		}
	}
	return marketentity.BotAdminRights{
		CanPostMessages:   cm.CanPostMessages,
		CanEditMessages:   cm.CanEditMessages,
		CanDeleteMessages: cm.CanDeleteMessages,
		CanPostStories:    cm.CanPostStories,
		CanEditStories:    cm.CanEditStories,
		CanDeleteStories:  cm.CanDeleteStories,
		CanPromoteMembers: cm.CanPromoteMembers,
	}
}
