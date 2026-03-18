package deal_post

import (
	"context"
	"time"

	telegram "ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/domain/entity"
)

type telegramClient interface {
	SendChannelMessage(ctx context.Context, chatID int64, text string, entities []telegram.MessageEntity) (int64, error)
	CheckMessageExists(ctx context.Context, chatID int64, messageID int64, probeChatID int64) (bool, error)
}

type channelRepository interface {
	GetChannelByID(ctx context.Context, id int64) (*entity.Channel, error)
}

type listingRepository interface {
	GetListingByID(ctx context.Context, id int64) (*entity.Listing, error)
}

type dealRepository interface {
	ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx context.Context) ([]*entity.Deal, error)
}

type dealPostMessageRepository interface {
	CreateDealPostMessageAndSetDealInProgress(ctx context.Context, m *entity.DealPostMessage) error
	UpdateDealPostMessageStatus(ctx context.Context, id int64, status entity.DealPostMessageStatus) error
	UpdateDealPostMessageStatusAndNextCheck(ctx context.Context, id int64, status entity.DealPostMessageStatus, nextCheck time.Time) error
	ListDealPostMessageExistsWithNextCheckBefore(ctx context.Context, before time.Time) ([]*entity.DealPostMessage, error)
}

type dealActionLockRepository interface {
	TakeDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (string, error)
	ReleaseDealActionLock(ctx context.Context, lockID string, status entity.DealActionLockStatus) error
	GetLastDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (*entity.DealActionLock, error)
}

// TODO: (@TheRealMal) Remove at all if userbot flow is choosen
type service struct {
	telegramClient      telegramClient
	channelRepo         channelRepository
	listingRepo         listingRepository
	dealRepo            dealRepository
	dealPostMessageRepo dealPostMessageRepository
	dealActionLockRepo  dealActionLockRepository
	serviceChatID       int64
}

func NewService(
	telegramClient telegramClient,
	channelRepo channelRepository,
	listingRepo listingRepository,
	dealRepo dealRepository,
	dealPostMessageRepo dealPostMessageRepository,
	dealActionLockRepo dealActionLockRepository,
	serviceChatID int64,
) *service {
	return &service{
		telegramClient:      telegramClient,
		channelRepo:         channelRepo,
		listingRepo:         listingRepo,
		dealRepo:            dealRepo,
		dealPostMessageRepo: dealPostMessageRepo,
		dealActionLockRepo:  dealActionLockRepo,
		serviceChatID:       serviceChatID,
	}
}
