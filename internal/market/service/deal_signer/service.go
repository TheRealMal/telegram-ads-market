package deal_signer

import (
	"context"
	"log/slog"
	"strings"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

type dealRepository interface {
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	SignDealInTx(ctx context.Context, dealID int64, userID int64, sig string) (*entity.Deal, error)
}

type userRepository interface {
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
}

type notificationAdder interface {
	AddTelegramNotificationEvent(ctx context.Context, event *evententity.EventTelegramNotification) error
}

type dealForumTopicRepository interface {
	GetDealForumTopicByDealID(ctx context.Context, dealID int64) (*entity.DealForumTopic, error)
}

type Service struct {
	dealRepo          dealRepository
	userRepo          userRepository
	notificationAdder notificationAdder
	forumTopicRepo    dealForumTopicRepository
}

func NewService(dealRepo dealRepository, userRepo userRepository, notificationAdder notificationAdder, forumTopicRepo dealForumTopicRepository) *Service {
	return &Service{
		dealRepo:          dealRepo,
		userRepo:          userRepo,
		notificationAdder: notificationAdder,
		forumTopicRepo:    forumTopicRepo,
	}
}

// SignDeal sets the current user's signature on the deal. Both payout addresses must be set; user's wallet must match.
// Returns the updated deal.
func (s *Service) SignDeal(ctx context.Context, userID int64, dealID int64) (*entity.Deal, error) {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || existing == nil {
		return nil, marketerrors.ErrNotFound
	}
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, marketerrors.ErrNotFound
	}
	if !user.HasWallet() {
		return nil, marketerrors.ErrWalletNotSet
	}
	if existing.LessorPayoutAddress == nil || *existing.LessorPayoutAddress == "" ||
		existing.LesseePayoutAddress == nil || *existing.LesseePayoutAddress == "" {
		return nil, marketerrors.ErrPayoutNotSet
	}
	if existing.IsStory() {
		mediaType, _ := entity.GetMediaFromDetails(existing.Details)
		if mediaType == "" {
			return nil, marketerrors.ErrDealDetailsMediaRequired
		}
	} else {
		if strings.TrimSpace(entity.GetMessageFromDetails(existing.Details)) == "" {
			return nil, marketerrors.ErrDealDetailsMessageRequired
		}
	}
	myPayout := *existing.LesseePayoutAddress
	if userID == existing.LessorID {
		myPayout = *existing.LessorPayoutAddress
	}
	if *user.WalletAddress != myPayout {
		return nil, marketerrors.ErrWalletNotSet
	}
	lessorPayout := *existing.LessorPayoutAddress
	lesseePayout := *existing.LesseePayoutAddress
	sig := entity.ComputeDealSignature(string(existing.Type), existing.Duration, existing.Price, existing.Details, userID, lessorPayout, lesseePayout)
	d, err := s.dealRepo.SignDealInTx(ctx, dealID, userID, sig)
	if err != nil {
		return nil, err
	}
	otherID := existing.LesseeID
	if userID == existing.LesseeID {
		otherID = existing.LessorID
	}

	notification := &evententity.EventTelegramNotification{
		ChatID:  otherID,
		Message: "Deal was signed by the other party",
	}

	// Try to route notification into the deal's forum topic
	if topic, err := s.forumTopicRepo.GetDealForumTopicByDealID(ctx, dealID); err == nil && topic != nil {
		if otherID == existing.LessorID {
			notification.ThreadID = topic.LessorMessageThreadID
		} else {
			notification.ThreadID = topic.LesseeMessageThreadID
		}
	}

	if err := s.notificationAdder.AddTelegramNotificationEvent(ctx, notification); err != nil {
		slog.Error("failed to add notification", "type", "deal_sign", "chat_id", otherID, "error", err)
	}
	return d, nil
}
