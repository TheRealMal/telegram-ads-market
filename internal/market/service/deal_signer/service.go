package deal_signer

import (
	"context"
	"log/slog"
	"strings"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

type dealRepository interface {
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	SignDealInTx(ctx context.Context, dealID int64, userID int64, sig string) error
}

type userRepository interface {
	GetUserByID(ctx context.Context, id int64) (*entity.User, error)
}

type notificationAdder interface {
	AddTelegramNotificationEvent(ctx context.Context, event *evententity.EventTelegramNotification) error
}

type Service struct {
	dealRepo          dealRepository
	userRepo          userRepository
	notificationAdder notificationAdder
}

func NewService(dealRepo dealRepository, userRepo userRepository, notificationAdder notificationAdder) *Service {
	return &Service{
		dealRepo:          dealRepo,
		userRepo:          userRepo,
		notificationAdder: notificationAdder,
	}
}

// SignDeal sets the current user's signature on the deal. Both payout addresses must be set; user's wallet must match.
func (s *Service) SignDeal(ctx context.Context, userID int64, dealID int64) error {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return marketerrors.ErrNotFound
	}
	if user.WalletAddress == nil || *user.WalletAddress == "" {
		return marketerrors.ErrWalletNotSet
	}
	if existing.LessorPayoutAddress == nil || *existing.LessorPayoutAddress == "" ||
		existing.LesseePayoutAddress == nil || *existing.LesseePayoutAddress == "" {
		return marketerrors.ErrPayoutNotSet
	}
	if strings.TrimSpace(domain.GetMessageFromDetails(existing.Details)) == "" {
		return marketerrors.ErrDealDetailsMessageRequired
	}
	myPayout := *existing.LesseePayoutAddress
	if userID == existing.LessorID {
		myPayout = *existing.LessorPayoutAddress
	}
	if *user.WalletAddress != myPayout {
		return marketerrors.ErrWalletNotSet
	}
	lessorPayout := *existing.LessorPayoutAddress
	lesseePayout := *existing.LesseePayoutAddress
	sig := domain.ComputeDealSignature(existing.Type, existing.Duration, existing.Price, existing.Details, userID, lessorPayout, lesseePayout)
	if err := s.dealRepo.SignDealInTx(ctx, dealID, userID, sig); err != nil {
		return err
	}
	otherID := existing.LesseeID
	if userID == existing.LesseeID {
		otherID = existing.LessorID
	}
	if err := s.notificationAdder.AddTelegramNotificationEvent(
		ctx,
		&evententity.EventTelegramNotification{
			ChatID:  otherID,
			Message: "Deal was signed by the other party",
		},
	); err != nil {
		slog.Error("failed to add notification", "type", "deal_sign", "chat_id", otherID, "error", err)

	}
	return nil
}
