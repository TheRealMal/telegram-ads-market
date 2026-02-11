package domain

import "math"

// EscrowConfig is used to compute escrow_amount from price.
type EscrowConfig struct {
	TransactionGasTON float64 // e.g. 0.1 TON
	CommissionPercent float64 // e.g. 2 for 2%
}

// ComputeEscrowAmount returns price + (transactionGasTON * 1e9 nanoton) + (price * commissionPercent / 100).
// Price and result are in nanoton.
func ComputeEscrowAmount(price int64, cfg EscrowConfig) int64 {
	gasNanoton := int64(cfg.TransactionGasTON * 1e9)
	commission := int64(math.Round(float64(price) * cfg.CommissionPercent / 100))
	return price + gasNanoton + commission
}
