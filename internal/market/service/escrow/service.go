package escrow

import (
	"ads-mrkt/internal/market/domain/entity"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type marketRepository interface {
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	ListDealsApprovedWithoutEscrow(ctx context.Context) ([]*entity.Deal, error)
	SetDealEscrowAddress(ctx context.Context, dealID int64, address string, privateKey string, releaseTime time.Time) error
}

type liteclient interface {
	Client() ton.APIClientWrapped
}

type service struct {
	marketRepository marketRepository
	liteclient       liteclient
}

func NewService(marketRepository marketRepository, liteclient liteclient) *service {
	return &service{
		marketRepository: marketRepository,
		liteclient:       liteclient,
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

	// Release time: now + deal duration (duration is in hours, e.g. 24 for 24hr)
	releaseTime := time.Now().Add(time.Duration(deal.Duration) * time.Hour)
	if err = s.marketRepository.SetDealEscrowAddress(
		ctx,
		dealID,
		wallet.Address().StringRaw(),
		strings.Join(seed, " "),
		releaseTime,
	); err != nil {
		return err
	}
	return nil
}
