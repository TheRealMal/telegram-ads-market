package blockchain_observer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"ads-mrkt/internal/liteclient"

	"github.com/redis/go-redis/v9"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
)

const (
	redisChannelExpired   = "__keyevent@%d__:expired"
	redisChannelNewExpire = "__keyevent@%d__:expire"

	lookupBlockTimeout = 400 * time.Millisecond
	lookupBlockDelay   = 400 * time.Millisecond

	handleMasterShardsAttemptsLimit    = 3
	serializeParentBlocksAttemptsLimit = 3
	shardsHandlerLimit                 = 10
)

func (o *Observer) startRedisEventsHandler(ctx context.Context) {
	expiredCh := fmt.Sprintf(redisChannelExpired, o.dbIndex)
	expireCh := fmt.Sprintf(redisChannelNewExpire, o.dbIndex)
	pubsub := o.rdb.PSubscribe(ctx, expiredCh, expireCh)
	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			o.log.Info("redis events handler stopped")
			return
		case msg := <-pubsub.Channel():
			if msg == nil {
				return
			}
			switch msg.Channel {
			case expiredCh:
				addr, err := address.ParseRawAddr(msg.Payload)
				if err != nil {
					o.log.Error("expired key not a valid address", "key", msg.Payload, "error", err)
					continue
				}
				o.removeAddress(WalletAddress(addr.Data()))
				if err := o.dealExpirer.SetDealStatusExpiredByEscrowAddress(ctx, msg.Payload); err != nil {
					o.log.Error("set deal expired", "address", msg.Payload, "error", err)
				} else {
					o.log.Info("deal marked expired", "address", msg.Payload)
				}
			case expireCh:
				addr, err := address.ParseRawAddr(msg.Payload)
				if err != nil {
					o.log.Debug("expire key not a valid address", "key", msg.Payload)
					continue
				}
				o.addAddress(WalletAddress(addr.Data()))
			default:
				o.log.Debug("redis keyevent", "channel", msg.Channel, "payload", msg.Payload)
			}
		}
	}
}

func (o *Observer) startDepositNotifier(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			o.log.Info("deposit notifier stopped")
			return
		case ev := <-o.depositEvents:
			if ev == nil {
				return
			}
			err := o.rdb.XAdd(ctx, &redis.XAddArgs{
				Stream: streamEscrowDeposit,
				Values: map[string]interface{}{
					"address":   ev.rawAddress,
					"amount":   ev.amount,
					"timestamp": ev.timestamp,
					"tx_hash":  ev.txHash,
				},
			}).Err()
			if err != nil {
				o.log.Error("xadd escrow deposit", "address", ev.rawAddress, "error", err)
			}
		}
	}
}

func (o *Observer) startMasterObserver(ctx context.Context) {
	currentMaster, err := o.lt.GetMasterchainInfo(ctx, lookupBlockTimeout)
	if err != nil {
		o.log.Error("get masterchain info", "error", err)
		return
	}
	lastSeqNo := currentMaster.SeqNo

	for {
		select {
		case <-ctx.Done():
			close(o.masterBlocks)
			o.log.Info("master observer stopped")
			return
		default:
			time.Sleep(lookupBlockDelay)
			currentMaster, err := o.lt.GetMasterchainInfo(ctx, lookupBlockTimeout)
			if err != nil {
				if !errors.Is(err, context.DeadlineExceeded) {
					o.log.Error("get masterchain info", "error", err)
				}
				continue
			}
			if currentMaster.SeqNo > lastSeqNo {
				lastSeqNo = currentMaster.SeqNo
				o.masterBlocks <- currentMaster
			}
		}
	}
}

func (o *Observer) startMastersHandler(ctx context.Context) {
	for master := range o.masterBlocks {
		o.handleNewMaster(ctx, master)
	}
	close(o.shardBlocks)
	o.log.Info("masters handler stopped")
}

