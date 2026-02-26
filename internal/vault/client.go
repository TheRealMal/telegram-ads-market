package vault

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	vaultclient "github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"

	"ads-mrkt/internal/vault/config"
)

const escrowSecretKey = "seed_phrase"

type Client struct {
	client    *vaultclient.Client
	mountPath string
	token     string
}

func NewClient(cfg config.Config) (*Client, error) {
	client, err := vaultclient.New(
		vaultclient.WithAddress(cfg.Address),
		vaultclient.WithRequestTimeout(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("vault new client: %w", err)
	}

	slog.Info("vault client initialized", "address", cfg.Address, "token", cfg.Token[:4]+"..."+cfg.Token[len(cfg.Token)-4:], "mount_path", cfg.MountPath)

	return &Client{
		client:    client,
		mountPath: cfg.MountPath,
		token:     cfg.Token,
	}, nil
}

func (c *Client) escrowPath(dealID int64) string {
	return "escrow/deal_" + strconv.FormatInt(dealID, 10)
}

func (c *Client) PutEscrowSeed(ctx context.Context, dealID int64, seedPhrase string) error {
	path := c.escrowPath(dealID)
	_, err := c.client.Secrets.KvV2Write(
		ctx,
		path,
		schema.KvV2WriteRequest{
			Data: map[string]any{
				escrowSecretKey: seedPhrase,
			},
		},
		vaultclient.WithMountPath(c.mountPath),
		vaultclient.WithToken(c.token),
	)
	if err != nil {
		return fmt.Errorf("vault write escrow seed for deal %d path %s: %w", dealID, path, err)
	}
	slog.Debug("vault: stored escrow seed", "deal_id", dealID)
	return nil
}

func (c *Client) GetEscrowSeed(ctx context.Context, dealID int64) (string, error) {
	path := c.escrowPath(dealID)
	resp, err := c.client.Secrets.KvV2Read(
		ctx, path,
		vaultclient.WithMountPath(c.mountPath),
		vaultclient.WithToken(c.token),
	)
	if err != nil {
		return "", fmt.Errorf("vault read escrow seed for deal %d path %s: %w", dealID, path, err)
	}
	if resp == nil || resp.Data.Data == nil {
		return "", fmt.Errorf("vault: no secret at path for deal %d: %w", dealID, ErrSecretNotFound)
	}

	raw, ok := resp.Data.Data[escrowSecretKey].(string)
	if !ok || raw == "" {
		return "", fmt.Errorf("vault: missing or invalid %q for deal %d", escrowSecretKey, dealID)
	}
	return raw, nil
}

var ErrSecretNotFound = errors.New("secret not found")
