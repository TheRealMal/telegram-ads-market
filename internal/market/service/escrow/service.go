package escrow

import (
	"ads-mrkt/internal/market/domain/entity"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

const escrowRedisTTL = 1 * time.Hour

var ErrPayoutAddressNotSet = errors.New("payout address not set for deal")

type marketRepository interface {
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	GetDealByEscrowAddress(ctx context.Context, escrowAddress string) (*entity.Deal, error)
	ListDealsApprovedWithoutEscrow(ctx context.Context) ([]*entity.Deal, error)
	ListDealsWaitingEscrowRelease(ctx context.Context) ([]*entity.Deal, error)
	ListDealsWaitingEscrowRefund(ctx context.Context) ([]*entity.Deal, error)
	SetDealEscrowAddress(ctx context.Context, dealID int64, address string, privateKey string) error
	SetDealStatusEscrowDepositConfirmed(ctx context.Context, dealID int64) error
	SetDealStatusEscrowReleaseConfirmed(ctx context.Context, dealID int64) error
	SetDealStatusEscrowRefundConfirmed(ctx context.Context, dealID int64) error
	TakeDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (string, error)
	ReleaseDealActionLock(ctx context.Context, lockID string, status entity.DealActionLockStatus) error
	GetLastDealActionLock(ctx context.Context, dealID int64, actionType entity.DealActionType) (*entity.DealActionLock, error)
}

type liteclient interface {
	Client() ton.APIClientWrapped
	HasOutgoingTxTo(ctx context.Context, fromAddrRaw *address.Address, amountNanoton int64, toAddr *address.Address) (bool, error)
}

type redisCache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type dealChatService interface {
	DeleteDealForumTopic(ctx context.Context, dealID int64) error
}

type service struct {
	marketRepository marketRepository
	liteclient       liteclient
	redis            redisCache
	dealChatService  dealChatService

	transactionGasNanoton int64
	comissionMultiplier   float64
}

func NewService(marketRepository marketRepository, liteclient liteclient, redis redisCache, dealChatService dealChatService, transactionGasTON float64, commissionPercent float64) *service {
	return &service{
		marketRepository:      marketRepository,
		liteclient:            liteclient,
		redis:                 redis,
		dealChatService:       dealChatService,
		transactionGasNanoton: int64(transactionGasTON * nanotonPerTON),
		comissionMultiplier:   1 + (commissionPercent / 100.0),
	}
}

func (s *service) CreateEscrow(ctx context.Context, dealID int64) error {
	deal, err := s.marketRepository.GetDealByID(ctx, dealID)
	if err != nil {
		return err
	}
	if deal.Status != entity.DealStatusApproved {
		return errors.New("deal is not approved")
	}

	seed := wallet.NewSeed()
	wallet, err := wallet.FromSeedWithOptions(
		s.liteclient.Client(),
		seed,
		wallet.ConfigV5R1Final{
			NetworkGlobalID: wallet.MainnetGlobalID,
		},
	)
	if err != nil {
		return err
	}

	rawAddr := wallet.Address().StringRaw()
	if err = s.marketRepository.SetDealEscrowAddress(ctx, dealID, rawAddr, strings.Join(seed, " ")); err != nil {
		return err
	}
	if err = s.redis.Set(ctx, rawAddr, "1", escrowRedisTTL); err != nil {
		slog.Error("failed to set escrow wallet for observer", "address", rawAddr, "deal_id", dealID)
	}
	return nil
}

// ReleaseOrRefundEscrow performs escrow release (to lessor) or refund (to lessee) using a short-lived lock.
// Takes a 5-minute lock, sends deal.EscrowAmount (nanoton) to the payout address, then marks lock and deal status.
// Deal must have lessor_payout_address (release) or lessee_payout_address (refund) set.
func (s *service) ReleaseOrRefundEscrow(ctx context.Context, logger *slog.Logger, dealID int64, release bool) error {
	deal, err := s.marketRepository.GetDealByID(ctx, dealID)
	if err != nil {
		return err
	}
	if deal == nil {
		return errors.New("deal not found")
	}

	actionType, destAddr, err := prepareAction(deal, release)
	if err != nil {
		return err
	}

	toAddr, err := address.ParseRawAddr(destAddr)
	if err != nil {
		return fmt.Errorf("failed to parse payout address: %w", err)
	}

	key, err := wallet.SeedToPrivateKeyWithOptions(
		strings.Split(
			strings.TrimSpace(*deal.EscrowPrivateKey),
			" ",
		),
	)
	if err != nil {
		return fmt.Errorf("failed to parse escrow private key: %w", err)
	}
	w, err := wallet.FromPrivateKey(s.liteclient.Client(), key, wallet.ConfigV5R1Final{
		NetworkGlobalID: wallet.MainnetGlobalID,
	})
	if err != nil {
		return fmt.Errorf("failed to create wallet from private key: %w", err)
	}

	amountNanoton := s.GetAmountWithoutGasAndCommission(deal.EscrowAmount)
	amount := tlb.FromNanoTONU(uint64(amountNanoton))

	if deal.EscrowAddress == nil || *deal.EscrowAddress == "" {
		return ErrPayoutAddressNotSet
	}

	escrowAddr, err := address.ParseRawAddr(*deal.EscrowAddress)
	if err != nil {
		return fmt.Errorf("failed to parse escrow address: %w", err)
	}

	// If the last lock for this action is Locked and expired, previous run may have transferred then crashed: try to find outgoing tx by amount and recover.
	lastLock, lerr := s.marketRepository.GetLastDealActionLock(ctx, dealID, actionType)
	if lerr == nil && lastLock != nil && lastLock.Status == entity.DealActionLockStatusLocked && !lastLock.ExpireAt.After(time.Now()) {
		found, _ := s.liteclient.HasOutgoingTxTo(ctx, escrowAddr, amountNanoton, toAddr)
		if found {
			if release {
				if err = s.marketRepository.SetDealStatusEscrowReleaseConfirmed(ctx, dealID); err != nil {
					return err
				}
			} else {
				if err = s.marketRepository.SetDealStatusEscrowRefundConfirmed(ctx, dealID); err != nil {
					return err
				}
			}
			_ = s.marketRepository.ReleaseDealActionLock(ctx, lastLock.ID, entity.DealActionLockStatusCompleted)
			_ = s.dealChatService.DeleteDealForumTopic(ctx, dealID)
			logger.Info("escrow release/refund recovered from expired lock", "deal_id", dealID, "release", release)
			return nil
		}
		_ = s.marketRepository.ReleaseDealActionLock(ctx, lastLock.ID, entity.DealActionLockStatusFailed)
	}

	err = func() error {
		lockID, err := s.marketRepository.TakeDealActionLock(ctx, dealID, actionType)
		if err != nil {
			return err
		}
		dealACtionLockStatus := entity.DealActionLockStatusFailed
		defer func() {
			_ = s.marketRepository.ReleaseDealActionLock(ctx, lockID, dealACtionLockStatus)
		}()

		if err = w.Transfer(ctx, toAddr, amount, string(actionType)); err != nil {
			logger.Error("escrow transfer failed", "deal_id", dealID, "release", release, "error", err)
			return err
		}

		if release {
			if err = s.marketRepository.SetDealStatusEscrowReleaseConfirmed(ctx, dealID); err != nil {
				return err
			}
		} else {
			if err = s.marketRepository.SetDealStatusEscrowRefundConfirmed(ctx, dealID); err != nil {
				return err
			}
		}
		_ = s.dealChatService.DeleteDealForumTopic(ctx, dealID)
		dealACtionLockStatus = entity.DealActionLockStatusCompleted
		return nil
	}()
	if err != nil {
		return err
	}

	logger.Info("escrow release/refund completed", "deal_id", dealID, "release", release)
	return nil
}

func prepareAction(deal *entity.Deal, release bool) (actionType entity.DealActionType, destAddr string, err error) {
	var wantStatus entity.DealStatus
	if release {
		actionType = entity.DealActionTypeEscrowRelease
		wantStatus = entity.DealStatusWaitingEscrowRelease
		if deal.LessorPayoutAddress == nil || *deal.LessorPayoutAddress == "" {
			return "", "", ErrPayoutAddressNotSet
		}
		destAddr = *deal.LessorPayoutAddress
	} else {
		actionType = entity.DealActionTypeEscrowRefund
		wantStatus = entity.DealStatusWaitingEscrowRefund
		if deal.LesseePayoutAddress == nil || *deal.LesseePayoutAddress == "" {
			return "", "", ErrPayoutAddressNotSet
		}
		destAddr = *deal.LesseePayoutAddress
	}
	if deal.Status != wantStatus {
		return "", "", errors.New("deal status is not " + string(wantStatus))
	}
	if deal.EscrowPrivateKey == nil || *deal.EscrowPrivateKey == "" {
		return "", "", errors.New("deal has no escrow private key")
	}
	return
}