func (o *Observer) handleNewMaster(ctx context.Context, master *ton.BlockIDExt) {
	var err error
	for attempts := 0; attempts < handleMasterShardsAttemptsLimit; attempts++ {
		shards, err := o.lt.GetBlockShardsInfo(ctx, master)
		if err != nil {
			if liteclient.IsNotReadyError(err) {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			o.log.Error("get block shards", "error", err, "seqno", master.SeqNo)
			attempts++
			continue
		}
		if len(o.workchain.Shards) == 0 {
			for _, shard := range shards {
				o.workchain.Shards[shard.Shard] = shard.SeqNo
				o.shardBlocks <- shard
			}
			return
		}
		newShards := make(map[int64]uint32, len(shards))
		for _, shard := range shards {
			stack := make(shardStack, 0)
			if err := o.handleShardBlock(ctx, shard, &stack); err != nil {
				o.log.Error("handle shard block", "error", err, "seqno", shard.SeqNo)
				continue
			}
			for top := stack.Pop(); top != nil; top = stack.Pop() {
				o.shardBlocks <- top
			}
			newShards[shard.Shard] = shard.SeqNo
		}
		o.workchain.Shards = newShards
		return
	}
	o.log.Error("handle new master failed", "error", err, "seqno", master.SeqNo)
}

func (o *Observer) handleShardBlock(ctx context.Context, shard *ton.BlockIDExt, stack *shardStack) error {
	oldSeq, ok := o.workchain.Shards[shard.Shard]
	if ok && oldSeq >= shard.SeqNo {
		return nil
	}
	stack.Push(shard)
	for attempts := 0; attempts < serializeParentBlocksAttemptsLimit; attempts++ {
		block, err := o.lt.GetBlockData(ctx, shard)
		if err != nil {
			if liteclient.IsNotReadyError(err) {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			attempts++
			continue
		}
		parents, err := ton.GetParentBlocks(&block.BlockInfo)
		if err != nil {
			attempts++
			continue
		}
		for _, parent := range parents {
			_ = o.handleShardBlock(ctx, parent, stack)
		}
		return nil
	}
	return fmt.Errorf("serialize parent blocks")
}

func (o *Observer) startShardsHandler(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < shardsHandlerLimit; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			o.shardHandleWorker(ctx, idx)
		}(i)
	}
	wg.Wait()
	close(o.depositEvents)
	o.log.Info("shards handler stopped")
}

func (o *Observer) shardHandleWorker(ctx context.Context, idx int) {
	for shardBlock := range o.shardBlocks {
		txIDs, err := o.lt.GetTransactionIDsFromBlock(ctx, shardBlock)
		if err != nil {
			o.log.Error("get block transactions", "error", err, "seqno", shardBlock.SeqNo)
			continue
		}
		for _, txInfo := range txIDs {
			if !o.isAddressWatched(WalletAddress(txInfo.Account)) {
				continue
			}
			addr := address.NewAddress(0, 0, txInfo.Account)
			tx, err := o.lt.GetTransaction(ctx, shardBlock, addr, txInfo.LT)
			if err != nil {
				o.log.Debug("get transaction failed", "addr", rawAddrFromAccount(txInfo.Account), "error", err)
				continue
			}
			amount, ts, hash := extractIncomingAmountAndTime(tx)
			if amount < 0 {
				continue
			}
			select {
			case o.depositEvents <- &depositEvent{
				rawAddress: rawAddrFromAccount(txInfo.Account),
				amount:    amount,
				timestamp: ts,
				txHash:    hash,
			}:
			default:
				o.log.Warn("deposit events channel full, drop")
			}
		}
	}
	o.log.Info("shard worker stopped", "idx", idx)
}

// extractIncomingAmountAndTime returns amount in nanoton, timestamp (unix), and tx hash hex. Returns -1 for amount if not a valid incoming transfer.
func extractIncomingAmountAndTime(tx *tlb.Transaction) (amount int64, timestamp int64, txHash string) {
	if tx.IO.In == nil || tx.IO.In.Msg == nil {
		return -1, 0, ""
	}
	internal, ok := tx.IO.In.Msg.(*tlb.InternalMessage)
	if !ok {
		return -1, 0, ""
	}
	amount = internal.Amount.Nano().Int64()
	timestamp = int64(tx.Now)
	txHash = hex.EncodeToString(tx.Hash)
	return amount, timestamp, txHash
}
