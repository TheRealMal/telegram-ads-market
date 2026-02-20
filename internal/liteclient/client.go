package liteclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ads-mrkt/internal/liteclient/config"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
)

const (
	// Timeouts for fetching TON global config (avoids TLS handshake / context deadline errors on slow networks)
	globalConfigTLSHandshakeTimeout = 45 * time.Second
	globalConfigClientTimeout       = 90 * time.Second
	globalConfigRetries             = 3
	globalConfigRetryBackoff        = 5 * time.Second
)

const (
	ErrBlockNotApplied = "block is not applied"
	ErrBlockNotInDB    = "code 651"

	GetShardsTXsLimit = 5
)

type client struct {
	api ton.APIClientWrapped
}

func NewClient(ctx context.Context, cfg config.Config, isTestnet bool, public bool) (*client, error) {
	pool := liteclient.NewConnectionPool()
	var globalConfig *liteclient.GlobalConfig
	var err error
	if cfg.GlobalConfigDir != "" {
		filename := config.GlobalConfigFilename[isTestnet]
		path := filepath.Join(cfg.GlobalConfigDir, filename)
		slog.Info("loading TON global config from file")
		globalConfig, err = loadGlobalConfigFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load global config from file %s: %w", path, err)
		}
		slog.Info("loaded TON global config from file", "path", path)
	} else {
		globalConfig, err = fetchGlobalConfig(ctx, config.GlobalConfigURL[isTestnet])
		if err != nil {
			return nil, fmt.Errorf("failed to get global config: %w", err)
		}
	}
	if !public {
		if err = pool.AddConnection(ctx, cfg.LiteserverHost, cfg.LiteserverKey); err != nil {
			return nil, fmt.Errorf("failed to add connection: %w", err)
		}

	} else {
		err = pool.AddConnectionsFromConfig(ctx, globalConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to add connections from config: %w", err)
		}
	}

	api := ton.NewAPIClient(pool, ton.ProofCheckPolicyFast)
	api.SetTrustedBlockFromConfig(globalConfig)

	slog.Info("fetching and checking proofs since config init block ...")
	_, err = api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current masterchain info: %w", err)
	}
	return &client{
		api: api,
	}, nil
}

// loadGlobalConfigFromFile reads TON global config from a JSON file (e.g. downloaded at build time).
func loadGlobalConfigFromFile(path string) (*liteclient.GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg liteclient.GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// fetchGlobalConfig fetches TON global config with long timeouts and retries to avoid TLS handshake
// and context deadline errors on slow or restricted networks (e.g. in Docker).
func fetchGlobalConfig(ctx context.Context, url string) (*liteclient.GlobalConfig, error) {
	transport := &http.Transport{
		TLSHandshakeTimeout:   globalConfigTLSHandshakeTimeout,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   globalConfigClientTimeout,
	}
	var lastErr error
	for attempt := 0; attempt < globalConfigRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(globalConfigRetryBackoff):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			slog.Debug("fetch global config attempt failed", "url", url, "attempt", attempt+1, "error", err)
			continue
		}
		var config liteclient.GlobalConfig
		if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		resp.Body.Close()
		return &config, nil
	}
	return nil, lastErr
}

func (c *client) Client() ton.APIClientWrapped {
	return c.api
}

func (c *client) GetTransactionIDsFromBlock(ctx context.Context, blockID *ton.BlockIDExt) ([]ton.TransactionShortInfo, error) {
	var (
		txIDList []ton.TransactionShortInfo
		after    *ton.TransactionID3
		next     = true
		attempts = 0
	)
	for next {
		fetchedIDs, more, err := c.GetBlockTransactionsV2(ctx, blockID, 256, after)
		if err != nil {
			if IsNotReadyError(err) {
				time.Sleep(time.Millisecond * 100)
				continue
			}

			attempts += 1
			if attempts == GetShardsTXsLimit {
				return nil, err // Retries limit exceeded for batch
			}

			logAfter := uint64(0)
			if after != nil {
				logAfter = after.LT
			}
			slog.Error("failed to get block transactions batch", "workchain", blockID.Workchain, "shard", blockID.Shard, "seqno", blockID.SeqNo, "logAfter", logAfter, "error", err)
			continue
		}
		txIDList = append(txIDList, fetchedIDs...)
		next = more
		if more {
			after = fetchedIDs[len(fetchedIDs)-1].ID3()
		}
		attempts = 0 // Refresh attempts for next batch
	}
	sort.Slice(txIDList, func(i, j int) bool {
		return txIDList[i].LT < txIDList[j].LT
	})
	return txIDList, nil
}

func (c *client) GetBlockTransactionsV2(ctx context.Context, block *ton.BlockIDExt, count uint32, after ...*ton.TransactionID3) ([]ton.TransactionShortInfo, bool, error) {
	return c.api.WithRetry().GetBlockTransactionsV2(ctx, block, count, after...)
}

func (c *client) GetMasterchainInfo(ctx context.Context, timeout time.Duration) (*ton.BlockIDExt, error) {
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	return c.api.WithTimeout(timeout).WithRetry().CurrentMasterchainInfo(ctx)
}

func (c *client) GetBlockShardsInfo(ctx context.Context, master *ton.BlockIDExt) ([]*ton.BlockIDExt, error) {
	return c.api.WithRetry().GetBlockShardsInfo(ctx, master)
}

func (c *client) GetBlockData(ctx context.Context, block *ton.BlockIDExt) (*tlb.Block, error) {
	return c.api.WithRetry().GetBlockData(ctx, block)
}

func (c *client) GetTransaction(ctx context.Context, block *ton.BlockIDExt, addr *address.Address, lt uint64) (*tlb.Transaction, error) {
	return c.api.WithRetry().GetTransaction(ctx, block, addr, lt)
}

func (c *client) LookupBlock(ctx context.Context, timeout time.Duration, workchain int32, shard int64, seqno uint32) (*ton.BlockIDExt, error) {
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	return c.api.WithTimeout(timeout).WithRetry().LookupBlock(ctx, workchain, shard, seqno)
}

func IsNotReadyError(err error) bool {
	return strings.Contains(err.Error(), ErrBlockNotApplied) || strings.Contains(err.Error(), ErrBlockNotInDB)
}

// HasOutgoingTxTo returns true if the account at fromAddrRaw has an outgoing internal transaction
// with the given amount (nanoton) to the given destination address.
// Used e.g. to recover when a previous run transferred but crashed before updating status.
func (c *client) HasOutgoingTxTo(ctx context.Context, fromAddr *address.Address, amountNanoton int64, toAddr *address.Address) (bool, error) {
	block, err := c.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return false, err
	}
	account, err := c.api.GetAccount(ctx, block, fromAddr)
	if err != nil {
		return false, err
	}
	if account == nil || account.LastTxLT == 0 {
		return false, nil
	}
	txs, err := c.api.ListTransactions(ctx, fromAddr, 20, account.LastTxLT, account.LastTxHash)
	if err != nil {
		return false, err
	}
	if len(txs) == 0 {
		return false, nil
	}
	want := big.NewInt(amountNanoton)
	for _, tx := range txs {
		if tx.IO.Out == nil {
			continue
		}
		msgs, err := tx.IO.Out.ToSlice()
		if err != nil {
			continue
		}
		for _, msg := range msgs {
			internal := msg.AsInternal()
			if internal == nil {
				continue
			}
			dst := internal.DestAddr()
			if dst == nil || !toAddr.Equals(dst) {
				continue
			}
			if internal.Amount.Nano().Cmp(want) == 0 {
				return true, nil
			}
		}
	}
	return false, nil
}
