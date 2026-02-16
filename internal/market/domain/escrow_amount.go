package domain

import "math"

const nanotonPerTON int64 = 1e9

// EscrowConfig is used to compute escrow_amount from price.
type EscrowConfig struct {
	TransactionGasTON float64 // e.g. 0.1 TON
	CommissionPercent float64 // e.g. 2 for 2%
}

// ComputeEscrowAmount returns escrow amount in nanotons.
// price is in TON (e.g. 10 for 10 TON); it is converted to nanotons (×1e9) then gas and commission are added.
func ComputeEscrowAmount(priceTON int64, cfg EscrowConfig) int64 {
	priceNanoton := priceTON * nanotonPerTON
	gasNanoton := int64(cfg.TransactionGasTON * float64(nanotonPerTON))
	commission := int64(math.Round(float64(priceNanoton) * cfg.CommissionPercent / 100))
	return priceNanoton + gasNanoton + commission
}
