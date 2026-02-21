package blockchain_observer

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ads-mrkt/internal/event/domain/entity"

	"github.com/redis/go-redis/v9"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
)

type WalletAddress [32]byte

type lt interface {
	GetTransactionIDsFromBlock(ctx context.Context, blockID *ton.BlockIDExt) ([]ton.TransactionShortInfo, error)
	GetBlockTransactionsV2(ctx context.Context, block *ton.BlockIDExt, count uint32, after ...*ton.TransactionID3) ([]ton.TransactionShortInfo, bool, error)
	GetMasterchainInfo(ctx context.Context, timeout time.Duration) (*ton.BlockIDExt, error)
	GetBlockShardsInfo(ctx context.Context, master *ton.BlockIDExt) ([]*ton.BlockIDExt, error)
	GetTransaction(ctx context.Context, block *ton.BlockIDExt, addr *address.Address, lt uint64) (*tlb.Transaction, error)
	GetBlockData(ctx context.Context, block *ton.BlockIDExt) (*tlb.Block, error)
}

type dealRepository interface {
	SetDealStatusExpiredByEscrowAddress(ctx context.Context, escrowAddress string) error
}

type escrowDepositEventService interface {
	AddEscrowDepositEvent(ctx context.Context, event *entity.EventEscrowDeposit) error
}

type depositEvent struct {
	rawAddress string
	amount     int64
	timestamp  int64
	txHash     string
}

// Observer watches TON chain for transactions to escrow wallets and Redis key expiration for deal expiry.
type Observer struct {
	lt             lt
	rdb            *redis.Client
	dealRepository dealRepository
	eventService   escrowDepositEventService
	dbIndex        int
	addresses      map[WalletAddress]struct{}
	addressesMutex sync.RWMutex
	workchain      *virtualWorkchain
	masterBlocks   chan *ton.BlockIDExt
	shardBlocks    chan *ton.BlockIDExt
	depositEvents  chan *depositEvent
	log            *slog.Logger
}

// New builds an observer. rdb is the Redis client (for keys, keyspace subscription). dbIndex is Redis DB index for keyevent channels. escrowDepositAdder is used to push deposit events to the escrow_deposit stream.
func New(lt lt, rdb *redis.Client, dealRepository dealRepository, eventService escrowDepositEventService, dbIndex int) *Observer {
	return &Observer{
		lt:             lt,
		rdb:            rdb,
		dealRepository: dealRepository,
		eventService:   eventService,
		dbIndex:        dbIndex,
		addresses:      make(map[WalletAddress]struct{}),
		workchain:      &virtualWorkchain{ID: 0, Shards: make(map[int64]uint32)},
		masterBlocks:   make(chan *ton.BlockIDExt),
		shardBlocks:    make(chan *ton.BlockIDExt),
		depositEvents:  make(chan *depositEvent, 256),
		log:            slog.With("component", "blockchain_observer"),
	}
}

// Start runs the observer until ctx is cancelled. Requires Redis notify-keyspace-events Egx.
func (o *Observer) Start(ctx context.Context) error {
	o.log.Info("loading escrow wallets from Redis...")
	if err := o.loadAddresses(ctx); err != nil {
		return err
	}
	o.log.Info("loading done")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.startRedisEventsHandler(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.startDepositNotifier(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.startShardsHandler(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.startMastersHandler(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.startMasterObserver(ctx)
	}()

	wg.Wait()
	return nil
}

func (o *Observer) loadAddresses(ctx context.Context) error {
	keys, err := o.rdb.Keys(ctx, "*").Result()
	if err != nil {
		return fmt.Errorf("redis keys: %w", err)
	}
	o.addressesMutex.Lock()
	defer o.addressesMutex.Unlock()
	for _, k := range keys {
		addr, err := address.ParseRawAddr(k)
		if err != nil {
			o.log.Debug("skip non-address redis key", "key", k)
			continue
		}
		o.log.Info("wallet loaded", "address", addr.StringRaw())
		o.addresses[WalletAddress(addr.Data())] = struct{}{}
	}
	return nil
}

func (o *Observer) isAddressWatched(key WalletAddress) bool {
	o.addressesMutex.RLock()
	defer o.addressesMutex.RUnlock()
	_, ok := o.addresses[key]
	return ok
}

func (o *Observer) addAddress(key WalletAddress) {
	o.addressesMutex.Lock()
	defer o.addressesMutex.Unlock()
	o.addresses[key] = struct{}{}
}

func (o *Observer) removeAddress(key WalletAddress) {
	o.addressesMutex.Lock()
	defer o.addressesMutex.Unlock()
	delete(o.addresses, key)
}

// rawAddrFromAccount formats workchain 0 account ID as raw address string.
func rawAddrFromAccount(account []byte) string {
	return fmt.Sprintf("0:%x", account)
}
